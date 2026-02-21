package platform

import (
	"os/exec"
	"runtime"
)

func OpenURL(url string) error {
	cmd := buildOpenCommand(url)
	return cmd.Start()
}

func buildOpenCommand(url string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url)
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return exec.Command("xdg-open", url)
	}
}
