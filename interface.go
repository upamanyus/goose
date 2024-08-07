package goose

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/goose-lang/goose/glang"
	"golang.org/x/tools/go/packages"
)

// declsOrError translates one top-level declaration,
// catching Goose translation errors and returning them as a regular Go error
func (ctx Ctx) declsOrError(stmt ast.Decl) (decls []glang.Decl, err error) {
	defer func() {
		if r := recover(); r != nil {
			if gooseErr, ok := r.(gooseError); ok {
				err = gooseErr.err
			} else {
				// r is an error from a non-goose error, indicating a bug
				panic(r)
			}
		}
	}()
	return ctx.maybeDecls(stmt), nil
}

func filterImports(decls []glang.Decl) (nonImports []glang.Decl, imports glang.ImportDecls) {
	for _, d := range decls {
		switch d := d.(type) {
		case glang.ImportDecl:
			imports = append(imports, d)
		default:
			nonImports = append(nonImports, d)
		}
	}
	return
}

type declId struct {
	fileIdx int
	declIdx int
}

type depTracker struct {
	names []string
	deps  []string
}

func (dt *depTracker) addName(s string) {
	dt.names = append(dt.names, s)
}

func (dt *depTracker) addDep(s string) {
	dt.deps = append(dt.deps, s)
}

// Decls converts an entire package (possibly multiple files) to a list of decls
func (ctx Ctx) Decls(fs ...NamedFile) (imports glang.ImportDecls, decls []glang.Decl, errs []error) {
	declGroups := make(map[declId][]glang.Decl)
	declDeps := make(map[declId][]string)
	nameDecls := make(map[string]declId)
	generated := make(map[declId]bool)

	// Translate every Go decl into a Glang decl and build up dependencies for
	// each of them.
	for fi, f := range fs {
		for di, d := range f.Ast.Decls {
			ctx.dep = &depTracker{}

			id := declId{fi, di}
			newDecls, err := ctx.declsOrError(d)
			if err != nil {
				errs = append(errs, err)
			}

			declGroups[id] = newDecls
			declDeps[id] = ctx.dep.deps
			for _, n := range ctx.dep.names {
				nameDecls[n] = id
			}
		}
	}

	// Sort Glang decls based on dependencies
	var lastFile int
	var processDecl func(id declId, ident string)

	processDecl = func(id declId, ident string) {
		if generated[id] {
			return
		}
		generated[id] = true

		for _, dep := range declDeps[id] {
			depid, ok := nameDecls[dep]
			if ok {
				processDecl(depid, dep)
			}
		}

		if lastFile != id.fileIdx && ident != "" {
			f := fs[id.fileIdx]
			decls = append(decls,
				glang.NewComment(fmt.Sprintf("%s from %s", ident, f.Name())))
			lastFile = id.fileIdx
		}

		newDecls, newImports := filterImports(declGroups[id])
		decls = append(decls, newDecls...)
		imports = append(imports, newImports...)
	}

	for fi, f := range fs {
		if len(fs) > 1 {
			decls = append(decls,
				glang.NewComment(fmt.Sprintf("%s", f.Name())))
		}
		if f.Ast.Doc != nil {
			decls = append(decls, glang.NewComment(f.Ast.Doc.Text()))
		}
		lastFile = fi
		for di := range f.Ast.Decls {
			processDecl(declId{fi, di}, "")
		}
	}
	return
}

type MultipleErrors []error

func (es MultipleErrors) Error() string {
	var errs []string
	for _, e := range es {
		errs = append(errs, e.Error())
	}
	errs = append(errs, fmt.Sprintf("%d errors", len(es)))
	return strings.Join(errs, "\n\n")
}

type NamedFile struct {
	Path string
	Ast  *ast.File
}

func (f NamedFile) Name() string {
	return path.Base(f.Path)
}

func sortedFiles(fileNames []string, fileAsts []*ast.File) []NamedFile {
	var flatFiles []NamedFile
	if len(fileNames) != len(fileAsts) {
		fmt.Printf("names: %+v\n", fileNames)
		fmt.Printf("asts: %+v\n", fileAsts)
		panic("sortedFiles(): fileNames must match fileAsts")
	}
	for i := range fileNames {
		flatFiles = append(flatFiles, NamedFile{Path: fileNames[i], Ast: fileAsts[i]})
	}
	sort.Slice(flatFiles, func(i, j int) bool {
		return flatFiles[i].Path < flatFiles[j].Path
	})
	return flatFiles
}

// Translator has global configuration for translation
//
// TODO: need a better name for this (Translator is somehow global stuff, Config
// is per-package)
//
// TODO: fix duplication with Config, perhaps embed a Translator in a Config
type Translator struct {
	TypeCheck             bool
	AddSourceFileComments bool
}

func pkgErrors(errors []packages.Error) error {
	var errs []error
	for _, err := range errors {
		errs = append(errs, err)
	}
	return MultipleErrors(errs)
}

// translatePackage translates an entire package to a single Coq file.
//
// If the source directory has multiple source files, these are processed in
// alphabetical order; this must be a topological sort of the definitions or the
// Coq code will be out-of-order. Sorting ensures the results are stable
// and not dependent on map or directory iteration order.
func (tr Translator) translatePackage(pkg *packages.Package) (glang.File, error) {
	if len(pkg.Errors) > 0 {
		return glang.File{}, errors.Errorf(
			"could not load package %v:\n%v", pkg.PkgPath,
			pkgErrors(pkg.Errors))
	}
	ctx := NewPkgCtx(pkg, tr)
	files := sortedFiles(pkg.CompiledGoFiles, pkg.Syntax)

	coqFile := glang.File{
		PkgPath:   pkg.PkgPath,
		GoPackage: pkg.Name,
	}
	coqFile.ImportHeader, coqFile.Footer = ffiHeaderFooter(ctx.Config.Ffi)

	imports, decls, errs := ctx.Decls(files...)
	coqFile.Imports = imports
	coqFile.Decls = decls
	if len(errs) != 0 {
		return coqFile, errors.Wrap(MultipleErrors(errs),
			"conversion failed")
	}
	return coqFile, nil
}

func ffiHeaderFooter(ffi string) (header string, footer string) {
	if ffi == "none" {
		header = "Section code.\n" +
			"Context `{ffi_syntax}.\n" +
			"Local Coercion Var' s: expr := Var s."
		footer = "\nEnd code.\n"
	} else {
		header += fmt.Sprintf("From New Require Import %s_prelude.", ffi)
	}
	return
}

// newPackageConfig creates a package loading configuration suitable for
// Goose translation.
func newPackageConfig(modDir string) *packages.Config {
	mode := packages.NeedName | packages.NeedCompiledGoFiles
	mode |= packages.NeedImports
	mode |= packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo
	return &packages.Config{
		Dir:        modDir,
		Mode:       mode,
		BuildFlags: []string{"-tags", "goose"},
		Fset:       token.NewFileSet(),
	}
}

// TranslatePackages loads packages by a list of patterns and translates them
// all, producing one file per matched package.
//
// The errs list contains errors corresponding to each package (in parallel with
// the files list). patternErr is only non-nil if the patterns themselves have
// a syntax error.
func (tr Translator) TranslatePackages(modDir string,
	pkgPattern ...string) (files []glang.File, errs []error, patternErr error) {
	pkgs, err := packages.Load(newPackageConfig(modDir), pkgPattern...)
	if err != nil {
		return nil, nil, err
	}
	if len(pkgs) == 0 {
		// consider matching nothing to be an error, unlike packages.Load
		return nil, nil,
			errors.New("patterns matched no packages")
	}
	files = make([]glang.File, len(pkgs))
	errs = make([]error, len(pkgs))
	var wg sync.WaitGroup
	wg.Add(len(pkgs))
	for i, pkg := range pkgs {
		go func(i int, pkg *packages.Package) {
			f, err := tr.translatePackage(pkg)
			files[i] = f
			errs[i] = err
			wg.Done()
		}(i, pkg)
	}
	wg.Wait()
	return
}
