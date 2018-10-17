// +build js,wasm

package main

import (
	"syscall/js"
)

var sourcePane js.Value
var previewPane js.Value

var updatePreview = js.NewCallback(func(args []js.Value) {
	source := args[0].Get("detail").String()
	previewPane.Call("setAttribute", "srcdoc", getPageForFile(source))
})

func main() {
	sourcePane = js.Global().Get("document").Call("getElementById", "codeInput")
	previewPane = js.Global().Get("document").Call("getElementById", "previewPane")
	previewPane.Call("addEventListener", "updatePreview", updatePreview)

	// Now that handler is registered, trigger a render to display initial content
	js.Global().Call("triggerRender")

	// keep program alive to process callbacks
	<-make(chan struct{})
}
