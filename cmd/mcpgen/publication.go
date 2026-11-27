package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/publish"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/qualify"
)

func publishChecked(ctx context.Context, repo string, report *checkReport, bootstrap bool, hook publish.Hook) (*publish.Receipt, error) {
	if !qualify.AllPassed(report.Qualification) || report.revalidate == nil {
		return nil, fmt.Errorf("generate: complete candidate qualification required")
	}
	options := publish.Options{Previous: report.previous, Files: report.files, Revalidate: report.revalidate, Hook: hook}
	if bootstrap {
		options.Bootstrap = func(ctx context.Context) error { return verifyLegacyGit(ctx, repo) }
	}
	return publish.Publish(ctx, repo, options)
}

type gitOutput struct{ body bytes.Buffer }

func (b *gitOutput) Write(body []byte) (int, error) {
	if b.body.Len()+len(body) > 8<<20 {
		return 0, fmt.Errorf("bootstrap: Git output exceeds bound")
	}
	return b.body.Write(body)
}

func gitRead(ctx context.Context, repo string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	var output gitOutput
	cmd.Stdout, cmd.Stderr, cmd.WaitDelay = &output, io.Discard, 2*time.Second
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("bootstrap: frozen Git evidence unavailable or owned output dirty: %w", err)
	}
	return output.body.Bytes(), nil
}

// verifyLegacyGit runs under the publication lock and checks independent pins,
// frozen object bytes, current regular files and both staged/unstaged state.
func verifyLegacyGit(ctx context.Context, repo string) (failure error) {
	return verifyLegacyGitReader(ctx, repo, gitRead)
}

func verifyLegacyGitReader(ctx context.Context, repo string, read func(context.Context, string, ...string) ([]byte, error)) (failure error) {
	root, err := os.OpenRoot(repo)
	if err != nil {
		return err
	}
	defer func() { failure = errors.Join(failure, root.Close()) }()
	pins := ownership.LegacyHashes()
	pins[ownership.ManifestName] = ownership.LegacyManifestSHA256
	paths := []string{}
	for _, name := range ownership.Sorted(pins) {
		logical := "protocol/" + name
		body, err := read(ctx, repo, "show", ownership.LegacyCommit+":"+logical)
		if err != nil {
			return err
		}
		current, err := ownership.Read(root, logical)
		if err != nil || ownership.Digest(body) != pins[name] || !bytes.Equal(body, current) {
			return fmt.Errorf("bootstrap: frozen/current output identity differs: %s", name)
		}
		paths = append(paths, logical)
	}
	for _, args := range [][]string{{"diff", "--quiet", "--"}, {"diff", "--cached", "--quiet", "--"}} {
		if _, err := read(ctx, repo, append(args, paths...)...); err != nil {
			return err
		}
	}
	return nil
}
