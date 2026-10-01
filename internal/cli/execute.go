package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/exitcode"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
	"github.com/procrastivity/lds/internal/surface"
)

// Execute runs root and returns the process exit code, per the contract's
// exit-code table (C2.4). It distinguishes a Cobra argument-parsing error
// (code 2) from a verb-raised structured error (codes 1/3/4, read off the
// error's own code field) and from a silent parity exit (the verb's own
// code, nothing rendered) — distinct return paths, not one blanket
// non-zero exit.
func Execute(root *cobra.Command, streams *iostreams.Streams) int {
	classifyRunEFailures(root)
	cmd, err := root.ExecuteC()
	if err == nil {
		return exitcode.Success
	}

	var serr *exitcode.SilentError
	if errors.As(err, &serr) {
		return serr.Code
	}

	var terr *ldserr.Error
	if errors.As(err, &terr) {
		jsonOut, _ := cmd.Flags().GetBool("json")
		ldserr.Render(streams.Err, verbPath(root, cmd), terr, jsonOut)
		return exitcode.FromError(terr)
	}

	// Anything that isn't our own structured error type reached here outside
	// a RunE callback (bad flags, unknown command, Args validation, hooks) —
	// a usage error, not a verb failure.
	_, _ = fmt.Fprintf(streams.Err, "lds: %s\n", err)
	return exitcode.Usage
}

// classifyRunEFailures preserves verb-owned structured and silent errors,
// while giving plain RunE failures the internal code and shared renderer.
// Cobra invokes Args and flag validation before RunE, so those failures
// remain on Execute's usage path.
func classifyRunEFailures(root *cobra.Command) {
	for _, cmd := range root.Commands() {
		classifyRunEFailures(cmd)
	}
	if root.RunE == nil {
		return
	}
	runE := root.RunE
	root.RunE = func(cmd *cobra.Command, args []string) error {
		err := runE(cmd, args)
		if err == nil {
			return nil
		}
		var structured *ldserr.Error
		var silent *exitcode.SilentError
		if errors.As(err, &structured) || errors.As(err, &silent) {
			return err
		}
		return ldserr.New("internal.command-failed", err.Error())
	}
}

// verbPath names the failing verb for the human-mode
// "lds: <verb>: <message>" line — the full subcommand path minus the
// root command's own name, so nested verbs (e.g. "step create") read
// naturally without redesign later.
func verbPath(root, cmd *cobra.Command) string {
	if canonical, ok := surface.AliasOf(cmd); ok {
		return canonical
	}
	path := strings.TrimPrefix(cmd.CommandPath(), root.Name())
	return strings.TrimSpace(path)
}
