package platform

import (
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
