package scalar

import "testing"

func TestSplitEndomorphismSignedMatchesOld(t *testing.T) {
	for _, k := range splitTestScalars(t) {
		old1, old2 := SplitEndomorphism(&k)
		got1, got2 := SplitEndomorphismSignedVartime(&k)
		if reconstructedSplitScalar(got1) != old1.Words() {
			t.Fatalf("signed k1 mismatch for %x", k.Bytes())
		}
		if reconstructedSplitScalar(got2) != old2.Words() {
			t.Fatalf("signed k2 mismatch for %x", k.Bytes())
		}
	}
}

func TestSplitEndomorphismWordsMatchesOld(t *testing.T) {
	for _, k := range splitTestScalars(t) {
		old1, old2 := SplitEndomorphism(&k)
		got1, got2 := SplitEndomorphismVartimeWords(k.Words())
		if reconstructedSplitScalar(got1) != old1.Words() {
			t.Fatalf("word k1 mismatch for %x: got %+v, want %x", k.Bytes(), got1, old1.Bytes())
		}
		if reconstructedSplitScalar(got2) != old2.Words() {
			t.Fatalf("word k2 mismatch for %x: got %+v, want %x", k.Bytes(), got2, old2.Bytes())
		}
		if got1.Bits > 130 || got2.Bits > 130 {
			t.Fatalf("oversized split for %x: %d/%d bits", k.Bytes(), got1.Bits, got2.Bits)
		}
	}
}

func TestSplitEndomorphismWordsRejectsNonCanonicalInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("non-canonical input did not panic")
		}
	}()
	SplitEndomorphismVartimeWords(orderWords)
}

func splitTestScalars(t *testing.T) []Element {
	t.Helper()
	inputs := [][Size]byte{
		fromHex("00"),
		fromHex("01"),
		fromHex("02"),
		fromHex("7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0"),
		fromHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		fromHex("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140"),
	}
	state := uint64(0xd1b54a32d192ed03)
	for range 2_000 {
		var words [4]uint64
		for i := range words {
			state ^= state >> 12
			state ^= state << 25
			state ^= state >> 27
			words[i] = state * 0x2545f4914f6cdd1d
		}
		var k Element
		k.SetWordsModOrder(words)
		inputs = append(inputs, k.Bytes())
	}

	values := make([]Element, 0, len(inputs))
	for _, in := range inputs {
		var k Element
		if !k.SetBytes(&in) {
			t.Fatalf("SetBytes(%x) failed", in)
		}
		values = append(values, k)
	}
	return values
}

func reconstructedSplitScalar(x SplitScalar) [4]uint64 {
	words := [4]uint64{x.Words[0], x.Words[1], x.Words[2]}
	if x.Neg {
		words = NegWords(words)
	}
	return words
}

func BenchmarkSplitEndomorphismVartimeWords(b *testing.B) {
	kb := fromHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	var k Element
	if !k.SetBytes(&kb) {
		b.Fatal("SetBytes failed")
	}
	words := k.Words()
	var sink1, sink2 SplitScalar
	b.ReportAllocs()
	for b.Loop() {
		sink1, sink2 = SplitEndomorphismVartimeWords(words)
	}
	benchmarkSplitScalarSink = [2]SplitScalar{sink1, sink2}
}

var benchmarkSplitScalarSink [2]SplitScalar
