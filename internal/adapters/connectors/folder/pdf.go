package folder

import (
	"bytes"
	"compress/zlib"
	"io"
	"strings"
	"unicode/utf8"
)

// maxStream is the most one decompressed PDF stream may grow to, so a small file can't
// expand without bound.
const maxStream = 16 << 20

// PDFText is the text of a PDF's content streams, read from their text operators (Tj, TJ,
// ' and "), uncompressed or Flate-compressed. It covers the PDFs office tools write with
// standard fonts; text drawn with a font that maps codes to glyphs some other way comes out
// as nothing, and the file is then indexed by name only. It never fails: what it can't read
// it leaves out.
func PDFText(b []byte) string {
	var out strings.Builder
	for len(b) > 0 {
		i := bytes.Index(b, []byte("stream"))
		if i < 0 {
			break
		}
		// "endstream" also contains "stream"; skip it.
		if i >= 3 && string(b[i-3:i]) == "end" {
			b = b[i+6:]
			continue
		}
		dict := b[:i]
		if d := bytes.LastIndex(dict, []byte("<<")); d >= 0 {
			dict = dict[d:]
		}
		body := b[i+6:]
		body = bytes.TrimPrefix(body, []byte("\r"))
		body = bytes.TrimPrefix(body, []byte("\n"))
		end := bytes.Index(body, []byte("endstream"))
		if end < 0 {
			break
		}
		data := body[:end]
		b = body[end+9:]
		if bytes.Contains(dict, []byte("/Image")) || bytes.Contains(dict, []byte("/XRef")) {
			continue
		}
		if bytes.Contains(dict, []byte("/FlateDecode")) {
			z, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				continue
			}
			dec, err := io.ReadAll(io.LimitReader(z, maxStream))
			_ = z.Close()
			if err != nil && len(dec) == 0 {
				continue
			}
			data = dec
		} else if bytes.Contains(dict, []byte("/Filter")) {
			continue // another filter: not read
		}
		if t := content(data); t != "" {
			out.WriteString(t)
			out.WriteString("\n")
		}
	}
	return strings.TrimSpace(out.String())
}

// content reads one content stream's text operators.
func content(s []byte) string {
	var out strings.Builder
	var pending []string // strings since the last operator
	emit := func(sep string) {
		for _, p := range pending {
			out.WriteString(p)
		}
		pending = pending[:0]
		out.WriteString(sep)
	}
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '(':
			str, n := literal(s[i:])
			pending = append(pending, str)
			i += n
		case c == '<' && i+1 < len(s) && s[i+1] != '<':
			str, n := hexString(s[i:])
			pending = append(pending, str)
			i += n
		case c == '%':
			for i < len(s) && s[i] != '\n' && s[i] != '\r' {
				i++
			}
		case c == '-' || (c >= '0' && c <= '9') || c == '.':
			j := i + 1
			for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == '.') {
				j++
			}
			// In a TJ array a large negative offset is a word gap.
			if c == '-' && j-i >= 4 && len(pending) > 0 {
				pending = append(pending, " ")
			}
			i = j
		case isLetter(c) || c == '\'' || c == '"' || c == '*':
			j := i + 1
			for j < len(s) && (isLetter(s[j]) || s[j] == '*') {
				j++
			}
			switch string(s[i:j]) {
			case "Tj", "TJ":
				emit("")
			case "'", "\"", "T*":
				emit("\n")
			case "Td", "TD", "Tm":
				if out.Len() > 0 || len(pending) > 0 {
					emit("\n")
				}
			case "ET":
				emit("\n")
			case "BT":
				pending = pending[:0]
			default:
				pending = pending[:0] // a string before another operator isn't text drawn
			}
			i = j
		default:
			i++
		}
	}
	lines := strings.Split(out.String(), "\n")
	kept := lines[:0]
	for _, l := range lines {
		if l = strings.TrimSpace(l); l != "" {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// literal reads a PDF literal string, escapes and nested parentheses included, and returns
// its text and the bytes it took.
func literal(s []byte) (string, int) {
	var b []byte
	depth := 0
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s):
			i++
			switch e := s[i]; e {
			case 'n':
				b = append(b, '\n')
			case 'r':
				b = append(b, '\r')
			case 't':
				b = append(b, '\t')
			case 'b', 'f':
			case '\r', '\n':
			case '0', '1', '2', '3', '4', '5', '6', '7':
				v, n := 0, 0
				for n < 3 && i < len(s) && s[i] >= '0' && s[i] <= '7' {
					v = v*8 + int(s[i]-'0')
					i++
					n++
				}
				b = append(b, byte(v))
				continue
			default:
				b = append(b, e)
			}
		case c == '(':
			if depth > 0 {
				b = append(b, c)
			}
			depth++
		case c == ')':
			depth--
			if depth == 0 {
				return decode(b), i + 1
			}
			b = append(b, c)
		default:
			b = append(b, c)
		}
		i++
	}
	return decode(b), i
}

func hexString(s []byte) (string, int) {
	end := bytes.IndexByte(s, '>')
	if end < 0 {
		return "", len(s)
	}
	var b []byte
	hi := -1
	for _, c := range s[1:end] {
		var v int
		switch {
		case c >= '0' && c <= '9':
			v = int(c - '0')
		case c >= 'a' && c <= 'f':
			v = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			v = int(c-'A') + 10
		default:
			continue
		}
		if hi < 0 {
			hi = v
		} else {
			b = append(b, byte(hi<<4|v))
			hi = -1
		}
	}
	if hi >= 0 {
		b = append(b, byte(hi<<4))
	}
	// A hex string without a byte-order mark is usually glyph codes of a font that maps
	// them some other way: left out, rather than indexed as noise.
	if len(b) < 2 || b[0] != 0xfe || b[1] != 0xff {
		return "", end + 1
	}
	return decode(b), end + 1
}

// decode turns a string's bytes into text: UTF-16 with its byte-order mark, else
// PDFDocEncoding read as Latin-1, keeping only printable characters.
func decode(b []byte) string {
	var r []rune
	if len(b) >= 2 && b[0] == 0xfe && b[1] == 0xff {
		for i := 2; i+1 < len(b); i += 2 {
			r = append(r, rune(b[i])<<8|rune(b[i+1]))
		}
	} else {
		for _, c := range b {
			r = append(r, rune(c))
		}
	}
	var out strings.Builder
	for _, c := range r {
		if c == '\n' || c == '\t' || c >= 0x20 && c != 0x7f && utf8.ValidRune(c) {
			out.WriteRune(c)
		}
	}
	return out.String()
}
