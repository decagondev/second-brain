package shamir

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestMulInv(t *testing.T) {
	for a := 1; a < 256; a++ {
		if mul(byte(a), inv(byte(a))) != 1 {
			t.Fatalf("inv(%d) is wrong", a)
		}
	}
	if mul(0x53, 0xca) != 0x01 { // the AES field's textbook pair
		t.Fatal("mul is not GF(2^8) with 0x11b")
	}
}

func TestAnyThreeOfFive(t *testing.T) {
	secret := make([]byte, 33)
	_, _ = rand.Read(secret)
	sh, err := Split(secret, 5, 3, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for a := 0; a < 5; a++ {
		for b := a + 1; b < 5; b++ {
			for c := b + 1; c < 5; c++ {
				got, err := Combine([]Share{sh[c], sh[a], sh[b]})
				if err != nil || !bytes.Equal(got, secret) {
					t.Fatalf("shares %d,%d,%d: %v", a+1, b+1, c+1, err)
				}
			}
			if got, _ := Combine([]Share{sh[a], sh[b]}); bytes.Equal(got, secret) {
				t.Fatalf("two shares %d,%d rebuilt the secret", a+1, b+1)
			}
		}
	}
	if got, _ := Combine(sh); !bytes.Equal(got, secret) {
		t.Fatal("all five")
	}
	if _, err := Combine([]Share{sh[0], sh[0], sh[1]}); err == nil {
		t.Fatal("a repeated share was accepted")
	}
	if _, err := Split(secret, 2, 3, rand.Reader); err == nil {
		t.Fatal("k > n was accepted")
	}
}

// Two shares say nothing: for any two shares and any candidate first byte, some third
// share makes it so. Exhaustively, for the first byte of a 1-byte secret with k=3.
func TestTwoSharesRevealNothing(t *testing.T) {
	sh, _ := Split([]byte{0x42}, 5, 3, rand.Reader)
	for cand := 0; cand < 256; cand++ {
		found := false
		for y := 0; y < 256 && !found; y++ {
			got, _ := Combine([]Share{sh[0], sh[1], {X: 5, Y: []byte{byte(y)}}})
			found = got[0] == byte(cand)
		}
		if !found {
			t.Fatalf("with shares 1 and 2, the secret can't be %d: they leak", cand)
		}
	}
}
