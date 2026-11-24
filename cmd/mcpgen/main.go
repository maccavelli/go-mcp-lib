// Command mcpgen generates the framework's reviewed protocol contracts
// from the pinned MCP schema.
//
// The generator is standard-library only and has no dependency on the
// runtime module, Node, or Python. Stages are implemented across stages
// 2-4 of the framework plan; until then, invoking an unimplemented stage
// returns an explicit incomplete error and exits nonzero.
package main

import (
	"fmt"
	"os"
	"sort"
)

// stages maps a stage name to its implementation status. A stage is
// implemented only when its function is wired in; nil means the plan
// stage has not landed yet and must fail explicitly.
var stages = map[string]func([]string) error{
	"acquire":    nil,
	"load":       nil,
	"resolve":    nil,
	"model":      nil,
	"sdkmap":     nil,
	"emit":       nil,
	"qualify":    nil,
	"publish":    nil,
	"diagnostic": nil,
}

const usage = `usage: mcpgen <stage> [args]

Stages: acquire, load, resolve, model, sdkmap, emit, qualify, publish, diagnostic

Unimplemented stages exit 2 with an incomplete error.`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	name := os.Args[1]
	run, ok := stages[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "mcpgen: unknown stage %q\n%s\n", name, usage)
		os.Exit(2)
	}
	if run == nil {
		fmt.Fprintf(os.Stderr, "mcpgen: stage %q is not implemented yet; exit incomplete\n", name)
		os.Exit(2)
	}
	if err := run(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "mcpgen: %s: %v\n", name, err)
		os.Exit(1)
	}
}

// stageNames is the sorted stage list for tests and diagnostics.
func stageNames() []string {
	names := make([]string, 0, len(stages))
	for name := range stages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
