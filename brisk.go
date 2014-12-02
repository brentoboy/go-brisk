package brisk

import (
	"encoding/json"
	"fmt"
	"github.com/brentoboy/go-jsph"
	"github.com/oleiade/reflections"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Site struct {
	Wireframes map[string]func(interface{}) string
	Widgets    map[string]*WidgetFactory
	Pages      map[string]*HtmlPage
	Routes     map[string]*Route
}

type Response struct {
	Body string
}

type Responder interface {
	BuildResponse() *Response
}

type Route struct {
	Regexp      *regexp.Regexp
	Action      string
	HandlerFunc http.HandlerFunc
}

type WidgetFactory struct {
	NewParamsObject func() interface{}
	Prepare         func(interface{}) interface{}
	Render          func(interface{}) string
}

type HtmlPage struct {
	Title     string
	Wireframe string
	Base      string
	BodyId    string
	Top       []string
	Left      []string
	Center    []string
	Right     []string
	Bottom    []string
}

func (site *Site) LoadFromBaseFolder(baseFolder string) {
	site.Wireframes = makeWireframeList(path.Join(baseFolder, "wireframes"))
	site.Widgets = makeWidgetList(path.Join(baseFolder, "widgets"))
	site.Pages = makeActionList(path.Join(baseFolder, "actions"))
}

func (this *Site) renderWidget(widgetName string, pageParams map[string]string) string {
	widget, ok := this.Widgets[widgetName]
	if !ok {
		return "Missing Widget: " + widgetName
	}
	var widgetParams interface{} = nil
	if widget.NewParamsObject != nil {
		widgetParams = widget.NewParamsObject()
		AssignValues(pageParams, widgetParams)
	}
	if widget.Prepare != nil {
		widgetParams = widget.Prepare(widgetParams)
	}

	if widget.Render == nil {
		return ""
	}
	return widget.Render(widgetParams)
}

func (this *Site) buildPage(page *HtmlPage, pageParams map[string]string) (*HtmlPage, interface{}) {
	inheritenceChain := []*HtmlPage{page}

	for chainLink := this.Pages[page.Base]; chainLink != nil; chainLink = this.Pages[chainLink.Base] {
		inheritenceChain = append([]*HtmlPage{chainLink}, inheritenceChain...)
	}

	pageJson := &HtmlPage{}
	for _, chainLink := range inheritenceChain {
		if chainLink.Title != "" {
			pageJson.Title = chainLink.Title
		}
		if chainLink.BodyId != "" {
			pageJson.BodyId = chainLink.BodyId
		}
		if chainLink.Wireframe != "" {
			pageJson.Wireframe = chainLink.Wireframe
		}
		for _, x := range chainLink.Top {
			pageJson.Top = append(pageJson.Top, this.renderWidget(x, pageParams))
		}
		for _, x := range chainLink.Left {
			pageJson.Left = append(pageJson.Left, this.renderWidget(x, pageParams))
		}
		for _, x := range chainLink.Center {
			pageJson.Center = append(pageJson.Center, this.renderWidget(x, pageParams))
		}
		for _, x := range chainLink.Right {
			pageJson.Right = append(pageJson.Right, this.renderWidget(x, pageParams))
		}
		for _, x := range chainLink.Bottom {
			pageJson.Bottom = append(pageJson.Bottom, this.renderWidget(x, pageParams))
		}
	}

	return pageJson, nil
}

func (this *Site) writeTo(page *HtmlPage, w http.ResponseWriter) {
	fmt.Fprintf(w, this.Wireframes[page.Wireframe](page))
}

func AssignValues(vals map[string]string, obj interface{}) {
	for k, v := range vals {
		k = strings.Title(k)
		kind, _ := reflections.GetFieldKind(obj, k)

		switch kind {
		case reflect.String:
			reflections.SetField(obj, k, v)
		case reflect.Int:
			val, err := strconv.Atoi(v)
			if err == nil {
				reflections.SetField(obj, k, val)
			}
		}
	}
}

func NewRegexpRoute(regexString string) *Route {
	regex, err := regexp.Compile(regexString)
	if err != nil {
		fmt.Printf("Regexp doesnt compile: `%s` \n", regexString)
		return nil
	}
	return &Route{Regexp: regex}
}

func NewRegexpRouteToAction(regexString string, actionName string) *Route {
	route := NewRegexpRoute(regexString)
	route.Action = actionName
	return route
}

func NewStaticRouteToAction(exactPath string, actionName string) *Route {
	regexString := `^\Q` + exactPath + `\E$`
	return NewRegexpRouteToAction(regexString, actionName)
}

func NewStaticRouteToFile(exactPath string, localFilePath string) *Route {
	regexString := `^\Q` + exactPath + `\E$`
	route := NewRegexpRoute(regexString)
	route.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, localFilePath)
	}
	return route
}

func NewRouteToMediaFolder(mediaFolder string, localMediaFolder string) *Route {
	regexString := `^\Q` + mediaFolder + `\E`
	route := NewRegexpRoute(regexString)
	handler := http.StripPrefix(mediaFolder, http.FileServer(http.Dir(localMediaFolder)))
	route.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}
	return route
}

func (site *Site) RunHttpServer(port string) {
	fmt.Println("HTTP Server, Listening on " + port)
	http.ListenAndServe(port, site)
}

func (site *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//context := site.NewContext(...?)
	//context.Site = site
	//context.Route := context.Site.chooseRoute(context)
	//context.Action := context.Route.chooseAction(context)
	//context.Ux := context.Action.chooseUx(context)
	//context.Response := context.Action.buildResponse(context)
	//sendResponse(context, w, r)
	//context.Site.Log(context)

	requestPath := r.URL.Path
	urlParams, _ := url.ParseQuery(r.URL.RawQuery)
	params := make(map[string]string)
	for k, v := range urlParams {
		params[k] = v[0]
	}

	for routeName, route := range site.Routes {
		if route.Regexp.MatchString(requestPath) {
			prettyParams := route.Regexp.SubexpNames()
			if len(prettyParams) > 0 {
				submatches := route.Regexp.FindStringSubmatch(requestPath)
				for index, name := range prettyParams {
					params[name] = submatches[index]
				}
			}

			if route.HandlerFunc != nil {
				fmt.Printf("found route: %s, for: %s\n", routeName, r.URL)
				route.HandlerFunc(w, r)
				return
			} else {
				fmt.Printf("found route: %s, action: %s \n", routeName, route.Action)
				page := site.Pages[route.Action]
				if page == nil {
					fmt.Printf("page info not found: %s\n", route.Action)
					http.NotFound(w, r)
					return
				}

				builtPage, err := site.buildPage(page, params)
				if builtPage == nil {
					fmt.Printf("buildPage failed: %s\n", err)
					http.NotFound(w, r) // how about an error 500 instead?
					return
				}

				wireframe := site.Wireframes[builtPage.Wireframe]
				if wireframe == nil {
					fmt.Printf("wireframe not found: %s\n", builtPage.Wireframe)
					return
				}

				fmt.Fprintf(w, wireframe(builtPage))
				return
			}
		}
	}
	http.NotFound(w, r)
}

func readDirTree(baseFolder string) []string {
	folders := []string{""}
	files := []string{}

	for i := 0; i < len(folders); i++ {
		folder := folders[i]
		dir, err := ioutil.ReadDir(path.Join(baseFolder, folder))
		if err != nil {
			fmt.Println(err)
		}
		for _, file := range dir {
			filePath := path.Join(folder, file.Name())
			if file.IsDir() {
				folders = append(folders, filePath)
			} else {
				files = append(files, filePath)
			}
		}
	}

	return files
}

func makeWidgetList(baseFolder string) map[string]*WidgetFactory {
	allFiles := readDirTree(baseFolder)
	widgets := make(map[string]*WidgetFactory)

	for _, file := range allFiles {
		folder := path.Dir(file)
		fileName := path.Base(file)
		widget := widgets[folder]
		if widget == nil {
			widget = &WidgetFactory{}
			widgets[folder] = widget
		}
		ext := path.Ext(fileName)
		switch ext {
		case ".html":
			widget.Render = jsph.CompileJsphFile(path.Join(baseFolder, file))
		}
	}

	return widgets
}

func makeWireframeList(baseFolder string) map[string]func(interface{}) string {
	allFiles := readDirTree(baseFolder)
	wireframes := make(map[string]func(interface{}) string)

	for _, file := range allFiles {
		folder := path.Dir(file)
		fileName := path.Base(file)
		ext := path.Ext(fileName)
		switch ext {
		case ".html":
			wireframes[folder] = jsph.CompileJsphFile(path.Join(baseFolder, file))
		}
	}

	return wireframes
}

func makeActionList(baseFolder string) map[string]*HtmlPage {
	allFiles := readDirTree(baseFolder)
	actions := make(map[string]*HtmlPage)

	for _, file := range allFiles {
		folder := path.Dir(file)
		fileName := path.Base(file)
		ext := path.Ext(fileName)
		switch ext {
		case ".json":
			action := &HtmlPage{}
			fileContents, err := ioutil.ReadFile(path.Join(baseFolder, file))
			if err != nil {
				fmt.Printf("readfile(%s): %s\n", fileName, err)
			} else {
				err := json.Unmarshal(fileContents, action)
				if err != nil {
					fmt.Printf("unmarshal(%s): %s\n", fileName, err)
				}
			}
			actions[folder] = action
		}
	}

	return actions
}

func getFileContentsAsString(path string) (string, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
