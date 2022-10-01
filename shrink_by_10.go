//+build ignore

package main

import (
	"image"
	"image/png"
	"os"
)

const scale = 10

func main() {
	for _, path := range os.Args[1:] {
		img, err := load(path)
		check(err)
		b := img.Bounds()
		out := image.NewRGBA(image.Rect(0, 0, b.Dx()/scale, b.Dy()/scale))
		for y := 0; y < out.Rect.Max.Y; y++ {
			for x := 0; x < out.Rect.Max.X; x++ {
				out.Set(x, y, img.At(b.Min.X+x*scale, b.Min.Y+y*scale))
			}
		}
		save(out, path)
	}
}

func load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func save(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
