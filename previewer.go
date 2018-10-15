package main

import (
	"bytes"
	"fmt"
	"github.com/davecgh/go-spew/spew"
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
	"text/template"
)

const source = `package monkey

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/pkg/errors"
)

// Patch lets you overwrite unexported fields of structs even when:
// * the fields are deeply nested
// * the fields are of an unexported type
//
// This is done by creating a shadow struct of the same layout.
// Patch then scans through the struct to the patched and for each field:
// * If a field of the same name exists in the shadow then it will be patched.
// * If the matching field is of the same type then it is overwritten directly.
// * If the matching field is of a different type then the two types are patched recursively.
func Patch(actualI interface{}, shadowI interface{}) error {
	actual := reflect.ValueOf(actualI)
	shadow := reflect.ValueOf(shadowI)
	if !actual.CanAddr() || !shadow.CanAddr() {
		if actual.Kind() != reflect.Ptr && actual.Kind() != reflect.Interface {
			// unaddressable so can't change values
			return errors.New("cannot patch unaddressable value")
		}

		actual = actual.Elem()

		if shadow.Kind() != reflect.Ptr && shadow.Kind() != reflect.Interface {
			// unaddressable so can't change use to change values
			return errors.New("cannot use unaddressable shadow")
		}

		shadow = shadow.Elem()
	}

	return patch(actual, shadow)
}

func patch(actual reflect.Value, shadow reflect.Value) error {
	switch actual.Kind() {
	// Indirections
	case reflect.Interface:
		return patchInterface(actual, shadow)
	case reflect.Ptr:
		return patchPtr(actual, shadow)

	// Collections
	case reflect.Struct:
		return patchStruct(actual, shadow)
	case reflect.Slice, reflect.Array:
		return patchSlice(actual, shadow)
	//case reflect.Map:
	//	return m.mapMap(iVal, parentID, inlineable)
	//
	//// Simple types
	//case reflect.Bool:
	//	return patchPrimitive(actual, shadow)
	//case reflect.String:
	//	return m.mapString(iVal, inlineable)
	//case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
	//	return m.mapInt(iVal, inlineable)
	//case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
	//	return m.mapUint(iVal, inlineable)

	// Anything else should just be overwritable
	default:
		fmt.Println("patching primitive")
		return patchPrimitive(actual, shadow)
	}
}

func patchInterface(actual reflect.Value, shadow reflect.Value) error {
	if shadow.Type() == actual.Type() {
		// valid to just assign directly
		return unsafeSet(actual, shadow)
	}

	// assume we're meant to be patching the underlying type
	// TODO: add check here that actual.Elem is the same Kind as shadow
	return patch(actual.Elem(), shadow)
}

func patchStruct(actual reflect.Value, shadow reflect.Value) error {
	// assume this is a struct
	for i := 0; i < actual.NumField(); i++ {
		actualStructField := actual.Type().Field(i)
		fieldName := actualStructField.Name

		shadowField := shadow.FieldByName(fieldName)

		// check if matching field found in shadow
		if shadowField.IsValid() {

			if actualStructField.Type == shadowField.Type() {
				// fields are same type so overwrite directly
				return unsafeSet(actual.FieldByName(fieldName), shadowField)
			}

			// fields not same type so need to patch recursively
			err := patch(actual.FieldByName(fieldName), shadowField)
			if err != nil {
				return errors.Wrap(err, fieldName)
			}
		}
	}

	return nil
}

func patchPrimitive(actual reflect.Value, shadow reflect.Value) error {
	return unsafeSet(actual, shadow)
}

func patchPtr(actual reflect.Value, shadow reflect.Value) error {
	if shadow.IsNil() {
		// no more overwriting to do
		return nil
	}

	if actual.IsNil() && !shadow.IsNil() {
		// need to create a new value for actual to point at
		pointee := reflect.New(actual.Type().Elem())
		actual = reflect.NewAt(actual.Type(), unsafe.Pointer(actual.UnsafeAddr())).Elem()
		actual.Set(pointee)
	}

	return patch(actual.Elem(), shadow.Elem())
}

func patchSlice(actual reflect.Value, shadow reflect.Value) error {
	// TODO: should we allow slices of different length here?
	if actual.Len() != shadow.Len() {
		return errors.New("cannot patch slices of different length")
	}

	for i := 0; i < actual.Len(); i++ {
		err := patch(actual.Index(i), shadow.Index(i))
		if err != nil {
			return errors.Wrapf(err, "index %v:", i)
		}
	}

	return nil
}

func unsafeSet(actual, shadow reflect.Value) error {
	actual = reflect.NewAt(actual.Type(), unsafe.Pointer(actual.UnsafeAddr())).Elem()
	shadow = reflect.NewAt(shadow.Type(), unsafe.Pointer(shadow.UnsafeAddr())).Elem()
	actual.Set(shadow)

	return nil
}

`

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

func main() {
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
	ctxt.OpenFile = func(name string) (io.ReadCloser, error) {
		fmt.Println("OpenFile", name)
		if name == "/input.go" {
			b := bytes.NewBufferString(source)
			return ioutil.NopCloser(b), nil
		}
		return nil, nil
	}

	pkgInfo, err := ctxt.ImportDir("/", 0)
	spew.Dump(pkgInfo)
	fmt.Println(err)

	info.FSet = token.NewFileSet()
	fileAST, err := parser.ParseFile(info.FSet, "input.go", source, parser.ParseComments)
	spew.Dump(fileAST)
	fmt.Println(err)

	files := map[string]*ast.File{
		"input.go": fileAST,
	}

	pkg, err := ast.NewPackage(info.FSet, files, poorMansImporter, nil)
	spew.Dump(pkg, err)
	info.PDoc = doc.New(pkg, pkg.Name, 0)

	presentation := godoc.NewPresentation(&godoc.Corpus{})
	presentation.PackageHTML, err = template.New("package.html").Funcs(presentation.FuncMap()).Parse(string(static.Files["package.html"]))
	if err != nil {
		panic(err)
	}
	presentation.GodocHTML, err = template.New("godoc.html").Funcs(presentation.FuncMap()).Parse(string(static.Files["godoc.html"]))

	spew.Dump(info)
	body := applyTemplate(presentation.PackageHTML, "packageHTML", info)

	resp := httptest.NewRecorder()
	presentation.ServePage(resp, godoc.Page{
		Title: "Package " + pkg.Name,
		//Tabtitle: "My tab tile",
		//Subtitle: "My subtitle",
		Body: body,
		//GoogleCN: info.GoogleCN,
	})

	fmt.Println(strings.Replace(resp.Body.String(), "/lib/godoc", "./lib/godoc", -1))
}

//
//func (h *handlerServer) GetPageInfo(abspath, relpath string, mode PageInfoMode, goos, goarch string) *PageInfo {
//	info := &PageInfo{Dirname: abspath, Mode: mode}
//
//	// Restrict to the package files that would be used when building
//	// the package on this system.  This makes sure that if there are
//	// separate implementations for, say, Windows vs Unix, we don't
//	// jumble them all together.
//	// Note: If goos/goarch aren't set, the current binary's GOOS/GOARCH
//	// are used.
//	ctxt := build.Default
//	ctxt.IsAbsPath = pathpkg.IsAbs
//	ctxt.IsDir = func(path string) bool {
//		fi, err := h.c.fs.Stat(filepath.ToSlash(path))
//		return err == nil && fi.IsDir()
//	}
//	ctxt.ReadDir = func(dir string) ([]os.FileInfo, error) {
//		f, err := h.c.fs.ReadDir(filepath.ToSlash(dir))
//		filtered := make([]os.FileInfo, 0, len(f))
//		for _, i := range f {
//			if mode&NoFiltering != 0 || i.Name() != "internal" {
//				filtered = append(filtered, i)
//			}
//		}
//		return filtered, err
//	}
//	ctxt.OpenFile = func(name string) (r io.ReadCloser, err error) {
//		data, err := vfs.ReadFile(h.c.fs, filepath.ToSlash(name))
//		if err != nil {
//			return nil, err
//		}
//		return ioutil.NopCloser(bytes.NewReader(data)), nil
//	}
//
//	// Make the syscall/js package always visible by default.
//	// It defaults to the host's GOOS/GOARCH, and golang.org's
//	// linux/amd64 means the wasm syscall/js package was blank.
//	// And you can't run godoc on js/wasm anyway, so host defaults
//	// don't make sense here.
//	if goos == "" && goarch == "" && relpath == "syscall/js" {
//		goos, goarch = "js", "wasm"
//	}
//	if goos != "" {
//		ctxt.GOOS = goos
//	}
//	if goarch != "" {
//		ctxt.GOARCH = goarch
//	}
//
//	pkginfo, err := ctxt.ImportDir(abspath, 0)
//	// continue if there are no Go source files; we still want the directory info
//	if _, nogo := err.(*build.NoGoError); err != nil && !nogo {
//		info.Err = err
//		return info
//	}
//
//	// collect package files
//	pkgname := pkginfo.Name
//	pkgfiles := append(pkginfo.GoFiles, pkginfo.CgoFiles...)
//	if len(pkgfiles) == 0 {
//		// Commands written in C have no .go files in the build.
//		// Instead, documentation may be found in an ignored file.
//		// The file may be ignored via an explicit +build ignore
//		// constraint (recommended), or by defining the package
//		// documentation (historic).
//		pkgname = "main" // assume package main since pkginfo.Name == ""
//		pkgfiles = pkginfo.IgnoredGoFiles
//	}
//
//	// get package information, if any
//	if len(pkgfiles) > 0 {
//		// build package AST
//		fset := token.NewFileSet()
//		files, err := h.c.parseFiles(fset, relpath, abspath, pkgfiles)
//		if err != nil {
//			info.Err = err
//			return info
//		}
//
//		// ignore any errors - they are due to unresolved identifiers
//		pkg, _ := ast.NewPackage(fset, files, poorMansImporter, nil)
//
//		// extract package documentation
//		info.FSet = fset
//		if mode&ShowSource == 0 {
//			// show extracted documentation
//			var m doc.Mode
//			if mode&NoFiltering != 0 {
//				m |= doc.AllDecls
//			}
//			if mode&AllMethods != 0 {
//				m |= doc.AllMethods
//			}
//			info.PDoc = doc.New(pkg, pathpkg.Clean(relpath), m) // no trailing '/' in importpath
//			if mode&NoTypeAssoc != 0 {
//				for _, t := range info.PDoc.Types {
//					info.PDoc.Consts = append(info.PDoc.Consts, t.Consts...)
//					info.PDoc.Vars = append(info.PDoc.Vars, t.Vars...)
//					info.PDoc.Funcs = append(info.PDoc.Funcs, t.Funcs...)
//					t.Consts = nil
//					t.Vars = nil
//					t.Funcs = nil
//				}
//				// for now we cannot easily sort consts and vars since
//				// go/doc.Value doesn't export the order information
//				sort.Sort(funcsByName(info.PDoc.Funcs))
//			}
//
//			// collect examples
//			testfiles := append(pkginfo.TestGoFiles, pkginfo.XTestGoFiles...)
//			files, err = h.c.parseFiles(fset, relpath, abspath, testfiles)
//			if err != nil {
//				log.Println("parsing examples:", err)
//			}
//			info.Examples = collectExamples(h.c, pkg, files)
//
//			// collect any notes that we want to show
//			if info.PDoc.Notes != nil {
//				// could regexp.Compile only once per godoc, but probably not worth it
//				if rx := h.p.NotesRx; rx != nil {
//					for m, n := range info.PDoc.Notes {
//						if rx.MatchString(m) {
//							if info.Notes == nil {
//								info.Notes = make(map[string][]*doc.Note)
//							}
//							info.Notes[m] = n
//						}
//					}
//				}
//			}
//
//		} else {
//			// show source code
//			// TODO(gri) Consider eliminating export filtering in this mode,
//			//           or perhaps eliminating the mode altogether.
//			if mode&NoFiltering == 0 {
//				packageExports(fset, pkg)
//			}
//			info.PAst = files
//		}
//		info.IsMain = pkgname == "main"
//	}
//
//	// get directory information, if any
//	var dir *Directory
//	var timestamp time.Time
//	if tree, ts := h.c.fsTree.Get(); tree != nil && tree.(*Directory) != nil {
//		// directory tree is present; lookup respective directory
//		// (may still fail if the file system was updated and the
//		// new directory tree has not yet been computed)
//		dir = tree.(*Directory).lookup(abspath)
//		timestamp = ts
//	}
//	if dir == nil {
//		// no directory tree present (happens in command-line mode);
//		// compute 2 levels for this page. The second level is to
//		// get the synopses of sub-directories.
//		// note: cannot use path filter here because in general
//		// it doesn't contain the FSTree path
//		dir = h.c.newDirectory(abspath, 2)
//		timestamp = time.Now()
//	}
//	info.Dirs = dir.listing(true, func(path string) bool { return h.includePath(path, mode) })
//
//	info.DirTime = timestamp
//	info.DirFlat = mode&FlatDir != 0
//
//	return info
//}
