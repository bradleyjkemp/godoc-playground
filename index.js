"use strict";

window.triggerRender = () => {
    const code = editor.session.getValue();
    window.localStorage.setItem('input.go', code);
    // trigger event on preview pane which wasm has an event handler for
    document.getElementById("previewPane").dispatchEvent(new CustomEvent('updatePreview', {detail: code}));
    console.log("sent event")
};

// if no saved code then initialise with default
if (window.localStorage.getItem('input.go') == null) {
    window.localStorage.setItem('input.go', `// Paste your go code here
package mypackage`);
}

const go_syntax = {
    comment: [{
        pattern: /(^|[^\\])\/\*[\s\S]*?(?:\*\/|$)/,
        lookbehind: true
    },
        {
            pattern: /(^|[^\\:])\/\/.*/,
            lookbehind: true,
            greedy: true
        }
    ],
    string: {
        pattern: /(["'`])(\\[\s\S]|(?!\1)[^\\])*\1/,
        greedy: true
    },
    keyword: /\b(?:break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go(?:to)?|if|import|interface|map|package|range|return|select|struct|switch|type|var)\b/,
    boolean: /\b(?:_|iota|nil|true|false)\b/,
    function: /[a-z0-9_]+(?=\()/i,
    number: /(?:\b0x[a-f\d]+|(?:\b\d+\.?\d*|\B\.\d+)(?:e[-+]?\d+)?)i?/i,
    operator: /[*\/%^!=]=?|\+[=+]?|-[=-]?|\|[=|]?|&(?:=|&|\^=?)?|>(?:>=?|=)?|<(?:<=?|=|-)?|:=|\.\.\./,
    punctuation: /[{}[\];(),.:]/,
    builtin: /\b(?:bool|byte|complex(?:64|128)|error|float(?:32|64)|rune|string|u?int(?:8|16|32|64)?|uintptr|append|cap|close|complex|copy|delete|imag|len|make|new|panic|print(?:ln)?|real|recover)\b/
};

let editor;

window.onload = async function() {
    editor = ace.edit("code-editor");
    editor.setTheme("ace/theme/monokai");
    editor.session.setMode("ace/mode/golang");
    editor.session.setValue(window.localStorage.getItem('input.go'));

    Split(['#codePane', '#previewPane'], {
        direction: 'horizontal'
    });

    const go = new Go();
    const response = await fetch("main.wasm");
    const buffer = await response.arrayBuffer();
    WebAssembly.instantiate(buffer, go.importObject).then((result) => {
        console.log(result);
        return go.run(result.instance)
    });

    let typingTimer;                //timer identifier
    let doneTypingInterval = 1000;  //pause length (in ms) after which preview is updated
    editor.on('change', () => {
        clearTimeout(typingTimer);
        typingTimer = setTimeout(window.triggerRender, doneTypingInterval);
    });
};
