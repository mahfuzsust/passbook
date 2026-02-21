package main

import (
	"flag"
	"fmt"
	"time"

	"passbook/internal/config"
	"passbook/internal/ui"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	cfg := config.LoadOrInit()

	h, err := ui.NewApp(cfg)
	if err != nil {
		panic(err)
	}

	go func() {
		for range time.Tick(1 * time.Second) {
			h.QueueUpdateDraw(func() { h.DrawTOTP() })
		}
	}()

	if err := h.Run(); err != nil {
		panic(err)
	}
}
