package random

import (
	"log"
	"math/rand"
	"testing"
)

func TestSameWeight(t *testing.T) {
	testsameWeightWithparams(t, 10, 10000)
	testsameWeightWithparams(t, 11, 10000)
	testsameWeightWithparams(t, 1, 10000)
	testsameWeightWithparams(t, 2, 10000)
	testsameWeightWithparams(t, 3, 10000)
	testsameWeightWithparams(t, 1000, 10000000)
}

func TestEmptySlice(t *testing.T) {
	_, err := NewDiscreteRandomVariable(rand.New(rand.NewSource(0)), []float64{})

	if err != ErrEmptyWeightsSlice {
		t.Fatal("ErrEmptyWeightsSlice should be returned")
	}
}

func testsameWeightWithparams(t *testing.T, numsCount, experimentsCount uint) {
	counters := make([]uint, numsCount)
	weights := make([]float64, numsCount)

	for i := range numsCount {
		weights[i] = 1
	}

	drv, err := NewDiscreteRandomVariable(rand.New(rand.NewSource(0)), weights)

	if err != nil {
		t.Fatal(err)
	}

	for range experimentsCount {
		if t.Context().Err() != nil {
			t.FailNow()
		}

		num := drv.Get()

		counters[num]++
	}

	average := experimentsCount / numsCount

	for _, counter := range counters {
		if max(counter, average)-min(counter, average) > average/10 {
			log.Println(counters)

			t.FailNow()
		}
	}
}

func TestDifferentWeights(t *testing.T) {
	numsCount := uint(10)
	experimentsCount := uint(100000)

	counters := make([]uint, numsCount)
	weights := make([]float64, numsCount)

	for i := range numsCount {
		weights[i] = 1
	}

	weights[0] = 2
	weights[1] = 3
	weights[3] = 0

	drv, err := NewDiscreteRandomVariable(rand.New(rand.NewSource(0)), weights)

	if err != nil {
		t.Fatal(err)
	}

	drv.SetWeight(2, 4)

	weights[2] = 4

	for range experimentsCount {
		if t.Context().Err() != nil {
			t.FailNow()
		}

		num := drv.Get()

		counters[num]++
	}

	weightsSum := float64(0)

	for _, weight := range weights {
		weightsSum += weight
	}

	for i, counter := range counters {
		expectedCounterValue := uint(float64(experimentsCount) * weights[i] / weightsSum)

		if max(counter, expectedCounterValue)-min(counter, expectedCounterValue) > expectedCounterValue/10 {
			log.Println(expectedCounterValue, counters)

			t.FailNow()
		}
	}
}
