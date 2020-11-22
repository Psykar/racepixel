package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
)

type Car struct {
	checkpointsComplete uint64
	nextCheckpoint      int

	velocity  float64
	direction float64
	position  pixel.Vec

	driver Driver

	dead    bool
	crashes uint64
	finish  time.Time
	lastInc time.Time

	distances []intersect

	source string
}

const maxCrashes = 10

func (c *Car) addCrash() {
	c.crashes++
	// if c.crashes > c.checkpointsComplete {
	c.dead = true
	// }
}

func (c *Car) Score() float64 {
	// TODO ?
	points := float64(c.checkpointsComplete)
	if c.checkpointsComplete > 5*numCheckpoints {
		points += float64(c.checkpointsComplete) / (c.finish.Sub(start).Seconds() / 100)
	}
	return points
}

type SortableCars struct {
	cars []*Car
}

func (s *SortableCars) Len() int {
	return len(s.cars)
}

func (s *SortableCars) Less(i, j int) bool {
	return s.cars[i].Score() > s.cars[j].Score()
}

func (s *SortableCars) Swap(i, j int) {
	left := s.cars[i]
	right := s.cars[j]
	s.cars[i] = right
	s.cars[j] = left
}

func breedCars(cars []*Car, initialPos pixel.Vec, initialDir float64) []*Car {
	// Select highest 10%
	s := &SortableCars{cars}
	sort.Sort(s)

	result := []*Car{}
	best := s.cars[:len(s.cars)/10]

	worst := 0
	total := 0.0
	for _, i := range cars {
		if i.checkpointsComplete == 0 {
			worst += 1
		}
		total += i.Score()
	}
	fmt.Printf("worst cars: %v, avg: %.2f\n", worst, float64(total)/float64(len(cars)))

	for _, i := range best {
		fmt.Printf("%.0f ", i.Score())
	}
	fmt.Printf("\n")
	// spew.Dump(best[0].driver)
	fmt.Printf("Best car: %.2f from: %v\n", best[0].Score(), best[0].source)

	for _, car := range best {
		result = append(result, &Car{
			driver:    car.driver,
			position:  initialPos,
			direction: initialDir,
			source:    "previous best",
		})
		result = append(result, &Car{
			driver:    MutateNeuralDriver(car.driver.(NeuralDriver)),
			position:  initialPos,
			direction: initialDir,
			source:    "mutated from best",
		})
	}

	for len(result) < len(cars) {
		result = append(result, MutateFromPool(best, initialPos, initialDir))
		result = append(result, BreedFromPool(best, initialPos, initialDir))
	}
	// for len(result) < len(cars) {
	// 	result = append(result, BreedFromPool(best, initialPos, initialDir))
	// }
	return result
}

func BreedFromPool(best []*Car, initialPos pixel.Vec, initialDir float64) *Car {
	leftIndex := rand.Int63n(int64(len(best)))
	rightIndex := rand.Int63n(int64(len(best)))
	left := best[leftIndex]
	right := best[rightIndex]
	driver := BreedNeuralDrivers(left.driver.(NeuralDriver), right.driver.(NeuralDriver))

	return &Car{
		driver:    driver,
		position:  initialPos,
		direction: initialDir,
		source:    "bred from pool",
	}
}

func MutateFromPool(best []*Car, initialPos pixel.Vec, initialDir float64) *Car {
	index := rand.Int63n(int64(len(best)))

	return &Car{
		driver:    MutateNeuralDriver(best[index].driver.(NeuralDriver)),
		position:  initialPos,
		direction: initialDir,
		source:    "mutated from pool",
	}
}

func (car *Car) processStep(
	lines []pixel.Line, delta float64,
	checkpoints []pixel.Line,

) {
	distances := getVision(car.position, car.direction, lines)

	car.distances = distances
	if car.dead {
		return
	}
	keys := car.driver.Drive(inputs{
		nextCheckpoint: car.checkpointsComplete % uint64(len(checkpoints)),
		speed:          car.velocity,
		distances:      distances,
	})

	if _, ok := keys[pixelgl.KeyDown]; ok {
		if car.velocity < 0 {
			car.velocity -= delta * velMulti / 100
		} else {
			car.velocity -= delta * velMulti
		}
	}
	if _, ok := keys[pixelgl.KeyUp]; ok {
		car.velocity += delta * velMulti
	}
	car.velocity -= car.velocity * car.velocity * friction * delta

	turnMulti := maxTurnRate * math.Exp(-math.Pow(car.velocity-peakTurnSpeed, 2)/(2*turnSpeedSpread*turnSpeedSpread))
	if _, ok := keys[pixelgl.KeyRight]; ok {
		if car.velocity < peakTurnSpeed {
			car.direction -= delta * maxTurnRate * car.velocity / peakTurnSpeed
		} else {
			car.direction -= (delta * turnMulti)
		}
	}
	if _, ok := keys[pixelgl.KeyLeft]; ok {
		if car.velocity < peakTurnSpeed {
			car.direction += delta * maxTurnRate * car.velocity / peakTurnSpeed
		} else {
			car.direction += delta * turnMulti
		}
	}

	dist := car.velocity * delta

	xDist := math.Cos(car.direction) * dist
	yDist := math.Sin(car.direction) * dist
	oldVec := car.position
	car.position = car.position.Add(pixel.Vec{X: xDist, Y: yDist})

	path := pixel.L(oldVec, car.position)

	for _, line := range lines {
		if _, ok := line.Intersect(path); ok {
			car.position = oldVec
			car.velocity = 0
			car.addCrash()
			break
		}
	}

	for i, line := range checkpoints {
		if _, ok := line.Intersect(path); ok {
			if i == car.nextCheckpoint {
				car.checkpointsComplete++
				// fmt.Println("Checkpoints: ", car.checkpointsComplete)
				car.nextCheckpoint++
				car.lastInc = clock.Now()
			} else {
				// fmt.Println("expected ", car.nextCheckpoint, " got : ", i)

				car.dead = true
			}
			if car.nextCheckpoint >= len(checkpoints) {
				car.nextCheckpoint = 0
			}
			break

		}
	}

	if car.lastInc.IsZero() {
		car.lastInc = clock.Now()
	}
	if clock.Now().Sub(car.lastInc) > time.Second*5 {
		car.dead = true
	}

	if car.checkpointsComplete > numCheckpoints*6 {
		car.finish = clock.Now()
		fmt.Println("!!!!Finish line!!!:: ", car.Score())
		car.dead = true
	}

	// if vec.X > win.Bounds().Max.X {
	// 	vec.X = win.Bounds().Min.X
	// }
	// if vec.Y > win.Bounds().Max.Y {
	// 	vec.Y = win.Bounds().Min.Y
	// }
	// if vec.X < win.Bounds().Min.X {
	// 	vec.X = win.Bounds().Max.X
	// }
	// if vec.Y < win.Bounds().Min.Y {
	// 	vec.Y = win.Bounds().Max.Y
	// }
}
