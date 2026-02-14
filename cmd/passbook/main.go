package main

import (
	"time"

	"passbook/internal/config"
	"passbook/internal/ui"
)

func main() {
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
