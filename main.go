package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/abey/clean-git/internal/config"
	"github.com/abey/clean-git/internal/errors"
	"github.com/abey/clean-git/internal/git"
)

const (
	Version     = "0.1.1"
	Description = "A tool for cleaning up stale and merged branches in your repository and make your life easier."
)

var (
	version    = flag.Bool("version", false, "Print version information")
	v          = flag.Bool("v", false, "Print version information (short)")
	help       = flag.Bool("help", false, "Show help information")
	h          = flag.Bool("h", false, "Show help information (short)")
	dryRun     = flag.Bool("dry-run", false, "Show what would be done without actually doing it")
	verbose    = flag.Bool("verbose", false, "Enable verbose output")
	configFlag = flag.Bool("config", false, "Show or update configuration")
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

	repoRoot, err := config.FindGitRepoRoot()
	if err != nil {
		errors.FatalError(errors.ExitGit, "Not in a Git repository: %v", err)
	}

	configService, err := config.NewService(repoRoot)
	if err != nil {
		errors.FatalError(errors.ExitConfig, "Failed to initialize configuration service: %v", err)
	}

	subcmd := flag.Arg(0)
	if subcmd == "" {
		if *configFlag {
			handleConfigCommand(nil, configService)
			return
		}
		flag.Usage()
		errors.FatalError(errors.ExitGeneral, "No command specified")
	}

	// Onboard repo if not onboarded
	if !configService.IsOnboarded() && subcmd != "config" {
		fmt.Println("Welcome to clean-git!")
		fmt.Println("It looks like this repository hasn't been configured yet.")
		fmt.Println("Let's set up the configuration to get started.")

		if err := runInteractiveConfiguration(configService); err != nil {
			errors.FatalError(errors.ExitConfig, "Configuration setup failed: %v", err)
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
		errors.FatalError(errors.ExitGeneral, "Unknown command '%s'", subcmd)
	}
}

func handleCleanCommand(args []string, configService config.Service) {
	// Parse subcommand flags
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

	if !configService.IsOnboarded() {
		errors.FatalError(errors.ExitConfig, "Repository not configured. Run 'clean-git config' first")
	}

	cfg := configService.Config()
	if cfg == nil {
		errors.FatalError(errors.ExitConfig, "Failed to load configuration")
	}

	branchService := git.NewBranchService(cfg.RemoteName)

	var qualifyingBranches []*git.Branch
	var totalProcessed int
	var errors []string

	for _, baseBranch := range cfg.BaseBranches {
		if *verbose {
			fmt.Printf("Processing base branch: %s\n", baseBranch)
		}

		mergedBranches, err := branchService.GetMergedBranches(baseBranch)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to get merged branches for %s: %v", baseBranch, err)
			errors = append(errors, errorMsg)
			if *verbose {
				fmt.Fprintf(os.Stderr, "Warning: %s\n", errorMsg)
			}
			continue
		}

		totalProcessed += len(mergedBranches)

		for _, branch := range mergedBranches {
			if branch.IsCurrent {
				if *verbose {
					fmt.Printf("Skipping current branch: %s\n", branch.Name)
				}
				continue
			}

			age := time.Since(branch.LastCommitAt)
			if age < cfg.MaxAge {
				if *verbose {
					fmt.Printf("Skipping branch %s: too recent (%s < %s)\n", branch.Name, formatDuration(age), formatDuration(cfg.MaxAge))
				}
				continue
			}

			if *localOnly && branch.IsRemote {
				if *verbose {
					fmt.Printf("Skipping remote branch %s: --local-only specified\n", branch.Name)
				}
				continue
			}
			if *remoteOnly && !branch.IsRemote {
				if *verbose {
					fmt.Printf("Skipping local branch %s: --remote-only specified\n", branch.Name)
				}
				continue
			}

			includeMatch := false
			for _, pattern := range cfg.IncludeRegex {
				matched, err := regexp.MatchString(pattern, branch.Name)
				if err != nil {
					if *verbose {
						fmt.Printf("Warning: Invalid include regex pattern '%s': %v\n", pattern, err)
					}
					continue
				}
				if matched {
					includeMatch = true
					break
				}
			}
			if !includeMatch {
				if *verbose {
					fmt.Printf("Skipping branch %s: no include pattern matches\n", branch.Name)
				}
				continue
			}

			if branchService.IsProtectedBranch(&branch, cfg.ProtectedRegex) {
				if *verbose {
					fmt.Printf("Skipping protected branch: %s\n", branch.Name)
				}
				continue
			}

			branchCopy := branch
			qualifyingBranches = append(qualifyingBranches, &branchCopy)
		}
	}

	if len(qualifyingBranches) == 0 {
		fmt.Println("No branches qualify for deletion.")
		if len(errors) > 0 {
			fmt.Printf("\nEncountered %d error(s) during processing:\n", len(errors))
			for _, err := range errors {
				fmt.Printf("  - %s\n", err)
			}
		}
		return
	}

	fmt.Printf("\nFound %d branch(es) qualifying for deletion:\n", len(qualifyingBranches))
	for _, branch := range qualifyingBranches {
		branchType := "local"
		if branch.IsRemote {
			branchType = "remote"
		}
		age := time.Since(branch.LastCommitAt)
		fmt.Printf("  - %s (%s): last commit %s ago by %s (%s)\n",
			branch.Name, branchType, formatDuration(age), branch.AuthorUserName, branch.LastCommitSHA)

		if *verbose {
			fmt.Printf("    Author email: %s\n", branch.AuthorEmail)
			if !branch.IsRemote {
				fmt.Printf("    Has unpushed commits: %v\n", branch.HasUnpushedCommits)
			}
			if branch.Remote != "" {
				fmt.Printf("    Remote: %s\n", branch.Remote)
			}
		}
	}

	if *dryRun {
		fmt.Printf("\n[DRY RUN] Would delete %d branch(es). No actual deletions performed.\n", len(qualifyingBranches))
		if len(errors) > 0 {
			fmt.Printf("\nEncountered %d error(s) during processing:\n", len(errors))
			for _, err := range errors {
				fmt.Printf("  - %s\n", err)
			}
		}
		return
	}

	fmt.Printf("\nDeleting %d branch(es)...\n", len(qualifyingBranches))
	var successCount, failCount int
	var deletionErrors []string

	for _, branch := range qualifyingBranches {
		branchType := "local"
		if branch.IsRemote {
			branchType = "remote"
		}

		if err := branchService.DeleteBranch(branch); err != nil {
			failCount++
			errorMsg := fmt.Sprintf("Failed to delete %s branch %s: %v", branchType, branch.Name, err)
			deletionErrors = append(deletionErrors, errorMsg)
			fmt.Printf("  ✗ %s\n", errorMsg)
		} else {
			successCount++
			fmt.Printf("  ✓ Deleted %s branch: %s\n", branchType, branch.Name)
		}
	}

	fmt.Printf("\n=== Deletion Summary ===\n")
	fmt.Printf("Successfully deleted: %d branch(es)\n", successCount)
	if failCount > 0 {
		fmt.Printf("Failed to delete: %d branch(es)\n", failCount)
		fmt.Println("\nDeletion errors:")
		for _, err := range deletionErrors {
			fmt.Printf("  - %s\n", err)
		}
	}

	if len(errors) > 0 {
		fmt.Printf("\nProcessing errors (%d):\n", len(errors))
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	fmt.Printf("\nProcessed %d total merged branch(es) across %d base branch(es).\n", totalProcessed, len(cfg.BaseBranches))
}

// ad-hoc config flow
func handleConfigCommand(args []string, configService config.Service) {
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
		errors.FatalError(errors.ExitConfig, "Configuration failed: %v", err)
	}

	fmt.Println("\nConfiguration updated successfully!")
}

func parseMaxAge(input string, defaultDuration time.Duration) (time.Duration, error) {
	if input == "" {
		return defaultDuration, nil
	}

	days, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid number '%s': expected integer number of days", input)
	}
	if days < 0 {
		return 0, fmt.Errorf("days must be positive, got %d", days)
	}
	return time.Duration(days) * 24 * time.Hour, nil
}

func parseCommaSeparatedList(input string, defaultList []string, validateRegex bool) ([]string, error) {
	if input == "" {
		return defaultList, nil
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		if validateRegex {
			if _, err := regexp.Compile(trimmed); err != nil {
				return nil, fmt.Errorf("invalid regex pattern '%s': %w", trimmed, err)
			}
		}

		result = append(result, trimmed)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("at least one value is required")
	}

	return result, nil
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	if hours%24 == 0 {
		return fmt.Sprintf("%d days", hours/24)
	}
	return fmt.Sprintf("%d hours", hours)
}

func runInteractiveConfiguration(configService config.Service) error {
	reader := bufio.NewReader(os.Stdin)
	currentConfig := configService.Config()
	newConfig := &config.Config{}

	fmt.Println("=== Clean-Git Configuration Setup ===")
	fmt.Println("Let's configure clean-git for your repository.")

	fmt.Printf("Base branches (branches to keep, comma-separated) [%s]: ", strings.Join(currentConfig.BaseBranches, ","))
	fmt.Println("  Press Enter to keep defaults or type comma-separated list to override")
	baseBranchesInput, _ := reader.ReadString('\n')
	baseBranchesInput = strings.TrimSpace(baseBranchesInput)

	var err error
	newConfig.BaseBranches, err = parseCommaSeparatedList(baseBranchesInput, currentConfig.BaseBranches, false)
	if err != nil {
		return fmt.Errorf("invalid base branches input: %w", err)
	}

	currentMaxAgeFormatted := formatDuration(currentConfig.MaxAge)
	fmt.Printf("Maximum age for stale branches [%s]: ", currentMaxAgeFormatted)
	fmt.Println("  Enter number of days (e.g., 30)")
	maxAgeInput, _ := reader.ReadString('\n')
	maxAgeInput = strings.TrimSpace(maxAgeInput)

	newConfig.MaxAge, err = parseMaxAge(maxAgeInput, currentConfig.MaxAge)
	if err != nil {
		return fmt.Errorf("invalid max age input: %w", err)
	}

	fmt.Printf("Protected branch patterns (regex, comma-separated) [%s]: ", strings.Join(currentConfig.ProtectedRegex, ","))
	fmt.Println("  Default patterns: release/*, hotfix/* - Press Enter to keep or edit")
	protectedInput, _ := reader.ReadString('\n')
	protectedInput = strings.TrimSpace(protectedInput)

	newConfig.ProtectedRegex, err = parseCommaSeparatedList(protectedInput, currentConfig.ProtectedRegex, true)
	if err != nil {
		return fmt.Errorf("invalid protected regex patterns: %w", err)
	}

	fmt.Printf("Include branch patterns (regex, comma-separated) [%s]: ", strings.Join(currentConfig.IncludeRegex, ","))
	fmt.Println("  Default pattern: .* (matches all) - Press Enter to keep or edit")
	includeInput, _ := reader.ReadString('\n')
	includeInput = strings.TrimSpace(includeInput)

	newConfig.IncludeRegex, err = parseCommaSeparatedList(includeInput, currentConfig.IncludeRegex, true)
	if err != nil {
		return fmt.Errorf("invalid include regex patterns: %w", err)
	}

	fmt.Printf("Remote name [%s]: ", currentConfig.RemoteName)
	fmt.Println("  Default: origin - Press Enter to keep or type new remote name")
	remoteInput, _ := reader.ReadString('\n')
	remoteInput = strings.TrimSpace(remoteInput)
	if remoteInput == "" {
		newConfig.RemoteName = currentConfig.RemoteName
	} else {
		// Basic validation for remote name (no spaces, no special chars except -_)
		if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, remoteInput); !matched {
			return fmt.Errorf("invalid remote name '%s': must contain only letters, numbers, hyphens, and underscores", remoteInput)
		}
		newConfig.RemoteName = remoteInput
	}

	fmt.Println("\n=== Configuration Summary ===")
	fmt.Printf("Base branches: %s\n", strings.Join(newConfig.BaseBranches, ", "))
	fmt.Printf("Max age: %s\n", formatDuration(newConfig.MaxAge))
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

	configPath := configService.ConfigPath()
	fmt.Println("\n=== Configuration Saved Successfully! ===")
	fmt.Printf("Configuration file: %s\n", configPath)
	fmt.Println("\nSaved configuration:")
	fmt.Printf("  • Base branches: %s\n", strings.Join(newConfig.BaseBranches, ", "))
	fmt.Printf("  • Max age: %s\n", formatDuration(newConfig.MaxAge))
	fmt.Printf("  • Protected patterns: %s\n", strings.Join(newConfig.ProtectedRegex, ", "))
	fmt.Printf("  • Include patterns: %s\n", strings.Join(newConfig.IncludeRegex, ", "))
	fmt.Printf("  • Remote name: %s\n", newConfig.RemoteName)
	fmt.Println("\nYou can now use clean-git to manage your repository branches!")
	return nil
}
