package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var contractImportPattern = regexp.MustCompile(`^github\.com/yeomyeonggeori/(?:bluecollar|blueprotocol)/(.+)$`)

var contractRoots = []string{"agentcontract", "toolcontract", "model", "taskstate", "holdrecord", "acpupdate", "evaltest"}

type symbolRef struct {
	space string
	name  string
}

type groupInfo struct {
	start   int
	end     int
	total   int
	removed int
}

type fragment struct {
	file    string
	start   int
	end     int
	text    string
	group   *groupInfo
	removed bool
}

type unit struct {
	space     string
	directory string
	names     []string
	kind      string
	file      string
	isTest    bool
	isEntry   bool
	fragments []*fragment
	deps      map[symbolRef]bool
}

type sourceFile struct {
	path        string
	relative    string
	directory   string
	space       string
	isTest      bool
	text        string
	parsed      *ast.File
	fileSet     *token.FileSet
	hasUnits    bool
	isAssetOnly bool
}

type universe struct {
	root    string
	files   []*sourceFile
	units   []*unit
	byName  map[string]map[string]*unit
	spaces  map[string]bool
	byFile  map[string][]*unit
	imports map[*sourceFile]map[string]string
}

func isContractDirectory(relative string) bool {
	first := strings.SplitN(filepath.ToSlash(relative), "/", 2)[0]
	for _, root := range contractRoots {
		if first == root {
			return true
		}
	}
	return false
}

func loadUniverse(root string) *universe {
	loaded := &universe{
		root:    root,
		byName:  map[string]map[string]*unit{},
		spaces:  map[string]bool{},
		byFile:  map[string][]*unit{},
		imports: map[*sourceFile]map[string]string{},
	}
	filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkError error) error {
		if walkError != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		relative, _ := filepath.Rel(root, path)
		if isContractDirectory(relative) {
			loaded.addFile(path, filepath.ToSlash(relative))
		}
		return nil
	})
	loaded.collectUnits()
	loaded.attachMethods()
	loaded.collectDependencies()
	return loaded
}

func (loaded *universe) addFile(path string, relative string) {
	text, errorValue := os.ReadFile(path)
	if errorValue != nil {
		fatal(errorValue)
	}
	fileSet := token.NewFileSet()
	parsed, errorValue := parser.ParseFile(fileSet, path, text, parser.ParseComments)
	if errorValue != nil {
		fatal(errorValue)
	}
	directory := filepath.ToSlash(filepath.Dir(relative))
	isTest := strings.HasSuffix(path, "_test.go")
	space := directory
	if isTest && strings.HasSuffix(parsed.Name.Name, "_test") {
		space = directory + "#external"
	}
	loaded.spaces[space] = true
	loaded.files = append(loaded.files, &sourceFile{
		path: path, relative: relative, directory: directory, space: space, isTest: isTest,
		text: string(text), parsed: parsed, fileSet: fileSet, isAssetOnly: parsed.Name.Name == "main",
	})
}

func (loaded *universe) collectUnits() {
	for _, file := range loaded.files {
		if file.isAssetOnly {
			continue
		}
		loaded.imports[file] = contractImports(file.parsed)
		for _, declaration := range file.parsed.Decls {
			for _, created := range loaded.unitsOf(file, declaration) {
				loaded.register(file, created)
			}
		}
	}
}

func (loaded *universe) register(file *sourceFile, created *unit) {
	file.hasUnits = true
	loaded.units = append(loaded.units, created)
	loaded.byFile[file.relative] = append(loaded.byFile[file.relative], created)
	if loaded.byName[created.space] == nil {
		loaded.byName[created.space] = map[string]*unit{}
	}
	for _, name := range created.names {
		loaded.byName[created.space][name] = created
	}
}

func newUnit(file *sourceFile, kind string, names ...string) *unit {
	created := &unit{
		space: file.space, directory: file.directory, names: names, kind: kind,
		file: file.relative, isTest: file.isTest, deps: map[symbolRef]bool{},
	}
	created.isEntry = isEntryUnit(created)
	return created
}

func isEntryUnit(candidate *unit) bool {
	if !candidate.isTest {
		return false
	}
	for _, name := range candidate.names {
		for _, prefix := range []string{"Test", "Benchmark", "Fuzz", "Example"} {
			if strings.HasPrefix(name, prefix) && candidate.kind == "func" {
				return true
			}
		}
		if name == "init" || name == "_" {
			return true
		}
	}
	return false
}

func (loaded *universe) unitsOf(file *sourceFile, declaration ast.Decl) []*unit {
	switch typed := declaration.(type) {
	case *ast.FuncDecl:
		if typed.Recv != nil {
			return nil
		}
		created := newUnit(file, "func", typed.Name.Name)
		created.fragments = []*fragment{wholeFragment(file, typed, typed.Doc)}
		return []*unit{created}
	case *ast.GenDecl:
		return loaded.genDeclUnits(file, typed)
	}
	return nil
}

func wholeFragment(file *sourceFile, node ast.Node, doc *ast.CommentGroup) *fragment {
	start := file.fileSet.Position(node.Pos()).Offset
	if doc != nil {
		start = file.fileSet.Position(doc.Pos()).Offset
	}
	end := file.fileSet.Position(node.End()).Offset
	return &fragment{file: file.relative, start: start, end: end, text: file.text[start:end]}
}

func (loaded *universe) genDeclUnits(file *sourceFile, declaration *ast.GenDecl) []*unit {
	if declaration.Tok == token.IMPORT {
		return nil
	}
	kind := strings.ToLower(declaration.Tok.String())
	if !declaration.Lparen.IsValid() || usesImplicitValues(declaration) {
		return []*unit{wholeDeclUnit(file, declaration, kind)}
	}
	group := &groupInfo{
		start: file.fileSet.Position(declStart(file, declaration)).Offset,
		end:   file.fileSet.Position(declaration.End()).Offset,
		total: len(declaration.Specs),
	}
	var units []*unit
	for _, spec := range declaration.Specs {
		created := newUnit(file, kind, specNames(spec)...)
		created.fragments = []*fragment{specFragment(file, declaration, spec, group)}
		units = append(units, created)
	}
	return units
}

func declStart(file *sourceFile, declaration *ast.GenDecl) token.Pos {
	if declaration.Doc != nil {
		return declaration.Doc.Pos()
	}
	return declaration.Pos()
}

func wholeDeclUnit(file *sourceFile, declaration *ast.GenDecl, kind string) *unit {
	var names []string
	for _, spec := range declaration.Specs {
		names = append(names, specNames(spec)...)
	}
	created := newUnit(file, kind, names...)
	created.fragments = []*fragment{wholeFragment(file, declaration, declaration.Doc)}
	return created
}

func specFragment(file *sourceFile, declaration *ast.GenDecl, spec ast.Spec, group *groupInfo) *fragment {
	start := file.fileSet.Position(spec.Pos()).Offset
	docStart := start
	switch typed := spec.(type) {
	case *ast.TypeSpec:
		if typed.Doc != nil {
			docStart = file.fileSet.Position(typed.Doc.Pos()).Offset
		}
	case *ast.ValueSpec:
		if typed.Doc != nil {
			docStart = file.fileSet.Position(typed.Doc.Pos()).Offset
		}
	}
	end := file.fileSet.Position(spec.End()).Offset
	body := file.text[start:end]
	keyword := strings.ToLower(declaration.Tok.String())
	text := file.text[docStart:start] + keyword + " " + body
	return &fragment{file: file.relative, start: docStart, end: end, text: text, group: group}
}

func usesImplicitValues(declaration *ast.GenDecl) bool {
	if declaration.Tok != token.CONST {
		return false
	}
	for _, spec := range declaration.Specs {
		if len(spec.(*ast.ValueSpec).Values) == 0 {
			return true
		}
	}
	return false
}

func specNames(spec ast.Spec) []string {
	switch typed := spec.(type) {
	case *ast.TypeSpec:
		return []string{typed.Name.Name}
	case *ast.ValueSpec:
		var names []string
		for _, name := range typed.Names {
			names = append(names, name.Name)
		}
		return names
	}
	return nil
}

func receiverName(function *ast.FuncDecl) string {
	expression := function.Recv.List[0].Type
	if star, isStar := expression.(*ast.StarExpr); isStar {
		expression = star.X
	}
	if index, isIndex := expression.(*ast.IndexExpr); isIndex {
		expression = index.X
	}
	if ident, isIdent := expression.(*ast.Ident); isIdent {
		return ident.Name
	}
	return ""
}

func (loaded *universe) attachMethods() {
	for _, file := range loaded.files {
		if file.isAssetOnly {
			continue
		}
		for _, declaration := range file.parsed.Decls {
			function, isFunction := declaration.(*ast.FuncDecl)
			if !isFunction || function.Recv == nil {
				continue
			}
			owner := loaded.byName[file.space][receiverName(function)]
			if owner == nil {
				fatal(fmt.Errorf("%s: method %s has no receiver type in its package", file.relative, function.Name.Name))
			}
			owner.fragments = append(owner.fragments, wholeFragment(file, function, function.Doc))
		}
	}
}

func contractImports(file *ast.File) map[string]string {
	imports := map[string]string{}
	for _, spec := range file.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		match := contractImportPattern.FindStringSubmatch(path)
		if match == nil {
			continue
		}
		name := filepath.Base(match[1])
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = match[1]
	}
	return imports
}

func isFileLevel(file *ast.File, ident *ast.Ident) bool {
	return ident.Obj == nil || file.Scope.Objects[ident.Name] == ident.Obj
}

func (loaded *universe) collectDependencies() {
	for _, file := range loaded.files {
		if file.isAssetOnly {
			continue
		}
		for _, declaration := range file.parsed.Decls {
			owner := loaded.ownerOf(file, declaration)
			if owner == nil {
				continue
			}
			for _, reference := range loaded.referencesIn(file, declaration) {
				owner.deps[reference] = true
			}
		}
	}
}

func (loaded *universe) ownerOf(file *sourceFile, declaration ast.Decl) *unit {
	switch typed := declaration.(type) {
	case *ast.FuncDecl:
		if typed.Recv == nil {
			return loaded.byName[file.space][typed.Name.Name]
		}
		return loaded.byName[file.space][receiverName(typed)]
	case *ast.GenDecl:
		if typed.Tok == token.IMPORT || len(typed.Specs) == 0 {
			return nil
		}
		return loaded.byName[file.space][specNames(typed.Specs[0])[0]]
	}
	return nil
}

func (loaded *universe) referencesIn(file *sourceFile, declaration ast.Decl) []symbolRef {
	var references []symbolRef
	loaded.walkReferences(file, declaration, func(_ ast.Node, reference symbolRef) {
		references = append(references, reference)
	})
	return references
}

func (loaded *universe) walkReferences(file *sourceFile, root ast.Node, visit func(node ast.Node, reference symbolRef)) {
	imports := loaded.imports[file]
	skipped := map[*ast.Ident]bool{}
	keysAreFields := keyedLiteralKinds(root)
	ast.Inspect(root, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			skipped[typed.Sel] = true
			qualifier, isIdent := typed.X.(*ast.Ident)
			if !isIdent || !isFileLevel(file.parsed, qualifier) {
				return true
			}
			if target, isContract := imports[qualifier.Name]; isContract && loaded.byName[target][typed.Sel.Name] != nil {
				visit(typed, symbolRef{target, typed.Sel.Name})
			}
		case *ast.FuncDecl:
			skipped[typed.Name] = true
		case *ast.KeyValueExpr:
			if key, isIdent := typed.Key.(*ast.Ident); isIdent && keysAreFields[typed] {
				skipped[key] = true
			}
		case *ast.Ident:
			if skipped[typed] || !isFileLevel(file.parsed, typed) {
				return true
			}
			if loaded.byName[file.space][typed.Name] != nil {
				visit(typed, symbolRef{file.space, typed.Name})
			}
		}
		return true
	})
}

func keyedLiteralKinds(root ast.Node) map[*ast.KeyValueExpr]bool {
	fields := map[*ast.KeyValueExpr]bool{}
	ast.Inspect(root, func(node ast.Node) bool {
		literal, isLiteral := node.(*ast.CompositeLit)
		if !isLiteral {
			return true
		}
		markFieldKeys(literal, literal.Type, fields)
		return true
	})
	return fields
}

func markFieldKeys(literal *ast.CompositeLit, literalType ast.Expr, fields map[*ast.KeyValueExpr]bool) {
	isStructLike := isStructLikeType(literalType)
	for _, element := range literal.Elts {
		pair, isPair := element.(*ast.KeyValueExpr)
		if isPair && isStructLike {
			fields[pair] = true
		}
		inner, isInner := innerLiteral(element)
		if isInner && inner.Type == nil {
			markFieldKeys(inner, elementType(literalType), fields)
		}
	}
}

func innerLiteral(element ast.Expr) (*ast.CompositeLit, bool) {
	if pair, isPair := element.(*ast.KeyValueExpr); isPair {
		element = pair.Value
	}
	inner, isInner := element.(*ast.CompositeLit)
	return inner, isInner
}

func elementType(literalType ast.Expr) ast.Expr {
	switch typed := literalType.(type) {
	case *ast.MapType:
		return typed.Value
	case *ast.ArrayType:
		return typed.Elt
	case *ast.StarExpr:
		return elementType(typed.X)
	}
	return nil
}

func isStructLikeType(literalType ast.Expr) bool {
	switch typed := literalType.(type) {
	case *ast.Ident, *ast.SelectorExpr, *ast.StructType, *ast.IndexExpr:
		return true
	case *ast.StarExpr:
		return isStructLikeType(typed.X)
	case nil:
		return false
	}
	return false
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func fatal(errorValue error) {
	fmt.Fprintln(os.Stderr, errorValue)
	os.Exit(1)
}
