package main

import (
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gonutz/tiled"
)

const (
	tileGuyFacingRight     = 288
	tileGuyFacingLeft      = 289
	tileJumpkinCentered    = 275
	tileJumpkinLeftAligned = 277
	tileBat                = 337
	tileCrawlerFacingLeft  = 369
	tileCrawlerFacingRight = 401
)

func loadLevel(f fs.File, lev *level) {
	info, _ := f.Stat()
	lev.modTime = info.ModTime()

	tileMap, err := tiled.Read(f)
	check(err)

	lev.tiles = parseCsvTiles(layerByName(&tileMap, "base").Data.Text)
	lev.width = tileMap.Width
	lev.tileImage = "assets/base.png"
	lev.tileSize = tileMap.TileWidth
	lev.tileCountX = 256 / tileMap.TileWidth

	lev.batStarts = lev.batStarts[:0]
	lev.leftFacingCrawlerStarts = lev.leftFacingCrawlerStarts[:0]
	lev.rightFacingCrawlerStarts = lev.rightFacingCrawlerStarts[:0]
	lev.centeredJumpkinStarts = lev.centeredJumpkinStarts[:0]
	lev.leftAlginedJumpkinStarts = lev.leftAlginedJumpkinStarts[:0]

	objects := parseCsvTiles(layerByName(&tileMap, "objects").Data.Text)
	w, h := lev.size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			tile := objects[lev.xyToIndex(x, y)]
			pos := tilePos{x: x, y: y}
			switch tile {
			case tileGuyFacingRight, tileGuyFacingLeft:
				lev.startTileX = x
				lev.startTileY = y + 1
				lev.startFacingLeft = tile == tileGuyFacingLeft
			case tileJumpkinCentered:
				lev.centeredJumpkinStarts = append(lev.centeredJumpkinStarts, pos)
			case tileJumpkinLeftAligned:
				lev.leftAlginedJumpkinStarts = append(lev.leftAlginedJumpkinStarts, pos)
			case tileBat:
				lev.batStarts = append(lev.batStarts, pos)
			case tileCrawlerFacingLeft:
				lev.leftFacingCrawlerStarts = append(lev.leftFacingCrawlerStarts, pos)
			case tileCrawlerFacingRight:
				lev.rightFacingCrawlerStarts = append(lev.rightFacingCrawlerStarts, pos)
			}
		}
	}
}

func canUpdateLevel(lev *level) bool {
	_, err := os.Stat(lev.filePath)
	return err == nil
}

func updateLevel(lev *level) {
	info, err := os.Stat(lev.filePath)
	if err != nil {
		return
	}

	if !info.ModTime().After(lev.modTime) {
		return
	}

	f, err := os.Open(lev.filePath)
	if err != nil {
		return
	}
	defer f.Close()

	loadLevel(f, lev)
}

func layerByName(m *tiled.Map, name string) *tiled.Layer {
	for i := range m.Layers {
		if m.Layers[i].Name == name {
			return &m.Layers[i]
		}
	}
	panic("no layer in map named " + name)
}

func parseCsvTiles(s string) []int {
	nums := strings.Split(s, ",")
	tiles := make([]int, len(nums))
	for i, n := range nums {
		tiles[i], _ = strconv.Atoi(n)
		tiles[i]--
	}
	return tiles
}

func newLevel(path string) *level {
	f, err := assets.Open(path)
	check(err)
	defer f.Close()

	lev := &level{filePath: path}
	loadLevel(f, lev)

	return lev
}

type level struct {
	filePath   string
	modTime    time.Time
	tiles      []int
	width      int
	tileImage  string
	tileSize   int
	tileCountX int
	// startTileX, startTileY is the tile that our character is standing on at
	// the beginning of the level.
	startTileX               int
	startTileY               int
	startFacingLeft          bool
	batStarts                []tilePos
	leftFacingCrawlerStarts  []tilePos
	rightFacingCrawlerStarts []tilePos
	centeredJumpkinStarts    []tilePos
	leftAlginedJumpkinStarts []tilePos
}

type tilePos struct {
	x, y int
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
	if 0 > tile || tile >= len(tileWalkability) {
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
	top, top, top, top, none, none, down45, up45, none, none, none, none, none, none, none, none,
	none, none, none, none, none, none, none, none, none, none, none, none, none, none, none, none,
	none, none, none, none, topDown22_5, centerDown22_5, bottomUp22_5, centerUp22_5, none, none, none, none, none, none, none, none,
	top, top, top, top, none, none, none, none, none, none, none, none, none, none, none, none,
	none, none, none, top, none, none, none, none, none, none, none, none, none, none, none, none,
	none, none, none, none, none, none, none, none, none, none, none, none, none, none, none, none,
	top, top, top, none, none, none, none, none, none, none, none, none, none, none, none, none,
	none, none, none, none, none, none, none, none, none, none, none, none, none, none, none, none,
}
