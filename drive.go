package main

import (
	"math/rand"

	"github.com/faiface/pixel/pixelgl"

	"github.com/NOX73/go-neural"
)

type inputs struct {
	nextCheckpoint uint64
	speed          float64
	distances      []intersect
}

type outputs map[pixelgl.Button]bool

type ManualDriver struct {
	win *pixelgl.Window
}

type Driver interface {
	Drive(inputs) outputs
}

func (d ManualDriver) Drive(i inputs) outputs {
	ret := make(outputs)

	keys := []pixelgl.Button{
		pixelgl.KeyUp,
		pixelgl.KeyDown,
		pixelgl.KeyLeft,
		pixelgl.KeyRight,
	}
	for _, key := range keys {
		if d.win.Pressed(key) {
			ret[key] = true
		}
	}
	return ret
}

type NeuralDriver struct {
	network *neural.Network
}

func (d NeuralDriver) Drive(i inputs) outputs {
	// d.network.RandomizeSynapses()
	in := []float64{
		float64(i.nextCheckpoint),
		i.speed,
	}
	for _, d := range i.distances {
		in = append(in, d.float64)
	}
	output := d.network.Calculate(in)

	keys := []pixelgl.Button{
		pixelgl.KeyUp,
		pixelgl.KeyDown,
		pixelgl.KeyLeft,
		pixelgl.KeyRight,
	}

	presses := make(outputs)
	for i, val := range output {
		if val >= 0.5 {
			presses[keys[i]] = true
		}
	}
	return presses
}

func NewNeuralDriver() NeuralDriver {
	// Last layer is network output.
	n := neural.NewNetwork(visionPoints*2+3, []int{15, 15, 4})
	// Randomize sypaseses weights
	n.RandomizeSynapses()

	return NeuralDriver{
		network: n,
	}
}

func NewDriver() ManualDriver {
	return ManualDriver{}
}

const stdev = 1

func BreedNeuralDrivers(a NeuralDriver, b NeuralDriver) NeuralDriver {
	new := NewNeuralDriver()

	for i, l := range a.network.Layers {
		for j, n := range l.Neurons {
			for k, leftSynapse := range n.InSynapses {
				rightSynapse := b.network.Layers[i].Neurons[j].InSynapses[k]
				newSynapse := new.network.Layers[i].Neurons[j].InSynapses[k]
				newSynapse.Weight = (leftSynapse.Weight + rightSynapse.Weight) / 2
				newSynapse.Weight += (rand.NormFloat64() * stdev)
			}
		}
	}
	return new
}

func MutateNeuralDriver(a NeuralDriver) NeuralDriver {
	new := NewNeuralDriver()

	for i, l := range a.network.Layers {
		for j, n := range l.Neurons {
			for k, oldSynapse := range n.InSynapses {
				newSynapse := new.network.Layers[i].Neurons[j].InSynapses[k]
				newSynapse.Weight = oldSynapse.Weight + (rand.NormFloat64() * stdev)
			}
		}
	}
	return new
}
