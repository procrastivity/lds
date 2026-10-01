package harness

import (
	"reflect"
	"testing"

	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
)

func TestProjectableRequiresPlumbingNamespaceAndKind(t *testing.T) {
	verbs := []manifest.Verb{
		{Name: "version", Kind: surface.Plumbing},
		{Name: "manifest", Kind: surface.Plumbing, AliasOf: "plumbing manifest"},
		{Name: "doctor", Kind: surface.Plumbing, AliasOf: "plumbing doctor"},
		{Name: "plumbing manifest", Kind: surface.Plumbing},
		{Name: "plumbing doctor", Kind: surface.Plumbing},
		{Name: "plumbing docs plan", Kind: surface.Plumbing},
		{Name: "plumbing future-llm", Kind: surface.LLM},
		{Name: "plumbing future", Kind: surface.ControlPlane},
		{Name: "docs plan", Kind: surface.Plumbing},
	}

	got := Projectable(verbs)
	var names []string
	for _, verb := range got {
		names = append(names, verb.Name)
	}
	want := []string{"plumbing manifest", "plumbing doctor", "plumbing docs plan"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("Projectable names = %v, want %v", names, want)
	}
}
