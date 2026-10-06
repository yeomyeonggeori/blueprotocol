package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const (
	sourceModule = "github.com/yeomyeonggeori/bluecollar"
	targetModule = "github.com/yeomyeonggeori/blueprotocol"
)

var versionSuffix = regexp.MustCompile(`^v\d+$`)

type edit struct {
	start       int
	end         int
	replacement string
}

type location struct {
	importPath string
	isKnown    bool
}

func applyEdits(text string, edits []edit) string {
	sort.Slice(edits, func(left, right int) bool { return edits[left].start > edits[right].start })
	for _, applied := range edits {
		text = text[:applied.start] + applied.replacement + text[applied.end:]
	}
	return text
}

func importName(importPath string) string {
	elements := strings.Split(importPath, "/")
	last := elements[len(elements)-1]
	if versionSuffix.MatchString(last) && len(elements) > 1 {
		return elements[len(elements)-2]
	}
	return last
}

func (built *plan) locate(reference symbolRef) location {
	target := built.universe.byName[reference.space][reference.name]
	switch {
	case target == nil || target.isTest || strings.Contains(reference.space, "#"):
		return location{}
	case built.moved[target]:
		return location{targetModule + "/" + reference.space, true}
	case built.inPlace[target.directory]:
		return location{sourceModule + "/" + reference.space, true}
	case built.destination[target] != "":
		return location{sourceModule + "/" + built.destination[target], true}
	}
	return location{}
}

func (built *plan) rewriteReferences(file *sourceFile, currentPackage string, removals []edit) (string, map[string]bool) {
	var edits []edit
	needed := map[string]bool{}
	built.universe.walkReferences(file, file.parsed, func(node ast.Node, reference symbolRef) {
		destination := built.locate(reference)
		position := func(pos token.Pos) int { return file.fileSet.Position(pos).Offset }
		if !destination.isKnown || isInsideAny(position(node.Pos()), removals) {
			return
		}
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			qualifier := typed.X.(*ast.Ident)
			if destination.importPath == currentPackage {
				edits = append(edits, edit{position(qualifier.Pos()), position(typed.Sel.Pos()), ""})
				return
			}
			needed[destination.importPath] = true
			edits = append(edits, edit{position(qualifier.Pos()), position(qualifier.End()), importName(destination.importPath)})
		case *ast.Ident:
			if destination.importPath == currentPackage {
				return
			}
			needed[destination.importPath] = true
			edits = append(edits, edit{position(typed.Pos()), position(typed.Pos()), importName(destination.importPath) + "."})
		}
	})
	return applyEdits(file.text, append(edits, removals...)), needed
}

func isInsideAny(offset int, removals []edit) bool {
	for _, removal := range removals {
		if offset >= removal.start && offset < removal.end {
			return true
		}
	}
	return false
}

func (built *plan) aliasRemovals(file *sourceFile, currentPackage string) []edit {
	var removals []edit
	for _, declaration := range file.parsed.Decls {
		generated, isGen := declaration.(*ast.GenDecl)
		if !isGen || generated.Tok == token.IMPORT {
			continue
		}
		var removable []ast.Spec
		for _, spec := range generated.Specs {
			if built.isAliasOfRelocatedUnit(file, spec, currentPackage) {
				removable = append(removable, spec)
			}
		}
		if len(removable) == len(generated.Specs) {
			removals = append(removals, wholeLineEdit(file.fileSet, file.text, generated))
			continue
		}
		for _, spec := range removable {
			removals = append(removals, wholeLineEdit(file.fileSet, file.text, spec))
		}
	}
	return removals
}

func (built *plan) isAliasOfRelocatedUnit(file *sourceFile, spec ast.Spec, currentPackage string) bool {
	name, value := aliasOf(spec)
	selector, isSelector := value.(*ast.SelectorExpr)
	if name == "" || !isSelector {
		return false
	}
	qualifier, isIdent := selector.X.(*ast.Ident)
	if !isIdent {
		return false
	}
	target := built.universe.imports[file][qualifier.Name]
	if target == "" || selector.Sel.Name != name {
		return false
	}
	return built.locate(symbolRef{target, name}).importPath == currentPackage
}

func aliasOf(spec ast.Spec) (string, ast.Expr) {
	switch typed := spec.(type) {
	case *ast.ValueSpec:
		if len(typed.Names) == 1 && len(typed.Values) == 1 {
			return typed.Names[0].Name, typed.Values[0]
		}
	case *ast.TypeSpec:
		if typed.Assign.IsValid() {
			return typed.Name.Name, typed.Type
		}
	}
	return "", nil
}

func (built *plan) isManagedImport(importPath string) bool {
	for _, module := range []string{sourceModule, targetModule} {
		relative, isInside := strings.CutPrefix(importPath, module+"/")
		if isInside && built.isManagedDirectory(relative) {
			return true
		}
	}
	return false
}

func (built *plan) isManagedDirectory(relative string) bool {
	return built.universe.spaces[relative]
}

func fixImports(text string, needed map[string]bool, isManaged func(string) bool) string {
	fileSet := token.NewFileSet()
	parsed, errorValue := parser.ParseFile(fileSet, "fixed.go", text, parser.ParseComments)
	if errorValue != nil {
		fatal(fmt.Errorf("%w\n%s", errorValue, text))
	}
	used := usedQualifiers(parsed)
	var kept []string
	var removals []edit
	for _, declaration := range parsed.Decls {
		generated, isGen := declaration.(*ast.GenDecl)
		if !isGen || generated.Tok != token.IMPORT {
			continue
		}
		removals = append(removals, wholeLineEdit(fileSet, text, generated))
		for _, spec := range generated.Specs {
			importSpec := spec.(*ast.ImportSpec)
			importPath, _ := strconv.Unquote(importSpec.Path.Value)
			if isManaged(importPath) || !isUsedImport(importSpec, importPath, used) {
				continue
			}
			kept = append(kept, importSpecText(importSpec, importPath))
		}
	}
	for importPath := range needed {
		if !slices.Contains(kept, strconv.Quote(importPath)) {
			kept = append(kept, strconv.Quote(importPath))
		}
	}
	rewritten := applyEdits(text, removals)
	return insertImportBlock(rewritten, kept)
}

func wholeLineEdit(fileSet *token.FileSet, text string, node ast.Node) edit {
	start := fileSet.Position(node.Pos()).Offset
	end := fileSet.Position(node.End()).Offset
	if end < len(text) && text[end] == '\n' {
		end++
	}
	return edit{start, end, ""}
}

func usedQualifiers(parsed *ast.File) map[string]bool {
	used := map[string]bool{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		selector, isSelector := node.(*ast.SelectorExpr)
		if !isSelector {
			return true
		}
		if qualifier, isIdent := selector.X.(*ast.Ident); isIdent {
			used[qualifier.Name] = true
		}
		return true
	})
	return used
}

func isUsedImport(importSpec *ast.ImportSpec, importPath string, used map[string]bool) bool {
	if importSpec.Name != nil {
		return importSpec.Name.Name == "_" || importSpec.Name.Name == "." || used[importSpec.Name.Name]
	}
	return used[importName(importPath)]
}

func importSpecText(importSpec *ast.ImportSpec, importPath string) string {
	if importSpec.Name != nil {
		return importSpec.Name.Name + " " + strconv.Quote(importPath)
	}
	return strconv.Quote(importPath)
}

func insertImportBlock(text string, specs []string) string {
	if len(specs) == 0 {
		return formatted(text)
	}
	var standard, others []string
	for _, spec := range specs {
		if isStandardLibrarySpec(spec) {
			standard = append(standard, spec)
		} else {
			others = append(others, spec)
		}
	}
	sort.Strings(standard)
	sort.Strings(others)
	block := "import (\n"
	for _, spec := range standard {
		block += "\t" + spec + "\n"
	}
	if len(standard) > 0 && len(others) > 0 {
		block += "\n"
	}
	for _, spec := range others {
		block += "\t" + spec + "\n"
	}
	block += ")\n"
	fileSet := token.NewFileSet()
	parsed, errorValue := parser.ParseFile(fileSet, "fixed.go", text, parser.PackageClauseOnly)
	if errorValue != nil {
		fatal(errorValue)
	}
	insertAt := fileSet.Position(parsed.Name.End()).Offset
	if newline := strings.Index(text[insertAt:], "\n"); newline >= 0 {
		insertAt += newline + 1
	}
	return formatted(text[:insertAt] + "\n" + block + text[insertAt:])
}

func isStandardLibrarySpec(spec string) bool {
	unquoted, _ := strconv.Unquote(spec[strings.Index(spec, "\""):])
	first := strings.SplitN(unquoted, "/", 2)[0]
	return !strings.Contains(first, ".")
}

func formatted(text string) string {
	collapsed := regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	result, errorValue := format.Source([]byte(collapsed))
	if errorValue != nil {
		fatal(fmt.Errorf("%w\n%s", errorValue, collapsed))
	}
	return string(result)
}

func packageDirectoryName(destination string) string {
	return path.Base(destination)
}
