package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/config"
	"github.com/antoniowav/sprawl/internal/app"
	"github.com/antoniowav/sprawl/internal/meta"
	"github.com/antoniowav/sprawl/render/sprites"
)

func main() {
	var (
		cfgPath = flag.String("config", filepath.Join(meta.ConfigDir(), "config.toml"), "config file")
		seed    = flag.Int64("seed", 0, "map seed (0 = random)")
		shot    = flag.String("screenshot", "", "save a PNG of the first frame to this path and exit")
		actions = flag.String("actions", "", "comma-separated actions to run at startup (with -screenshot)")
		ticks   = flag.Int("ticks", 0, "simulate this many ticks after -actions (with -screenshot)")
		iconOut = flag.String("write-icon", "", "write the 256×256 app icon PNG to this path and exit")
		iconDir = flag.String("write-icons", "", "write icons at 32-512 px into this hicolor directory and exit")
		version = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()
	if *version {
		fmt.Println(meta.Name, meta.Version)
		return
	}
	if *iconDir != "" {
		for _, scale := range []int{1, 2, 4, 8, 16} {
			n := 32 * scale
			path := filepath.Join(*iconDir, fmt.Sprintf("%dx%d", n, n), "apps", meta.ID()+".png")
			if err := writePNG(path, sprites.Icon(scale)); err != nil {
				log.Fatal(err)
			}
		}
		return
	}
	if *iconOut != "" {
		if err := writePNG(*iconOut, sprites.Icon(8)); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *seed == 0 {
		*seed = time.Now().UnixNano() % 1_000_000
	}
	cfg, cfgErr := config.Load(*cfgPath)
	if path := os.Getenv("SPRAWL_CPUPROFILE"); path != "" {
		f, err := os.Create(path)
		if err == nil && pprof.StartCPUProfile(f) == nil {
			// Developer aid: profile the first 15 seconds of a session.
			time.AfterFunc(15*time.Second, func() { pprof.StopCPUProfile(); f.Close() })
		}
	}

	ebiten.SetWindowTitle(meta.Name)
	ebiten.SetWindowIcon([]image.Image{sprites.Icon(1), sprites.Icon(2), sprites.Icon(4)})
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowClosingHandled(true) // ask before losing unsaved work
	ebiten.SetRunnableOnUnfocused(!cfg.PauseUnfocused || *shot != "")

	game := app.New(app.Options{Config: cfg, ConfigPath: *cfgPath, ConfigErr: cfgErr, Seed: *seed, Screenshot: *shot, Actions: splitList(*actions), Ticks: *ticks})
	// WM_CLASS / app-id "sprawl", so Hyprland window rules can target it.
	opts := &ebiten.RunGameOptions{X11ClassName: meta.ID(), X11InstanceName: meta.ID()}
	if err := ebiten.RunGameWithOptions(game, opts); err != nil {
		log.Fatal(err)
	}
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
