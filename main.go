package main

import (
	"embed"
	"fmt"
	"io"
	"math"

	"github.com/gonutz/prototype/draw"
)

//go:embed assets/*.png assets/*.tmx
var assets embed.FS

const (
	title           = "Ludum Dare 51 - Every 10 Seconds"
	tileSize        = 16
	visibleTilesInX = 25
	maxXSpeed       = 2
	jumpSpeed       = -2.5
	jumpHoldDelay   = 10
	edgeJumpLeeway  = 50
	gravity         = 0.18
	batRadius       = 3 * tileSize
	batSpeed        = 2
	cameraDrag      = 0.15
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

type worldPoint struct {
	x, y int
}

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
	jumpSince  int
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
	if c.nextRun >= 5 {
		c.nextRun = 0
		c.runIndex = (c.runIndex + 1) % 8
	}

	if c.speedX == 0 {
		c.runIndex = 6
		c.nextRun = 0
	}

	c.jumpSince++
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

type bat struct {
	center    worldPoint
	angle     int
	frame     int
	nextFrame int
}

func (b *bat) bounds() worldRect {
	dy, dx := math.Sincos(degToRad(float64(b.angle)))
	return worldRect{
		x:      b.center.x + round(dx*batRadius) - 48/2,
		y:      b.center.y + round(dy*batRadius) - 48/2,
		width:  48,
		height: 48,
	}
}

func degToRad(deg float64) float64 {
	return deg / 180 * math.Pi
}

func (b *bat) sourceBounds() worldRect {
	return worldRect{x: 0, y: 0, width: 48, height: 48}
}

func (b *bat) update() {
	b.angle = (b.angle + batSpeed) % 360

	b.nextFrame++
	if b.nextFrame >= 9 {
		b.nextFrame = 0
		b.frame = (b.frame + 1) % 4
	}
}

func (b *bat) image() string {
	return fmt.Sprintf("assets/bat%d.png", b.frame)
}

type jumpkinState int

const (
	jumpkinJumping jumpkinState = iota
	jumpkinBouncing
)

type jumpkin struct {
	foot   worldPoint
	speedY float64
	dy     float64
	bounce int
	state  jumpkinState
}

func (j *jumpkin) update() {
	switch j.state {
	case jumpkinJumping:
		j.dy += j.speedY
		j.speedY += 0.13
		if j.dy >= 0 {
			j.state = jumpkinBouncing
			j.bounce = 0
		}
	case jumpkinBouncing:
		j.bounce++
		if 6 <= j.bounce && j.bounce <= 10 {
			j.dy = -1
		}
		if j.bounce >= 10 {
			j.speedY = -4.7
			j.dy = 0
			j.state = jumpkinJumping
		}
	}
}

func (j *jumpkin) bounds() worldRect {
	return worldRect{
		x:      j.foot.x - 48/2,
		y:      j.foot.y - 48 + round(j.dy),
		width:  48,
		height: 48,
	}
}

func (j *jumpkin) sourceBounds() worldRect {
	return worldRect{x: 0, y: 0, width: 48, height: 48}
}

func (j *jumpkin) image() string {
	if j.state == jumpkinJumping {
		if j.speedY < -1.5 {
			return "assets/jumpkin4.png"
		}
		if j.speedY <= 1 {
			return "assets/jumpkin0.png"
		}
		return "assets/jumpkin0.png"
	} else {
		if j.bounce > 6 {
			return "assets/jumpkin2.png"
		}
		return "assets/jumpkin3.png"
	}
}

func main() {
	fmt.Print()

	draw.OpenFile = func(path string) (io.ReadCloser, error) {
		return assets.Open(path)
	}

	var (
		fullscreen  = true
		world       = newLevel("assets/world.tmx")
		guy         character
		cam         = newCamera()
		screenShake [][2]float64
		bats        []bat
		jumpkins    []jumpkin
		wasUp       bool
	)

	guy.facingLeft = world.startFacingLeft
	guy.bounds.width = 3 * tileSize
	guy.bounds.height = 3 * tileSize
	guy.bounds.x = tileSize*world.startTileX + tileSize/2 - guy.bounds.width/2
	guy.bounds.y = tileSize*world.startTileY - guy.bounds.height
	guy.bounds.y += 2

	cam.centerX = float64(guy.bounds.x + guy.bounds.width/2)
	cam.centerY = float64(guy.bounds.y + guy.bounds.height/2)

	for i, startTile := range world.batStarts {
		bats = append(bats, bat{
			center: worldPoint{
				x: startTile.x * tileSize,
				y: startTile.y*tileSize + tileSize - batRadius - 24,
			},
			frame:     i % 4,
			nextFrame: 3 * i,
		})
	}

	addJumpkin := func(x, y int) {
		jumpkins = append(jumpkins, jumpkin{foot: worldPoint{x: x, y: y + 3}})
	}
	for _, startTile := range world.centeredJumpkinStarts {
		addJumpkin(startTile.x*tileSize+tileSize/2, (startTile.y+1)*tileSize)
	}
	for _, startTile := range world.leftAlginedJumpkinStarts {
		addJumpkin((1+startTile.x)*tileSize, (startTile.y+1)*tileSize)
	}

	shakeScreen := func(intensity float64) {
		if len(screenShake) == 0 {
			x := intensity
			screenShake = [][2]float64{
				{-3 * x, 0},
				{0, -2 * x},
				{1 * x, 0},
				{0, 4 * x},
				{2 * x, 0},
				{0, -2 * x},
				{2 * x, -3 * x},
				{-2 * x, 3 * x},
			}
		}
	}

	updateMapOnSave := canUpdateLevel(world)

	check(draw.RunWindow(title, 800, 600, func(window draw.Window) {
		if window.WasKeyPressed(draw.KeyEscape) {
			window.Close()
			return
		}

		if updateMapOnSave {
			updateLevel(world)
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
		upPressed := up && !wasUp

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

		if !(up && guy.jumpSince <= jumpHoldDelay) {
			guy.speedY += gravity
		}
		if !up {
			guy.jumpSince = jumpHoldDelay
		}

		guyX, guyY = guy.collider()
		guy.onGround = world.collidesDownwards(guyX, guyY+1) && guy.speedY >= 0
		if upPressed && guy.speedY != 0 && guy.onGround {
			guy.speedY = jumpSpeed
			guy.jumpSince = 0
		}
		dy := round(guy.speedY)
		for dy > 0 {
			guy.move(0, 1)
			if world.collidesDownwards(guy.collider()) {
				guy.move(0, -1)
				if guy.speedY >= 5 {
					shakeScreen((guy.speedY - 5) / 10)
				}
				guy.speedY = 0
				dy = 0
			} else {
				dy--
			}
		}
		guy.move(0, dy)

		guyCenterX := float64(guy.bounds.x + guy.bounds.width/2)
		guyCenterY := float64(guy.bounds.y + guy.bounds.height/2)
		cam.centerX = cameraDrag*guyCenterX + (1.0-cameraDrag)*cam.centerX
		cam.centerY = cameraDrag*guyCenterY + (1.0-cameraDrag)*cam.centerY

		if len(screenShake) > 0 {
			s := screenShake[0]
			cam.centerX += s[0]
			cam.centerY += s[1]
			screenShake = screenShake[1:]
		}

		guy.update()
		for i := range bats {
			bats[i].update()
		}
		for i := range jumpkins {
			jumpkins[i].update()
		}

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

		for _, b := range bats {
			source := b.sourceBounds()
			dest := cam.worldToCameraRect(b.bounds())
			check(window.DrawImageFilePart(
				b.image(),
				source.x, source.y, source.width, source.height,
				dest.x, dest.y, dest.width, dest.height,
				0,
			))
		}

		for _, j := range jumpkins {
			source := j.sourceBounds()
			dest := cam.worldToCameraRect(j.bounds())
			check(window.DrawImageFilePart(
				j.image(),
				source.x, source.y, source.width, source.height,
				dest.x, dest.y, dest.width, dest.height,
				0,
			))
		}

		wasUp = up
	}))
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
