package random

import (
	"math"
	"math/rand"
)

type DiscreteRandomVariable struct {
	intervals                []float64
	valuesWeights            []float64
	weightsSum               float64
	randSource               *rand.Rand
	recalculateintervalsFrom int
}

func NewDiscreteRandomVariable(randSource *rand.Rand, weights []float64) (*DiscreteRandomVariable, error) {
	res := &DiscreteRandomVariable{
		randSource: randSource,
	}

	if len(weights) <= 0 {
		return nil, ErrEmptyWeightsSlice
	}

	res.SetWeights(weights)

	return res, nil
}

func (rv *DiscreteRandomVariable) GetWeights() []float64 {
	res := make([]float64, len(rv.valuesWeights))

	copy(res, rv.valuesWeights)

	return res
}

func (rv *DiscreteRandomVariable) GetWeight(i int) float64 {
	return rv.valuesWeights[i]
}

func (rv *DiscreteRandomVariable) SetWeights(weights []float64) error {
	if len(weights) <= 0 {
		return ErrEmptyWeightsSlice
	}

	if len(rv.valuesWeights) >= len(weights) {
		rv.valuesWeights = rv.valuesWeights[:len(weights)]
		rv.intervals = rv.intervals[:len(weights)-1]
	} else {
		rv.valuesWeights = make([]float64, len(weights))
		rv.intervals = make([]float64, len(weights)-1)
	}

	copy(rv.valuesWeights, weights)

	rv.weightsSum = 0

	for _, weight := range rv.valuesWeights {
		rv.weightsSum += weight
	}

	rv.recalculateintervalsFrom = 0

	return nil
}

func (rv *DiscreteRandomVariable) recalculateIntervals(from int) {
	prevSectionValue := float64(0)

	if from > 0 {
		prevSectionValue = rv.intervals[from-1]
	}

	for intervalIndex := from; intervalIndex < len(rv.intervals); intervalIndex++ {
		currentSectionValue := prevSectionValue + rv.valuesWeights[intervalIndex]

		rv.intervals[intervalIndex] = currentSectionValue

		prevSectionValue = currentSectionValue
	}
}

func (rv *DiscreteRandomVariable) SetWeight(i int, weight float64) {
	oldWeight := rv.valuesWeights[i]

	//To avoid extra recalculation of intervals.
	if weight == oldWeight {
		return
	}

	rv.weightsSum -= oldWeight

	rv.weightsSum += weight

	rv.valuesWeights[i] = weight

	if rv.recalculateintervalsFrom == -1 || i < rv.recalculateintervalsFrom {
		rv.recalculateintervalsFrom = i
	}
}

func (rv *DiscreteRandomVariable) Get() int {
	if rv.recalculateintervalsFrom != -1 {
		rv.recalculateIntervals(rv.recalculateintervalsFrom)

		rv.recalculateintervalsFrom = -1
	}

	//binary search

	var (
		randomNumberToFindSection = float64(rv.randSource.Uint64()) * rv.weightsSum / float64(math.MaxUint64)

		left       = 0
		right      = len(rv.valuesWeights)
		valueIndex int
	)

	for {
		valueIndex = (right + left) / 2

		if valueIndex > 0 {
			sectionBegin := rv.intervals[valueIndex-1]

			if randomNumberToFindSection < sectionBegin {
				right = valueIndex

				continue
			}
		}

		if valueIndex < len(rv.intervals) {
			sectionEnd := rv.intervals[valueIndex]

			if randomNumberToFindSection >= sectionEnd {
				left = valueIndex

				continue
			}
		}

		break
	}

	return valueIndex
}
