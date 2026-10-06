package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (loaded *universe) fileByRelative(relative string) *sourceFile {
	for _, file := range loaded.files {
		if file.relative == relative {
			return file
		}
	}
	return nil
}

func cutProtocol(root string, decisions map[string]decision) {
	loaded := loadUniverse(root)
	for _, candidate := range loaded.units {
		entry, isPlanned := decisions[unitKey(candidate)]
		if isPlanned && !entry.Protocol {
			markRemoved(candidate)
		}
	}
	for _, file := range loaded.files {
		if file.hasUnits {
			rewriteCutFile(loaded, file)
		}
	}
	removeUnsharedPackages(loaded, decisions)
}

func removeUnsharedPackages(loaded *universe, decisions map[string]decision) {
	shared := map[string]bool{}
	for _, candidate := range loaded.units {
		if decisions[unitKey(candidate)].Protocol {
			shared[candidate.directory] = true
		}
	}
	for _, file := range loaded.files {
		if !shared[file.directory] && !file.isAssetOnly {
			os.Remove(file.path)
		}
	}
}

func markRemoved(candidate *unit) {
	for _, piece := range candidate.fragments {
		piece.removed = true
		if piece.group != nil {
			piece.group.removed++
		}
	}
}

func rewriteCutFile(loaded *universe, file *sourceFile) {
	var edits []edit
	remaining := 0
	for _, candidate := range loaded.byFile[file.relative] {
		if !candidate.fragments[0].removed {
			remaining++
		}
	}
	for _, candidate := range loaded.units {
		for _, piece := range candidate.fragments {
			if piece.file == file.relative && piece.removed {
				edits = append(edits, removalEdit(piece))
			}
		}
	}
	if remaining == 0 && !hasKeptMethods(loaded, file) {
		os.Remove(file.path)
		return
	}
	text := dedupeEdits(file.text, edits)
	text = fixImports(text, nil, func(string) bool { return false })
	os.WriteFile(file.path, []byte(text), 0o644)
}

func hasKeptMethods(loaded *universe, file *sourceFile) bool {
	for _, candidate := range loaded.units {
		for _, piece := range candidate.fragments[min(1, len(candidate.fragments)):] {
			if piece.file == file.relative && !piece.removed {
				return true
			}
		}
	}
	return false
}

func removalEdit(piece *fragment) edit {
	if piece.group != nil && piece.group.removed == piece.group.total {
		return edit{piece.group.start, piece.group.end, ""}
	}
	return edit{piece.start, piece.end, ""}
}

func dedupeEdits(text string, edits []edit) string {
	seen := map[edit]bool{}
	var unique []edit
	for _, candidate := range edits {
		if !seen[candidate] {
			seen[candidate] = true
			unique = append(unique, candidate)
		}
	}
	return applyEdits(text, unique)
}

type generatedFile struct {
	destination string
	origin      *sourceFile
	fragments   []*fragment
	isTest      bool
}

func (built *plan) relocate(root string) {
	generated := built.collectGenerated()
	built.writeGenerated(root, generated)
	built.rewriteConsumers(root, generated)
	built.removeMovedDirectories(root)
}

func (built *plan) collectGenerated() map[string]*generatedFile {
	generated := map[string]*generatedFile{}
	for _, candidate := range built.universe.units {
		if built.inPlace[candidate.directory] {
			continue
		}
		for _, destination := range built.destinationsOf(candidate) {
			for _, piece := range candidate.fragments {
				key := destination + "|" + piece.file
				if generated[key] == nil {
					origin := built.universe.fileByRelative(piece.file)
					generated[key] = &generatedFile{destination: destination, origin: origin, isTest: origin.isTest}
				}
				generated[key].fragments = append(generated[key].fragments, piece)
			}
		}
	}
	return generated
}

func (built *plan) destinationsOf(candidate *unit) []string {
	if candidate.isTest {
		return sortedKeys(built.stayTests[candidate])
	}
	if destination := built.destination[candidate]; destination != "" {
		return []string{destination}
	}
	return nil
}

func (built *plan) writeGenerated(root string, generated map[string]*generatedFile) {
	keys := make([]string, 0, len(generated))
	for key := range generated {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		item := generated[key]
		name := built.generatedName(root, item)
		text := built.generatedText(root, item)
		parsed, errorValue := parser.ParseFile(token.NewFileSet(), name, text, parser.ParseComments)
		if errorValue != nil {
			fatal(errorValue)
		}
		scratch := &sourceFile{
			relative: item.origin.relative, space: item.origin.space, isTest: item.isTest,
			text: text, parsed: parsed, fileSet: tokenFileSet(name, text),
		}
		built.universe.imports[scratch] = contractImports(parsed)
		rewritten, needed := built.rewriteReferences(scratch, sourceModule+"/"+item.destination, nil)
		final := fixImports(rewritten, needed, built.isManagedImport)
		os.MkdirAll(filepath.Dir(name), 0o755)
		os.WriteFile(name, []byte(final), 0o644)
	}
}

func tokenFileSet(name string, text string) *token.FileSet {
	fileSet := token.NewFileSet()
	parser.ParseFile(fileSet, name, text, parser.ParseComments)
	return fileSet
}

func (built *plan) generatedName(root string, item *generatedFile) string {
	base := filepath.Base(item.origin.relative)
	candidate := filepath.Join(root, item.destination, base)
	if _, errorValue := os.Stat(candidate); errorValue != nil {
		return candidate
	}
	prefix := strings.ReplaceAll(item.origin.directory, "/", "_")
	return filepath.Join(root, item.destination, prefix+"_"+base)
}

func (built *plan) generatedText(root string, item *generatedFile) string {
	name := packageNameOf(root, item.destination)
	if item.origin.space != item.origin.directory {
		name += "_test"
	}
	var pieces []string
	for _, piece := range item.fragments {
		pieces = append(pieces, piece.text)
	}
	return "package " + name + "\n\n" + importDeclarations(item.origin) + "\n\n" + strings.Join(pieces, "\n\n") + "\n"
}

func importDeclarations(origin *sourceFile) string {
	var texts []string
	for _, declaration := range origin.parsed.Decls {
		if generated, isGen := declaration.(*ast.GenDecl); isGen && generated.Tok == token.IMPORT {
			start := origin.fileSet.Position(generated.Pos()).Offset
			end := origin.fileSet.Position(generated.End()).Offset
			texts = append(texts, origin.text[start:end])
		}
	}
	return strings.Join(texts, "\n")
}

func packageNameOf(root string, destination string) string {
	matches, _ := filepath.Glob(filepath.Join(root, destination, "*.go"))
	for _, match := range matches {
		parsed, errorValue := parser.ParseFile(token.NewFileSet(), match, nil, parser.PackageClauseOnly)
		if errorValue == nil && !strings.HasSuffix(parsed.Name.Name, "_test") {
			return parsed.Name.Name
		}
	}
	return packageDirectoryName(destination)
}

func (built *plan) rewriteConsumers(root string, generated map[string]*generatedFile) {
	filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return nil
		}
		relative, _ := filepath.Rel(root, path)
		if entry.IsDir() {
			return consumerDirectoryDecision(entry.Name(), filepath.ToSlash(relative), built)
		}
		if strings.HasSuffix(path, ".go") && !built.isOriginFile(filepath.ToSlash(relative)) {
			built.rewriteConsumer(path, filepath.ToSlash(relative))
		}
		return nil
	})
}

func consumerDirectoryDecision(name string, relative string, built *plan) error {
	switch name {
	case ".git", ".dependency", ".claude", "node_modules", "vendor":
		return filepath.SkipDir
	}
	return nil
}

func (built *plan) isOriginFile(relative string) bool {
	file := built.universe.fileByRelative(relative)
	return file != nil && !built.inPlace[file.directory]
}

func (built *plan) rewriteConsumer(path string, relative string) {
	text, errorValue := os.ReadFile(path)
	if errorValue != nil {
		return
	}
	fileSet := token.NewFileSet()
	parsed, errorValue := parser.ParseFile(fileSet, path, text, parser.ParseComments)
	if errorValue != nil {
		return
	}
	scratch := &sourceFile{relative: relative, text: string(text), parsed: parsed, fileSet: fileSet}
	imports := contractImports(parsed)
	if len(imports) == 0 {
		return
	}
	built.universe.imports[scratch] = imports
	current := sourceModule + "/" + filepath.ToSlash(filepath.Dir(relative))
	rewritten, needed := built.rewriteReferences(scratch, current, built.aliasRemovals(scratch, current))
	final := fixImports(rewritten, needed, built.isManagedImport)
	if final != string(text) {
		os.WriteFile(path, []byte(final), 0o644)
	}
}

func (built *plan) removeMovedDirectories(root string) {
	for directory := range built.universe.spaces {
		if strings.Contains(directory, "#") || built.inPlace[directory] {
			continue
		}
		entries, _ := os.ReadDir(filepath.Join(root, directory))
		for _, entry := range entries {
			location := filepath.Join(root, directory, entry.Name())
			if !entry.IsDir() {
				os.Remove(location)
			} else if !holdsGoFiles(location) {
				os.RemoveAll(location)
			}
		}
	}
	for _, directory := range removalOrder(built) {
		os.Remove(filepath.Join(root, directory))
	}
}

func removalOrder(built *plan) []string {
	var directories []string
	for directory := range built.universe.spaces {
		if !strings.Contains(directory, "#") && !built.inPlace[directory] {
			directories = append(directories, directory)
		}
	}
	sort.Slice(directories, func(left, right int) bool { return len(directories[left]) > len(directories[right]) })
	return directories
}

func holdsGoFiles(directory string) bool {
	found := false
	filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkError error) error {
		if walkError == nil && !entry.IsDir() && strings.HasSuffix(path, ".go") {
			found = true
		}
		return nil
	})
	return found
}
