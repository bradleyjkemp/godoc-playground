/* global CustomEvent, ace, Split, Toastify, Go, WebAssembly, fetch */
'use strict'

let editor

window.triggerRender = () => {
  if (editor === undefined) {
    // wasm was loaded and called this method before dom was ready
    // so schedule trigger again after a delay
    clearTimeout(this.timeout)
    this.timeout = setTimeout(window.triggerRender, 100)
    return
  }

  const code = editor.session.getValue()
  window.localStorage.setItem('input.go', code)
  // trigger event on preview pane which wasm has an event handler for
  document.getElementById('previewPane').dispatchEvent(new CustomEvent('updatePreview', { detail: code }))
}

window.updatePreview = (htmlContents) => {
  const iframeWindow = document.getElementById('previewPane').contentWindow
  const iframeDoc = iframeWindow.document
  const selectedHash = iframeWindow.location.hash
  iframeDoc.open('text/html', 'replace')
  iframeDoc.write(htmlContents)
  iframeDoc.close()

  if (iframeWindow.location.hash !== selectedHash) {
    iframeWindow.location.hash = selectedHash
  }
}

window.showErrorToast = (errorMessage) => {
  Toastify({
    text: errorMessage,
    gravity: 'bottom',
    backgroundColor: 'orangered',
    duration: 3000
  }).showToast()
}

// if no saved code then initialise with default
if (window.localStorage.getItem('input.go') == null) {
  window.localStorage.setItem('input.go', `// Write your go code in the editor on the left and watch it previewed here on the right.
//
// Features
//
// * Supports all the GoDoc syntax
//
// * That's because this is using the actual godoc renderer compiled to WebAssembly and running in your browser!
//
// * You don't even have to give a full working sample: unresolved symbols are automagically fixed so even just a small snippet will work fine.
package mypackage
`)
}

const go = new Go()
const mainWasm = fetch('main.wasm')
const instantiateWasm = WebAssembly.instantiateStreaming
  ? WebAssembly.instantiateStreaming(mainWasm, go.importObject)
  : mainWasm
    .then(response => response.arrayBuffer())
    .then(bytes => WebAssembly.instantiate(bytes, go.importObject))

window.onload = async function () {
  editor = ace.edit('code-editor')
  editor.setTheme('ace/theme/chrome')
  editor.session.setMode('ace/mode/golang')
  editor.session.setValue(window.localStorage.getItem('input.go'))

  Split(['#codePane', '#previewPane'], {
    direction: 'horizontal'
  })

  let typingTimer // timer identifier
  let doneTypingInterval = 1000 // pause length (in ms) after which preview is updated
  editor.on('change', () => {
    clearTimeout(typingTimer)
    typingTimer = setTimeout(window.triggerRender, doneTypingInterval)
  })

  instantiateWasm.then((result) => {
    return go.run(result.instance)
  })
}
