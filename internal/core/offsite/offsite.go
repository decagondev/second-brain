// Package offsite seals files to a recovery key, so a copy off the box is unreadable without
// it (sb-selfhost(7), Sealed off-box copies; SB-1901). Each file gets a fresh AES-256-GCM
// key, wrapped to the recipient's X25519 public key with an ephemeral key pair and HKDF-
// SHA256; the content is encrypted in 64 KiB chunks, each authenticated, the last one
// marked, so a changed, dropped, reordered or truncated chunk fails to open. Standard
// library only.
//
// A sealed file is:
//
//	"sb-seal/v1\n" | recipient id (8) | ephemeral public key (32) | wrapped file key (48) | chunks
//
// where each chunk is AES-256-GCM over at most ChunkSize bytes with the nonce
// counter (11 bytes, big-endian) | last (1 byte, 1 on the final chunk), and the header as
// the first chunk's additional data.
package offsite

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Magic starts every sealed file.
const Magic = "sb-seal/v1\n"

// ChunkSize is the plaintext size of every chunk but the last.
const ChunkSize = 64 << 10

const (
	pubPrefix  = "sb-recovery-pub:"
	privPrefix = "sb-recovery-key:"
	headerLen  = len(Magic) + 8 + 32 + 48
)

// ErrOpen is a file that can't be opened: sealed to another key, changed, or not sealed.
var ErrOpen = errors.New("offsite: can't open")

// ErrKey is a key that can't be read.
var ErrKey = errors.New("offsite: bad key")

// PublicKey is a recovery key's public half: what a box keeps, to seal to.
type PublicKey struct{ k *ecdh.PublicKey }

// PrivateKey is a recovery key's private half: never kept on the box (SB-1902).
type PrivateKey struct{ k *ecdh.PrivateKey }

// Generate makes a recovery key pair.
func Generate(rnd io.Reader) (PrivateKey, error) {
	k, err := ecdh.X25519().GenerateKey(rnd)
	if err != nil {
		return PrivateKey{}, err
	}
	return PrivateKey{k}, nil
}

// Public is the private key's public half.
func (p PrivateKey) Public() PublicKey { return PublicKey{p.k.PublicKey()} }

// String is "sb-recovery-pub:BASE64".
func (p PublicKey) String() string {
	return pubPrefix + base64.RawURLEncoding.EncodeToString(p.k.Bytes())
}

// String is "sb-recovery-key:BASE64".
func (p PrivateKey) String() string {
	return privPrefix + base64.RawURLEncoding.EncodeToString(p.k.Bytes())
}

// ID is the public key's 8-byte id, written into every file sealed to it.
func (p PublicKey) ID() []byte {
	s := sha256.Sum256(append([]byte("sb-seal id"), p.k.Bytes()...))
	return s[:8]
}

// ParsePublic reads "sb-recovery-pub:BASE64".
func ParsePublic(s string) (PublicKey, error) {
	b, err := decodeKey(s, pubPrefix)
	if err != nil {
		return PublicKey{}, err
	}
	k, err := ecdh.X25519().NewPublicKey(b)
	if err != nil {
		return PublicKey{}, fmt.Errorf("%w: not an X25519 public key", ErrKey)
	}
	return PublicKey{k}, nil
}

// ParsePrivate reads "sb-recovery-key:BASE64".
func ParsePrivate(s string) (PrivateKey, error) {
	b, err := decodeKey(s, privPrefix)
	if err != nil {
		return PrivateKey{}, err
	}
	k, err := ecdh.X25519().NewPrivateKey(b)
	if err != nil {
		return PrivateKey{}, fmt.Errorf("%w: not an X25519 private key", ErrKey)
	}
	return PrivateKey{k}, nil
}

func decodeKey(s, prefix string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, prefix) {
		return nil, fmt.Errorf("%w: want %s…", ErrKey, prefix)
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, prefix))
	if err != nil || len(b) != 32 {
		return nil, fmt.Errorf("%w: want 32 bytes in base64url", ErrKey)
	}
	return b, nil
}

func wrapKey(shared, eph, rcpt []byte) (cipher.AEAD, error) {
	kek, err := hkdf.Key(sha256.New, shared, append(append([]byte{}, eph...), rcpt...), "sb-seal/v1 wrap", 32)
	if err != nil {
		return nil, err
	}
	return gcm(kek)
}

func gcm(key []byte) (cipher.AEAD, error) {
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(b)
}

// Seal writes r, sealed to the recipient, to w.
func Seal(w io.Writer, r io.Reader, to PublicKey, rnd io.Reader) error {
	eph, err := ecdh.X25519().GenerateKey(rnd)
	if err != nil {
		return err
	}
	shared, err := eph.ECDH(to.k)
	if err != nil {
		return err
	}
	wrap, err := wrapKey(shared, eph.PublicKey().Bytes(), to.k.Bytes())
	if err != nil {
		return err
	}
	fileKey := make([]byte, 32)
	if _, err := io.ReadFull(rnd, fileKey); err != nil {
		return err
	}
	head := append([]byte(Magic), to.ID()...)
	head = append(head, eph.PublicKey().Bytes()...)
	head = append(head, wrap.Seal(nil, make([]byte, 12), fileKey, head)...)
	if _, err := w.Write(head); err != nil {
		return err
	}
	return sealChunks(w, r, fileKey, head)
}

// sealChunks encrypts r in chunks under key, with the header bound into the first.
func sealChunks(w io.Writer, r io.Reader, key, head []byte) error {
	a, err := gcm(key)
	if err != nil {
		return err
	}
	buf := make([]byte, ChunkSize+1) // one byte of look-ahead tells the last chunk
	n, err := io.ReadFull(r, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return err
	}
	var counter uint64
	for {
		last := n <= ChunkSize
		size := min(n, ChunkSize)
		aad := []byte(nil)
		if counter == 0 {
			aad = head
		}
		if _, err := w.Write(a.Seal(nil, nonce(counter, last), buf[:size], aad)); err != nil {
			return err
		}
		if last {
			return nil
		}
		counter++
		copy(buf, buf[ChunkSize:n])
		m, err := io.ReadFull(r, buf[n-ChunkSize:])
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
			return err
		}
		n = n - ChunkSize + m
	}
}

func nonce(counter uint64, last bool) []byte {
	n := make([]byte, 12)
	for i := 0; i < 8; i++ {
		n[10-i] = byte(counter >> (8 * i))
	}
	if last {
		n[11] = 1
	}
	return n
}

// Open writes the content of a file sealed to key. Only authenticated chunks are written,
// but a file changed after its first chunk fails part way: write to a temporary file and
// keep it only when Open returns nil.
func Open(w io.Writer, r io.Reader, key PrivateKey) error {
	head := make([]byte, headerLen)
	if _, err := io.ReadFull(r, head); err != nil || !bytes.HasPrefix(head, []byte(Magic)) {
		return fmt.Errorf("%w: not a sealed file", ErrOpen)
	}
	id := head[len(Magic) : len(Magic)+8]
	if !bytes.Equal(id, key.Public().ID()) {
		return fmt.Errorf("%w: sealed to another recovery key", ErrOpen)
	}
	ephB := head[len(Magic)+8 : len(Magic)+40]
	eph, err := ecdh.X25519().NewPublicKey(ephB)
	if err != nil {
		return fmt.Errorf("%w: bad header", ErrOpen)
	}
	shared, err := key.k.ECDH(eph)
	if err != nil {
		return fmt.Errorf("%w: bad header", ErrOpen)
	}
	wrap, err := wrapKey(shared, ephB, key.k.PublicKey().Bytes())
	if err != nil {
		return err
	}
	fileKey, err := wrap.Open(nil, make([]byte, 12), head[len(Magic)+40:], head[:len(Magic)+40])
	if err != nil {
		return fmt.Errorf("%w: the file key doesn't open (changed, or another key)", ErrOpen)
	}
	return openChunks(w, r, fileKey, head)
}

// openChunks decrypts chunks under key, with the header bound into the first.
func openChunks(w io.Writer, r io.Reader, key, aadHead []byte) error {
	a, err := gcm(key)
	if err != nil {
		return err
	}
	const sealed = ChunkSize + 16
	buf := make([]byte, sealed+1)
	n, err := io.ReadFull(r, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return err
	}
	var counter uint64
	for {
		last := n <= sealed
		size := min(n, sealed)
		aad := []byte(nil)
		if counter == 0 {
			aad = aadHead
		}
		p, err := a.Open(nil, nonce(counter, last), buf[:size], aad)
		if err != nil {
			return fmt.Errorf("%w: chunk %d is changed, missing or out of order", ErrOpen, counter)
		}
		if _, err := w.Write(p); err != nil {
			return err
		}
		if last {
			return nil
		}
		counter++
		copy(buf, buf[sealed:n])
		m, err := io.ReadFull(r, buf[n-sealed:])
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
			return err
		}
		n = n - sealed + m
	}
}

// SealWithKey seals r under a 32-byte symmetric key (the recovery kit's archive, SB-1902):
// the same chunks, with no key wrap. The header is "sb-kit/v1\n".
func SealWithKey(w io.Writer, r io.Reader, key []byte) error {
	if len(key) != 32 {
		return fmt.Errorf("%w: a kit key is 32 bytes", ErrKey)
	}
	head := []byte(kitMagic)
	if _, err := w.Write(head); err != nil {
		return err
	}
	return sealChunks(w, r, key, head)
}

// OpenWithKey opens what SealWithKey sealed.
func OpenWithKey(w io.Writer, r io.Reader, key []byte) error {
	if len(key) != 32 {
		return fmt.Errorf("%w: a kit key is 32 bytes", ErrKey)
	}
	head := make([]byte, len(kitMagic))
	if _, err := io.ReadFull(r, head); err != nil || string(head) != kitMagic {
		return fmt.Errorf("%w: not a sealed kit", ErrOpen)
	}
	return openChunks(w, r, key, head)
}

const kitMagic = "sb-kit/v1\n"

// NewKey is 32 random bytes for SealWithKey.
func NewKey() ([]byte, error) {
	k := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, k)
	return k, err
}

// IsSealed is whether data starts as a sealed file does: to a recovery key, or a kit's
// archive. It says nothing about whether it opens.
func IsSealed(data []byte) bool {
	return bytes.HasPrefix(data, []byte(Magic)) || bytes.HasPrefix(data, []byte(kitMagic))
}
