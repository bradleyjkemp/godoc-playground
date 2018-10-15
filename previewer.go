// +build js,wasm

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"golang.org/x/tools/godoc"
	"golang.org/x/tools/godoc/static"
	"io"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"syscall/js"
	"text/template"
)

// Copied from godoc package:
// by the last path component of the provided package path
// (as is the convention for packages). This is sufficient
// to resolve package identifiers without doing an actual
// import. It never returns an error.
//
func poorMansImporter(imports map[string]*ast.Object, path string) (*ast.Object, error) {
	pkg := imports[path]
	if pkg == nil {
		// note that strings.LastIndex returns -1 if there is no "/"
		pkg = ast.NewObj(ast.Pkg, path[strings.LastIndex(path, "/")+1:])
		pkg.Data = ast.NewScope(nil) // required by ast.NewPackage for dot-import
		imports[path] = pkg
	}
	return pkg, nil
}

// Copied from godoc package:
func applyTemplate(t *template.Template, name string, data interface{}) []byte {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		log.Printf("%s.Execute: %s", name, err)
	}
	return buf.Bytes()
}

var sourcePane js.Value
var previewPane js.Value

var updatePreview = js.NewCallback(func(_ []js.Value) {
	fmt.Println("updatePreview in wasm")
	source := sourcePane.Get("value").String()
	fmt.Println(source)
	previewPane.Call("setAttribute", "srcdoc", getPageForFile(source))
})

func main() {
	fmt.Println("hello webassembly!")
	sourcePane = js.Global().Get("document").Call("getElementById", "codeInput")
	previewPane = js.Global().Get("document").Call("getElementById", "preview")
	previewPane.Call("addEventListener", "updatePreview", updatePreview)

	// keep program alive to process callbacks
	<-make(chan struct{})
}

func getPageForFile(file string) string {
	info := &godoc.PageInfo{Dirname: "/", Mode: godoc.NoFiltering}
	ctxt := build.Default
	ctxt.IsAbsPath = func(path string) bool {
		fmt.Println("IsAbsPath", path)
		return false
	}
	ctxt.IsDir = func(path string) bool {
		fmt.Println("IsDir", path)
		if path == "/" {
			return true
		}
		return false
	}
	ctxt.ReadDir = func(dir string) ([]os.FileInfo, error) {
		fmt.Println("ReadDir", dir)
		if dir == "/" {
			return []os.FileInfo{FakeFile("input.go")}, nil
		}
		return nil, nil
	}
	ctxt.HasSubdir = func(root, dir string) (rel string, ok bool) {
		return "", false
	}
	ctxt.OpenFile = func(name string) (io.ReadCloser, error) {
		fmt.Println("OpenFile", name)
		if name == "/input.go" {
			b := bytes.NewBufferString(file)
			return ioutil.NopCloser(b), nil
		}
		return nil, nil
	}

	//pkgInfo, err := ctxt.ImportDir("/", 0)
	//spew.Dump(pkgInfo)
	//fmt.Println(err)

	info.FSet = token.NewFileSet()
	fileAST, err := parser.ParseFile(info.FSet, "input.go", file, parser.ParseComments)
	//spew.Dump(fileAST)
	//fmt.Println(err)

	files := map[string]*ast.File{
		"input.go": fileAST,
	}

	pkg, err := ast.NewPackage(info.FSet, files, poorMansImporter, nil)
	//spew.Dump(pkg, err)
	info.PDoc = doc.New(pkg, pkg.Name, 0)

	presentation := godoc.NewPresentation(&godoc.Corpus{})
	presentation.PackageHTML, err = template.New("package.html").Funcs(presentation.FuncMap()).Parse(string(static.Files["package.html"]))
	if err != nil {
		panic(err)
	}
	presentation.GodocHTML, err = template.New("godoc.html").Funcs(presentation.FuncMap()).Parse(string(static.Files["godoc.html"]))

	//spew.Dump(info)
	body := applyTemplate(presentation.PackageHTML, "packageHTML", info)

	resp := httptest.NewRecorder()
	presentation.ServePage(resp, godoc.Page{
		Title: "Package " + pkg.Name,
		//Tabtitle: "My tab tile",
		//Subtitle: "My subtitle",
		Body: body,
		//GoogleCN: info.GoogleCN,
	})

	return strings.Replace(resp.Body.String(), "/lib/godoc", "./ext", -1)
}
