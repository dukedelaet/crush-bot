package config

import (
	"os"
	"path/filepath"
)

// Paths are XDG locations for crushbot.
type Paths struct {
	ConfigDir  string
	ConfigFile string
	Home       string // CRUSHBOT_HOME
}

func ResolvePaths() Paths {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(homeDir(), ".config")
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(homeDir(), ".local", "share")
	}
	home := os.Getenv("CRUSHBOT_HOME")
	if home == "" {
		home = filepath.Join(dataHome, "crushbot")
	}
	cfgDir := filepath.Join(configHome, "crushbot")
	return Paths{
		ConfigDir:  cfgDir,
		ConfigFile: filepath.Join(cfgDir, "config.yaml"),
		Home:       home,
	}
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return "."
	}
	return h
}
