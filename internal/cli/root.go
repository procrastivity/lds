// Package cli is the one registration point for every verb (C1.3): it
// builds the root Cobra command, binds the two global flags once, and maps
// whatever Execute returns to the process exit code. cmd/lds/main.go
// does nothing beyond calling into this package.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/buildinfo"
	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/surface"
	adoptverb "github.com/procrastivity/lds/internal/verbs/adopt"
	docsverb "github.com/procrastivity/lds/internal/verbs/docs"
	doctorverb "github.com/procrastivity/lds/internal/verbs/doctor"
	installverb "github.com/procrastivity/lds/internal/verbs/install"
	manifestverb "github.com/procrastivity/lds/internal/verbs/manifest"
	uninstallverb "github.com/procrastivity/lds/internal/verbs/uninstall"
	versionverb "github.com/procrastivity/lds/internal/verbs/version"
)

// NewRootCommand builds the lds root command with both global flags
// bound and every verb registered. It is the only place any verb package
// gets imported — no ad hoc init() side effects live anywhere else (C1.3).
func NewRootCommand(streams *iostreams.Streams, build buildinfo.Info) *cobra.Command {
	root := &cobra.Command{
		Use:   "lds",
		Short: "lds — Layered Documentation System CLI (scaffold)",
		// We render every error ourselves (see Execute) so human and
		// --json modes come from one code path; Cobra's own printing
		// would double up or bypass the --json envelope (C2.5).
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			jsonOut, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			verbose, err := cmd.Flags().GetBool("verbose")
			if err != nil {
				return err
			}
			cmd.SetContext(cliflags.WithFlags(cmd.Context(), cliflags.Flags{JSON: jsonOut, Verbose: verbose}))
			return nil
		},
	}
	root.SetOut(streams.Out)
	root.SetErr(streams.Err)

	root.PersistentFlags().Bool("json", false, "emit the success payload as one JSON value")
	root.PersistentFlags().BoolP("verbose", "v", false, "extra diagnostic lines on stderr")

	root.AddCommand(versionverb.Command(streams, build))
	plumbing := &cobra.Command{Use: "plumbing", Short: "deterministic commands for skills and scripts"}
	plumbing.AddCommand(adoptverb.Command(streams))
	plumbing.AddCommand(manifestverb.Command(streams, build, root))
	plumbing.AddCommand(doctorverb.Command(streams, build, root))
	plumbing.AddCommand(docsverb.Command(streams))
	root.AddCommand(plumbing)

	manifestAlias := manifestverb.Command(streams, build, root)
	surface.Alias(manifestAlias, "plumbing manifest")
	root.AddCommand(manifestAlias)
	root.AddCommand(installverb.Command(streams, build, root))
	root.AddCommand(uninstallverb.Command(streams))
	doctorAlias := doctorverb.Command(streams, build, root)
	surface.Alias(doctorAlias, "plumbing doctor")
	root.AddCommand(doctorAlias)

	// Register the tool's own verbs here, one package per verb under
	// internal/verbs/ (C1.4). Every Command constructor ends with
	// surface.Annotate — the manifest walk hard-errors without it (C3.2).

	return root
}
