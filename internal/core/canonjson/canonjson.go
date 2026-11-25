// Package canonjson is RFC 8785 canonical JSON for the subset the platform hashes:
// objects, arrays, strings, booleans, null, and integers within ±2^53 (approval payloads,
// release bundles). Other numbers are refused so every implementation hashes the same bytes.
package canonjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
)

// ErrUnsupported is input outside the subset.
var ErrUnsupported = errors.New("canonjson: not canonicalisable")

// Encode is the RFC 8785 JSON of raw for the subset the platform hashes: objects, arrays, strings,
// booleans, null, and integers within ±2^53 (sb-approvals(7), The payload hash).
func Encode(raw []byte) ([]byte, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupported, err)
	}
	if d.More() {
		return nil, fmt.Errorf("%w: trailing data", ErrUnsupported)
	}
	var b bytes.Buffer
	if err := canon(&b, v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func canon(b *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(x))
	case json.Number:
		i, err := strconv.ParseInt(string(x), 10, 64)
		if err != nil || i > 1<<53 || i < -(1<<53) {
			return fmt.Errorf("%w: number %s is not an integer within ±2^53", ErrUnsupported, x)
		}
		b.WriteString(strconv.FormatInt(i, 10))
	case string:
		enc := json.NewEncoder(b)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(x); err != nil {
			return err
		}
		b.Truncate(b.Len() - 1)
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := canon(b, e); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return utf16Less(keys[i], keys[j]) })
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := canon(b, k); err != nil {
				return err
			}
			b.WriteByte(':')
			if err := canon(b, x[k]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	default:
		return fmt.Errorf("%w: unsupported value %T", ErrUnsupported, v)
	}
	return nil
}

// utf16Less orders keys by UTF-16 code units, as RFC 8785 requires.
func utf16Less(a, b string) bool {
	ua, ub := utf16(a), utf16(b)
	for i := 0; i < len(ua) && i < len(ub); i++ {
		if ua[i] != ub[i] {
			return ua[i] < ub[i]
		}
	}
	return len(ua) < len(ub)
}

func utf16(s string) []uint16 {
	var out []uint16
	for _, r := range s {
		if r >= 0x10000 {
			r -= 0x10000
			out = append(out, uint16(0xD800+(r>>10)), uint16(0xDC00+(r&0x3FF)))
		} else {
			out = append(out, uint16(r))
		}
	}
	return out
}

// EncodeDocument is Encode for a signed document (an agent's record, the AI inventory):
// a number that isn't an integer within ±2^53, such as an amount in dollars or a rate, is
// hashed as the string of its shortest decimal form ("0.6", "12.75"), so two
// implementations hash the same bytes without agreeing on how to print a float.
func EncodeDocument(raw []byte) ([]byte, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupported, err)
	}
	if d.More() {
		return nil, fmt.Errorf("%w: trailing data", ErrUnsupported)
	}
	var b bytes.Buffer
	if err := canon(&b, decimals(v)); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// decimals turns every number outside the integer subset into its decimal string.
func decimals(v any) any {
	switch x := v.(type) {
	case json.Number:
		if i, err := strconv.ParseInt(string(x), 10, 64); err == nil && i <= 1<<53 && i >= -(1<<53) {
			return x
		}
		f, err := strconv.ParseFloat(string(x), 64)
		if err != nil {
			return string(x)
		}
		if f == float64(int64(f)) && f <= 1<<53 && f >= -(1<<53) {
			return json.Number(strconv.FormatInt(int64(f), 10))
		}
		return strconv.FormatFloat(f, 'f', -1, 64)
	case []any:
		for i := range x {
			x[i] = decimals(x[i])
		}
	case map[string]any:
		for k := range x {
			x[k] = decimals(x[k])
		}
	}
	return v
}
