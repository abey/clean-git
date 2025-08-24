# clean-git

A tool for cleaning up stale and merged branches in your Git repository.

## Installation

Install clean-git using Go's built-in package manager:

```bash
go install github.com/abey/clean-git@latest
```

## Usage

After installation, you can use `clean-git` from anywhere in your terminal:

```bash
# Show version
clean-git --version

# Configure clean-git for a repository (run from within a Git repo)
clean-git config

# Clean branches with dry-run (recommended first)
clean-git --dry-run clean

# Actually clean branches
clean-git clean

# Clean only local branches
clean-git clean --local-only

# Clean only remote branches  
clean-git clean --remote-only

# Verbose output
clean-git --verbose clean
```

## Configuration

Run `clean-git config` in any Git repository to set up:

- **Base branches**: Branches to keep (e.g., main, develop)
- **Max age**: How old branches must be before deletion
- **Protected patterns**: Regex patterns for branches to never delete
- **Include patterns**: Regex patterns for branches to consider for deletion
- **Remote name**: Name of your Git remote (usually "origin")

## Requirements

- Go 1.22 or later
- Git repository