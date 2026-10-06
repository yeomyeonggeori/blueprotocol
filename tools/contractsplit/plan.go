package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type layout struct {
	ForceMoved  []string          `json:"forceMoved"`
	Homes       map[string]string `json:"homes"`
	SymbolHomes map[string]string `json:"symbolHomes"`
	NotShared   []string          `json:"notShared"`
	HostIgnore  []string          `json:"hostIgnore"`
}

type decision struct {
	Key          string   `json:"key"`
	Protocol     bool     `json:"protocol"`
	Stay         bool     `json:"stay"`
	Destination  string   `json:"destination,omitempty"`
	Destinations []string `json:"destinations,omitempty"`
}

type plan struct {
	universe      *universe
	layout        layout
	moved         map[*unit]bool
	destination   map[*unit]string
	protocolTests map[*unit]bool
	stayTests     map[*unit]map[string]bool
	inPlace       map[string]bool
	dropped       map[*unit]bool
	droppedTests  []string
	direct        map[*unit]bool
	users         map[symbolRef]map[string]bool
	problems      []string
}

func unitKey(candidate *unit) string {
	return candidate.file + "|" + candidate.kind + "|" + strings.Join(candidate.names, ",")
}

func buildPlan(contractTree string, hostTree string, usersTree string, chosen layout) *plan {
	built := &plan{
		universe:      loadUniverse(contractTree),
		layout:        chosen,
		moved:         map[*unit]bool{},
		destination:   map[*unit]string{},
		protocolTests: map[*unit]bool{},
		stayTests:     map[*unit]map[string]bool{},
		inPlace:       map[string]bool{},
		dropped:       map[*unit]bool{},
	}
	built.users = scanUses(usersTree, built.universe, nil)
	built.closeOverHost(scanUses(hostTree, built.universe, chosen.HostIgnore))
	built.findInPlacePackages()
	built.assignDestinations()
	built.classifyTests()
	built.checkUnexportedReferences()
	built.checkDestinationsServeTheirUsers()
	return built
}

func scanUses(root string, loaded *universe, ignored []string) map[symbolRef]map[string]bool {
	uses := map[symbolRef]map[string]bool{}
	filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return nil
		}
		relative, _ := filepath.Rel(root, path)
		if entry.IsDir() {
			if slices.Contains(ignored, filepath.ToSlash(relative)) {
				return filepath.SkipDir
			}
			return skipDirectory(entry.Name(), relative)
		}
		if strings.HasSuffix(path, ".go") && !isContractDirectory(relative) {
			recordUses(uses, loaded, path, filepath.ToSlash(filepath.Dir(relative)))
		}
		return nil
	})
	return uses
}

func skipDirectory(name string, relative string) error {
	switch name {
	case ".git", ".dependency", ".claude", "node_modules", "vendor":
		return filepath.SkipDir
	}
	if isContractDirectory(relative) {
		return filepath.SkipDir
	}
	return nil
}

func recordUses(uses map[symbolRef]map[string]bool, loaded *universe, path string, directory string) {
	fileSet := token.NewFileSet()
	parsed, errorValue := parser.ParseFile(fileSet, path, nil, 0)
	if errorValue != nil {
		return
	}
	imports := contractImports(parsed)
	if len(imports) == 0 {
		return
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		selector, isSelector := node.(*ast.SelectorExpr)
		if !isSelector {
			return true
		}
		qualifier, isIdent := selector.X.(*ast.Ident)
		if !isIdent || !isFileLevel(parsed, qualifier) {
			return true
		}
		target, isContract := imports[qualifier.Name]
		if isContract && loaded.byName[target][selector.Sel.Name] != nil {
			reference := symbolRef{target, selector.Sel.Name}
			if uses[reference] == nil {
				uses[reference] = map[string]bool{}
			}
			uses[reference][normalizeUserDirectory(directory)] = true
		}
		return true
	})
}

func normalizeUserDirectory(directory string) string {
	parent, last := filepath.Split(directory)
	if parent != "" && last == filepath.Base(parent)+"test" {
		return strings.TrimSuffix(parent, "/")
	}
	return directory
}

func (built *plan) isNotShared(reference symbolRef) bool {
	for _, entry := range built.layout.NotShared {
		if entry == reference.space+"."+reference.name {
			return true
		}
	}
	return false
}

func (built *plan) forced(candidate *unit) bool {
	for _, entry := range built.layout.ForceMoved {
		if candidate.directory == entry || candidate.file == entry {
			return true
		}
	}
	return false
}

func (built *plan) closeOverHost(hostUses map[symbolRef]map[string]bool) {
	var pending []*unit
	built.direct = map[*unit]bool{}
	for reference := range hostUses {
		if !built.isNotShared(reference) {
			target := built.universe.byName[reference.space][reference.name]
			built.direct[target] = true
			pending = append(pending, target)
		}
	}
	for _, candidate := range built.universe.units {
		if !candidate.isTest && built.forced(candidate) {
			pending = append(pending, candidate)
		}
	}
	for len(pending) > 0 {
		built.closeFrom(pending)
		pending = built.stayUnitsNeedingMovedHelpers()
	}
}

func (built *plan) closeFrom(pending []*unit) {
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if current.isTest || built.moved[current] {
			continue
		}
		built.moved[current] = true
		for dependency := range current.deps {
			if target := built.universe.byName[dependency.space][dependency.name]; target != nil {
				pending = append(pending, target)
			}
		}
	}
}

func (built *plan) stayUnitsNeedingMovedHelpers() []*unit {
	var promoted []*unit
	for _, candidate := range built.universe.units {
		if candidate.isTest || built.moved[candidate] {
			continue
		}
		for dependency := range candidate.deps {
			target := built.universe.byName[dependency.space][dependency.name]
			if target != nil && built.moved[target] && !ast.IsExported(dependency.name) {
				promoted = append(promoted, candidate)
				break
			}
		}
	}
	return promoted
}

func (built *plan) findInPlacePackages() {
	movedIn := map[string]bool{}
	for candidate := range built.moved {
		movedIn[candidate.directory] = true
	}
	for _, candidate := range built.universe.units {
		if !movedIn[candidate.directory] {
			built.inPlace[candidate.directory] = true
		}
	}
}

func (built *plan) isStay(candidate *unit) bool {
	return !candidate.isTest && !built.moved[candidate] && !built.inPlace[candidate.directory]
}

func (built *plan) usersOf(candidate *unit) map[string]bool {
	merged := map[string]bool{}
	for _, name := range candidate.names {
		for user := range built.users[symbolRef{candidate.directory, name}] {
			merged[user] = true
		}
	}
	return merged
}

func (built *plan) homeOf(candidate *unit) string {
	for _, name := range candidate.names {
		if home, isSet := built.layout.SymbolHomes[candidate.directory+"."+name]; isSet {
			return home
		}
	}
	return ""
}

func (built *plan) assignDestinations() {
	for _, candidate := range built.universe.units {
		if built.isStay(candidate) {
			built.assignFromEvidence(candidate)
		}
	}
	built.settleComponents()
	built.assignFromDependents()
}

func (built *plan) settleComponents() {
	for _, component := range built.stayComponents() {
		homes := map[string]bool{}
		evidence := map[string]bool{}
		for _, member := range component {
			if home, isSet := built.layout.Homes[member.file]; isSet {
				homes[home] = true
			}
			if built.destination[member] != "" {
				evidence[built.destination[member]] = true
			}
		}
		chosen := built.chooseComponentDestination(component, homes, evidence)
		for _, member := range component {
			if chosen != "" {
				built.destination[member] = chosen
			}
		}
	}
}

func (built *plan) chooseComponentDestination(component []*unit, homes map[string]bool, evidence map[string]bool) string {
	switch {
	case len(homes) == 1:
		return sortedKeys(homes)[0]
	case len(homes) == 0 && len(evidence) == 1:
		return sortedKeys(evidence)[0]
	case len(homes) > 1 || len(evidence) > 1:
		built.problems = append(built.problems, fmt.Sprintf("units tied by unexported names disagree on a home: %s (homes %v, users %v)", unitKey(component[0]), sortedKeys(homes), sortedKeys(evidence)))
	}
	return ""
}

func (built *plan) stayComponents() [][]*unit {
	parent := map[*unit]*unit{}
	var find func(*unit) *unit
	find = func(candidate *unit) *unit {
		if parent[candidate] == nil || parent[candidate] == candidate {
			parent[candidate] = candidate
			return candidate
		}
		parent[candidate] = find(parent[candidate])
		return parent[candidate]
	}
	for _, candidate := range built.universe.units {
		if !built.isStay(candidate) {
			continue
		}
		find(candidate)
		for dependency := range candidate.deps {
			target := built.universe.byName[dependency.space][dependency.name]
			if target != nil && built.isStay(target) && !ast.IsExported(dependency.name) {
				parent[find(candidate)] = find(target)
			}
		}
	}
	groups := map[*unit][]*unit{}
	for candidate := range parent {
		groups[find(candidate)] = append(groups[find(candidate)], candidate)
	}
	var components [][]*unit
	for _, members := range groups {
		if len(members) > 1 {
			sort.Slice(members, func(left, right int) bool { return unitKey(members[left]) < unitKey(members[right]) })
			components = append(components, members)
		}
	}
	return components
}

func (built *plan) assignFromEvidence(candidate *unit) {
	if home := built.homeOf(candidate); home != "" {
		built.destination[candidate] = home
		return
	}
	if home, isSet := built.layout.Homes[candidate.file]; isSet {
		built.destination[candidate] = home
		return
	}
	if users := built.usersOf(candidate); len(users) == 1 {
		built.destination[candidate] = sortedKeys(users)[0]
	}
}

func (built *plan) assignFromDependents() {
	for changed := true; changed; {
		changed = false
		for _, candidate := range built.universe.units {
			if !built.isStay(candidate) || built.destination[candidate] != "" {
				continue
			}
			if destination := built.destinationOfDependents(candidate); destination != "" {
				built.destination[candidate] = destination
				changed = true
			}
		}
	}
}

func (built *plan) destinationOfDependents(candidate *unit) string {
	found := map[string]bool{}
	for _, other := range built.universe.units {
		if !built.isStay(other) || built.destination[other] == "" {
			continue
		}
		for _, name := range candidate.names {
			if other.deps[symbolRef{candidate.directory, name}] {
				found[built.destination[other]] = true
			}
		}
	}
	if len(found) == 1 {
		return sortedKeys(found)[0]
	}
	return ""
}

func (built *plan) classifyTests() {
	stayDependent := map[*unit]bool{}
	for changed := true; changed; {
		changed = false
		for _, candidate := range built.universe.units {
			if candidate.isTest && !stayDependent[candidate] && built.dependsOnStay(candidate, stayDependent) {
				stayDependent[candidate] = true
				changed = true
			}
		}
	}
	protocolReachable := built.reachableTests(func(candidate *unit) bool { return !stayDependent[candidate] })
	for candidate := range protocolReachable {
		built.protocolTests[candidate] = true
	}
	built.assignStayTests(stayDependent)
}

func (built *plan) assignStayTests(stayDependent map[*unit]bool) {
	for _, entry := range built.universe.units {
		if !entry.isTest || !entry.isEntry || !stayDependent[entry] || built.inPlace[entry.directory] {
			continue
		}
		destination := built.testDestination(entry, map[*unit]bool{})
		if destination == "" {
			built.droppedTests = append(built.droppedTests, unitKey(entry))
			continue
		}
		for helper := range built.testsReachableFrom(entry) {
			if built.stayTests[helper] == nil {
				built.stayTests[helper] = map[string]bool{}
			}
			built.stayTests[helper][destination] = true
		}
	}
}

func (built *plan) testsReachableFrom(entry *unit) map[*unit]bool {
	reached := map[*unit]bool{}
	pending := []*unit{entry}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if reached[current] {
			continue
		}
		reached[current] = true
		for dependency := range current.deps {
			target := built.universe.byName[dependency.space][dependency.name]
			if target != nil && target.isTest {
				pending = append(pending, target)
			}
		}
	}
	return reached
}

func (built *plan) dependsOnStay(candidate *unit, stayDependent map[*unit]bool) bool {
	if built.inPlace[candidate.directory] {
		return false
	}
	for dependency := range candidate.deps {
		target := built.universe.byName[dependency.space][dependency.name]
		if target == nil {
			continue
		}
		if built.isStay(target) || stayDependent[target] {
			return true
		}
	}
	return false
}

func (built *plan) reachableTests(isRoot func(*unit) bool) map[*unit]bool {
	reached := map[*unit]bool{}
	var pending []*unit
	for _, candidate := range built.universe.units {
		if candidate.isTest && candidate.isEntry && isRoot(candidate) && !built.inPlace[candidate.directory] {
			pending = append(pending, candidate)
		}
	}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if reached[current] {
			continue
		}
		reached[current] = true
		for dependency := range current.deps {
			target := built.universe.byName[dependency.space][dependency.name]
			if target != nil && target.isTest {
				pending = append(pending, target)
			}
		}
	}
	return reached
}

func (built *plan) testDestination(candidate *unit, visiting map[*unit]bool) string {
	if visiting[candidate] {
		return ""
	}
	visiting[candidate] = true
	found := map[string]bool{}
	for dependency := range candidate.deps {
		target := built.universe.byName[dependency.space][dependency.name]
		switch {
		case target == nil:
		case target.isTest:
			if destination := built.testDestination(target, visiting); destination != "" {
				found[destination] = true
			}
		case built.isStay(target) && built.destination[target] != "":
			found[built.destination[target]] = true
		}
	}
	if len(found) > 1 {
		built.problems = append(built.problems, fmt.Sprintf("test %s needs stays in several packages: %v", unitKey(candidate), sortedKeys(found)))
	}
	if len(found) == 0 {
		return ""
	}
	return sortedKeys(found)[0]
}

func (built *plan) checkUnexportedReferences() {
	for _, candidate := range built.universe.units {
		if !built.isStay(candidate) && len(built.stayTests[candidate]) == 0 {
			continue
		}
		for dependency := range candidate.deps {
			built.checkReference(candidate, dependency)
		}
	}
	for _, candidate := range built.universe.units {
		if built.isStay(candidate) && built.destination[candidate] == "" {
			built.dropUnusedOrAskForHome(candidate)
		}

	}
}

func (built *plan) dropUnusedOrAskForHome(candidate *unit) {
	if len(built.usersOf(candidate)) == 0 {
		built.dropped[candidate] = true
		return
	}
	built.problems = append(built.problems, fmt.Sprintf("%s is used from %v and needs a home in the layout", unitKey(candidate), sortedKeys(built.usersOf(candidate))))
}

func (built *plan) checkDestinationsServeTheirUsers() {
	for _, candidate := range built.universe.units {
		destination := built.destination[candidate]
		if destination == "" || !built.isExistingPackage(destination) {
			continue
		}
		for user := range built.usersOf(candidate) {
			if user != destination {
				built.problems = append(built.problems, fmt.Sprintf("%s would move into %s, but %s uses it too and cannot import %s; give it a home in the layout", unitKey(candidate), destination, user, destination))
			}
		}
	}
}

func (built *plan) isExistingPackage(destination string) bool {
	matches, _ := filepath.Glob(filepath.Join(built.universe.root, destination, "*.go"))
	return len(matches) > 0
}

func (built *plan) checkReference(from *unit, dependency symbolRef) {
	target := built.universe.byName[dependency.space][dependency.name]
	if target == nil || ast.IsExported(dependency.name) || target.isTest {
		return
	}
	if built.moved[target] {
		built.problems = append(built.problems, fmt.Sprintf("%s needs the unexported %s.%s, which moved", unitKey(from), dependency.space, dependency.name))
		return
	}
	fromDestination := built.destination[from]
	if from.isTest {
		fromDestination = onlyKey(built.stayTests[from])
	}
	if built.isStay(target) && built.destination[target] != fromDestination {
		built.problems = append(built.problems, fmt.Sprintf("%s in %s needs the unexported %s.%s in %s", unitKey(from), fromDestination, dependency.space, dependency.name, built.destination[target]))
	}
}

func (built *plan) decisions() []decision {
	var decided []decision
	for _, candidate := range built.universe.units {
		entry := decision{Key: unitKey(candidate)}
		switch {
		case candidate.isTest:
			entry.Protocol = built.protocolTests[candidate]
			entry.Destinations = sortedKeys(built.stayTests[candidate])
			entry.Stay = len(entry.Destinations) > 0 || built.inPlace[candidate.directory]
		case built.moved[candidate]:
			entry.Protocol = true
		case built.inPlace[candidate.directory]:
			entry.Stay = true
		default:
			entry.Stay = built.destination[candidate] != ""
			entry.Destination = built.destination[candidate]
		}
		decided = append(decided, entry)
	}
	sort.Slice(decided, func(left, right int) bool { return decided[left].Key < decided[right].Key })
	return decided
}

func (built *plan) droppedReport() string {
	var names []string
	for candidate := range built.dropped {
		names = append(names, unitKey(candidate))
	}
	sort.Strings(names)
	sort.Strings(built.droppedTests)
	return "unused, dropped: " + strings.Join(names, " ") + "\ntests of dropped units: " + strings.Join(built.droppedTests, " ")
}

func (built *plan) carriedReport() string {
	var names []string
	for candidate := range built.moved {
		if candidate.kind == "type" && !built.direct[candidate] && ast.IsExported(candidate.names[0]) && !built.forced(candidate) {
			names = append(names, candidate.directory+"."+candidate.names[0])
		}
	}
	sort.Strings(names)
	return "exported types moved only because a shared type needs them: " + strings.Join(names, " ")
}

func (built *plan) summary() string {
	counts := map[string][2]int{}
	for _, candidate := range built.universe.units {
		if candidate.isTest {
			continue
		}
		counted := counts[candidate.directory]
		if built.moved[candidate] {
			counted[0]++
		} else {
			counted[1]++
		}
		counts[candidate.directory] = counted
	}
	directories := make([]string, 0, len(counts))
	for directory := range counts {
		directories = append(directories, directory)
	}
	sort.Strings(directories)
	var lines []string
	for _, directory := range directories {
		lines = append(lines, fmt.Sprintf("%-34s moved %4d  stays %4d", directory, counts[directory][0], counts[directory][1]))
	}
	return strings.Join(lines, "\n")
}

func onlyKey(set map[string]bool) string {
	keys := sortedKeys(set)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}
