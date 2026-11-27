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
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := runCLI(ctx, os.Args[1:])
	stop()
	os.Exit(code)
}

func runCLI(ctx context.Context, args []string) int {
	return runCLIPrinter(ctx, args, newPrinter("text"))
}

func runCLIPrinter(ctx context.Context, args []string, out *printer) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, helpText())
		return exitUsage
	}
	name := args[0]
	if name == "help" || name == "--help" || name == "-h" {
		fmt.Print(helpText())
		return exitOK
	}
	run, ok := commands[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "mcpgen: unknown command %q\n\n%s\n", name, helpText())
		return exitUsage
	}
	out.ctx = ctx
	err := run(args[1:], out)
	if err == nil || errors.Is(err, errHelp) {
		return exitOK
	}
	if _, isUsage := err.(*usageExit); isUsage {
		return exitUsage
	}
	// A failed operation has a machine-readable result only when JSON was
	// selected; human diagnostics remain on stderr in every mode.
	if out.format == "json" && !out.emitted {
		if emitErr := out.emit(name, "failed", map[string]any{"error": err.Error()}); emitErr != nil {
			fmt.Fprintf(os.Stderr, "mcpgen: result write: %v\n", emitErr)
		}
	}
	fmt.Fprintf(os.Stderr, "mcpgen %s: %v\n", name, err)
	return exitFail
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
