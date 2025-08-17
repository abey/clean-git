package git

type GitClient interface {
	Run(args ...string) (string, error)
}
