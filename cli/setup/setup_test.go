package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMain(m *testing.M) {
	root, err := os.ReadFile(filepath.Join("..", "..", "config_example.yml"))
	if err != nil {
		panic(err)
	}
	SetConfigTemplate(root)
	os.Exit(m.Run())
}

type fakePrompts struct {
	lines    []string
	password string
}

func (f *fakePrompts) ReadLine(prompt string) (string, error) {
	if len(f.lines) == 0 {
		return "", nil
	}
	line := f.lines[0]
	f.lines = f.lines[1:]
	return line, nil
}

func (f *fakePrompts) ReadPassword(prompt string) (string, error) {
	return f.password, nil
}

func TestRunCreatesConfig(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "pikpakcli", "config.yml")

	path, err := Run(Options{
		TargetPath: targetPath,
		Prompts: &fakePrompts{
			lines:    []string{"user@example.com", "http://127.0.0.1:7890", "~/Downloads/pikpak-open"},
			password: "secret",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, targetPath, path)

	bs, err := os.ReadFile(targetPath)
	require.NoError(t, err)

	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(bs, &cfg))
	assert.Equal(t, "http://127.0.0.1:7890", cfg["proxy"])
	assert.Equal(t, "user@example.com", cfg["username"])
	assert.Equal(t, "secret", cfg["password"])

	openCfg, ok := cfg["open"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "~/Downloads/pikpak-open", openCfg["download_dir"])
	assert.Equal(t, []any{}, openCfg["default"])
	assert.Contains(t, string(bs), "# PikPak account username")
}

func TestRunRejectsExistingConfigWithoutForce(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(targetPath, []byte("existing"), 0600))

	_, err := Run(Options{
		TargetPath: targetPath,
		Prompts: &fakePrompts{
			lines:    []string{"user@example.com", "", ""},
			password: "secret",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	bs, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	assert.Equal(t, "existing", string(bs))
}

func TestRunOverwritesExistingConfigWithForce(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(targetPath, []byte("existing"), 0600))

	_, err := Run(Options{
		Force:      true,
		TargetPath: targetPath,
		Prompts: &fakePrompts{
			lines:    []string{"user@example.com", "", ""},
			password: "secret",
		},
	})
	require.NoError(t, err)

	bs, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(bs), "username: user@example.com")
	assert.NotEqual(t, "existing", string(bs))
}

func TestRunRequiresUsername(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "config.yml")

	_, err := Run(Options{
		TargetPath: targetPath,
		Prompts: &fakePrompts{
			lines:    []string{" ", "", ""},
			password: "secret",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
	assert.NoFileExists(t, targetPath)
}

func TestRunRequiresPassword(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "config.yml")

	_, err := Run(Options{
		TargetPath: targetPath,
		Prompts: &fakePrompts{
			lines:    []string{"user@example.com", "", ""},
			password: " ",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
	assert.NoFileExists(t, targetPath)
}

func TestRunRejectsInvalidProxy(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "config.yml")

	_, err := Run(Options{
		TargetPath: targetPath,
		Prompts: &fakePrompts{
			lines:    []string{"user@example.com", "127.0.0.1:7890", ""},
			password: "secret",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "proxy should contains ://")
	assert.NoFileExists(t, targetPath)
}

func TestConfigTemplateIsLoaded(t *testing.T) {
	assert.Contains(t, string(configTemplate), "username: xxx")
}
