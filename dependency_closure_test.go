package blueprotocol_test

import (
	"os/exec"
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
	output, err := exec.Command("go", "list", "-deps", "./...").Output()
	if err != nil {
		t.Fatalf("go list -deps ./...: %v", err)
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
