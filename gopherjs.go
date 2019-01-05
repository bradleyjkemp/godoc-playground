package main

import (
	"github.com/bradleyjkemp/godoc-playground/preview"
	"github.com/gopherjs/gopherjs/js"
	"regexp"
	"strings"
)

var previewPane *js.Object

var updatePreview = func(source *js.Object) {
	page, err := preview.GetPageForFile(source.Get("detail").String())
	if err != nil {
		js.Global.Call("showErrorToast", err.Error())
		return
	}

	js.Global.Call("updatePreview", sanitize(page))
}

var nonAnchorHref = regexp.MustCompile(`<a href="[^#].*?"`)

func sanitize(page string) string {
	// Rewrite static assets to point to local copies
	page = strings.Replace(page, "/lib/godoc", "./ext", -1)

	// Remove href's which will break the iframe if clicked on
	// This is any href which isn't an anchor
	page = nonAnchorHref.ReplaceAllString(page, `$0 style="pointer-events:none"`)
	return page
}

func onload() {
	previewPane = js.Global.Get("document").Call("getElementById", "previewPane")
	previewPane.Call("addEventListener", "updatePreview", updatePreview)

	// Now that handler is registered, trigger a render to display initial content
	js.Global.Call("triggerRender")
}

func main() {
	js.Global.Set("onload", onload)
}
