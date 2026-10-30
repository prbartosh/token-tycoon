// Package paths resolves the game's configuration directory.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// AppDir is the per-user directory name of the game.
const AppDir = "token-tycoon"

// ConfigDir returns the directory holding save.json and the balance override:
// %APPDATA%\token-tycoon on Windows, $XDG_CONFIG_HOME/token-tycoon or
// ~/.config/token-tycoon elsewhere (macOS included, per DESIGN section 16).
// It does not create the directory.
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
