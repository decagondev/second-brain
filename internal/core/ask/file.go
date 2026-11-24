package ask

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// Ask about a file (sb-ask(3), Request `file`; SB-3802): one file's text, for one ask, read
// as ingest reads a document, and kept nowhere.

const (
	MaxFileBytes    = 2 << 20
	MaxFilePassages = 3
	// FileKind is the cite kind of the asker's own file.
	FileKind = "file"
)

// FileUpload is the file an ask is about.
type FileUpload struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
}

// fileName is the upload's base name as a cite can carry it.
func fileName(name string) string {
	n := path.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	n = strings.Map(func(r rune) rune {
		if r == '@' || r == ':' || r < ' ' {
			return '_'
		}
		return r
	}, n)
	if rs := []rune(n); len(rs) > 120 {
		n = string(rs[:120])
	}
	return n
}

// checkFile refuses a file the ask can't take, before anything else runs.
func (a Agent) checkFile(f *FileUpload) error {
	if f == nil {
		return nil
	}
	if len(f.Data) == 0 || len(f.Data) > MaxFileBytes {
		return fmt.Errorf("%w: a file is 1 byte to 2 MB", ErrInvalid)
	}
	if a.FileText == nil {
		return fmt.Errorf("%w: this cell doesn't read files", ErrInvalid)
	}
	if n := fileName(f.Name); n == "" || n == "." || n == "/" {
		return fmt.Errorf("%w: a file needs a name with its extension", ErrInvalid)
	}
	return nil
}

// fileChunks are the file's passages most like q, at most MaxFilePassages, as context
// chunks: stripped, flagged and with secrets dropped as ingest does (sb-ingest(7)).
func (a Agent) fileChunks(ctx context.Context, q string, f *FileUpload) ([]ports.Chunk, error) {
	name := fileName(f.Name)
	text, ok := a.FileText(name, f.Data)
	if !ok {
		return nil, fmt.Errorf("%w: the folder connector doesn't read this type of file", ErrInvalid)
	}
	san := ingest.Sanitiser{}
	clean, docFlags := san.Sanitise(text)
	body := ingest.Normalise(clean)
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("%w: the file has no text to read", ErrInvalid)
	}
	rs := []rune(body)
	cite := FileKind + ":" + name
	var cs []ports.Chunk
	for _, sp := range ingest.Chunks(body) {
		t := string(rs[sp.Start:sp.End])
		if ingest.SecretIn(t) != "" {
			continue // a secret is never sent to the model (LAW-07)
		}
		flags := append(append([]string(nil), docFlags...), san.Flag(t)...)
		if ingest.HasPII(t) {
			flags = append(flags, ingest.FlagPersonal)
		}
		cs = append(cs, ports.Chunk{ID: fmt.Sprintf("file_%d", sp.Start), Tenant: a.Tenant, Cite: cite, Kind: FileKind, SourceID: name,
			Start: sp.Start, End: sp.End, Text: t, Flags: flags, Title: "Your file: " + name, Seen: a.Now(), Section: ingest.Section(body, sp.Start)})
	}
	if len(cs) == 0 {
		return nil, fmt.Errorf("%w: the file has no text that can be sent", ErrInvalid)
	}
	if len(cs) <= MaxFilePassages || a.Embedder == nil {
		if len(cs) > MaxFilePassages {
			cs = cs[:MaxFilePassages]
		}
		return cs, nil
	}
	texts := []string{q}
	for _, c := range cs {
		texts = append(texts, c.Text)
	}
	vs, err := a.Embedder.Embed(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("%w: embedder: %v", ErrUnavailable, err)
	}
	if len(vs) != len(texts) {
		return cs[:MaxFilePassages], nil
	}
	idx := make([]int, len(cs))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool { return cosine(vs[0], vs[idx[i]+1]) > cosine(vs[0], vs[idx[j]+1]) })
	idx = idx[:MaxFilePassages]
	sort.Ints(idx) // in the file's order
	out := make([]ports.Chunk, 0, len(idx))
	for _, i := range idx {
		out = append(out, cs[i])
	}
	return out, nil
}
