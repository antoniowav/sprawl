package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Watcher polls the theme location. Polling two or three stat calls every
// couple of seconds is cheaper than it sounds and survives Omarchy replacing
// the whole directory on a theme switch, which trips up inotify watches.
type Watcher struct {
	Interval time.Duration
	sig      string
	next     time.Time
}

// NewWatcher records the current state as the baseline.
func NewWatcher() *Watcher {
	return &Watcher{Interval: 2 * time.Second, sig: signature()}
}

// Changed reports whether the theme changed since the last call that
// returned true. It does nothing until Interval has passed since the last poll.
func (w *Watcher) Changed(now time.Time) bool {
	if now.Before(w.next) {
		return false
	}
	w.next = now.Add(w.Interval)
	s := signature()
	if s == w.sig {
		return false
	}
	w.sig = s
	return true
}

func signature() string {
	dir := FindDir()
	if dir == "" {
		return ""
	}
	real, _ := filepath.EvalSymlinks(dir)
	sig := real
	for _, p := range []string{
		dir,
		filepath.Join(dir, "colors.toml"),
		filepath.Join(dir, "alacritty.toml"),
		filepath.Join(dir, "..", "theme.name"),
	} {
		fi, err := os.Stat(p)
		if err != nil {
			sig += "|-"
			continue
		}
		var ino uint64
		if st, ok := fi.Sys().(*syscall.Stat_t); ok {
			ino = st.Ino
		}
		sig += fmt.Sprintf("|%d:%d:%d", fi.ModTime().UnixNano(), fi.Size(), ino)
	}
	return sig
}
