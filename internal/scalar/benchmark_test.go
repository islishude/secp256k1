package scalar

import "testing"

var benchmarkElementSink Element

func BenchmarkScalarMul(b *testing.B) {
	x := benchmarkElement(0x1234567890abcdef)
	y := benchmarkElement(0xfedcba9876543210)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkElementSink.Mul(&x, &y)
	}
}

func BenchmarkScalarSquare(b *testing.B) {
	x := benchmarkElement(0x1234567890abcdef)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkElementSink.Square(&x)
	}
}

func benchmarkElement(seed uint64) Element {
	var x Element
	x.SetWordsModOrder([4]uint64{
		seed,
		seed*0x9e3779b97f4a7c15 + 1,
		seed*0xbf58476d1ce4e5b9 + 2,
		seed*0x94d049bb133111eb + 3,
	})
	return x
}
