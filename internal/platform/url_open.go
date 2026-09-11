package platform

import (
	"os/exec"
	"runtime"
)

// startCommand is overridden in tests to avoid actually launching a browser.
var startCommand = func(cmd *exec.Cmd) error { return cmd.Start() }

func OpenURL(url string) error {
	cmd := buildOpenCommand(url)
	return startCommand(cmd)
}

func buildOpenCommand(url string) *exec.Cmd {
	return buildOpenCommandForOS(runtime.GOOS, url)
}

func buildOpenCommandForOS(goos, url string) *exec.Cmd {
	switch goos {
	case "darwin":
		return exec.Command("open", url)
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return exec.Command("xdg-open", url)
	}
}
