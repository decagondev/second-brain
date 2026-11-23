package laws

import (
	"bytes"
	"crypto/rand"
	"fmt"
	mrand "math/rand"

	"example.com/second-brain/internal/core/offsite"
	"example.com/second-brain/internal/core/shamir"
)

// LAW-11 and LAW-07 off the box (SB-1901): every copy the box sends off it is sealed, so
// what sits on SB_OFFSITE reveals none of its content, opens only with the recovery key,
// and fails to open after any change. LAW-18 for the recovery kit (SB-1902): any three of
// the five shares rebuild the kit key and any two don't.

type sealPair struct {
	seal func(plain []byte, to offsite.PublicKey) ([]byte, error)
	open func(sealed []byte, key offsite.PrivateKey) ([]byte, error)
}

var realSeal = sealPair{
	seal: func(p []byte, to offsite.PublicKey) ([]byte, error) {
		var b bytes.Buffer
		err := offsite.Seal(&b, bytes.NewReader(p), to, rand.Reader)
		return b.Bytes(), err
	},
	open: func(s []byte, k offsite.PrivateKey) ([]byte, error) {
		var b bytes.Buffer
		err := offsite.Open(&b, bytes.NewReader(s), k)
		return b.Bytes(), err
	},
}

func law11Sealed(r *mrand.Rand, sp sealPair) error {
	key, err := offsite.Generate(rand.Reader)
	if err != nil {
		return err
	}
	plain := make([]byte, 64+r.Intn(3*offsite.ChunkSize))
	r.Read(plain)
	sealed, err := sp.seal(plain, key.Public())
	if err != nil {
		return err
	}
	for i := 0; i+32 <= len(plain); i += 997 {
		if bytes.Contains(sealed, plain[i:i+32]) {
			return fmt.Errorf("the sealed copy holds 32 bytes of its content at offset %d", i)
		}
	}
	got, err := sp.open(sealed, key)
	if err != nil || !bytes.Equal(got, plain) {
		return fmt.Errorf("the recovery key doesn't open the sealed copy to its content: %v", err)
	}
	other, _ := offsite.Generate(rand.Reader)
	if _, err := sp.open(sealed, other); err == nil {
		return fmt.Errorf("another key opened the sealed copy")
	}
	c := append([]byte{}, sealed...)
	i := r.Intn(len(c))
	c[i] ^= byte(1 + r.Intn(255))
	if _, err := sp.open(c, key); err == nil {
		return fmt.Errorf("byte %d of %d was changed and the copy still opened", i, len(c))
	}
	return nil
}

type kitScheme struct {
	split   func(secret []byte) ([]shamir.Share, error)
	combine func([]shamir.Share) ([]byte, error)
}

var realKit = kitScheme{
	split: func(s []byte) ([]shamir.Share, error) {
		return shamir.Split(s, offsite.KitShares, offsite.KitNeed, rand.Reader)
	},
	combine: shamir.Combine,
}

func law18Kit(r *mrand.Rand, k kitScheme) error {
	secret := make([]byte, 32)
	r.Read(secret)
	shares, err := k.split(secret)
	if err != nil {
		return err
	}
	if len(shares) != offsite.KitShares {
		return fmt.Errorf("%d shares, want %d", len(shares), offsite.KitShares)
	}
	pick := r.Perm(len(shares))
	three := []shamir.Share{shares[pick[0]], shares[pick[1]], shares[pick[2]]}
	got, err := k.combine(three)
	if err != nil || !bytes.Equal(got, secret) {
		return fmt.Errorf("shares %d, %d and %d don't rebuild the key: %v", three[0].X, three[1].X, three[2].X, err)
	}
	two := []shamir.Share{shares[pick[3]], shares[pick[4]]}
	if got, err := k.combine(two); err == nil && bytes.Equal(got, secret) {
		return fmt.Errorf("two shares, %d and %d, rebuilt the key", two[0].X, two[1].X)
	}
	for _, sh := range shares {
		if bytes.Contains(sh.Y, secret[:8]) {
			return fmt.Errorf("share %d holds part of the key as it is", sh.X)
		}
	}
	return nil
}
