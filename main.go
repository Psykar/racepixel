package main

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	_ "image/png"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/jdeal-mediamath/clockwork"
	"golang.org/x/image/colornames"
)

const (
	velMulti        = 500.0
	friction        = 0.001
	maxTurnRate     = 6.0
	peakTurnSpeed   = 100.0
	turnSpeedSpread = 100.0
	vsync           = false
	frameRate       = 30
	numCheckpoints  = 100
)

var (
	clock clockwork.FakeClock
	start time.Time
)

func main() {
	pixelgl.Run(run)
}

func run() {
	rand.Seed(time.Now().UnixNano())
	// all of our code will be fired up from here

	clock = clockwork.NewFakeClock()

	spritesheet, err := loadPicture("car.png")
	if err != nil {
		panic(err)
	}

	carIcon := pixel.NewSprite(spritesheet, pixel.R(0, 0, spritesheet.Bounds().W(), spritesheet.Bounds().H()))

	last := clock.Now()

	batch := pixel.NewBatch(&pixel.TrianglesData{}, spritesheet)

	imd := imdraw.New(nil)

	imd.Color = colornames.Blueviolet
	imd.EndShape = imdraw.RoundEndShape

	ch := imdraw.New(nil)
	ch.Color = colornames.Green

	visionDraw := imdraw.New(nil)
	visionDraw.Color = colornames.Pink

	minX := outer[0][0]
	maxX := outer[0][0]
	minY := outer[0][1]
	maxY := outer[0][1]

	var lines []pixel.Line

	var lastPoint *pixel.Vec
	var checkpoints []pixel.Line
	for index, p := range outer {
		imd.Push(pixel.V(p[0], p[1]))
		point := pixel.V(p[0], p[1])
		if lastPoint != nil {
			lines = append(lines, pixel.L(point, *lastPoint))
		} else {
			l := len(outer)
			lines = append(lines, pixel.L(point, pixel.V(outer[l-1][0], outer[l-1][1])))
		}
		lastPoint = &point

		if index%(len(outer)/numCheckpoints) == 0 {
			other := getClosest(point, inner)
			ch.Push(pixel.V(p[0], p[1]))
			ch.Push(pixel.V(other.X, other.Y))
			ch.Line(1)
			checkpoints = append(checkpoints, pixel.L(point, other))

		}

		if p[0] > maxX {
			maxX = p[0]
		}
		if p[1] > maxY {
			maxY = p[1]
		}

		if p[0] < minX {
			minX = p[0]
		}
		if p[1] < minY {
			minY = p[1]
		}
	}

	var reversedCheckpoints []pixel.Line
	checkpointDir := 1
	for i := 12; len(reversedCheckpoints) < len(checkpoints); i += checkpointDir {
		if i >= len(checkpoints) {
			i = 0
		}
		if i < 0 {
			i = len(checkpoints)
		}
		reversedCheckpoints = append(reversedCheckpoints, checkpoints[i])
	}
	checkpoints = reversedCheckpoints
	imd.Polygon(2)

	lastPoint = nil
	for _, p := range inner {
		imd.Push(pixel.V(p[0], p[1]))

		point := pixel.V(p[0], p[1])
		if lastPoint != nil {
			lines = append(lines, pixel.L(point, *lastPoint))
		} else {
			l := len(inner)
			lines = append(lines, pixel.L(point, pixel.V(inner[l-1][0], inner[l-1][1])))
		}
		lastPoint = &point
	}
	imd.Polygon(2)

	cfg := pixelgl.WindowConfig{
		Title:  "Pixel Rocks!",
		Bounds: pixel.R(minX, minY, maxX, maxY),
		VSync:  vsync,
		// Bounds: pixel.R(10, 0, 700, 1100),
	}

	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}
	win.SetSmooth(true)

	win.Clear(colornames.Grey)

	canvas := pixelgl.NewCanvas(win.Bounds())
	imd.Draw(canvas)
	ch.Draw(canvas)

	// initialPos := pixel.Vec{(outer[0][0] + inner[0][0]) / 2, (outer[0][1] + inner[0][1]) / 2}
	initialPos := pixel.V(110.0544269, 722.1342655)
	// p1 := getClosest(initialPos, outer)
	// p2 := getClosest(initialPos, inner)
	initialDir := 70 * math.Pi / 180

	cars := []*Car{}
	numCars := 500
	for i := 0; i < numCars; i++ {
		cars = append(cars, &Car{
			direction: initialDir,
			position:  initialPos,
			driver:    NewNeuralDriver(),
			source:    "initial",
		})
	}

	lastIncrease := clock.Now()
	start = lastIncrease
	lastCheckpoints := make([]uint64, len(cars))

	for !win.Closed() {
		clock.Advance(time.Second / frameRate)

		win.Clear(colornames.Aliceblue)

		now := clock.Now()
		delta := now.Sub(last).Seconds()
		if false {
			fmt.Println(delta)
		}
		last = now

		batch.Clear()

		canvas.Draw(win, pixel.IM.Moved(win.Bounds().Center()))

		if win.JustPressed(pixelgl.MouseButtonLeft) {
			fmt.Println(win.MousePosition())
		}

		g := sync.WaitGroup{}
		for _, car := range cars {
			// doCar(car, lines, delta, checkpoints)
			g.Add(1)
			go func(c *Car) {
				c.processStep(lines, delta, checkpoints)
				g.Done()
			}(car)
		}
		g.Wait()

		for i, car := range cars {
			if car.checkpointsComplete > lastCheckpoints[i] {
				lastIncrease = clock.Now()
				lastCheckpoints[i] = car.checkpointsComplete
				// fmt.Println("inc: ", lastIncrease)
			}
			if car.dead {
				continue
			}
			carIcon.Draw(batch, pixel.IM.Scaled(pixel.ZV, 0.05).Rotated(pixel.ZV, car.direction).Moved(car.position))
			// visionDraw.Clear()
			// for _, d := range car.distances {
			// 	visionDraw.Push(car.position)
			// 	visionDraw.Push(d.Vec)
			// 	visionDraw.Line(1)
			// }
			// visionDraw.Draw(win)

		}

		batch.Draw(win)

		win.Update()

		goods := 0
		alive := 0
		for _, car := range cars {
			if !car.dead {
				alive++
			}
			if car.checkpointsComplete > 0 && !car.dead {
				goods += 1
			}
			if car.checkpointsComplete < 1 && clock.Now().Sub(start) > time.Second/2 {
				car.dead = true
			}
		}

		if alive == 0 {
			lastIncrease = clock.Now()

			cars = breedCars(cars, initialPos, initialDir)
			start = lastIncrease

		}
	}
}

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}
