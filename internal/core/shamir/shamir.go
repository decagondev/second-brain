// Package shamir splits a secret into shares, any k of which rebuild it and fewer reveal
// nothing (sb-selfhost(7), The recovery kit; SB-1902): Shamir's scheme over GF(256) with
// the AES polynomial, one random polynomial of degree k-1 per byte. Standard library only.
package shamir

import (
	"errors"
	"fmt"
	"io"
)

// Share is one share: its x (1 to 255) and one y per byte of the secret.
type Share struct {
	X byte
	Y []byte
}

// ErrShares is a set of shares that can't rebuild a secret.
var ErrShares = errors.New("shamir: can't rebuild")

func mul(a, b byte) byte {
	var p byte
	for b > 0 {
		if b&1 != 0 {
			p ^= a
		}
		hi := a & 0x80
		a <<= 1
		if hi != 0 {
			a ^= 0x1b // x^8 + x^4 + x^3 + x + 1
		}
		b >>= 1
	}
	return p
}

func inv(a byte) byte { // a^254, since a^255 = 1 for a != 0
	r := byte(1)
	for i := 0; i < 254; i++ {
		r = mul(r, a)
	}
	return r
}

// Split makes n shares of secret, any k of which rebuild it (2 <= k <= n <= 255).
func Split(secret []byte, n, k int, rnd io.Reader) ([]Share, error) {
	if k < 2 || n < k || n > 255 {
		return nil, fmt.Errorf("shamir: need 2 <= k <= n <= 255, got k=%d n=%d", k, n)
	}
	if len(secret) == 0 {
		return nil, errors.New("shamir: an empty secret")
	}
	shares := make([]Share, n)
	for i := range shares {
		shares[i] = Share{X: byte(i + 1), Y: make([]byte, len(secret))}
	}
	coef := make([]byte, k-1)
	for j, s := range secret {
		if _, err := io.ReadFull(rnd, coef); err != nil {
			return nil, err
		}
		for i := range shares {
			x := shares[i].X
			// Horner: s + c0 x + c1 x^2 + …
			y := byte(0)
			for c := len(coef) - 1; c >= 0; c-- {
				y = mul(y, x) ^ coef[c]
			}
			shares[i].Y[j] = mul(y, x) ^ s
		}
	}
	return shares, nil
}

// Combine rebuilds the secret from at least k shares by Lagrange interpolation at 0. It
// can't tell k-1 shares from k; callers check what they rebuilt (the kit's checksum).
func Combine(shares []Share) ([]byte, error) {
	if len(shares) < 2 {
		return nil, fmt.Errorf("%w: need at least 2 shares", ErrShares)
	}
	size := len(shares[0].Y)
	seen := map[byte]bool{}
	for _, s := range shares {
		if s.X == 0 || seen[s.X] || len(s.Y) != size || size == 0 {
			return nil, fmt.Errorf("%w: shares must have distinct numbers and the same length", ErrShares)
		}
		seen[s.X] = true
	}
	out := make([]byte, size)
	for i, si := range shares {
		// l_i(0) = prod_{j != i} x_j / (x_j - x_i); in GF(256) minus is xor.
		l := byte(1)
		for j, sj := range shares {
			if i != j {
				l = mul(l, mul(sj.X, inv(sj.X^si.X)))
			}
		}
		for b := range out {
			out[b] ^= mul(l, si.Y[b])
		}
	}
	return out, nil
}
