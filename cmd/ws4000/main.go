package main

import (
	"fmt"
	"os"

	"github.com/amcchord/ws4000/internal/app"
	"github.com/amcchord/ws4000/internal/config"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	flags := config.ParseFlags()
	if flags.Version {
		fmt.Printf("ws4000 %s\n", version)
		return
	}
	if flags.ListDisplays {
		config.PrintDisplays()
		return
	}

	cfgPath := flags.ConfigPath
	if cfgPath == "" {
		if path, err := config.DefaultPath(); err == nil {
			cfgPath = path
		}
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	cfg = config.Merge(cfg, flags)
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "config invalid: %v\n", err)
		os.Exit(1)
	}

	headless := flags.Screenshot != ""
	application, err := app.New(cfg, headless)
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup error: %v\n", err)
		os.Exit(1)
	}
	defer application.Close()

	if err := application.Run(flags.Screenshot, flags.Display); err != nil {
		fmt.Fprintf(os.Stderr, "runtime error: %v\n", err)
		os.Exit(1)
	}
}
