package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"passbook/internal/config"
	"passbook/internal/importer"
	"passbook/internal/ui"

	"golang.org/x/term"
)

var version = "3.0.9"

const iCloudDataDir = "~/Library/Mobile Documents/com~apple~CloudDocs/PassBook"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	importSource := flag.String("import", "", "import entries from an external source (e.g. bitwarden)")
	enableICloud := flag.Bool("icloud", false, "set vault data directory to iCloud Drive (macOS only)")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if *enableICloud {
		setupICloud()
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

func setupICloud() {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "iCloud sync is only supported on macOS.")
		os.Exit(1)
	}

	cfg := config.LoadOrInit()
	if cfg.DataDir == iCloudDataDir {
		fmt.Println("iCloud sync is already enabled.")
		fmt.Printf("Data directory: %s\n", config.ExpandPath(iCloudDataDir))
		return
	}

	oldDir := config.ExpandPath(cfg.DataDir)
	newDir := config.ExpandPath(iCloudDataDir)

	if err := os.MkdirAll(newDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create iCloud directory: %v\n", err)
		os.Exit(1)
	}

	if err := moveDBFiles(oldDir, newDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to move database: %v\n", err)
		os.Exit(1)
	}

	cfg.DataDir = iCloudDataDir
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("iCloud sync enabled.")
	fmt.Printf("Data directory set to: %s\n", newDir)
}

func moveDBFiles(oldDir, newDir string) error {
	dbFiles := []string{"passbook.db", "passbook.db-wal", "passbook.db-shm"}
	moved := 0
	for _, name := range dbFiles {
		src := filepath.Join(oldDir, name)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		dst := filepath.Join(newDir, name)
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("destination already exists: %s", dst)
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("moving %s: %w", name, err)
		}
		moved++
		fmt.Printf("Moved %s → %s\n", src, dst)
	}
	if moved == 0 {
		fmt.Println("No existing database to move.")
	}
	return nil
}

