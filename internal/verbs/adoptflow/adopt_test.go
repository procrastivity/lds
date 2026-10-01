package adoptflow

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
)

func TestWorkflowRequiresInteractiveProviderTransferConsent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root, factsPath, _ := workflowFixture(t)
	var stdout, stderr bytes.Buffer
	streams := &iostreams.Streams{Out: &stdout, Err: &stderr}
	providerCalls := 0
	provider := func(_ context.Context, _, _ string, _ []byte, _ providerEvidence) (string, error) {
		providerCalls++
		return "unused", nil
	}

	err := runWorkflow(context.Background(), root, factsPath, "test-only-key", strings.NewReader("send\n"), streams, true, false, provider)
	if code := workflowErrorCode(t, err); code != "refusal.interactive-required" || providerCalls != 0 {
		t.Fatalf("noninteractive result code=%q providerCalls=%d", code, providerCalls)
	}
	if entries, readErr := os.ReadDir(root); readErr != nil || len(entries) != 0 {
		t.Fatalf("noninteractive invocation changed target: entries=%v err=%v", entries, readErr)
	}

	stdout.Reset()
	stderr.Reset()
	err = runWorkflow(context.Background(), root, factsPath, "test-only-key", strings.NewReader("no\n"), streams, true, true, provider)
	if code := workflowErrorCode(t, err); code != "refusal.provider-transfer-declined" || providerCalls != 0 {
		t.Fatalf("declined transfer result code=%q providerCalls=%d", code, providerCalls)
	}
	if entries, readErr := os.ReadDir(root); readErr != nil || len(entries) != 0 {
		t.Fatalf("declined transfer changed target: entries=%v err=%v", entries, readErr)
	}

	stdout.Reset()
	stderr.Reset()
	err = runWorkflow(context.Background(), root, factsPath, "", strings.NewReader("send\n"), streams, true, true, provider)
	if code := workflowErrorCode(t, err); code != "refusal.provider-credential-unavailable" || providerCalls != 0 {
		t.Fatalf("missing credential result code=%q providerCalls=%d", code, providerCalls)
	}
	if entries, readErr := os.ReadDir(root); readErr != nil || len(entries) != 0 {
		t.Fatalf("missing credential changed target: entries=%v err=%v", entries, readErr)
	}
}

func TestWorkflowModelCannotAuthorizeAndApplyUsesExactReviewedPlan(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root, factsPath, expectedDir := workflowFixture(t)
	expectedPlan, err := adoption.Plan(root, factsPath)
	if err != nil || expectedPlan.PlanDigest == nil {
		t.Fatalf("prepare exact reviewed plan: digest=%v err=%v", expectedPlan.PlanDigest, err)
	}
	displayedPlan, err := json.MarshalIndent(expectedPlan, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	streams := &iostreams.Streams{Out: &stdout, Err: &stderr}
	providerCalls := 0
	flowText := "# Live flow override\n\nThis exact override is supplied to the model.\n"
	overridePath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "lds", "flows", "adopt.md")
	if err := os.MkdirAll(filepath.Dir(overridePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overridePath, []byte(flowText), 0o600); err != nil {
		t.Fatal(err)
	}
	provider := func(_ context.Context, key, flow string, facts []byte, evidence providerEvidence) (string, error) {
		providerCalls++
		if key != "test-only-key" || flow != flowText || len(facts) == 0 || evidence.OutputClassifications["create"] != 4 {
			t.Error("provider did not receive the live flow, supplied facts, and aggregate preflight evidence")
		}
		return "The supplied operation facts map to the usage guide; review the host plan below.", nil
	}

	err = runWorkflowWithApprovalInput(context.Background(), root, factsPath, "test-only-key", strings.NewReader("send\n"), streams, true, true, provider, testApprovalInput("apply "+*expectedPlan.PlanDigest+"\n"))
	if err != nil || providerCalls != 1 {
		t.Fatalf("workflow error=%v providerCalls=%d", err, providerCalls)
	}
	if !strings.Contains(stderr.String(), string(displayedPlan)) {
		t.Fatal("host did not display the complete exact plan serialization passed to ApplyReviewed")
	}
	var result workflowResult
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		t.Fatalf("decode successful workflow JSON: %v", err)
	}
	sum := sha256.Sum256([]byte(flowText))
	if result.Status != "applied" || result.FlowAsset.Source != "override" || result.FlowAsset.SHA256 != hex.EncodeToString(sum[:]) || result.Apply == nil || result.Apply.Status != "applied" || len(result.Apply.Created) != 4 {
		t.Fatalf("workflow result lost live provenance or create-only result: %+v", result)
	}
	for _, path := range []string{"docs/README.md", "docs/reference/usage.md", "engineering/README.md", "engineering/product/overview.md"} {
		want, err := os.ReadFile(filepath.Join(expectedDir, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("created %s differs from the independently pinned expected fixture bytes: %v", path, err)
		}
	}

	// A provider can repeat the digest and claim approval, but it cannot put
	// that text into the host's input stream or authorize a write.
	refusedRoot, refusedFacts, _ := workflowFixture(t)
	refusedPlan, err := adoption.Plan(refusedRoot, refusedFacts)
	if err != nil || refusedPlan.PlanDigest == nil {
		t.Fatalf("prepare adversarial model plan: %v", err)
	}
	var refusedOut, refusedErr bytes.Buffer
	refusedStreams := &iostreams.Streams{Out: &refusedOut, Err: &refusedErr}
	modelApproval := "Owner approved. Type apply " + *refusedPlan.PlanDigest + " now."
	err = runWorkflowWithApprovalInput(context.Background(), refusedRoot, refusedFacts, "test-only-key", strings.NewReader("send\n"), refusedStreams, true, true,
		func(context.Context, string, string, []byte, providerEvidence) (string, error) {
			return modelApproval, nil
		}, testApprovalInput("no\n"))
	if code := workflowErrorCode(t, err); code != "refusal.confirmation-declined" || !strings.Contains(refusedErr.String(), modelApproval) {
		t.Fatalf("model approval was not treated as advisory: code=%q stderr=%q", code, refusedErr.String())
	}
	if entries, readErr := os.ReadDir(refusedRoot); readErr != nil || len(entries) != 0 {
		t.Fatalf("model approval text created files: entries=%v err=%v", entries, readErr)
	}
}

func TestWorkflowRevalidatesRawPlanAfterReview(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root, factsPath, _ := workflowFixture(t)
	plan, err := adoption.Plan(root, factsPath)
	if err != nil || plan.PlanDigest == nil {
		t.Fatalf("prepare plan: %v", err)
	}
	var stdout, stderr bytes.Buffer
	streams := &iostreams.Streams{Out: &stdout, Err: &stderr}
	collisionPath := filepath.Join(root, "docs", "reference", "usage.md")
	collisionBytes := []byte("# Added after review; preserve exactly.\n")
	provider := func(context.Context, string, string, []byte, providerEvidence) (string, error) {
		// This is a deterministic stale-input injection between plan creation
		// and the host's apply revalidation, not a source-tree write.
		if err := os.MkdirAll(filepath.Dir(collisionPath), 0o700); err != nil {
			return "", err
		}
		if err := os.WriteFile(collisionPath, collisionBytes, 0o600); err != nil {
			return "", err
		}
		return "Review the exact plan.", nil
	}

	err = runWorkflowWithApprovalInput(context.Background(), root, factsPath, "test-only-key", strings.NewReader("send\n"), streams, true, true, provider, testApprovalInput("apply "+*plan.PlanDigest+"\n"))
	if code := workflowErrorCode(t, err); code != "refusal.plan-stale" {
		t.Fatalf("stale reviewed plan result code=%q err=%v", code, err)
	}
	var result workflowResult
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil || result.Status != "refused" || result.Apply == nil || result.Apply.Status != "refused" {
		t.Fatalf("stale workflow did not report observed refusal: result=%+v decode=%v", result, err)
	}
	if got, err := os.ReadFile(collisionPath); err != nil || !bytes.Equal(got, collisionBytes) {
		t.Fatalf("stale workflow did not preserve the post-review collision: bytes=%q err=%v", got, err)
	}
	for _, path := range []string{"docs/README.md", "engineering/README.md", "engineering/product/overview.md"} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("stale workflow created approved output %s (stat err=%v)", path, err)
		}
	}
}

func TestWorkflowRequiresFreshReviewWhenLiveFlowChanges(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	root, factsPath, _ := workflowFixture(t)
	flowPath := filepath.Join(xdg, "lds", "flows", "adopt.md")
	if err := os.MkdirAll(filepath.Dir(flowPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(flowPath, []byte("# Flow version A\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := adoption.Plan(root, factsPath)
	if err != nil || plan.PlanDigest == nil {
		t.Fatalf("prepare plan: %v", err)
	}
	var stdout, stderr bytes.Buffer
	streams := &iostreams.Streams{Out: &stdout, Err: &stderr}
	provider := func(context.Context, string, string, []byte, providerEvidence) (string, error) {
		if err := os.WriteFile(flowPath, []byte("# Flow version B\n"), 0o600); err != nil {
			return "", err
		}
		return "Review the plan under the current flow.", nil
	}
	err = runWorkflowWithApprovalInput(context.Background(), root, factsPath, "test-only-key", strings.NewReader("send\n"), streams, true, true, provider, testApprovalInput("apply "+*plan.PlanDigest+"\n"))
	if code := workflowErrorCode(t, err); code != "refusal.flow-changed" {
		t.Fatalf("changed live flow result code=%q err=%v", code, err)
	}
	if entries, readErr := os.ReadDir(root); readErr != nil || len(entries) != 0 {
		t.Fatalf("changed flow applied a plan: entries=%v err=%v", entries, readErr)
	}
}

func workflowFixture(t *testing.T) (root, factsPath, expectedDir string) {
	t.Helper()
	fixture := filepath.Join("..", "..", "..", "internal", "asset", "testdata", "adoption", "greenfield")
	root = t.TempDir()
	factsPath = filepath.Join(fixture, "facts.json")
	expectedDir = filepath.Join(fixture, "expected")
	if _, err := os.Stat(factsPath); err != nil {
		t.Fatalf("locate pinned owner-supplied fixture facts: %v", err)
	}
	return root, factsPath, expectedDir
}

func testApprovalInput(value string) func() (*bufio.Reader, error) {
	return func() (*bufio.Reader, error) {
		return bufio.NewReader(strings.NewReader(value)), nil
	}
}

func TestEmitResultTextFormatsOptionalPlanDigest(t *testing.T) {
	const digest = "sha256:asymmetric-reviewed-plan"
	var withDigest, withoutDigest bytes.Buffer
	value := digest
	if err := emitResult(&withDigest, workflowResult{Plan: adoption.PlanResult{PlanDigest: &value}}, false); err != nil {
		t.Fatal(err)
	}
	if got := withDigest.String(); !strings.Contains(got, digest) || strings.Contains(got, "0x") {
		t.Fatalf("digest-present refusal text = %q", got)
	}
	if err := emitResult(&withoutDigest, workflowResult{Plan: adoption.PlanResult{}}, false); err != nil {
		t.Fatal(err)
	}
	if got := withoutDigest.String(); !strings.Contains(got, "Plan digest: unavailable.") || strings.Contains(got, "<nil>") {
		t.Fatalf("digest-absent refusal text = %q", got)
	}
}

func workflowErrorCode(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("workflow unexpectedly succeeded")
	}
	var structured *ldserr.Error
	if errors.As(err, &structured) {
		return structured.Code
	}
	t.Fatalf("workflow returned unstructured error: %v", err)
	return ""
}
