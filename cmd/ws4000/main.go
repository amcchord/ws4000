package main

import (
	"fmt"
	"os"

	"github.com/amcchord/ws4000/internal/app"
	"github.com/amcchord/ws4000/internal/config"
)

func main() {
	flags := config.ParseFlags()
	if flags.ListDisplays {
		config.PrintDisplays()
		return
	}

	cfgPath := flags.ConfigPath
	if cfgPath == "" {
		path, err := config.DefaultPath()
		if err == nil {
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

	application, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup error: %v\n", err)
		os.Exit(1)
	}
	defer application.Close()

	if err := application.Run(flags.Screenshot); err != nil {
		fmt.Fprintf(os.Stderr, "runtime error: %v\n", err)
		os.Exit(1)
	}
}
