package brisk

import (
	//"fmt"
	//"path"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_LoadFromBaseFolder(t *testing.T) {
	site := Site{}
	site.LoadFromBaseFolder("C:/code/go/src/bflarsen.org")

	assert.NotNil(t, site.Widgets["banner"])
	assert.NotNil(t, site.Widgets["biography"])
	assert.NotNil(t, site.Widgets["fineprint"])
	assert.NotNil(t, site.Widgets["pageHeader"])
	assert.NotNil(t, site.Widgets["resources"])
	assert.NotNil(t, site.Widgets["searchPaintings"])
	assert.NotNil(t, site.Widgets["terms"])
	assert.NotNil(t, site.Widgets["topMenu"])

	assert.NotNil(t, site.Wireframes["default"])

	assert.NotNil(t, site.Pages["contactUsPage"])
	assert.NotNil(t, site.Pages["defaultLayout"])
	assert.NotNil(t, site.Pages["homePage"])
	assert.NotNil(t, site.Pages["paintingPage"])
	assert.NotNil(t, site.Pages["resourcesPage"])
	assert.NotNil(t, site.Pages["searchPage"])
	assert.NotNil(t, site.Pages["termsPage"])
}
