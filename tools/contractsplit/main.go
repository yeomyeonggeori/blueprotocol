package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fatal(fmt.Errorf("usage: contractsplit plan|cut|relocate [flags]"))
	}
	flags := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	contractTree := flags.String("contracts", ".", "tree holding the contract packages as bluecollar has them")
	hostTree := flags.String("host", "", "tree whose use of the contracts decides what is shared")
	usersTree := flags.String("users", "", "tree of the code that keeps what is not shared")
	layoutPath := flags.String("layout", "", "layout JSON")
	decisionsPath := flags.String("decisions", "", "decisions JSON (written by plan, read by cut)")
	root := flags.String("root", ".", "tree to modify")
	flags.Parse(os.Args[2:])

	switch os.Args[1] {
	case "plan":
		runPlan(*contractTree, *hostTree, *usersTree, *layoutPath, *decisionsPath)
	case "cut":
		runCut(*root, *decisionsPath)
	case "relocate":
		runRelocate(*root, *hostTree, *layoutPath)
	default:
		fatal(fmt.Errorf("unknown command %s", os.Args[1]))
	}
}

func readLayout(path string) layout {
	var chosen layout
	content, errorValue := os.ReadFile(path)
	if errorValue != nil {
		fatal(errorValue)
	}
	if errorValue := json.Unmarshal(content, &chosen); errorValue != nil {
		fatal(errorValue)
	}
	return chosen
}

func reportProblems(built *plan) {
	fmt.Println(built.summary())
	fmt.Println(built.droppedReport())
	fmt.Println(built.carriedReport())
	for _, problem := range built.problems {
		fmt.Fprintln(os.Stderr, "problem:", problem)
	}
	if len(built.problems) > 0 {
		os.Exit(1)
	}
}

func runPlan(contractTree string, hostTree string, usersTree string, layoutPath string, decisionsPath string) {
	built := buildPlan(contractTree, hostTree, usersTree, readLayout(layoutPath))
	if decisionsPath != "" {
		encoded, _ := json.MarshalIndent(built.decisions(), "", "  ")
		os.WriteFile(decisionsPath, encoded, 0o644)
	}
	reportProblems(built)
}

func runCut(root string, decisionsPath string) {
	content, errorValue := os.ReadFile(decisionsPath)
	if errorValue != nil {
		fatal(errorValue)
	}
	var decided []decision
	if errorValue := json.Unmarshal(content, &decided); errorValue != nil {
		fatal(errorValue)
	}
	byKey := map[string]decision{}
	for _, entry := range decided {
		byKey[entry.Key] = entry
	}
	cutProtocol(root, byKey)
}

func runRelocate(root string, hostTree string, layoutPath string) {
	built := buildPlan(root, hostTree, root, readLayout(layoutPath))
	reportProblems(built)
	built.relocate(root)
}
