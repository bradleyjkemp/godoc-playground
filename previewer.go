package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"golang.org/x/tools/godoc"
	"golang.org/x/tools/godoc/static"
	"log"
	"net/http/httptest"
	"strings"
	"text/template"
)

// Copied from godoc package:
// by the last path component of the provided package path
// (as is the convention for packages). This is sufficient
// to resolve package identifiers without doing an actual
// import. It never returns an error.
//
func poorMansImporter(imports map[string]*ast.Object, path string) (*ast.Object, error) {
	fmt.Println(imports, path)
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

// get any function receivers which are of undeclared type
// these stop the function appearing in the preview
// unless we fake them
func getUnresolvedReceiverTypes(fileDecls []ast.Decl, unresolved []*ast.Ident) []string {
	unresolvedIdents := map[string]bool{}
	for _, ident := range unresolved {
		unresolvedIdents[ident.Name] = true
	}

	var unresolvedReceiverTypes []string
	for _, decl := range fileDecls {
		funcNode, ok := decl.(*ast.FuncDecl)
		if ok && funcNode.Recv != nil {
			for _, recv := range funcNode.Recv.List {
				var name string
				recvType := recv.Type
				switch recvType.(type) {
				case *ast.StarExpr:
					name = recvType.(*ast.StarExpr).X.(*ast.Ident).Name
				case *ast.Ident:
					name = recvType.(*ast.Ident).Name
				}

				if unresolvedIdents[name] {
					unresolvedReceiverTypes = append(unresolvedReceiverTypes, name)
				}
			}
		}
	}

	return unresolvedReceiverTypes
}

func generateFakeTypesFile(unresolved []string, packageName string) string {
	file := &bytes.Buffer{}
	file.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	for _, ident := range unresolved {
		file.WriteString("// Undeclared type, presumably this is declared in another file\n")
		file.WriteString(fmt.Sprintf("type %s = undeclaredType\n", ident))
	}
	return file.String()
}

func getPageForFile(fileContents string) string {
	info := &godoc.PageInfo{Dirname: "/", Mode: godoc.NoFiltering}

	info.FSet = token.NewFileSet()
	parsedFile, err := parser.ParseFile(info.FSet, "input.go", fileContents, parser.ParseComments)
	//spew.Dump(parsedFile)
	fmt.Println(err)

	packageName := parsedFile.Name.Name

	fakeTypes, err := parser.ParseFile(
		info.FSet,
		"types.go",
		generateFakeTypesFile(
			getUnresolvedReceiverTypes(parsedFile.Decls, parsedFile.Unresolved),
			packageName),
		parser.ParseComments)

	files := map[string]*ast.File{
		"input.go": parsedFile,
		"types.go": fakeTypes,
	}

	pkg, err := ast.NewPackage(info.FSet, files, poorMansImporter, nil)

	fmt.Println("ast.NewPackage err:", err)

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
