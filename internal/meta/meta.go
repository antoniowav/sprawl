// Package meta holds the game's name and the paths derived from it.
// Renaming the game means changing Name (and the module path).
package meta

import (
	"os"
	"path/filepath"
	"strings"
)

// Name is the display name of the game.
const Name = "Sprawl"

// Version is overridden at build time with -ldflags "-X .../meta.Version=...".
var Version = "0.3.0-dev"

// ID is the lowercase name used for directories and the binary.
func ID() string { return strings.ToLower(Name) }

// ConfigDir is $XDG_CONFIG_HOME/<id> or ~/.config/<id>.
func ConfigDir() string { return xdg("XDG_CONFIG_HOME", ".config") }

// DataDir is $XDG_DATA_HOME/<id> or ~/.local/share/<id>.
func DataDir() string { return xdg("XDG_DATA_HOME", filepath.Join(".local", "share")) }

func xdg(env, fallback string) string {
	if v := os.Getenv(env); v != "" {
		return filepath.Join(v, ID())
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, fallback, ID())
}
