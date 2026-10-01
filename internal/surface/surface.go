// Package surface names the three manifest surface kinds every verb declares
// on its registration (CONTRACT.md C3.2) and records alias links on the
// porcelain registration (C3.9). Both live on Cobra commands so consumers
// walk one registration tree rather than maintaining parallel registries.
package surface

import "github.com/spf13/cobra"

// Kind is a verb's manifest surface kind.
type Kind string

const (
	// Plumbing verbs are deterministic: JSON + exit codes, no LLM. A verb
	// projects into a harness only when both its kind and qualified namespace
	// are plumbing.
	Plumbing Kind = "plumbing"
	// LLM verbs do CLI-side LLM shaping; no MCP. Absent from harness
	// projections because inside a harness the model already is the LLM.
	LLM Kind = "llm"
	// ControlPlane verbs require a live MCP surface; they exist only in the
	// agent porcelain.
	ControlPlane Kind = "control-plane"
)

// annotationKind is the cobra.Command.Annotations key a verb's kind is
// recorded under. The registration lives on the command itself so the
// manifest generator can walk the command tree without a parallel registry.
const (
	annotationKind    = "lds.surface-kind"
	annotationAliasOf = "lds.alias-of"
)

// Annotate records kind as cmd's explicit, machine-readable surface kind.
func Annotate(cmd *cobra.Command, kind Kind) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[annotationKind] = string(kind)
}

// Of returns the kind recorded on cmd by Annotate, and whether one was found.
func Of(cmd *cobra.Command) (Kind, bool) {
	k, ok := cmd.Annotations[annotationKind]
	return Kind(k), ok
}

// Alias records the canonical qualified command path for a compatibility
// alias. Call it only on the porcelain member of an alias pair.
func Alias(cmd *cobra.Command, canonical string) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[annotationAliasOf] = canonical
}

// AliasOf returns the canonical qualified path recorded on an alias
// registration, if cmd is the porcelain member of an alias pair.
func AliasOf(cmd *cobra.Command) (string, bool) {
	path, ok := cmd.Annotations[annotationAliasOf]
	return path, ok
}
