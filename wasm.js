/* global Go, fetch, WebAssembly */
'use strict'

const go = new Go()
const mainWasm = fetch('main.wasm')

WebAssembly.instantiateStreaming(mainWasm, go.importObject).then((result) => {
  return go.run(result.instance)
})
