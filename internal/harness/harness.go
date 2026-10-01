// Package harness holds the pure filter that selects which verbs project
// into a harness (C3.2's mechanical enforcement point), plus the shared
// comparisons every per-harness generator's callers use. The manifest's
// own --json output is never filtered — this package is the only consumer
// of the filter, and each subpackage (e.g. claudecode) is the only
// consumer of that harness's rendered output.
package harness

import (
	"strings"

	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
)

// Projectable selects canonical plumbing-kind members beneath the plumbing
// namespace (C4.3, C8.3, C8.5). Kind alone is not an audience decision:
// root porcelain stays out even when deterministic, and only qualified
// namespace members are projected.
func Projectable(verbs []manifest.Verb) []manifest.Verb {
	out := make([]manifest.Verb, 0, len(verbs))
	for _, v := range verbs {
		if v.Kind == surface.Plumbing && strings.HasPrefix(v.Name, "plumbing ") {
			out = append(out, v)
		}
	}
	return out
}
