package main

import (
	"embed"

	"errors"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed assets/*.png
var assets embed.FS

func main() {
	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Ludum Dare 51 - Every 10 Seconds")
	g := newGame()
	ebiten.RunGame(g)
}

func newGame() *game {
	return &game{
		tiles: ebiten.NewImageFromImage(loadImage("assets/stand1.png")),
	}
}

type game struct {
	tiles  *ebiten.Image
	dx, dy float64
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{255, 255, 255, 255})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.dx, g.dy)
	screen.DrawImage(g.tiles, op)
}

func (g *game) Layout(w, h int) (int, int) {
	return 320, 240
}

func (g *game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return errors.New("exit")
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.dx--
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.dx++
	}
	return nil
}

func loadImage(path string) image.Image {
	f, err := assets.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}

	return img
}
