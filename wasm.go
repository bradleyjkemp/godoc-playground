// +build js,wasm

package main

import (
	"github.com/bradleyjkemp/godoc-playground/preview"
	"regexp"
	"strings"
	"syscall/js"
)

var sourcePane js.Value
var previewPane js.Value

type ToastifyOptions struct {
	text     string
	duration int
}

var updatePreview = js.NewCallback(func(args []js.Value) {
	source := args[0].Get("detail").String()
	page, err := preview.GetPageForFile(source)
	if err != nil {
		js.Global().Call("showErrorToast", err.Error())
		return
	}

	js.Global().Call("updatePreview", sanitize(page))
})

var nonAnchorHref = regexp.MustCompile(`<a href="[^#].*?"`)

func sanitize(page string) string {
	// Rewrite static assets to point to local copies
	page = strings.Replace(page, "/lib/godoc", "./ext", -1)

	// Remove href's which will break the iframe if clicked on
	// This is any href which isn't an anchor
	page = nonAnchorHref.ReplaceAllString(page, `$0 style="pointer-events:none"`)
	return page
}

func main() {
	sourcePane = js.Global().Get("document").Call("getElementById", "codeInput")
	previewPane = js.Global().Get("document").Call("getElementById", "previewPane")
	previewPane.Call("addEventListener", "updatePreview", updatePreview)

	// Now that handler is registered, trigger a render to display initial content
	js.Global().Call("triggerRender")

	// keep program alive to process callbacks
	<-make(chan struct{})
}
