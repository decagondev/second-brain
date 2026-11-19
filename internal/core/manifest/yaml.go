package manifest

import (
	"strings"
	"unicode/utf8"
)

// The strict YAML subset of sb-tenant(5) SYNTAX: one document, block mappings and
// lists, one-line flow lists and maps, plain and quoted scalars, comments. Anything
// else is a syntax error with a position, never a silent reinterpretation.

type kind int

const (
	scalarNode kind = iota
	mapNode
	seqNode
)

type node struct {
	kind      kind
	line, col int
	value     string // scalarNode
	quoted    bool   // scalarNode: written in quotes
	keys      []*node
	vals      []*node // mapNode: vals[i] belongs to keys[i]
	items     []*node // seqNode
}

type syntaxError struct {
	line, col int
	msg       string
}

type srcLine struct {
	n, indent int
	text      []rune // content after the indent, comment removed, right-trimmed
}

func (l srcLine) col(i int) int { return l.indent + i + 1 }

// parse returns the document's root node, or nil for an empty document.
func parse(src []byte) (*node, *syntaxError) {
	lines, err := lex(src)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, nil
	}
	p := &parser{lines: lines}
	if lines[0].indent != 0 {
		return nil, &syntaxError{lines[0].n, lines[0].col(0), "the first field must start at column 1"}
	}
	root, err := p.block(0)
	if err != nil {
		return nil, err
	}
	if p.i < len(lines) {
		l := lines[p.i]
		return nil, &syntaxError{l.n, l.col(0), "unexpected indentation"}
	}
	return root, nil
}

func lex(src []byte) ([]srcLine, *syntaxError) {
	if !utf8.Valid(src) {
		return nil, &syntaxError{1, 1, "the file is not valid UTF-8"}
	}
	s := strings.TrimPrefix(string(src), "\ufeff")
	var out []srcLine
	for i, raw := range strings.Split(s, "\n") {
		n := i + 1
		r := []rune(strings.TrimSuffix(raw, "\r"))
		indent := 0
		for indent < len(r) && (r[indent] == ' ' || r[indent] == '\t') {
			if r[indent] == '\t' {
				return nil, &syntaxError{n, indent + 1, "tabs are not allowed in indentation"}
			}
			indent++
		}
		text, err := stripComment(r[indent:], n, indent)
		if err != nil {
			return nil, err
		}
		for len(text) > 0 && (text[len(text)-1] == ' ' || text[len(text)-1] == '\t') {
			text = text[:len(text)-1]
		}
		if len(text) == 0 {
			continue
		}
		t := string(text)
		if indent == 0 && (t == "---" || strings.HasPrefix(t, "--- ") || t == "...") {
			if t == "---" && len(out) == 0 {
				continue // an explicit start of the only document
			}
			return nil, &syntaxError{n, 1, "only one YAML document is allowed"}
		}
		if indent == 0 && text[0] == '%' {
			return nil, &syntaxError{n, 1, "YAML directives are not allowed"}
		}
		out = append(out, srcLine{n: n, indent: indent, text: text})
	}
	return out, nil
}

// stripComment removes a `#` comment, which starts at the beginning of the content or
// after a space, outside quotes. A quote opens a quoted scalar only at a token start.
func stripComment(r []rune, n, indent int) ([]rune, *syntaxError) {
	var q rune
	qAt := 0
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch {
		case q == '"':
			switch c {
			case '\\':
				i++
			case '"':
				q = 0
			}
		case q == '\'':
			if c == '\'' {
				if i+1 < len(r) && r[i+1] == '\'' {
					i++
				} else {
					q = 0
				}
			}
		case c == '#' && (i == 0 || r[i-1] == ' ' || r[i-1] == '\t'):
			return r[:i], nil
		case (c == '"' || c == '\'') && (i == 0 || strings.ContainsRune(" \t[{,:", r[i-1])):
			q, qAt = c, i
		}
	}
	if q != 0 {
		return nil, &syntaxError{n, indent + qAt + 1, "a quoted value must close on the same line"}
	}
	return r, nil
}

type parser struct {
	lines []srcLine
	i     int
}

func isItem(t []rune) bool { return len(t) > 0 && t[0] == '-' && (len(t) == 1 || t[1] == ' ') }

// block parses the mapping or list whose lines start at exactly indent.
func (p *parser) block(indent int) (*node, *syntaxError) {
	if isItem(p.lines[p.i].text) {
		return p.seq(indent)
	}
	return p.mapping(indent)
}

func (p *parser) mapping(indent int) (*node, *syntaxError) {
	first := p.lines[p.i]
	m := &node{kind: mapNode, line: first.n, col: first.col(0)}
	for p.i < len(p.lines) {
		l := p.lines[p.i]
		if l.indent < indent {
			break
		}
		if l.indent > indent {
			return nil, &syntaxError{l.n, l.col(0), "unexpected indentation"}
		}
		if isItem(l.text) {
			return nil, &syntaxError{l.n, l.col(0), "a list item can't appear among fields; indent it under a field"}
		}
		colon := keyColon(l.text)
		if colon < 0 {
			return nil, &syntaxError{l.n, l.col(0), "expected `field: value`"}
		}
		key, err := scalarAt(l, 0, colon)
		if err != nil {
			return nil, err
		}
		p.i++
		var val *node
		rest := colon + 1
		for rest < len(l.text) && l.text[rest] == ' ' {
			rest++
		}
		switch {
		case rest < len(l.text):
			if val, err = inline(l, rest); err != nil {
				return nil, err
			}
			if p.i < len(p.lines) && p.lines[p.i].indent > indent {
				nl := p.lines[p.i]
				return nil, &syntaxError{nl.n, nl.col(0), "unexpected indentation; values must fit on one line"}
			}
		case p.i < len(p.lines) && p.lines[p.i].indent > indent:
			if val, err = p.block(p.lines[p.i].indent); err != nil {
				return nil, err
			}
		case p.i < len(p.lines) && p.lines[p.i].indent == indent && isItem(p.lines[p.i].text):
			if val, err = p.seq(indent); err != nil {
				return nil, err
			}
		default:
			val = &node{kind: scalarNode, line: key.line, col: key.col} // empty value
		}
		m.keys, m.vals = append(m.keys, key), append(m.vals, val)
	}
	return m, nil
}

func (p *parser) seq(indent int) (*node, *syntaxError) {
	first := p.lines[p.i]
	s := &node{kind: seqNode, line: first.n, col: first.col(0)}
	for p.i < len(p.lines) {
		l := p.lines[p.i]
		if l.indent != indent || !isItem(l.text) {
			if l.indent > indent {
				return nil, &syntaxError{l.n, l.col(0), "unexpected indentation"}
			}
			break
		}
		p.i++
		start := 1
		for start < len(l.text) && l.text[start] == ' ' {
			start++
		}
		if start >= len(l.text) {
			return nil, &syntaxError{l.n, l.col(0), "a list item needs a value on the same line"}
		}
		if keyColon(l.text[start:]) >= 0 {
			return nil, &syntaxError{l.n, l.col(start), "list items must be single values, not fields"}
		}
		v, err := inline(l, start)
		if err != nil {
			return nil, err
		}
		if p.i < len(p.lines) && p.lines[p.i].indent > indent {
			nl := p.lines[p.i]
			return nil, &syntaxError{nl.n, nl.col(0), "unexpected indentation; values must fit on one line"}
		}
		s.items = append(s.items, v)
	}
	return s, nil
}

// keyColon finds the `:` that ends a block key: outside quotes and brackets, followed
// by a space or the end of the line. It returns -1 if there is none.
func keyColon(t []rune) int {
	var q rune
	depth := 0
	for i := 0; i < len(t); i++ {
		c := t[i]
		switch {
		case q == '"':
			switch c {
			case '\\':
				i++
			case '"':
				q = 0
			}
		case q == '\'':
			if c == '\'' {
				q = 0
			}
		case (c == '"' || c == '\'') && (i == 0 || strings.ContainsRune(" [{,", t[i-1])):
			q = c
		case c == '[' || c == '{':
			depth++
		case c == ']' || c == '}':
			depth--
		case c == ':' && depth == 0 && (i+1 == len(t) || t[i+1] == ' '):
			return i
		}
	}
	return -1
}

// scalarAt parses t[from:to] of l as a single key scalar.
func scalarAt(l srcLine, from, to int) (*node, *syntaxError) {
	f := &flow{l: l, s: l.text[:to], pos: from}
	n, err := f.value(ctxKey)
	if err != nil {
		return nil, err
	}
	if n.kind != scalarNode {
		return nil, &syntaxError{l.n, n.col, "field names must be plain words"}
	}
	f.space()
	if f.pos != len(f.s) {
		return nil, &syntaxError{l.n, l.col(f.pos), "unexpected text in field name"}
	}
	return n, nil
}

// inline parses the rest of a line from position from as one value.
func inline(l srcLine, from int) (*node, *syntaxError) {
	f := &flow{l: l, s: l.text, pos: from}
	n, err := f.value(ctxBlock)
	if err != nil {
		return nil, err
	}
	f.space()
	if f.pos != len(f.s) {
		return nil, &syntaxError{l.n, l.col(f.pos), "unexpected text after the value"}
	}
	return n, nil
}

type ctx int

const (
	ctxBlock ctx = iota // a value that runs to the end of the line
	ctxFlow             // a value inside [ ] or { }
	ctxKey              // a key, ended by `:`
)

type flow struct {
	l   srcLine
	s   []rune
	pos int
}

func (f *flow) err(i int, msg string) *syntaxError {
	return &syntaxError{f.l.n, f.l.col(i), msg}
}

func (f *flow) space() {
	for f.pos < len(f.s) && f.s[f.pos] == ' ' {
		f.pos++
	}
}

func (f *flow) value(c ctx) (*node, *syntaxError) {
	f.space()
	if f.pos >= len(f.s) {
		return &node{kind: scalarNode, line: f.l.n, col: f.l.col(f.pos)}, nil
	}
	start := f.pos
	switch ch := f.s[f.pos]; ch {
	case '[':
		if c == ctxKey {
			return nil, f.err(start, "field names must be plain words")
		}
		return f.list()
	case '{':
		if c == ctxKey {
			return nil, f.err(start, "field names must be plain words")
		}
		return f.dict()
	case '"', '\'':
		return f.quoted(ch)
	case '&', '*', '!':
		return nil, f.err(start, "anchors, aliases and tags are not allowed")
	case '|', '>':
		return nil, f.err(start, "block scalars are not allowed; put the value on one line in quotes")
	case '@', '`':
		return nil, f.err(start, "values can't start with "+string(ch)+"; quote the value")
	case ',', ']', '}':
		if c == ctxFlow {
			return &node{kind: scalarNode, line: f.l.n, col: f.l.col(start)}, nil
		}
		return nil, f.err(start, "unexpected "+string(ch))
	}
	return f.plain(c)
}

func (f *flow) plain(c ctx) (*node, *syntaxError) {
	start := f.pos
	for f.pos < len(f.s) {
		ch := f.s[f.pos]
		colonEnd := ch == ':' && (f.pos+1 == len(f.s) || strings.ContainsRune(" ,]}", f.s[f.pos+1]))
		if c == ctxBlock && ch == ':' && f.pos+1 < len(f.s) && f.s[f.pos+1] == ' ' {
			return nil, f.err(f.pos, `a value can't contain ": "; quote the value`)
		}
		if c != ctxBlock && (ch == ',' || ch == ']' || ch == '}' || colonEnd) {
			break
		}
		if c == ctxKey && colonEnd {
			break
		}
		f.pos++
	}
	end := f.pos
	for end > start && f.s[end-1] == ' ' {
		end--
	}
	return &node{kind: scalarNode, line: f.l.n, col: f.l.col(start), value: string(f.s[start:end])}, nil
}

func (f *flow) quoted(q rune) (*node, *syntaxError) {
	start := f.pos
	f.pos++
	var b strings.Builder
	for f.pos < len(f.s) {
		ch := f.s[f.pos]
		switch {
		case q == '\'' && ch == '\'':
			if f.pos+1 < len(f.s) && f.s[f.pos+1] == '\'' {
				b.WriteRune('\'')
				f.pos += 2
				continue
			}
			f.pos++
			return &node{kind: scalarNode, line: f.l.n, col: f.l.col(start), value: b.String(), quoted: true}, nil
		case q == '"' && ch == '"':
			f.pos++
			return &node{kind: scalarNode, line: f.l.n, col: f.l.col(start), value: b.String(), quoted: true}, nil
		case q == '"' && ch == '\\':
			if f.pos+1 >= len(f.s) {
				return nil, f.err(f.pos, "a quoted value must close on the same line")
			}
			esc, ok := map[rune]rune{'"': '"', '\\': '\\', '/': '/', 'n': '\n', 't': '\t', 'r': '\r'}[f.s[f.pos+1]]
			if !ok {
				return nil, f.err(f.pos, `unknown escape; use \" \\ \/ \n \t or \r`)
			}
			b.WriteRune(esc)
			f.pos += 2
		default:
			b.WriteRune(ch)
			f.pos++
		}
	}
	return nil, f.err(start, "a quoted value must close on the same line")
}

func (f *flow) list() (*node, *syntaxError) {
	open := f.pos
	n := &node{kind: seqNode, line: f.l.n, col: f.l.col(open)}
	f.pos++
	for {
		f.space()
		if f.pos >= len(f.s) {
			return nil, f.err(open, "a [ list must close with ] on the same line")
		}
		if f.s[f.pos] == ']' {
			f.pos++
			return n, nil
		}
		item, err := f.value(ctxFlow)
		if err != nil {
			return nil, err
		}
		if item.kind == scalarNode && item.value == "" && !item.quoted {
			return nil, f.err(f.pos, "empty list item")
		}
		n.items = append(n.items, item)
		f.space()
		if f.pos < len(f.s) && f.s[f.pos] == ',' {
			f.pos++
			continue
		}
		if f.pos < len(f.s) && f.s[f.pos] == ']' {
			continue
		}
		if f.pos >= len(f.s) {
			return nil, f.err(open, "a [ list must close with ] on the same line")
		}
		return nil, f.err(f.pos, "expected , or ] in the list")
	}
}

func (f *flow) dict() (*node, *syntaxError) {
	open := f.pos
	n := &node{kind: mapNode, line: f.l.n, col: f.l.col(open)}
	f.pos++
	for {
		f.space()
		if f.pos >= len(f.s) {
			return nil, f.err(open, "a { map must close with } on the same line")
		}
		if f.s[f.pos] == '}' {
			f.pos++
			return n, nil
		}
		key, err := f.value(ctxKey)
		if err != nil {
			return nil, err
		}
		if key.kind != scalarNode || (key.value == "" && !key.quoted) {
			return nil, f.err(f.pos, "expected a field name in the map")
		}
		f.space()
		if f.pos >= len(f.s) || f.s[f.pos] != ':' {
			return nil, f.err(f.pos, "expected : after the field name")
		}
		f.pos++
		val, err := f.value(ctxFlow)
		if err != nil {
			return nil, err
		}
		if val.kind == scalarNode && val.value == "" && !val.quoted {
			val.line, val.col = key.line, key.col
		}
		n.keys, n.vals = append(n.keys, key), append(n.vals, val)
		f.space()
		if f.pos < len(f.s) && f.s[f.pos] == ',' {
			f.pos++
			continue
		}
		if f.pos < len(f.s) && f.s[f.pos] == '}' {
			continue
		}
		if f.pos >= len(f.s) {
			return nil, f.err(open, "a { map must close with } on the same line")
		}
		return nil, f.err(f.pos, "expected , or } in the map")
	}
}
