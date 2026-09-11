package platform

import (
	"errors"
	"os/exec"
	"runtime"
	"testing"
)

func TestBuildOpenCommand(t *testing.T) {
	url := "https://example.com"
	cmd := buildOpenCommand(url)

	if cmd == nil {
		t.Fatalf("expected command to be created")
	}
	if len(cmd.Args) == 0 {
		t.Fatalf("expected command args to be set")
	}

	switch runtime.GOOS {
	case "darwin":
		if cmd.Args[0] != "open" || len(cmd.Args) < 2 || cmd.Args[1] != url {
			t.Fatalf("unexpected command args: %v", cmd.Args)
		}
	case "windows":
		if cmd.Args[0] != "rundll32" || len(cmd.Args) < 3 || cmd.Args[1] != "url.dll,FileProtocolHandler" || cmd.Args[2] != url {
			t.Fatalf("unexpected command args: %v", cmd.Args)
		}
	default:
		if cmd.Args[0] != "xdg-open" || len(cmd.Args) < 2 || cmd.Args[1] != url {
			t.Fatalf("unexpected command args: %v", cmd.Args)
		}
	}
}

func TestOpenURLSuccess(t *testing.T) {
	orig := startCommand
	t.Cleanup(func() { startCommand = orig })

	var gotArgs []string
	startCommand = func(cmd *exec.Cmd) error {
		gotArgs = cmd.Args
		return nil
	}

	if err := OpenURL("https://example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArgs) == 0 {
		t.Fatalf("expected the built command to be passed to startCommand")
	}
}

func TestOpenURLPropagatesStartError(t *testing.T) {
	orig := startCommand
	t.Cleanup(func() { startCommand = orig })

	wantErr := errors.New("boom")
	startCommand = func(cmd *exec.Cmd) error { return wantErr }

	if err := OpenURL("https://example.com"); err != wantErr {
		t.Fatalf("expected propagated error, got %v", err)
	}
}

func TestBuildOpenCommandForOS(t *testing.T) {
	cases := []struct {
		goos     string
		wantArgs []string
	}{
		{"darwin", []string{"open", "https://example.com"}},
		{"windows", []string{"rundll32", "url.dll,FileProtocolHandler", "https://example.com"}},
		{"linux", []string{"xdg-open", "https://example.com"}},
		{"freebsd", []string{"xdg-open", "https://example.com"}},
	}
	for _, c := range cases {
		cmd := buildOpenCommandForOS(c.goos, "https://example.com")
		if len(cmd.Args) != len(c.wantArgs) {
			t.Fatalf("%s: expected args %v, got %v", c.goos, c.wantArgs, cmd.Args)
		}
		for i, a := range c.wantArgs {
			if cmd.Args[i] != a {
				t.Fatalf("%s: expected args %v, got %v", c.goos, c.wantArgs, cmd.Args)
			}
		}
	}
}
