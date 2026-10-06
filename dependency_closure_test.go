package blueprotocol_test

import (
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

var forbiddenModulePaths = []string{
	"github.com/yeomyeonggeori/bluecollar",
	"github.com/yeomyeonggeori/blueclaw",
}

func TestDependencyClosureExcludesHarnessAndHost(t *testing.T) {
	for _, dependency := range listDependencies(t) {
		if forbiddenModule, isForbidden := forbiddenModuleOf(dependency); isForbidden {
			t.Errorf("dependency %s belongs to %s", dependency, forbiddenModule)
		}
	}
}

func listDependencies(t *testing.T) []string {
	t.Helper()
	output, errorValue := exec.Command("go", "list", "-deps", "./...").Output()
	if errorValue != nil {
		t.Fatalf("go list -deps ./...: %v", errorValue)
	}
	return strings.Fields(string(output))
}

func forbiddenModuleOf(dependency string) (string, bool) {
	for _, forbiddenModule := range forbiddenModulePaths {
		if strings.Contains(dependency, forbiddenModule) {
			return forbiddenModule, true
		}
	}
	return "", false
}

func TestNoFileNamesTheHarnessOrTheHost(t *testing.T) {
	words := forbiddenWords()
	filepath.WalkDir(".", func(location string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return nil
		}
		if isExemptFromVocabulary(location, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			reportForbiddenWords(t, location, words)
		}
		return nil
	})
}

func forbiddenWords() []string {
	var words []string
	for _, modulePath := range forbiddenModulePaths {
		words = append(words, strings.ToLower(path.Base(modulePath)))
	}
	return words
}

func isExemptFromVocabulary(location string, entry fs.DirEntry) bool {
	if entry.IsDir() {
		return entry.Name() == ".git"
	}
	return location == "README.md" || location == "dependency_closure_test.go"
}

func reportForbiddenWords(t *testing.T, location string, words []string) {
	t.Helper()
	content, errorValue := os.ReadFile(location)
	if errorValue != nil || location == "go.sum" {
		return
	}
	for number, line := range strings.Split(strings.ToLower(string(content)), "\n") {
		for _, word := range words {
			if strings.Contains(line, word) {
				t.Errorf("%s:%d names %q; the contracts belong to no harness or host", location, number+1, word)
			}
		}
	}
}
