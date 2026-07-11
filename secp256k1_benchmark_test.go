package secp256k1

import (
	"crypto/sha256"
	"testing"

	"github.com/islishude/secp256k1/internal/field"
	"github.com/islishude/secp256k1/internal/scalar"
)

var (
	benchmarkSignatureSink            Signature
	benchmarkRecoverableSignatureSink RecoverableSignature
	benchmarkPublicKeySink            PublicKey
	benchmarkVerifyResult             bool
	benchmarkScalarSink               scalar.Element
	benchmarkPointSink                point
	benchmarkProjectivePointSink      projectivePoint
	benchmarkFieldElementSink         field.Element
	benchmarkWNAFSink                 [257]int16
	benchmarkHalfWNAFSink             [halfWNAFSize]int16
	benchmarkSplitScalarSink          [2]scalar.SplitScalar
	benchmarkIntSink                  int
	benchmarkGeneratorWNAFTableSink   [generatorWNAFSize]affinePoint
)

type benchmarkMaterial struct {
	priv         *PrivateKey
	digest       Digest
	signature    Signature
	recoverable  RecoverableSignature
	publicKey    PublicKey
	compressed   [PublicKeyCompressedSize]byte
	uncompressed [PublicKeyUncompressedSize]byte
}

func newBenchmarkMaterial(b *testing.B) benchmarkMaterial {
	b.Helper()
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 workload"))
	signature, err := priv.SignDigest(digest)
	if err != nil {
		b.Fatal(err)
	}
	recoverable, err := priv.SignRecoverableDigest(digest)
	if err != nil {
		b.Fatal(err)
	}
	publicKey, err := priv.PublicKey()
	if err != nil {
		b.Fatal(err)
	}
	compressed, err := publicKey.BytesCompressed()
	if err != nil {
		b.Fatal(err)
	}
	uncompressed, err := publicKey.BytesUncompressed()
	if err != nil {
		b.Fatal(err)
	}
	return benchmarkMaterial{
		priv:         priv,
		digest:       digest,
		signature:    signature,
		recoverable:  recoverable,
		publicKey:    publicKey,
		compressed:   compressed,
		uncompressed: uncompressed,
	}
}

func BenchmarkSignDigest(b *testing.B) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 sign"))

	b.ReportAllocs()
	b.SetBytes(32)

	for b.Loop() {
		sig, err := priv.SignDigest(digest)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkSignatureSink = sig
	}
}

func BenchmarkSignRecoverableDigest(b *testing.B) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 recoverable sign"))

	b.ReportAllocs()
	b.SetBytes(32)

	for b.Loop() {
		sig, err := priv.SignRecoverableDigest(digest)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkRecoverableSignatureSink = sig
	}
}

func BenchmarkVerifyDigest(b *testing.B) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 verify"))
	sig, err := priv.SignDigest(digest)
	if err != nil {
		b.Fatal(err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(32)
	for b.Loop() {
		benchmarkVerifyResult = VerifyDigest(pub, digest, sig)
	}
	if !benchmarkVerifyResult {
		b.Fatal("verification failed")
	}
}

func BenchmarkRecoverDigest(b *testing.B) {
	privBytes := must32("1e99423a4ed27608a15a2616b4c1b5d1f765a9f6a5f5a2d8e81f6f8a6a88b8d8")
	priv, err := ParsePrivateKey(privBytes[:])
	if err != nil {
		b.Fatal(err)
	}
	digest := sha256.Sum256([]byte("benchmark secp256k1 recover"))
	sig, err := priv.SignRecoverableDigest(digest)
	if err != nil {
		b.Fatal(err)
	}
	want, err := priv.PublicKey()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(32)
	for b.Loop() {
		pub, err := RecoverDigest(digest, sig)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkPublicKeySink = pub
	}
	if !benchmarkPublicKeySink.Equal(want) {
		b.Fatal("recovery failed")
	}
}

func BenchmarkVerifyHotPublicKey(b *testing.B) {
	m := newBenchmarkMaterial(b)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkVerifyResult = VerifyDigest(m.publicKey, m.digest, m.signature)
	}
	if !benchmarkVerifyResult {
		b.Fatal("verification failed")
	}
}

func BenchmarkVerifyParseCompressedCold(b *testing.B) {
	m := newBenchmarkMaterial(b)
	b.ReportAllocs()
	for b.Loop() {
		pub, err := ParsePublicKey(m.compressed[:])
		if err != nil {
			b.Fatal(err)
		}
		benchmarkVerifyResult = VerifyDigest(pub, m.digest, m.signature)
	}
	if !benchmarkVerifyResult {
		b.Fatal("verification failed")
	}
}

func BenchmarkVerifyParseUncompressedCold(b *testing.B) {
	m := newBenchmarkMaterial(b)
	b.ReportAllocs()
	for b.Loop() {
		pub, err := ParsePublicKey(m.uncompressed[:])
		if err != nil {
			b.Fatal(err)
		}
		benchmarkVerifyResult = VerifyDigest(pub, m.digest, m.signature)
	}
	if !benchmarkVerifyResult {
		b.Fatal("verification failed")
	}
}

func BenchmarkSignCompact(b *testing.B) {
	m := newBenchmarkMaterial(b)
	b.ReportAllocs()
	for b.Loop() {
		sig, err := m.priv.SignDigest(m.digest)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkSignatureSink = sig
	}
}

func BenchmarkSignRecoverable(b *testing.B) {
	m := newBenchmarkMaterial(b)
	b.ReportAllocs()
	for b.Loop() {
		sig, err := m.priv.SignRecoverableDigest(m.digest)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkRecoverableSignatureSink = sig
	}
}

func BenchmarkPublicKeyDerive(b *testing.B) {
	m := newBenchmarkMaterial(b)
	b.ReportAllocs()
	for b.Loop() {
		pub, err := m.priv.PublicKey()
		if err != nil {
			b.Fatal(err)
		}
		benchmarkPublicKeySink = pub
	}
}

func BenchmarkScalarInv(b *testing.B) {
	m := newBenchmarkMaterial(b)
	_, s, ok := parseSignature((*[SignatureSize]byte)(m.signature[:]), false)
	if !ok {
		b.Fatal("invalid benchmark signature")
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkScalarSink.Inv(&s)
	}
}

func BenchmarkScalarInvVartime(b *testing.B) {
	m := newBenchmarkMaterial(b)
	_, s, ok := parseSignature((*[SignatureSize]byte)(m.signature[:]), false)
	if !ok {
		b.Fatal("invalid benchmark signature")
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkScalarSink.InvVartime(&s)
	}
}

func BenchmarkSplitEndomorphism(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k scalar.Element
	k.SetBytesModOrder(&m.digest)
	b.ReportAllocs()
	for b.Loop() {
		k1, k2 := scalar.SplitEndomorphism(&k)
		benchmarkScalarSink.Add(&k1, &k2)
	}
}

func BenchmarkSplitEndomorphismVartimeWords(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k scalar.Element
	k.SetBytesModOrder(&m.digest)
	words := k.Words()
	b.ReportAllocs()
	for b.Loop() {
		k1, k2 := scalar.SplitEndomorphismVartimeWords(words)
		benchmarkSplitScalarSink = [2]scalar.SplitScalar{k1, k2}
	}
}

func BenchmarkSignedWNAF256(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k scalar.Element
	k.SetBytesModOrder(&m.digest)
	b.ReportAllocs()
	for b.Loop() {
		naf, length, _ := signedWNAFVartime(&k, varWNAFWindow)
		benchmarkWNAFSink = naf
		benchmarkIntSink = length
	}
}

func BenchmarkSignedWNAFHalf(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k scalar.Element
	k.SetBytesModOrder(&m.digest)
	k1, _ := scalar.SplitEndomorphismVartimeWords(k.Words())
	b.ReportAllocs()
	for b.Loop() {
		var out [halfWNAFSize]int16
		benchmarkIntSink = signedWNAFHalfVartime(&out, k1, varWNAFWindow)
		benchmarkHalfWNAFSink = out
	}
}

func BenchmarkDoubleScalarBaseMultPrecomputedVartime(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k1, k2 scalar.Element
	k1.SetBytesModOrder(&m.digest)
	_, k2, ok := parseSignature((*[SignatureSize]byte)(m.signature[:]), false)
	if !ok {
		b.Fatal("invalid benchmark signature")
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkPointSink = doubleScalarBaseMultPrecomputedVartime(
			&k1,
			&k2,
			&m.publicKey.precomputed.wnafTable,
			&m.publicKey.precomputed.endoWNAFTable,
		)
	}
}

func BenchmarkDoubleScalarBaseMultCombVartime(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k1, k2 scalar.Element
	k1.SetBytesModOrder(&m.digest)
	_, k2, ok := parseSignature((*[SignatureSize]byte)(m.signature[:]), false)
	if !ok {
		b.Fatal("invalid benchmark signature")
	}
	var q point
	q.setAffine(&m.publicKey.x, &m.publicKey.y)
	combTable := newVerifyCombTable(&q)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkPointSink = doubleScalarBaseMultCombVartime(
			&k1,
			&k2,
			&combTable,
		)
	}
}

func BenchmarkScalarBaseMultProjective(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k scalar.Element
	k.SetBytesModOrder(&m.digest)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkProjectivePointSink = scalarBaseMultProjective(&k)
	}
}

func BenchmarkScalarBaseMultProjectiveW4(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k scalar.Element
	k.SetBytesModOrder(&m.digest)
	table := newGeneratorAffineTableW4()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkProjectivePointSink = scalarBaseMultProjectiveW4(&k, &table)
	}
}

func BenchmarkScalarBaseMultProjectiveW6AffineScan(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k scalar.Element
	k.SetBytesModOrder(&m.digest)
	table := newGeneratorAffineTableW6ForTest()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkProjectivePointSink = scalarBaseMultProjectiveW6ForTest(&k, &table)
	}
}

func BenchmarkScalarBaseMultProjectiveW5Affine(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k scalar.Element
	k.SetBytesModOrder(&m.digest)
	table := newGeneratorAffineTableW5()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkProjectivePointSink = scalarBaseMultProjectiveW5Affine(&k, &table)
	}
}

func BenchmarkNewAffineOddTable(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var p point
	p.setAffine(&m.publicKey.x, &m.publicKey.y)
	b.ReportAllocs()
	for b.Loop() {
		table := newAffineOddTable(&p)
		benchmarkFieldElementSink = table[len(table)-1].x
	}
}

func BenchmarkAddAffine(b *testing.B) {
	var base point
	base.double(&generator)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkPointSink.addAffine(&base, &generator.x, &generator.y)
	}
}

func BenchmarkAddAffineWNAFVartime(b *testing.B) {
	var base point
	base.double(&generator)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkPointSink.addAffineWNAFVartime(&base, &generator.x, &generator.y)
	}
}

func BenchmarkAddCompleteMixedGo(b *testing.B) {
	m := newBenchmarkMaterial(b)
	var k scalar.Element
	k.SetBytesModOrder(&m.digest)
	p1 := scalarBaseMultProjective(&k)
	p2 := affinePoint{x: generator.x, y: generator.y}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkProjectivePointSink.addCompleteMixed(&p1, &p2)
	}
}

func BenchmarkPointDouble(b *testing.B) {
	base := generator
	base.double(&base)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkPointSink.double(&base)
	}
}

func BenchmarkPointDoubleSquare(b *testing.B) {
	base := generator
	base.double(&base)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkPointSink.doubleSquare(&base)
	}
}

func BenchmarkCurveYFromX(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		y, ok := curveYFromX(&generator.x, false)
		if !ok {
			b.Fatal("generator x has no curve y")
		}
		benchmarkFieldElementSink = y
	}
}

func BenchmarkGeneratorPrecomputeDynamic(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		wnaf := newGeneratorWNAFTable()
		endo := newGeneratorEndomorphismWNAFTable(&wnaf)
		benchmarkGeneratorWNAFTableSink = endo
	}
}

func BenchmarkGeneratorPrecomputeStaticLoad(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		wnaf, endo := loadGeneratorWNAFTables(&generatorWNAFTableWords, &generatorEndoWNAFXWords)
		benchmarkGeneratorWNAFTableSink = wnaf
		benchmarkGeneratorWNAFTableSink = endo
	}
}
