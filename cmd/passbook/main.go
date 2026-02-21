package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"passbook/internal/config"
	"passbook/internal/importer"
	"passbook/internal/ui"

	"golang.org/x/term"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	importSource := flag.String("import", "", "import entries from an external source (e.g. bitwarden)")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if *importSource != "" {
		runImport(*importSource, flag.Args())
		return
	}

	cfg := config.LoadOrInit()

	h, err := ui.NewApp(cfg)
	if err != nil {
		panic(err)
	}

	go func() {
		for range time.Tick(1 * time.Second) {
			h.QueueUpdateDraw(func() { h.DrawTOTP() })
		}
	}()

	if err := h.Run(); err != nil {
		panic(err)
	}
}

func runImport(source string, args []string) {
	supported := map[string]string{
		"bitwarden":  ".json",
		"1password":  ".1pux",
		"lastpass":   ".csv",
	}

	ext, ok := supported[source]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unsupported import source: %q (supported: bitwarden, 1password, lastpass)\n", source)
		os.Exit(1)
	}

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: passbook --import %s <path_to_%s_file>\n", source, ext)
		os.Exit(1)
	}
	filePath := args[0]

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "File not found: %s\n", filePath)
		os.Exit(1)
	}

	fmt.Print("Master Password: ")
	pwdBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
		os.Exit(1)
	}
	password := string(pwdBytes)

	cfg := config.LoadOrInit()

	switch source {
	case "bitwarden":
		err = importer.ImportBitwarden(filePath, password, cfg)
	case "1password":
		err = importer.Import1Password(filePath, password, cfg)
	case "lastpass":
		err = importer.ImportLastPass(filePath, password, cfg)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Import failed: %v\n", err)
		os.Exit(1)
	}
}

