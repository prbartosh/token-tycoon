// Package paths resolves the game's configuration directory.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

const AppDir = "token-tycoon"

func ConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		d, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(d, AppDir), nil
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" && filepath.IsAbs(x) {
		return filepath.Join(x, AppDir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", AppDir), nil
}
