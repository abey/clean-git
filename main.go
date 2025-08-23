package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	Version     = "0.1.0"
	Description = "A tool for cleaning up stale and merged branches in your repository and make you life easier."
)

var (
	version = flag.Bool("version", false, "Print version information")
	v       = flag.Bool("v", false, "Print version information (short)")
	help    = flag.Bool("help", false, "Show help information")
	h       = flag.Bool("h", false, "Show help information (short)")
	dryRun  = flag.Bool("dry-run", false, "Show what would be done without actually doing it")
	verbose = flag.Bool("verbose", false, "Enable verbose output")
	config  = flag.Bool("config", false, "Show or update configuration")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n\n", Description)
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] COMMAND\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  clean     Clean up stale and merged branches\n")
		fmt.Fprintf(os.Stderr, "  config   	Setup or update configuration\n")
		fmt.Fprintf(os.Stderr, "\nGlobal Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --version\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s clean --dry-run\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s onboard\n", os.Args[0])
	}

	flag.Parse()

	if *version || *v {
		fmt.Printf("clean-git version %s\n", Version)
		return
	}
	if *help || *h {
		flag.Usage()
		return
	}
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := args[0]
	switch command {
	case "clean":
		handleCleanCommand(args[1:])
	case "config":
		handleConfigCommand(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

func handleCleanCommand(args []string) {
	cleanFlags := flag.NewFlagSet("clean", flag.ExitOnError)
	localOnly := cleanFlags.Bool("local-only", false, "Only clean local branches")
	remoteOnly := cleanFlags.Bool("remote-only", false, "Only clean remote branches")

	cleanFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s clean [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Clean up stale and merged branches.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		cleanFlags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nGlobal options like --dry-run, --verbose are also available.\n")
	}

	cleanFlags.Parse(args)

	fmt.Println("Clean command executed with options:")
	fmt.Printf("  Dry run: %v\n", *dryRun)
	fmt.Printf("  Verbose: %v\n", *verbose)
	fmt.Printf("  Local only: %v\n", *localOnly)
	fmt.Printf("  Remote only: %v\n", *remoteOnly)
}

func handleConfigCommand(args []string) {
	configFlags := flag.NewFlagSet("config", flag.ExitOnError)

	configFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s config [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Set up or update clean-git configuration for this repository.\n\n")
		fmt.Fprintf(os.Stderr, "This command will guide you through configuring clean-git\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		configFlags.PrintDefaults()
	}

	configFlags.Parse(args)
}
