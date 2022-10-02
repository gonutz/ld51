package main

import (
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gonutz/tiled"
)

func loadLevel(f fs.File, lev *level) {
	tileMap, err := tiled.Read(f)
	check(err)

	tileStrings := strings.Split(tileMap.Layers[0].Data.Text, ",")
	tiles := make([]int, len(tileStrings))
	for i, t := range tileStrings {
		tiles[i], _ = strconv.Atoi(t)
	}

	lev.tiles = tiles
	lev.width = tileMap.Width
	lev.tileImage = "assets/tiles.png"
	lev.tileSize = tileMap.TileWidth
	lev.tileCountX = 128 / tileMap.TileWidth

	for i := range lev.tiles {
		lev.tiles[i]--
	}

	w, h := lev.size()
out:
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if isStartTile(lev.tileAt(x, y)) {
				lev.startTileX = x
				lev.startTileY = y + 1
				lev.startFacingLeft = tileFacesLeft(lev.tileAt(x, y))

				lev.setTileAt(x, y, -1)
				lev.setTileAt(x, y-1, -1)
				lev.setTileAt(x, y-2, -1)

				break out
			}
		}
	}
}

const worldFile = "assets/world.tmx"

var lastWorldUpdate time.Time

func canUpdateLevel() bool {
	_, err := os.Stat(worldFile)
	return err == nil
}

func updateLevel(lev *level) {
	info, err := os.Stat(worldFile)
	if err != nil {
		return
	}
	if info.ModTime().After(lastWorldUpdate) {
		lastWorldUpdate = info.ModTime()
	} else {
		return
	}

	f, err := os.Open(worldFile)
	if err != nil {
		return
	}
	defer f.Close()

	loadLevel(f, lev)
}

func newLevel() *level {
	f, err := assets.Open(worldFile)
	check(err)
	defer f.Close()

	lev := &level{}
	loadLevel(f, lev)

	return lev
}

func isStartTile(tile int) bool {
	return tile == 62 || tile == 63
}

func tileFacesLeft(tile int) bool {
	return tile == 62
}

type level struct {
	tiles      []int
	width      int
	tileImage  string
	tileSize   int
	tileCountX int
	// startTileX, startTileY is the tile that our character is standing on at
	// the beginning of the level.
	startTileX      int
	startTileY      int
	startFacingLeft bool
}

func (l *level) size() (width, height int) {
	return l.width, len(l.tiles) / l.width
}

func (l *level) xyToIndex(x, y int) int {
	return x + y*l.width
}

func (l *level) tileAt(x, y int) int {
	return l.tiles[l.xyToIndex(x, y)]
}

func (l *level) setTileAt(x, y, to int) {
	l.tiles[l.xyToIndex(x, y)] = to
}

func (l *level) tileImageBounds(tile int) (x, y, width, height int) {
	x = l.tileSize * (tile % l.tileCountX)
	y = l.tileSize * (tile / l.tileCountX)
	width = l.tileSize
	height = l.tileSize
	return
}

func (l *level) collidesDownwards(x, y int) bool {
	tileX, tileY := x/tileSize, y/tileSize
	width, height := l.size()
	if x < 0 || y < 0 ||
		tileX < 0 || tileX >= width ||
		tileY < 0 || tileY >= height {
		return false
	}
	tile := l.tileAt(tileX, tileY)
	if tile < 0 {
		return false
	}

	relX := x - tileX*tileSize
	relY := y - tileY*tileSize

	switch tileWalkability[tile] {
	case top:
		return relY == 0
	case down45:
		return relX == relY
	case up45:
		return relX == tileSize-1-relY
	case topDown22_5:
		y := relY * 2
		return relX == y || relX == y+1
	case centerDown22_5:
		y := (relY - tileSize/2) * 2
		return relX == y || relX == y+1
	case bottomUp22_5:
		x := tileSize - 1 - relX
		y := (relY - tileSize/2) * 2
		return x == y || x == y+1
	case centerUp22_5:
		x := tileSize - 1 - relX
		y := relY * 2
		return x == y || x == y+1
	default:
		return false
	}
}

func (l *level) walkableAt(x, y int) bool {
	return l.tiles[l.xyToIndex(x, y)] != 0
}

type walkability int

const (
	none walkability = iota
	top
	down45
	up45
	topDown22_5
	centerDown22_5
	bottomUp22_5
	centerUp22_5
)

var tileWalkability = []walkability{
	top, top, top, top, none, none, down45, up45,
	none, none, none, none, none, none, none, none,
	none, none, none, none, topDown22_5, centerDown22_5, bottomUp22_5, centerUp22_5,
	top, top, top, top, none, none, none, none,
}
