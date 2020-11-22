package main

import (
	"math"

	"github.com/faiface/pixel"
)

func getClosest(o pixel.Vec, points [][2]float64) pixel.Vec {
	var ret pixel.Vec
	closestDist := -1.0
	for _, p := range points {
		pVec := pixel.V(p[0], p[1])
		dist := distBetweenPoints(o, pVec)
		if closestDist < 0 || dist < closestDist {
			closestDist = dist
			ret = pixel.V(p[0], p[1])
		}

	}
	return ret
}

func getClosestIntersection(p pixel.Line, lines []pixel.Line) (pixel.Vec, float64) {
	minDist := -1.0
	minPoint := pixel.Vec{}
	for _, line := range lines {
		i, ok := line.Intersect(p)
		if ok {
			dist := distBetweenPoints(p.A, i)
			if minDist < 0 || dist < minDist {
				minDist = dist
				minPoint = i
			}
		}

	}
	return minPoint, minDist
}

func distBetweenPoints(a pixel.Vec, b pixel.Vec) float64 {
	return math.Sqrt(math.Pow(a.X-b.X, 2) + math.Pow(a.Y-b.Y, 2))
}
