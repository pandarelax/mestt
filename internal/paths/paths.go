package paths

import (
	"os"
	"path/filepath"
)

const appName = "mestt"

type Paths struct {
	ConfigDir   string
	ConfigFile  string
	DataDir     string
	StateDir    string
	SecretsFile string
	HistoryDB   string
	LogFile     string
}

func Resolve() Paths {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	configHome := getenvDefault("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	dataHome := getenvDefault("XDG_DATA_HOME", filepath.Join(homeDir, ".local", "share"))
	stateHome := getenvDefault("XDG_STATE_HOME", filepath.Join(homeDir, ".local", "state"))

	configDir := filepath.Join(configHome, appName)
	dataDir := filepath.Join(dataHome, appName)
	stateDir := filepath.Join(stateHome, appName)

	return Paths{
		ConfigDir:   configDir,
		ConfigFile:  filepath.Join(configDir, "config.toml"),
		DataDir:     dataDir,
		StateDir:    stateDir,
		SecretsFile: filepath.Join(dataDir, "credentials.json"),
		HistoryDB:   filepath.Join(dataDir, "history.sqlite3"),
		LogFile:     filepath.Join(stateDir, "mestt.log"),
	}
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (p Paths) Ensure() error {
	for _, dir := range []string{p.ConfigDir, p.DataDir, p.StateDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
