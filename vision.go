package main

import (
	"fmt"
	"math"

	"github.com/faiface/pixel"
)

const visionPoints = 3

type intersect struct {
	float64
	pixel.Vec
}

func calcVis(i float64, pos pixel.Vec, lines []pixel.Line) intersect {
	length := 1000000.0
	rayEnd := pixel.V(pos.X+length*math.Cos(i), pos.Y+length*math.Sin(i))
	ray := pixel.L(pos, rayEnd)
	p, d := getClosestIntersection(ray, lines)
	return intersect{float64: d, Vec: p}
}

func getVision(pos pixel.Vec, facing float64, lines []pixel.Line) []intersect {
	distances := []intersect{}

	distances = append(distances, calcVis(facing, pos, lines))

	step := math.Pi / visionPoints * 0.999 / 2
	for i := step; i <= math.Pi/2; i += step {
		distances = append(distances, calcVis(facing+i, pos, lines))
		distances = append(distances, calcVis(facing-i, pos, lines))
	}
	expected := visionPoints*2 + 1
	if len(distances) != expected {
		panic(fmt.Sprint("what? ", len(distances), " != ", expected))
	}
	return distances
}
