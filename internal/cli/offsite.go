package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/offsite"
	"example.com/second-brain/internal/core/shamir"
	"example.com/second-brain/internal/errs"
)

// flags reads --name VALUE options (repeatable ones collected) and the positional
// arguments, refusing an option it doesn't know.
func flags(cmd string, args []string, known ...string) (map[string][]string, []string, error) {
	ok := map[string]bool{}
	for _, k := range known {
		ok[k] = true
	}
	opts := map[string][]string{}
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}
		if !ok[a] {
			return nil, nil, errs.New(errs.EUSAGE, "%s: unknown option %q", cmd, a)
		}
		if i+1 >= len(args) {
			return nil, nil, errs.New(errs.EUSAGE, "%s: %s needs a value", cmd, a)
		}
		opts[a] = append(opts[a], args[i+1])
		i++
	}
	return opts, pos, nil
}

func one(opts map[string][]string, k string) string {
	if v := opts[k]; len(v) > 0 {
		return v[len(v)-1]
	}
	return ""
}

// writeAtomic writes through fn to a temporary file beside path, kept as path (mode 0600)
// only when fn returns nil.
func writeAtomic(path string, fn func(w io.Writer) error) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if err := fn(f); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// backupKeygen makes a recovery key pair (sb-selfhost(7), Sealed off-box copies; SB-1901):
// `sb backup keygen --public FILE --private FILE`.
func (e *env) backupKeygen(args []string) int {
	const cmd = "sb backup keygen"
	opts, pos, err := flags(cmd, args, "--public", "--private")
	if err != nil {
		return e.fail(cmd, err)
	}
	pub, priv := one(opts, "--public"), one(opts, "--private")
	if pub == "" || priv == "" || len(pos) > 0 {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb backup keygen --public FILE --private FILE"))
	}
	for _, p := range []string{pub, priv} {
		if _, err := os.Stat(p); err == nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "%s exists; a recovery key is made once", p))
		}
	}
	k, err := offsite.Generate(rand.Reader)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	if err := writeAtomic(priv, func(w io.Writer) error { _, err := fmt.Fprintln(w, k.String()); return err }); err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't write %s: %v", priv, err))
	}
	if err := os.WriteFile(pub, []byte(k.Public().String()+"\n"), 0o644); err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't write %s: %v", pub, err))
	}
	if e.g.JSON {
		return e.json(map[string]any{"public": k.Public().String(), "id": hex.EncodeToString(k.Public().ID())})
	}
	fmt.Fprintf(e.out, "recovery key %s: public half in %s, private half in %s\n", hex.EncodeToString(k.Public().ID()), pub, priv)
	return 0
}

func readKeyFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// backupSeal seals a file to the recovery key's public half: `sb backup seal --to PUBFILE
// [--out FILE] FILE` (default FILE.sealed).
func (e *env) backupSeal(args []string) int {
	const cmd = "sb backup seal"
	opts, pos, err := flags(cmd, args, "--to", "--out")
	if err != nil {
		return e.fail(cmd, err)
	}
	if len(pos) != 1 || one(opts, "--to") == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb backup seal --to PUBFILE [--out FILE] FILE"))
	}
	s, err := readKeyFile(one(opts, "--to"))
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", one(opts, "--to")))
	}
	pub, err := offsite.ParsePublic(s)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	out := one(opts, "--out")
	if out == "" {
		out = pos[0] + ".sealed"
	}
	in, err := os.Open(pos[0])
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", pos[0]))
	}
	defer func() { _ = in.Close() }()
	if err := writeAtomic(out, func(w io.Writer) error { return offsite.Seal(w, in, pub, rand.Reader) }); err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	if !e.g.Quiet {
		fmt.Fprintf(e.out, "sealed %s to recovery key %s\n", out, hex.EncodeToString(pub.ID()))
	}
	return 0
}

// backupOpen opens a sealed file with the recovery key's private half: `sb backup open
// --key KEYFILE [--out FILE] FILE.sealed`. A file that was changed, or sealed to another
// key, writes nothing.
func (e *env) backupOpen(args []string) int {
	const cmd = "sb backup open"
	opts, pos, err := flags(cmd, args, "--key", "--out")
	if err != nil {
		return e.fail(cmd, err)
	}
	if len(pos) != 1 || one(opts, "--key") == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb backup open --key KEYFILE [--out FILE] FILE.sealed"))
	}
	s, err := readKeyFile(one(opts, "--key"))
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", one(opts, "--key")))
	}
	key, err := offsite.ParsePrivate(s)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	out := one(opts, "--out")
	if out == "" {
		out = strings.TrimSuffix(pos[0], ".sealed")
		if out == pos[0] {
			return e.fail(cmd, errs.New(errs.EUSAGE, "%s doesn't end in .sealed; name the output with --out", pos[0]))
		}
	}
	in, err := os.Open(pos[0])
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", pos[0]))
	}
	defer func() { _ = in.Close() }()
	if err := writeAtomic(out, func(w io.Writer) error { return offsite.Open(w, in, key) }); err != nil {
		return e.fail(cmd, errs.New(errs.ECHAIN, "%v; nothing was written", err))
	}
	if !e.g.Quiet {
		fmt.Fprintf(e.out, "opened %s\n", out)
	}
	return 0
}

// kitRecord is the recovery kit's record, kitID.json: who holds which share, never a share.
type kitRecord struct {
	Kit        string            `json:"kit"`
	Tenant     string            `json:"tenant"`
	Made       string            `json:"made"`
	Shares     int               `json:"shares"`
	Need       int               `json:"need"`
	KeyCheck   string            `json:"key_check"`
	Custodians map[string]string `json:"custodians,omitempty"`
}

// recoverySplit makes a recovery kit (sb-selfhost(7), The recovery kit; SB-1902):
// `sb recovery split --tenant T --out DIR [--custodian N=NAME]... ARCHIVE`. A random kit
// key seals ARCHIVE into DIR/KIT.sealed and is split into five share pages,
// DIR/KIT-share-N.txt, any three of which rebuild it; DIR/KIT.json records the kit and who
// holds each share.
func (e *env) recoverySplit(args []string) int {
	const cmd = "sb recovery split"
	opts, pos, err := flags(cmd, args, "--tenant", "--out", "--custodian")
	if err != nil {
		return e.fail(cmd, err)
	}
	tenant, dir := one(opts, "--tenant"), one(opts, "--out")
	if tenant == "" && e.g.Tenant != "" {
		tenant = e.g.Tenant
	}
	if len(pos) != 1 || tenant == "" || dir == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb recovery split --tenant T --out DIR [--custodian N=NAME]... ARCHIVE"))
	}
	holders := map[string]string{}
	for _, c := range opts["--custodian"] {
		n, name, ok := strings.Cut(c, "=")
		x, err := strconv.Atoi(n)
		if !ok || err != nil || x < 1 || x > offsite.KitShares || strings.TrimSpace(name) == "" {
			return e.fail(cmd, errs.New(errs.EUSAGE, "--custodian is N=NAME, N from 1 to %d", offsite.KitShares))
		}
		holders[n] = strings.TrimSpace(name)
	}
	key, err := offsite.NewKey()
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	idb := make([]byte, 4)
	_, _ = rand.Read(idb)
	kit := "kit_" + hex.EncodeToString(idb)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't make %s", dir))
	}
	in, err := os.Open(pos[0])
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", pos[0]))
	}
	defer func() { _ = in.Close() }()
	sealed := filepath.Join(dir, kit+".sealed")
	if err := writeAtomic(sealed, func(w io.Writer) error { return offsite.SealWithKey(w, in, key) }); err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	shares, err := shamir.Split(key, offsite.KitShares, offsite.KitNeed, rand.Reader)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	made := time.Now().UTC().Format("2006-01-02")
	var pages []string
	for _, sh := range shares {
		p := filepath.Join(dir, fmt.Sprintf("%s-share-%d.txt", kit, sh.X))
		page := offsite.SharePage(kit, tenant, made, sh)
		if err := writeAtomic(p, func(w io.Writer) error { _, err := io.WriteString(w, page); return err }); err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "can't write %s", p))
		}
		pages = append(pages, p)
	}
	rec := kitRecord{Kit: kit, Tenant: tenant, Made: made, Shares: offsite.KitShares, Need: offsite.KitNeed, KeyCheck: offsite.KeyCheck(key), Custodians: holders}
	b, _ := json.MarshalIndent(rec, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, kit+".json"), append(b, '\n'), 0o644); err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't write the kit's record"))
	}
	if e.g.JSON {
		return e.json(map[string]any{"kit": kit, "sealed": sealed, "shares": pages})
	}
	fmt.Fprintf(e.out, "%s: %s sealed; print each share and give it to its custodian, then delete it from this machine:\n", kit, sealed)
	for _, p := range pages {
		fmt.Fprintf(e.out, "  %s\n", p)
	}
	return 0
}

// recoveryJoin rebuilds a kit's archive from three or more share pages: `sb recovery join
// --kit KIT.sealed --out FILE SHARE...`. Each page's checksum is checked first, then the
// rebuilt key against the kit's record, then the archive opens or nothing is written.
func (e *env) recoveryJoin(args []string) int {
	const cmd = "sb recovery join"
	opts, pos, err := flags(cmd, args, "--kit", "--out")
	if err != nil {
		return e.fail(cmd, err)
	}
	sealed, out := one(opts, "--kit"), one(opts, "--out")
	if sealed == "" || out == "" || len(pos) == 0 {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb recovery join --kit KIT.sealed --out FILE SHARE..."))
	}
	rb, err := os.ReadFile(strings.TrimSuffix(sealed, ".sealed") + ".json")
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "no record beside %s (KIT.json)", sealed))
	}
	var rec kitRecord
	if err := json.Unmarshal(rb, &rec); err != nil || rec.Kit == "" {
		return e.fail(cmd, errs.New(errs.EINVAL, "the kit's record can't be read"))
	}
	var shares []shamir.Share
	var used []int
	seen := map[byte]bool{}
	for _, p := range pos {
		b, err := os.ReadFile(p)
		if err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", p))
		}
		kit, tenant, sh, err := offsite.ParseSharePage(string(b))
		if err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "%s: %v", p, err))
		}
		if kit != rec.Kit || tenant != rec.Tenant {
			return e.fail(cmd, errs.New(errs.EINVAL, "%s is a share of %s (%s), not of %s", p, kit, tenant, rec.Kit))
		}
		if !seen[sh.X] {
			seen[sh.X] = true
			shares = append(shares, sh)
			used = append(used, int(sh.X))
		}
	}
	if len(shares) < rec.Need {
		return e.fail(cmd, errs.New(errs.EINVAL, "%d different share(s); %s needs %d", len(shares), rec.Kit, rec.Need))
	}
	key, err := shamir.Combine(shares)
	if err != nil || offsite.KeyCheck(key) != rec.KeyCheck {
		return e.fail(cmd, errs.New(errs.EINVAL, "the shares don't rebuild %s's key", rec.Kit))
	}
	in, err := os.ReadFile(sealed)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", sealed))
	}
	if err := writeAtomic(out, func(w io.Writer) error { return offsite.OpenWithKey(w, bytes.NewReader(in), key) }); err != nil {
		return e.fail(cmd, errs.New(errs.ECHAIN, "%v; nothing was written", err))
	}
	sort.Ints(used)
	if e.g.JSON {
		return e.json(map[string]any{"kit": rec.Kit, "shares": used, "out": out})
	}
	fmt.Fprintf(e.out, "%s: rebuilt from shares %s into %s\n", rec.Kit, strings.Trim(strings.Join(strings.Fields(fmt.Sprint(used)), ", "), "[]"), out)
	return 0
}
