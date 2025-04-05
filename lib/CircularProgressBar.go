package lib

import (
	"image"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type CircularProgressBar struct {
	widget.BaseWidget
	progress float64 // 0.0 - 1.0
	raster   *canvas.Raster
}

func NewCircularProgressBar() *CircularProgressBar {
	c := &CircularProgressBar{}
	c.raster = canvas.NewRaster(c.draw)
	c.ExtendBaseWidget(c)
	return c
}

func (c *CircularProgressBar) SetProgress(p float64) {
	if p < 0 {
		p = 0
	} else if p > 1 {
		p = 1
	}
	c.progress = p
	canvas.Refresh(c.raster)
}

func (c *CircularProgressBar) CreateRenderer() fyne.WidgetRenderer {
	return &circularRenderer{c: c, raster: c.raster, objects: []fyne.CanvasObject{c.raster}}
}

type circularRenderer struct {
	c       *CircularProgressBar
	raster  *canvas.Raster
	objects []fyne.CanvasObject
}

func (r *circularRenderer) Layout(size fyne.Size) {
	r.raster.Resize(size)
}

func (r *circularRenderer) MinSize() fyne.Size {
	return fyne.NewSize(50, 50)
}

func (r *circularRenderer) Refresh() {
	canvas.Refresh(r.raster)
}

func (r *circularRenderer) Destroy() {}

func (r *circularRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (c *CircularProgressBar) draw(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	centerX := float64(w) / 2
	centerY := float64(h) / 2
	radius := math.Min(centerX, centerY) * 0.9
	thickness := radius * 0.2

	angleLimit := c.progress * 2 * math.Pi

	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			dist := math.Hypot(dx, dy)

			if dist < radius && dist > (radius-thickness) {
				angle := math.Atan2(dy, dx)
				if angle < 0 {
					angle += 2 * math.Pi
				}
				if angle <= angleLimit {
					img.Set(x, y, color.RGBA{0x33, 0x99, 0xff, 0xff}) // filled
				} else {
					img.Set(x, y, color.RGBA{0xdd, 0xdd, 0xdd, 0xff}) // background
				}
			} else {
				img.Set(x, y, color.RGBA{0, 0, 0, 0}) // transparent
			}
		}
	}
	return img
}
