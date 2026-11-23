package offsite

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"
)

func roundTrip(t *testing.T, size int) {
	t.Helper()
	k, err := Generate(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plain := make([]byte, size)
	_, _ = rand.Read(plain)
	var sealed bytes.Buffer
	if err := Seal(&sealed, bytes.NewReader(plain), k.Public(), rand.Reader); err != nil {
		t.Fatal(err)
	}
	if size > 32 && bytes.Contains(sealed.Bytes(), plain[:32]) {
		t.Fatal("the sealed file holds the plaintext")
	}
	var out bytes.Buffer
	if err := Open(&out, bytes.NewReader(sealed.Bytes()), k); err != nil {
		t.Fatalf("size %d: %v", size, err)
	}
	if !bytes.Equal(out.Bytes(), plain) {
		t.Fatalf("size %d: opened content differs", size)
	}
	// Any changed byte fails.
	b := sealed.Bytes()
	for _, i := range []int{0, len(Magic) + 3, headerLen - 1, headerLen, len(b) / 2, len(b) - 1} {
		c := append([]byte{}, b...)
		c[i] ^= 0x40
		if err := Open(&bytes.Buffer{}, bytes.NewReader(c), k); !errors.Is(err, ErrOpen) {
			t.Fatalf("size %d: byte %d changed and it opened: %v", size, i, err)
		}
	}
	// A truncated file fails, even at a chunk boundary.
	if size > ChunkSize { // more than one chunk: drop all but the first
		if err := Open(&bytes.Buffer{}, bytes.NewReader(b[:headerLen+ChunkSize+16]), k); !errors.Is(err, ErrOpen) {
			t.Fatalf("size %d: truncated at a chunk boundary and it opened: %v", size, err)
		}
	}
	// Another key fails.
	other, _ := Generate(rand.Reader)
	if err := Open(&bytes.Buffer{}, bytes.NewReader(b), other); !errors.Is(err, ErrOpen) {
		t.Fatalf("another key: %v", err)
	}
}

func TestSealOpen(t *testing.T) {
	for _, n := range []int{0, 1, 100, ChunkSize - 1, ChunkSize, ChunkSize + 1, 3*ChunkSize + 77} {
		roundTrip(t, n)
	}
}

func TestKeysRoundTrip(t *testing.T) {
	k, _ := Generate(rand.Reader)
	pk, err := ParsePrivate(k.String())
	if err != nil || pk.Public().String() != k.Public().String() {
		t.Fatalf("private: %v", err)
	}
	pub, err := ParsePublic(k.Public().String())
	if err != nil || !bytes.Equal(pub.ID(), k.Public().ID()) {
		t.Fatalf("public: %v", err)
	}
	for _, bad := range []string{"", "sb-recovery-pub:", "sb-recovery-key:AAAA", k.String()} {
		if _, err := ParsePublic(bad); !errors.Is(err, ErrKey) {
			t.Errorf("ParsePublic(%q): %v", bad, err)
		}
	}
}

func TestSealWithKey(t *testing.T) {
	key, _ := NewKey()
	plain := bytes.Repeat([]byte("vault "), 30000)
	var s bytes.Buffer
	if err := SealWithKey(&s, bytes.NewReader(plain), key); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := OpenWithKey(&out, bytes.NewReader(s.Bytes()), key); err != nil || !bytes.Equal(out.Bytes(), plain) {
		t.Fatalf("open: %v", err)
	}
	other, _ := NewKey()
	if err := OpenWithKey(&bytes.Buffer{}, bytes.NewReader(s.Bytes()), other); !errors.Is(err, ErrOpen) {
		t.Fatalf("other key: %v", err)
	}
	if err := SealWithKey(&bytes.Buffer{}, bytes.NewReader(nil), key[:16]); !errors.Is(err, ErrKey) {
		t.Fatalf("short key: %v", err)
	}
}
