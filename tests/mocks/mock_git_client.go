package mocks

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type MockGitClient struct {
	Output string
}

func NewMockGitClient(filename string) *MockGitClient {
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Join(filepath.Dir(thisFile), "..", "fixtures")
	fixturePath := filepath.Join(baseDir, filename)

	fmt.Println("Reading fixture from:", fixturePath)

	content, err := os.ReadFile(filepath.Clean(fixturePath))
	if err != nil {
		panic("Failed to load fixture: " + err.Error())
	}
	return &MockGitClient{Output: string(content)}
}

func (m *MockGitClient) Run(args ...string) (string, error) {
	return m.Output, nil
}