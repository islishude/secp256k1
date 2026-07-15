package scalar

import "testing"

func TestInvVartimeMatchesInv(t *testing.T) {
	values := [][Size]byte{
		fromHex("00"),
		fromHex("01"),
		fromHex("02"),
		fromHex("7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0"),
		fromHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		fromHex("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140"),
	}

	state := uint64(0x9e3779b97f4a7c15)
	for range 1_000 {
		var words [4]uint64
		for i := range words {
			state ^= state << 13
			state ^= state >> 7
			state ^= state << 17
			words[i] = state
		}
		var x Element
		x.SetWordsModOrder(words)
		values = append(values, x.Bytes())
	}

	for _, xb := range values {
		var x, got, want Element
		if !x.SetBytes(&xb) {
			t.Fatalf("SetBytes(%x) failed", xb)
		}
		got.InvVartime(&x)
		want.Inv(&x)
		if !got.Equal(&want) {
			t.Fatalf("InvVartime(%x) = %x, want %x", xb, got.Bytes(), want.Bytes())
		}
	}
}

func TestInvVartimeMulIdentity(t *testing.T) {
	var one Element
	one.SetOne()
	state := uint64(1)
	for range 1_000 {
		state = state*6364136223846793005 + 1442695040888963407
		var x Element
		x.SetWordsModOrder([4]uint64{state, state << 1, state << 7, state << 19})
		if x.IsZero() {
			x.SetOne()
		}
		var inverse, product Element
		inverse.InvVartime(&x)
		product.Mul(&x, &inverse)
		if !product.Equal(&one) {
			t.Fatalf("x * InvVartime(x) = %x for x %x", product.Bytes(), x.Bytes())
		}
	}
}

func TestInvVartimeAliasing(t *testing.T) {
	b := fromHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	var got, want Element
	if !got.SetBytes(&b) || !want.SetBytes(&b) {
		t.Fatal("SetBytes failed")
	}
	got.InvVartime(&got)
	want.Inv(&want)
	if !got.Equal(&want) {
		t.Fatalf("aliased InvVartime = %x, want %x", got.Bytes(), want.Bytes())
	}
}

func TestInvVartimeBackendMatchesGo(t *testing.T) {
	edges := invVartimeBackendEdges()
	for i, input := range edges {
		checkInvVartimeWordsBackend(t, i, input)
	}

	state := uint64(0xd1b54a32d192ed03)
	for i := range 100_000 {
		var input [4]uint64
		for j := range input {
			state ^= state >> 12
			state ^= state << 25
			state ^= state >> 27
			input[j] = state * 0x2545f4914f6cdd1d
		}
		input = reduceWordsModOrder(input)
		checkInvVartimeWordsBackend(t, i+len(edges), input)
	}
}

func invVartimeBackendEdges() [][4]uint64 {
	return [][4]uint64{
		{},
		{1},
		{2},
		{^uint64(0)},
		{0, ^uint64(0)},
		{0, 0, ^uint64(0)},
		{orderLimb3 - 1, orderLimb2, orderLimb1, orderLimb0},
		{0x0123456789abcdef, 0xfedcba9876543210, 0x55aa55aa55aa55aa, 0x7fffffffffffffff},
	}
}

func checkInvVartimeWordsBackend(t *testing.T, index int, input [4]uint64) {
	t.Helper()
	want := invVartimeWordsGo(input)
	var got [4]uint64
	invVartimeWords(&got, &input)
	if got != want {
		t.Fatalf("InvVartime backend mismatch at input %d: got %x want %x", index, got, want)
	}
	if got != [4]uint64{} && !LessThanOrderWords(got) {
		t.Fatalf("InvVartime backend returned non-canonical value at input %d: %x", index, got)
	}

	aliased := input
	invVartimeWords(&aliased, &aliased)
	if aliased != want {
		t.Fatalf("aliased InvVartime backend mismatch at input %d: got %x want %x", index, aliased, want)
	}

	var inputElement, gotElement, constantTime, product Element
	inputElement.SetWords(input)
	gotElement.SetWords(got)
	constantTime.Inv(&inputElement)
	if !gotElement.Equal(&constantTime) {
		t.Fatalf("InvVartime/Inv mismatch at input %d: got %x want %x", index, gotElement.Bytes(), constantTime.Bytes())
	}
	if input != [4]uint64{} {
		var one Element
		one.SetOne()
		product.Mul(&inputElement, &gotElement)
		if !product.Equal(&one) {
			t.Fatalf("input * InvVartime(input) != 1 at input %d: got %x", index, product.Bytes())
		}
	}
}

func BenchmarkScalarInvVartime(b *testing.B) {
	xb := fromHex("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	var x, sink Element
	if !x.SetBytes(&xb) {
		b.Fatal("SetBytes failed")
	}
	b.ReportAllocs()
	for b.Loop() {
		sink.InvVartime(&x)
	}
	benchmarkScalarElementSink = sink
}

var benchmarkScalarElementSink Element
