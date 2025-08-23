package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	configpkg "clean-git/internal/config"
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
		fmt.Fprintf(os.Stderr, "Usage: %s [GLOBAL OPTIONS] COMMAND [SUBCOMMAND OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Subcommands:\n")
		fmt.Fprintf(os.Stderr, "  clean     Clean up stale and merged branches\n")
		fmt.Fprintf(os.Stderr, "  config    Setup or update configuration\n")
		fmt.Fprintf(os.Stderr, "\nGlobal Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nRun '%s COMMAND -h' for subcommand options.\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --version\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --dry-run clean --local-only\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s config\n", os.Args[0])
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

	// Set up working directory

	repoRoot, err := configpkg.FindGitRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Not in a Git repository: %v\n", err)
		os.Exit(1)
	}

	configService, err := configpkg.NewService(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize configuration service: %v\n", err)
		os.Exit(1)
	}

	subcmd := flag.Arg(0)
	if subcmd == "" {
		if *config {
			handleConfigCommand(nil, configService)
			return
		}
		flag.Usage()
		os.Exit(1)
	}

	// Onboard repo if not onboarded
	if !configService.IsOnboarded() && subcmd != "config" {
		fmt.Println("Welcome to clean-git!")
		fmt.Println("It looks like this repository hasn't been configured yet.")
		fmt.Println("Let's set up the configuration to get started.")

		if err := runInteractiveConfiguration(configService); err != nil {
			fmt.Fprintf(os.Stderr, "Error during configuration setup: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\nConfiguration complete! You can now use clean-git.")
		fmt.Println("Run 'clean-git config' anytime to modify your settings.")
	}

	switch subcmd {
	case "clean":
		handleCleanCommand(flag.Args()[1:], configService)
	case "config":
		handleConfigCommand(flag.Args()[1:], configService)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n\n", subcmd)
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

// ad-hoc config flow
func handleConfigCommand(args []string, configService configpkg.Service) {
	configFlags := flag.NewFlagSet("config", flag.ExitOnError)

	configFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s config [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Set up or update clean-git configuration for this repository.\n\n")
		fmt.Fprintf(os.Stderr, "This command will guide you through configuring clean-git\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		configFlags.PrintDefaults()
	}

	configFlags.Parse(args)

	if err := runInteractiveConfiguration(configService); err != nil {
		fmt.Fprintf(os.Stderr, "Error during configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nConfiguration updated successfully!")
}

func runInteractiveConfiguration(configService configpkg.Service) error {
	reader := bufio.NewReader(os.Stdin)
	currentConfig := configService.Config()
	newConfig := &configpkg.Config{}

	fmt.Println("=== Clean-Git Configuration Setup ===")
	fmt.Println("Let's configure clean-git for your repository.\n")

	fmt.Printf("Base branches (branches to keep, comma-separated) [%s]: ", strings.Join(currentConfig.BaseBranches, ","))
	baseBranchesInput, _ := reader.ReadString('\n')
	baseBranchesInput = strings.TrimSpace(baseBranchesInput)
	if baseBranchesInput == "" {
		newConfig.BaseBranches = currentConfig.BaseBranches
	} else {
		newConfig.BaseBranches = strings.Split(baseBranchesInput, ",")
		for i, branch := range newConfig.BaseBranches {
			newConfig.BaseBranches[i] = strings.TrimSpace(branch)
		}
	}

	currentMaxAgeDays := int(currentConfig.MaxAge.Hours() / 24)
	fmt.Printf("Maximum age for stale branches (days) [%d]: ", currentMaxAgeDays)
	maxAgeInput, _ := reader.ReadString('\n')
	maxAgeInput = strings.TrimSpace(maxAgeInput)
	if maxAgeInput == "" {
		newConfig.MaxAge = currentConfig.MaxAge
	} else {
		days, err := strconv.Atoi(maxAgeInput)
		if err != nil {
			return fmt.Errorf("invalid number of days: %w", err)
		}
		newConfig.MaxAge = time.Duration(days) * 24 * time.Hour
	}

	fmt.Printf("Protected branch patterns (regex, comma-separated) [%s]: ", strings.Join(currentConfig.ProtectedRegex, ","))
	protectedInput, _ := reader.ReadString('\n')
	protectedInput = strings.TrimSpace(protectedInput)
	if protectedInput == "" {
		newConfig.ProtectedRegex = currentConfig.ProtectedRegex
	} else {
		newConfig.ProtectedRegex = strings.Split(protectedInput, ",")
		for i, pattern := range newConfig.ProtectedRegex {
			newConfig.ProtectedRegex[i] = strings.TrimSpace(pattern)
		}
	}

	fmt.Printf("Include branch patterns (regex, comma-separated) [%s]: ", strings.Join(currentConfig.IncludeRegex, ","))
	includeInput, _ := reader.ReadString('\n')
	includeInput = strings.TrimSpace(includeInput)
	if includeInput == "" {
		newConfig.IncludeRegex = currentConfig.IncludeRegex
	} else {
		newConfig.IncludeRegex = strings.Split(includeInput, ",")
		for i, pattern := range newConfig.IncludeRegex {
			newConfig.IncludeRegex[i] = strings.TrimSpace(pattern)
		}
	}

	fmt.Printf("Remote name [%s]: ", currentConfig.RemoteName)
	remoteInput, _ := reader.ReadString('\n')
	remoteInput = strings.TrimSpace(remoteInput)
	if remoteInput == "" {
		newConfig.RemoteName = currentConfig.RemoteName
	} else {
		newConfig.RemoteName = remoteInput
	}

	fmt.Println("\n=== Configuration Summary ===")
	fmt.Printf("Base branches: %s\n", strings.Join(newConfig.BaseBranches, ", "))
	fmt.Printf("Max age: %d days\n", int(newConfig.MaxAge.Hours()/24))
	fmt.Printf("Protected patterns: %s\n", strings.Join(newConfig.ProtectedRegex, ", "))
	fmt.Printf("Include patterns: %s\n", strings.Join(newConfig.IncludeRegex, ", "))
	fmt.Printf("Remote name: %s\n", newConfig.RemoteName)

	fmt.Print("\nSave this configuration? (y/N): ")
	confirmInput, _ := reader.ReadString('\n')
	confirmInput = strings.TrimSpace(strings.ToLower(confirmInput))
	if confirmInput != "y" && confirmInput != "yes" {
		fmt.Println("Configuration cancelled.")
		return nil
	}

	if err := configService.Update(newConfig); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("Configuration saved successfully!")
	return nil
}
