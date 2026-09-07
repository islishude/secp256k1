package secp256k1

import (
	"sync"
	"sync/atomic"

	"github.com/islishude/secp256k1/internal/field"
)

// PublicKey is a secp256k1 verification key represented by an affine curve
// point. Values may be copied and used concurrently; copies share verification
// caches, which are allocated only when needed.
type PublicKey struct {
	x           field.Element
	y           field.Element
	precomputed *publicKeyPrecompute
	valid       bool
}

// publicKeyPrecompute is a small shared handle. PublicKey copies share lazy,
// immutable verification tables; parsing and encoding do not build them.
type publicKeyPrecompute struct {
	initialOnce sync.Once
	table       atomic.Pointer[publicKeyTable]
	verifyUses  atomic.Uint32
	promoting   atomic.Bool
}

// The two algorithms use the same storage layout. Publishing a new immutable
// table lets in-flight wNAF readers finish while the old allocation is reclaimed.
type publicKeyTable struct {
	points [max(verifyCombTableSize, 2*varWNAFTableSize)]affinePoint
	comb   bool
}

func (t *publicKeyTable) wnafTables() (*[varWNAFTableSize]affinePoint, *[varWNAFTableSize]affinePoint) {
	return (*[varWNAFTableSize]affinePoint)(t.points[:varWNAFTableSize]),
		(*[varWNAFTableSize]affinePoint)(t.points[varWNAFTableSize : 2*varWNAFTableSize])
}

// verificationTable initializes wNAF once, on the first valid-scalar verification.
func (p PublicKey) verificationTable() *publicKeyTable {
	if table := p.precomputed.table.Load(); table != nil {
		return table
	}
	p.precomputed.initialOnce.Do(func() {
		var q point
		q.setAffine(&p.x, &p.y)
		table := new(publicKeyTable)
		wnaf, endo := table.wnafTables()
		*wnaf = newAffineOddTable(&q)
		*endo = newEndomorphismWNAFTable(wnaf)
		p.precomputed.table.Store(table)
	})
	return p.precomputed.table.Load()
}

// ParsePublicKey parses a SEC 1 compressed or uncompressed public key.
func ParsePublicKey(b []byte) (PublicKey, error) {
	switch len(b) {
	case PublicKeyCompressedSize:
		if b[0] != 0x02 && b[0] != 0x03 {
			return PublicKey{}, ErrInvalidPublicKey
		}
		x, y, ok := affineFromXBytes((*[field.Size]byte)(b[1:]), b[0] == 0x03)
		if !ok {
			return PublicKey{}, ErrInvalidPublicKey
		}
		return newPublicKey(&x, &y), nil
	case PublicKeyUncompressedSize:
		if b[0] != 0x04 {
			return PublicKey{}, ErrInvalidPublicKey
		}
		var x, y field.Element
		if !x.SetBytes((*[field.Size]byte)(b[1:33])) ||
			!y.SetBytes((*[field.Size]byte)(b[33:])) ||
			!isOnCurve(&x, &y) {
			return PublicKey{}, ErrInvalidPublicKey
		}
		return newPublicKey(&x, &y), nil
	default:
		return PublicKey{}, ErrInvalidPublicKey
	}
}

// BytesCompressed returns the SEC 1 compressed public key encoding.
func (p PublicKey) BytesCompressed() ([PublicKeyCompressedSize]byte, error) {
	var out [PublicKeyCompressedSize]byte
	if !p.isValid() {
		return out, ErrInvalidPublicKey
	}
	out[0] = 0x02
	if p.y.IsOdd() {
		out[0] = 0x03
	}
	p.x.PutBytes((*[field.Size]byte)(out[1:]))
	return out, nil
}

// BytesUncompressed returns the SEC 1 uncompressed public key encoding.
func (p PublicKey) BytesUncompressed() ([PublicKeyUncompressedSize]byte, error) {
	var out [PublicKeyUncompressedSize]byte
	if !p.isValid() {
		return out, ErrInvalidPublicKey
	}
	out[0] = 0x04
	p.x.PutBytes((*[field.Size]byte)(out[1:33]))
	p.y.PutBytes((*[field.Size]byte)(out[33:]))
	return out, nil
}

// Equal reports whether p and q encode the same public key.
func (p PublicKey) Equal(q PublicKey) bool {
	if !p.isValid() || !q.isValid() {
		return false
	}
	return p.x.Equal(&q.x) && p.y.Equal(&q.y)
}

func (p PublicKey) isValid() bool {
	// Constructors provide a shared handle, but its expensive tables may be nil.
	return p.valid && p.precomputed != nil
}

func publicKeyFromPoint(p *point) (PublicKey, bool) {
	x, y, ok := p.affineVartime()
	if !ok {
		return PublicKey{}, false
	}
	return newPublicKey(&x, &y), true
}

func newPublicKey(x, y *field.Element) PublicKey {
	return PublicKey{x: *x, y: *y, precomputed: new(publicKeyPrecompute), valid: true}
}
