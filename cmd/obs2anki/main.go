package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/glebnaz/obsidian2anki/internal/config"
	"github.com/glebnaz/obsidian2anki/internal/obsidian"
)

// Exit codes.
const (
	ExitSuccess = 0
	ExitFatal   = 1
	ExitPartial = 2
)

// GlobalFlags holds flags shared across all commands.
type GlobalFlags struct {
	Config  string
	DryRun  bool
	Verbose bool
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return ExitFatal
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "init-config":
		return runInitConfig(cmdArgs)
	case "scan":
		return runScan(cmdArgs)
	case "sync":
		return runSync(cmdArgs)
	case "tui":
		return runTUI(cmdArgs)
	case "help", "--help", "-h":
		printUsage()
		return ExitSuccess
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printUsage()
		return ExitFatal
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: obs2anki <command> [flags]

Commands:
  init-config  Create default configuration file
  scan         Scan markdown files and show status
  sync         Sync cards to Anki
  tui          Interactive terminal UI

Global flags:
  --config     Path to config file (default: ~/.config/obs2anki/config.json)
  --dry-run    Preview actions without making changes
  --verbose    Enable verbose output`)
}

func addGlobalFlags(fs *flag.FlagSet, gf *GlobalFlags) {
	fs.StringVar(&gf.Config, "config", "", "path to config file")
	fs.BoolVar(&gf.DryRun, "dry-run", false, "preview actions without making changes")
	fs.BoolVar(&gf.Verbose, "verbose", false, "enable verbose output")
}

func runInitConfig(args []string) int {
	fs := flag.NewFlagSet("init-config", flag.ContinueOnError)
	var gf GlobalFlags
	addGlobalFlags(fs, &gf)
	if err := fs.Parse(args); err != nil {
		return ExitFatal
	}

	path := gf.Config
	if err := config.InitConfig(path); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return ExitFatal
	}

	if path == "" {
		p, _ := config.DefaultConfigPath()
		path = p
	}
	fmt.Printf("config written to %s\n", path)
	return ExitSuccess
}

func runScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	var gf GlobalFlags
	addGlobalFlags(fs, &gf)
	if err := fs.Parse(args); err != nil {
		return ExitFatal
	}

	cfg, err := config.Load(gf.Config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return ExitFatal
	}

	files, err := obsidian.ScanFiles(cfg.NotesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return ExitFatal
	}

	for _, f := range files {
		fmt.Printf("%s  synced=%t  has_table=%t  cards=%d\n", f.Path, f.Synced, f.HasTable, f.CardsCount)
	}
	return ExitSuccess
}

func runSync(args []string) int {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	var gf GlobalFlags
	addGlobalFlags(fs, &gf)
	if err := fs.Parse(args); err != nil {
		return ExitFatal
	}
	fmt.Println("sync: not yet implemented")
	return ExitSuccess
}

func runTUI(args []string) int {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	var gf GlobalFlags
	addGlobalFlags(fs, &gf)
	if err := fs.Parse(args); err != nil {
		return ExitFatal
	}
	fmt.Println("tui: not yet implemented")
	return ExitSuccess
}
