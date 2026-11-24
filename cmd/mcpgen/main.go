// Command mcpgen generates the framework's reviewed protocol contracts
// from the pinned MCP schema.
//
// Commands: inspect, check, generate, diff, update, recover, diagnostic.
// Help exits 0; usage errors exit 2; failed validation, drift, or
// incomplete recovery exit 1. Diagnostics go to stderr; machine-readable
// results go to stdout with a format version. The generator is standard
// library only and has no dependency on the runtime module, Node, or
// Python.
package main

import (
	"fmt"
	"os"
	"sort"
)

// commands maps a command name to its implementation.
var commands = map[string]func([]string, *printer) error{
	"inspect":    cmdInspect,
	"check":      cmdCheck,
	"generate":   cmdGenerate,
	"diff":       cmdDiff,
	"update":     cmdUpdate,
	"recover":    cmdRecover,
	"diagnostic": cmdDiagnostic,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, helpText())
		os.Exit(exitUsage)
	}
	name := os.Args[1]
	if name == "help" || name == "--help" || name == "-h" {
		fmt.Print(helpText())
		os.Exit(exitOK)
	}
	run, ok := commands[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "mcpgen: unknown command %q\n\n%s\n", name, helpText())
		os.Exit(exitUsage)
	}
	out := newPrinter("text")
	err := run(os.Args[2:], out)
	if err == nil {
		os.Exit(exitOK)
	}
	if _, isUsage := err.(*usageExit); isUsage {
		os.Exit(exitUsage)
	}
	fmt.Fprintf(os.Stderr, "mcpgen %s: %v\n", name, err)
	os.Exit(exitFail)
}

// commandNames is the sorted command list for tests.
func commandNames() []string {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
