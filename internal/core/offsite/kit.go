package offsite

import (
	"bufio"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"example.com/second-brain/internal/core/shamir"
)

// The recovery kit (sb-selfhost(7), The recovery kit; SB-1902): a random kit key seals an
// archive of the box's keys and the recovery private key (SealWithKey), and is split into
// KitShares shares, any KitNeed of which rebuild it. Each share is a page to print, with its
// number and a checksum, so a mistyped share is refused before it is used.
const (
	KitShares = 5
	KitNeed   = 3
)

// ErrShare is a share page that can't be read or fails its checksum.
var ErrShare = errors.New("offsite: bad share")

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// KeyCheck is a short check of a kit key, kept in the kit's record (not secret), so a
// rebuild from the wrong shares is told apart from a changed archive.
func KeyCheck(key []byte) string {
	s := sha256.Sum256(append([]byte("sb-kit check"), key...))
	return hex.EncodeToString(s[:4])
}

func shareSum(kit, tenant string, sh shamir.Share) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\n%s\n%d\n", kit, tenant, sh.X)
	h.Write(sh.Y)
	return hex.EncodeToString(h.Sum(nil)[:4])
}

// SharePage is one share as a page of text.
func SharePage(kit, tenant, made string, sh shamir.Share) string {
	enc := b32.EncodeToString(sh.Y)
	var groups []string
	for i := 0; i < len(enc); i += 5 {
		groups = append(groups, enc[i:min(i+5, len(enc))])
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Second Brain recovery kit: share %d of %d. Any %d shares rebuild the box's keys.\n", sh.X, KitShares, KitNeed)
	b.WriteString("Keep it on paper, somewhere only you can reach. Never photograph, scan or email it.\n\n")
	fmt.Fprintf(&b, "tenant: %s\nkit: %s\nmade: %s\nshare: %d\n\n", tenant, kit, made, sh.X)
	for i := 0; i < len(groups); i += 4 {
		b.WriteString(strings.Join(groups[i:min(i+4, len(groups))], " ") + "\n")
	}
	fmt.Fprintf(&b, "\nchecksum: %s\n", shareSum(kit, tenant, sh))
	return b.String()
}

// ParseSharePage reads a share page, typed back in or as printed, and checks its checksum.
func ParseSharePage(text string) (kit, tenant string, sh shamir.Share, err error) {
	var body, sum string
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		k, v, ok := strings.Cut(line, ":")
		v = strings.TrimSpace(v)
		switch {
		case ok && k == "tenant":
			tenant = v
		case ok && k == "kit":
			kit = v
		case ok && k == "share":
			n, perr := strconv.Atoi(v)
			if perr != nil || n < 1 || n > 255 {
				return "", "", sh, fmt.Errorf("%w: the share number %q", ErrShare, v)
			}
			sh.X = byte(n)
		case ok && k == "checksum":
			sum = strings.ToLower(v)
		case !ok && line != "" && isB32Line(line):
			body += strings.ReplaceAll(line, " ", "")
		}
	}
	if kit == "" || tenant == "" || sh.X == 0 || body == "" || sum == "" {
		return "", "", sh, fmt.Errorf("%w: a share page has tenant, kit, share, the groups and checksum", ErrShare)
	}
	y, derr := b32.DecodeString(strings.ToUpper(body))
	if derr != nil {
		return "", "", sh, fmt.Errorf("%w: share %d's groups aren't base32: check each character (no 0, 1, 8 or 9)", ErrShare, sh.X)
	}
	sh.Y = y
	if shareSum(kit, tenant, sh) != sum {
		return "", "", sh, fmt.Errorf("%w: share %d fails its checksum: a character is mistyped", ErrShare, sh.X)
	}
	return kit, tenant, sh, nil
}

func isB32Line(s string) bool {
	for _, c := range strings.ToUpper(s) {
		if c != ' ' && (c < 'A' || c > 'Z') && (c < '2' || c > '7') {
			return false
		}
	}
	return true
}
