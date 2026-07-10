package field

import (
	"testing"

	fiat "github.com/islishude/secp256k1/internal/fiat/basefield"
)

var benchmarkElementSink Element
var benchmarkFiatElementSink fiat.MontgomeryDomainFieldElement

func BenchmarkFieldMul(b *testing.B) {
	x := benchmarkElement(0x1234567890abcdef)
	y := benchmarkElement(0xfedcba9876543210)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkElementSink.Mul(&x, &y)
	}
}

func BenchmarkFieldSquare(b *testing.B) {
	x := benchmarkElement(0x1234567890abcdef)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkElementSink.Square(&x)
	}
}

func BenchmarkFieldMulFiat(b *testing.B) {
	x := benchmarkElement(0x1234567890abcdef)
	y := benchmarkElement(0xfedcba9876543210)
	b.ReportAllocs()
	for b.Loop() {
		fiat.Mul(&benchmarkFiatElementSink, &x.x, &y.x)
	}
}

func BenchmarkFieldSquareFiat(b *testing.B) {
	x := benchmarkElement(0x1234567890abcdef)
	b.ReportAllocs()
	for b.Loop() {
		fiat.Square(&benchmarkFiatElementSink, &x.x)
	}
}

func BenchmarkFieldAdd(b *testing.B) {
	x := benchmarkElement(0x1234567890abcdef)
	y := benchmarkElement(0xfedcba9876543210)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkElementSink.Add(&x, &y)
	}
}

func BenchmarkFieldAddFiat(b *testing.B) {
	x := benchmarkElement(0x1234567890abcdef)
	y := benchmarkElement(0xfedcba9876543210)
	b.ReportAllocs()
	for b.Loop() {
		fiat.Add(&benchmarkFiatElementSink, &x.x, &y.x)
	}
}

func BenchmarkFieldSub(b *testing.B) {
	x := benchmarkElement(0x1234567890abcdef)
	y := benchmarkElement(0xfedcba9876543210)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkElementSink.Sub(&x, &y)
	}
}

func BenchmarkFieldSubFiat(b *testing.B) {
	x := benchmarkElement(0x1234567890abcdef)
	y := benchmarkElement(0xfedcba9876543210)
	b.ReportAllocs()
	for b.Loop() {
		fiat.Sub(&benchmarkFiatElementSink, &x.x, &y.x)
	}
}

func benchmarkElement(seed uint64) Element {
	var x Element
	x.SetNonMontgomeryWords([4]uint64{
		seed,
		seed*0x9e3779b97f4a7c15 + 1,
		seed*0xbf58476d1ce4e5b9 + 2,
		seed*0x94d049bb133111eb + 3,
	})
	return x
}
