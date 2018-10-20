/* global CustomEvent, ace, Split */
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

// if no saved code then initialise with default
if (window.localStorage.getItem('input.go') == null) {
  window.localStorage.setItem('input.go', `// Paste your go code here
package mypackage`)
}

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
}
