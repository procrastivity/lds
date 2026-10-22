//go:build linux

package adoptflow

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/iostreams"
)

func TestCommandApprovalInputStartsFreshPTYEpoch(t *testing.T) {
	for _, test := range []struct {
		name         string
		queueEarly   bool
		approveFresh bool
	}{
		{name: "early-pasted-digest-is-discarded", queueEarly: true},
		{name: "fresh-post-prompt-digest-authorizes", approveFresh: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			t.Setenv(apiKeyEnv, "test-only-key")
			root, factsPath, expectedDir := workflowFixture(t)
			expectedPlan, err := adoption.Plan(root, factsPath, "")
			if err != nil || expectedPlan.PlanDigest == nil {
				t.Fatalf("precompute expected plan digest: digest=%v err=%v", expectedPlan.PlanDigest, err)
			}

			master, slave := openTestPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := Command(streams)
			command.SilenceUsage = true
			command.SilenceErrors = true
			command.SetIn(slave)
			command.SetArgs([]string{root, "--facts", factsPath})

			entered := make(chan struct{}, 1)
			release := make(chan struct{})
			released := false
			releaseProvider := func() {
				if !released {
					close(release)
					released = true
				}
			}
			t.Cleanup(releaseProvider)
			previousTransport := http.DefaultTransport
			http.DefaultTransport = approvalTestTransport{entered: entered, release: release}
			t.Cleanup(func() { http.DefaultTransport = previousTransport })

			input := "send\n"
			if test.queueEarly {
				input += "apply " + *expectedPlan.PlanDigest + "\n"
			}
			if _, err := unix.Write(int(master.Fd()), []byte(input)); err != nil {
				t.Fatalf("stage terminal input: %v", err)
			}
			done := make(chan error, 1)
			go func() { done <- command.Execute() }()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("provider transport was not reached")
			}

			beforeRelease, err := drainPTY(master)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(beforeRelease), "Complete deterministic plan") {
				t.Fatal("complete plan was displayed while the provider transport was still blocked")
			}
			releaseProvider()

			prompt := "To authorize only this displayed plan, type exactly: apply " + *expectedPlan.PlanDigest
			output, err := waitForPTYText(master, beforeRelease, prompt)
			if err != nil {
				t.Fatalf("wait for exact approval prompt: %v; observed %q", err, output)
			}
			if !strings.Contains(string(output), "Complete deterministic plan") {
				t.Fatal("approval prompt appeared without the complete plan")
			}

			if test.approveFresh {
				if _, err := unix.Write(int(master.Fd()), []byte("apply "+*expectedPlan.PlanDigest+"\n")); err != nil {
					t.Fatalf("provide fresh approval input: %v", err)
				}
				select {
				case err := <-done:
					if err != nil {
						t.Fatalf("fresh approval command failed: %v", err)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("command did not finish after fresh approval")
				}
				if !strings.Contains(stdout.String(), "Adoption applied; 4 file(s) created") {
					t.Fatalf("fresh approval result = %q", stdout.String())
				}
				for _, path := range []string{"docs/README.md", "docs/reference/usage.md", "engineering/README.md", "engineering/product/overview.md"} {
					want, err := os.ReadFile(filepath.Join(expectedDir, filepath.FromSlash(path)))
					if err != nil {
						t.Fatal(err)
					}
					got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
					if err != nil || !bytes.Equal(got, want) {
						t.Errorf("created %s differs from independent fixture: %v", path, err)
					}
				}
				return
			}

			select {
			case err := <-done:
				t.Fatalf("pre-display paste authorized or prematurely ended command: %v", err)
			case <-time.After(150 * time.Millisecond):
			}
			for _, path := range []string{"docs/README.md", "docs/reference/usage.md", "engineering/README.md", "engineering/product/overview.md"} {
				if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !os.IsNotExist(err) {
					t.Fatalf("early input created %s (lstat err=%v)", path, err)
				}
			}
			if _, err := unix.Write(int(master.Fd()), []byte("no\n")); err != nil {
				t.Fatalf("end refusal test with fresh negative input: %v", err)
			}
			select {
			case err := <-done:
				if workflowErrorCode(t, err) != "refusal.confirmation-declined" {
					t.Fatalf("fresh negative approval code: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("command did not finish after fresh negative input")
			}
			for _, path := range []string{"docs/README.md", "docs/reference/usage.md", "engineering/README.md", "engineering/product/overview.md"} {
				if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !os.IsNotExist(err) {
					t.Errorf("discarded early input/refusal created %s (lstat err=%v)", path, err)
				}
			}
		})
	}
}

type approvalTestTransport struct {
	entered chan struct{}
	release <-chan struct{}
}

func (transport approvalTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	select {
	case transport.entered <- struct{}{}:
	default:
	}
	select {
	case <-transport.release:
	case <-request.Context().Done():
		return nil, request.Context().Err()
	}
	body := `{"model":"deepseek-v4-flash","choices":[{"message":{"content":"Review the complete host plan."}}]}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}, nil
}

func openTestPTY(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	fd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatalf("open pty master: %v", err)
	}
	master := os.NewFile(uintptr(fd), "test-pty-master")
	t.Cleanup(func() { _ = master.Close() })
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatalf("unlock pty slave: %v", err)
	}
	number, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		t.Fatalf("get pty slave number: %v", err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatalf("open pty slave: %v", err)
	}
	t.Cleanup(func() { _ = slave.Close() })
	if err := unix.SetNonblock(fd, true); err != nil {
		t.Fatalf("set pty master nonblocking: %v", err)
	}
	return master, slave
}

func drainPTY(master *os.File) ([]byte, error) {
	var output []byte
	buffer := make([]byte, 8192)
	for {
		n, err := unix.Read(int(master.Fd()), buffer)
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			return output, nil
		}
		if err != nil {
			return output, err
		}
		if n == 0 {
			return output, nil
		}
		output = append(output, buffer[:n]...)
	}
}

func waitForPTYText(master *os.File, output []byte, expected string) ([]byte, error) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		more, err := drainPTY(master)
		if err != nil {
			return output, err
		}
		output = append(output, more...)
		if strings.Contains(string(output), expected) {
			return output, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return output, fmt.Errorf("timed out waiting for terminal text %q", expected)
}
