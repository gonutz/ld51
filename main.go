package main

import (
	"embed"
	"fmt"
	"io"

	"github.com/gonutz/prototype/draw"
)

//go:embed "assets/*.png"
var assets embed.FS

const (
	title           = "Ludum Dare 51 - Every 10 Seconds"
	tileSize        = 16
	visibleTilesInX = 25
	maxXSpeed       = 2
)

/*

Coordinate systems
==================

World
-----

World coordinates start top left with 0,0 and are in untransformed pixels.
X goes right, Y goes down.
One tile is 16x16 pixels in size, we are in this original pixel space, before
any scale or camera pan happens.
This is an integer pixel grid, our character moves at full integers on this
grid.

Camera
------

The camera has a center point, in world coordinates, and an integer scale
factor.

*/

type worldRect struct {
	x, y, width, height int
}

type cameraRect struct {
	x, y, width, height int
}

func newCamera() *camera {
	return &camera{}
}

type camera struct {
	// centerX, centerY are the sub-pixel camera center in world coordinates.
	centerX      float64
	centerY      float64
	scale        int
	windowWidth  int
	windowHeight int
}

func (c *camera) worldToCameraRect(p worldRect) cameraRect {
	return cameraRect{
		x:      c.windowWidth/2 + round(-float64(c.scale)*c.centerX) + c.scale*p.x,
		y:      c.windowHeight/2 + round(-float64(c.scale)*c.centerY) + c.scale*p.y,
		width:  c.scale * p.width,
		height: c.scale * p.height,
	}
}

func round(x float64) int {
	if x < 0 {
		return int(x - 0.5)
	}
	return int(x + 0.5)
}

type character struct {
	bounds     worldRect
	facingLeft bool
	speedX     int
	speedY     float64
	standIndex int
	nextStand  int
	onGround   bool
	runIndex   int
	nextRun    int
}

func (c *character) sourceBounds() worldRect {
	r := worldRect{x: 0, y: 0, width: c.bounds.width, height: c.bounds.height}
	if c.facingLeft {
		r.x = c.bounds.width
		r.width = -r.width
	}
	return r
}
func (c *character) collider() (x, y int) {
	return c.bounds.x + c.bounds.width/2, c.bounds.y + c.bounds.height - 3
}

func (c *character) move(dx, dy int) {
	c.bounds.x += dx
	c.bounds.y += dy
}

func (c *character) update() {
	c.nextStand++
	if c.nextStand >= 15 {
		c.nextStand = 0
		c.standIndex = (c.standIndex + 1) % 3
	}

	c.nextRun++
	if c.nextRun >= 4 {
		c.nextRun = 0
		c.runIndex = (c.runIndex + 1) % 8
	}

	if c.speedX == 0 {
		c.runIndex = 6
		c.nextRun = 0
	}
}

func (c *character) image() string {
	if c.onGround && c.speedX == 0 {
		return fmt.Sprintf("assets/stand%d.png", c.standIndex)
	}
	if !c.onGround && c.speedY >= 0 {
		return "assets/jump3.png"
	}
	if !c.onGround && c.speedY < 0 {
		return "assets/jump2.png"
	}

	return fmt.Sprintf("assets/run%d.png", c.runIndex)
}

func main() {
	fmt.Print()

	draw.OpenFile = func(path string) (io.ReadCloser, error) {
		return assets.Open(path)
	}

	var (
		fullscreen  = true
		world       = newLevel()
		guy         character
		cam         = newCamera()
		screenShake [][2]float64
	)

	guy.facingLeft = world.startFacingLeft
	guy.bounds.width = 3 * tileSize
	guy.bounds.height = 3 * tileSize
	guy.bounds.x = tileSize*world.startTileX + tileSize/2 - guy.bounds.width/2
	guy.bounds.y = tileSize*world.startTileY - guy.bounds.height
	guy.bounds.y += 2

	cam.centerX = float64(guy.bounds.x + guy.bounds.width/2)
	cam.centerY = float64(guy.bounds.y + guy.bounds.height/2)

	check(draw.RunWindow(title, 800, 600, func(window draw.Window) {
		if window.WasKeyPressed(draw.KeyEscape) {
			window.Close()
			return
		}

		alt := window.IsKeyDown(draw.KeyLeftAlt) ||
			window.IsKeyDown(draw.KeyRightAlt)
		enter := window.WasKeyPressed(draw.KeyEnter) ||
			window.WasKeyPressed(draw.KeyNumEnter)
		if window.WasKeyPressed(draw.KeyF11) || (alt && enter) {
			fullscreen = !fullscreen
		}

		window.SetFullscreen(fullscreen)
		window.ShowCursor(!fullscreen)

		cam.windowWidth, cam.windowHeight = window.Size()
		window.FillRect(0, 0, cam.windowWidth, cam.windowHeight, draw.RGB(0, 0.7, 1))

		cam.scale = cam.windowWidth / (visibleTilesInX * tileSize)
		if cam.scale < 1 {
			cam.scale = 1
		}

		left := window.IsKeyDown(draw.KeyLeft)
		right := window.IsKeyDown(draw.KeyRight)
		up := window.IsKeyDown(draw.KeyUp)

		xAcceleration := 0
		if left {
			xAcceleration = -1
		}
		if right {
			xAcceleration = 1
		}
		if left == right {
			// Stop if neither left nor right are pressed.
			// Also stop if left and right are pressed at the same time.
			xAcceleration = 0
			guy.speedX = 0
		}
		guy.speedX += xAcceleration
		if guy.speedX > maxXSpeed {
			guy.speedX = maxXSpeed
		}
		if guy.speedX < -maxXSpeed {
			guy.speedX = -maxXSpeed
		}
		if guy.speedX < 0 {
			guy.facingLeft = true
		}
		if guy.speedX > 0 {
			guy.facingLeft = false
		}

		// Move our guy horizontally and if we land on any piece of ground, we
		// follow its trail.
		guyX, guyY := guy.collider()
		dx := guy.speedX
		step := 1
		if dx < 0 {
			step = -1
		}
		for dx != 0 {
			guy.move(step, 0)
			dx -= step

			guyX += step
			if world.collidesDownwards(guyX, guyY+2) {
				guy.move(0, 1)
				guyY++
			} else if world.collidesDownwards(guyX, guyY) {
				guy.move(0, -1)
				guyY--
			}
		}

		guyX, guyY = guy.collider()
		guy.onGround = world.collidesDownwards(guyX, guyY+1)
		guy.speedY += 0.2
		if guy.onGround && up && guy.speedY >= 0 {
			guy.speedY = -4
		}
		dy := round(guy.speedY)
		for dy > 0 {
			guy.move(0, 1)
			if world.collidesDownwards(guy.collider()) {
				guy.move(0, -1)
				guy.speedY = 0
				dy = 0
			} else {
				dy--
			}
		}
		guy.move(0, dy)

		guyCenterX := float64(guy.bounds.x + guy.bounds.width/2)
		guyCenterY := float64(guy.bounds.y + guy.bounds.height/2)
		const camDrag = 0.15
		cam.centerX = camDrag*guyCenterX + (1.0-camDrag)*cam.centerX
		cam.centerY = camDrag*guyCenterY + (1.0-camDrag)*cam.centerY

		if len(screenShake) > 0 {
			s := screenShake[0]
			cam.centerX += s[0]
			cam.centerY += s[1]
			screenShake = screenShake[1:]
		} else if window.WasKeyPressed(draw.KeySpace) {
			// TODO For debugging we shake the screen on SPACE.
			screenShake = [][2]float64{
				{-3, 0},
				{0, -2},
				{1, 0},
				{0, 4},
				{2, 0},
				{0, -2},
				{2, -3},
				{-2, 3},
			}
		}

		guy.update()

		worldWidth, worldHeight := world.size()
		for y := 0; y < worldHeight; y++ {
			for x := 0; x < worldWidth; x++ {
				tile := world.tileAt(x, y)
				if tile < 0 {
					continue
				}
				sourceX, sourceY, sourceWidth, sourceHeight :=
					world.tileImageBounds(tile)
				dest := cam.worldToCameraRect(worldRect{
					x:      x * tileSize,
					y:      y * tileSize,
					width:  tileSize,
					height: tileSize,
				})
				window.DrawImageFilePart(
					world.tileImage,
					sourceX, sourceY, sourceWidth, sourceHeight,
					dest.x, dest.y, dest.width, dest.height,
					0,
				)
			}
		}
		guySource := guy.sourceBounds()
		guyDest := cam.worldToCameraRect(guy.bounds)
		check(window.DrawImageFilePart(
			guy.image(),
			guySource.x, guySource.y, guySource.width, guySource.height,
			guyDest.x, guyDest.y, guyDest.width, guyDest.height,
			0,
		))
	}))
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
