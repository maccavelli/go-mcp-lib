package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/publish"
)

// Exit codes are contract: 0 help/success, 2 usage errors, 1 failed
// validation, drift, or incomplete recovery.
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

// resultVersion is the machine-readable output envelope version.
const resultVersion = 1

// result is the envelope every command prints to stdout on success.
// Diagnostics always go to stderr.
type result struct {
	Version int    `json:"version"`
	Command string `json:"command"`
	Status  string `json:"status"`
	Data    any    `json:"data,omitempty"`
}

// printer handles --format text|json output.
type printer struct {
	format          string
	ctx             context.Context
	emitted         bool
	publicationHook publish.Hook
}

func newPrinter(format string) *printer {
	if format != "json" {
		format = "text"
	}
	return &printer{format: format, ctx: context.Background()}
}

func (p *printer) selectFormat(format string) error {
	if format != "text" && format != "json" {
		return usageError("", "--format must be text or json")
	}
	p.format = format
	return nil
}

func (p *printer) emit(command, status string, data any) error {
	p.emitted = true
	if p.format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result{Version: resultVersion, Command: command, Status: status, Data: data})
	}
	// Text format: key: value lines from a map, or plain status.
	if m, ok := data.(map[string]any); ok {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("%s: %v\n", k, m[k])
		}
		return nil
	}
	if data != nil {
		fmt.Printf("%s: %v\n", status, data)
		return nil
	}
	fmt.Println(status)
	return nil
}

// usageError prints to stderr and returns the usage exit code.
func usageError(command, message string) error {
	fmt.Fprintf(os.Stderr, "mcpgen %s: %s\n", command, message)
	fmt.Fprintf(os.Stderr, "mcpgen %s --help for usage\n", command)
	return errUsage
}

// errUsage routes a usage failure to exit 2 through the command table.
var errUsage = &usageExit{}
var errHelp = &helpExit{}

type helpExit struct{}

func (*helpExit) Error() string { return "help" }

type usageExit struct{}

func (e *usageExit) Error() string { return "usage" }

// flagSet builds a strict flag set: unknown flags fail.
func flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

// requireNoPositional rejects stray positional arguments.
func requireNoPositional(fs *flag.FlagSet) error {
	if fs.NArg() > 0 {
		return usageError(fs.Name(), fmt.Sprintf("unexpected positional arguments: %v", fs.Args()))
	}
	return nil
}

var commonUsage = `common flags:
  --format text|json   output format (default text)
  --manifest FILE     verified generation profile for inspect/check/generate

exit codes:
  0 success, 2 usage error, 1 failed validation, drift, or recovery conflict`

// commandDoc is one command's help entry.
type commandDoc struct {
	Name string
	Args string
	Help string
}

var commandDocs = []commandDoc{
	{"inspect", "[--repo DIR] [--manifest FILE] [--format text|json]", "report provenance, dispositions, mappings, unsupported features"},
	{"check", "[--repo DIR] [--manifest FILE] [--format text|json]", "render and qualify in scratch; compare bytes and inventory without publishing"},
	{"generate", "[--repo DIR] [--manifest FILE] [--bootstrap-legacy] [--format text|json]", "qualify exact candidate bytes, then publish with verified recovery"},
	{"diff", "--from M --to M [--format text|json]", "report semantic, capability, method, and inventory changes between manifests"},
	{"update", "--source URL --revision C --sha256 D --candidate DIR", "bounded network acquisition into a candidate directory; never promotes"},
	{"recover", "--action inspect|complete|restore [--repo DIR]", "inspect or resolve an interrupted publication from its journal"},
	{"diagnostic", "[--repo DIR] [--format text|json]", "report generator and environment diagnostics"},
}

func helpText() string {
	var b strings.Builder
	b.WriteString("usage: mcpgen <command> [flags]\n\ncommands:\n")
	for _, doc := range commandDocs {
		fmt.Fprintf(&b, "  %-12s %s\n\t%s\n", doc.Name, doc.Args, doc.Help)
	}
	b.WriteString("\n" + commonUsage + "\n")
	return b.String()
}
