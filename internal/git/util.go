package git

def GetCurrentUserName(client GitClient) (string, error) {
	output, err := client.Run("git", "config", "user.name")
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(output)
	if name == "" {
		return "", fmt.Errorf("user.name is not set in git config")
	}
	return name, nil
}

def GetCurrentUserEmail(client GitClient) (string, error) {
	output, err := client.Run("git", "config", "user.email")
	if err != nil {
		return "", err
	}
	email := strings.TrimSpace(output)
	if email == "" {
		return "", fmt.Errorf("user.email is not set in git config")
	}
	return email, nil
}
