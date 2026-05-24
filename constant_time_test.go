package secp256k1

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/islishude/secp256k1/internal/scalar"
)

var constantTimePointSink point

func TestConstantTimeScalarBaseMultSmoke(t *testing.T) {
	if os.Getenv("SECP256K1_CT_TEST") != "1" {
		t.Skip("set SECP256K1_CT_TEST=1 to run timing smoke tests")
	}

	lowBytes := must32("0000000000000000000000000000000000000000000000000000000000000001")
	highBytes := must32("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140")
	var low, high scalar.Element
	if !low.SetBytes(&lowBytes) || !high.SetBytes(&highBytes) {
		t.Fatal("test scalars are invalid")
	}

	const samples = 1000
	lowTimes := make([]float64, 0, samples)
	highTimes := make([]float64, 0, samples)
	for i := range samples {
		if i%2 == 0 {
			lowTimes = append(lowTimes, measureScalarBaseMult(&low))
			highTimes = append(highTimes, measureScalarBaseMult(&high))
		} else {
			highTimes = append(highTimes, measureScalarBaseMult(&high))
			lowTimes = append(lowTimes, measureScalarBaseMult(&low))
		}
	}

	if tScore := welchTScore(lowTimes, highTimes); tScore > 20 {
		t.Fatalf("scalarBaseMult timing smoke test exceeded threshold: t=%0.2f", tScore)
	}
}

func measureScalarBaseMult(k *scalar.Element) float64 {
	start := time.Now()
	constantTimePointSink = scalarBaseMult(k)
	return float64(time.Since(start).Nanoseconds())
}

func welchTScore(a, b []float64) float64 {
	meanA, varianceA := meanAndVariance(a)
	meanB, varianceB := meanAndVariance(b)
	denom := math.Sqrt(varianceA/float64(len(a)) + varianceB/float64(len(b)))
	if denom == 0 {
		return 0
	}
	return math.Abs(meanA-meanB) / denom
}

func meanAndVariance(values []float64) (float64, float64) {
	var mean float64
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	var variance float64
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(values) - 1)
	return mean, variance
}
