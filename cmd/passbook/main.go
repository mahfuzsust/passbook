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
	if source != "bitwarden" {
		fmt.Fprintf(os.Stderr, "Unsupported import source: %q (supported: bitwarden)\n", source)
		os.Exit(1)
	}

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: passbook --import bitwarden <path_to_json_file>")
		os.Exit(1)
	}
	jsonPath := args[0]

	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "File not found: %s\n", jsonPath)
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
	if err := importer.ImportBitwarden(jsonPath, password, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Import failed: %v\n", err)
		os.Exit(1)
	}
}

