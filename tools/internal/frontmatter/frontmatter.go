package frontmatter

import (
	"fmt"
	"regexp"
	"strings"
)

type Kind int

const (
	Scalar Kind = iota
	List
	Map
)

type Value struct {
	Kind   Kind
	Str    string
	List   []string
	Map    map[string]string
	Line   int
	Quoted bool
}

type Doc struct {
	Keys   []string
	Fields map[string]Value
	Body   string
}

type Error struct {
	Line int
	Msg  string
}

func (e *Error) Error() string { return fmt.Sprintf("line %d: %s", e.Line, e.Msg) }

func errf(line int, format string, args ...any) error {
	return &Error{Line: line, Msg: fmt.Sprintf(format, args...)}
}

var keyRe = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_.-]*)[ ]*:(?:[ ]+(.*))?$`)

func (d *Doc) Has(key string) bool {
	_, ok := d.Fields[key]
	return ok
}

func (d *Doc) String(key string) (string, bool) {
	v, ok := d.Fields[key]
	if !ok || v.Kind != Scalar {
		return "", false
	}
	return v.Str, true
}

func Parse(data []byte) (*Doc, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if strings.TrimRight(lines[0], " \t") != "---" {
		return nil, errf(1, "missing frontmatter (file does not start with \"---\")")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, errf(1, "frontmatter not terminated with a closing \"---\"")
	}
	doc := &Doc{Fields: map[string]Value{}, Body: strings.Join(lines[end+1:], "\n")}
	i := 1
	for i < end {
		raw := lines[i]
		lineNo := i + 1
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}
		if raw[0] == ' ' || raw[0] == '\t' {
			return nil, errf(lineNo, "unexpected indented line")
		}
		m := keyRe.FindStringSubmatch(strings.TrimRight(raw, " \t"))
		if m == nil {
			return nil, errf(lineNo, "expected \"key: value\", got %q", raw)
		}
		key, rest := m[1], strings.TrimSpace(m[2])
		if _, dup := doc.Fields[key]; dup {
			return nil, errf(lineNo, "duplicate key %q", key)
		}
		val, next, err := parseValue(lines, i, end, rest)
		if err != nil {
			return nil, err
		}
		val.Line = lineNo
		doc.Keys = append(doc.Keys, key)
		doc.Fields[key] = val
		i = next
	}
	return doc, nil
}

func parseValue(lines []string, i, end int, rest string) (Value, int, error) {
	lineNo := i + 1
	switch {
	case rest == "" || strings.HasPrefix(rest, "#"):
		return parseNested(lines, i, end)
	case rest[0] == '>' || rest[0] == '|':
		return parseBlock(lines, i, end, rest)
	case rest[0] == '"' || rest[0] == '\'':
		s, err := unquote(rest, lineNo)
		if err != nil {
			return Value{}, 0, err
		}
		if err := noIndentedFollow(lines, i+1, end); err != nil {
			return Value{}, 0, err
		}
		return Value{Kind: Scalar, Str: s, Quoted: true}, i + 1, nil
	case rest[0] == '[':
		items, err := parseFlowList(rest, lineNo)
		if err != nil {
			return Value{}, 0, err
		}
		return Value{Kind: List, List: items}, i + 1, nil
	case rest[0] == '{':
		return Value{}, 0, errf(lineNo, "flow maps are not supported")
	}
	if strings.Contains(rest, ": ") || strings.HasSuffix(rest, ":") {
		return Value{}, 0, errf(lineNo, "plain scalar contains \": \" — quote it")
	}
	if idx := strings.Index(rest, " #"); idx >= 0 {
		rest = strings.TrimSpace(rest[:idx])
	}
	if err := noIndentedFollow(lines, i+1, end); err != nil {
		return Value{}, 0, err
	}
	return Value{Kind: Scalar, Str: rest}, i + 1, nil
}

func noIndentedFollow(lines []string, from, end int) error {
	if from < end && lines[from] != "" && (lines[from][0] == ' ' || lines[from][0] == '\t') && strings.TrimSpace(lines[from]) != "" {
		return errf(from+1, "unexpected indented line after a single-line value")
	}
	return nil
}

func indentOf(s string) (int, error) {
	n := 0
	for _, c := range s {
		switch c {
		case ' ':
			n++
		case '\t':
			return 0, fmt.Errorf("tab in indentation")
		default:
			return n, nil
		}
	}
	return n, nil
}

func parseNested(lines []string, i, end int) (Value, int, error) {
	j := i + 1
	for j < end && strings.TrimSpace(lines[j]) == "" {
		j++
	}
	if j >= end {
		return Value{Kind: Scalar}, i + 1, nil
	}
	first := lines[j]
	ind, err := indentOf(first)
	if err != nil {
		return Value{}, 0, errf(j+1, "%v", err)
	}
	ft := strings.TrimSpace(first)
	if strings.HasPrefix(ft, "#") {
		return Value{Kind: Scalar}, i + 1, nil
	}
	if ind == 0 && !(ft == "-" || strings.HasPrefix(ft, "- ")) {
		return Value{Kind: Scalar}, i + 1, nil
	}
	if ft == "-" || strings.HasPrefix(ft, "- ") {
		var items []string
		k := j
		for k < end {
			t := strings.TrimSpace(lines[k])
			if t == "" || strings.HasPrefix(t, "#") {
				k++
				continue
			}
			ki, err := indentOf(lines[k])
			if err != nil {
				return Value{}, 0, errf(k+1, "%v", err)
			}
			if ki != ind {
				if ki < ind {
					break
				}
				return Value{}, 0, errf(k+1, "inconsistent list indentation")
			}
			if !(t == "-" || strings.HasPrefix(t, "- ")) {
				if ind == 0 {
					break
				}
				return Value{}, 0, errf(k+1, "expected list item \"- value\"")
			}
			item := strings.TrimSpace(strings.TrimPrefix(t, "-"))
			if item == "" {
				return Value{}, 0, errf(k+1, "empty list item")
			}
			if item[0] == '"' || item[0] == '\'' {
				item, err = unquote(item, k+1)
				if err != nil {
					return Value{}, 0, err
				}
			} else {
				if idx := strings.Index(item, " #"); idx >= 0 {
					item = strings.TrimSpace(item[:idx])
				}
				if strings.Contains(item, ": ") {
					return Value{}, 0, errf(k+1, "nested maps in lists are not supported")
				}
			}
			items = append(items, item)
			k++
		}
		return Value{Kind: List, List: items}, k, nil
	}
	m := map[string]string{}
	k := j
	for k < end {
		t := strings.TrimSpace(lines[k])
		if t == "" || strings.HasPrefix(t, "#") {
			k++
			continue
		}
		ki, err := indentOf(lines[k])
		if err != nil {
			return Value{}, 0, errf(k+1, "%v", err)
		}
		if ki == 0 {
			break
		}
		if ki != ind {
			return Value{}, 0, errf(k+1, "inconsistent map indentation (only one nesting level is supported)")
		}
		mm := keyRe.FindStringSubmatch(t)
		if mm == nil {
			return Value{}, 0, errf(k+1, "expected \"key: value\", got %q", t)
		}
		v := strings.TrimSpace(mm[2])
		if v == "" {
			return Value{}, 0, errf(k+1, "nested structures below one level are not supported")
		}
		if v[0] == '"' || v[0] == '\'' {
			v, err = unquote(v, k+1)
			if err != nil {
				return Value{}, 0, err
			}
		} else if idx := strings.Index(v, " #"); idx >= 0 {
			v = strings.TrimSpace(v[:idx])
		}
		if _, dup := m[mm[1]]; dup {
			return Value{}, 0, errf(k+1, "duplicate key %q", mm[1])
		}
		m[mm[1]] = v
		k++
	}
	return Value{Kind: Map, Map: m}, k, nil
}

func parseBlock(lines []string, i, end int, header string) (Value, int, error) {
	lineNo := i + 1
	folded := header[0] == '>'
	chomp := byte(0)
	for _, c := range []byte(header[1:]) {
		switch c {
		case '-', '+':
			chomp = c
		case ' ', '#':
		default:
			if c >= '0' && c <= '9' {
				return Value{}, 0, errf(lineNo, "explicit block indentation indicators are not supported")
			}
			if c != ' ' {
				return Value{}, 0, errf(lineNo, "bad block scalar header %q", header)
			}
		}
		if c == '#' {
			break
		}
	}
	j := i + 1
	var body []string
	ind := -1
	for j < end {
		l := lines[j]
		if strings.TrimSpace(l) == "" {
			body = append(body, "")
			j++
			continue
		}
		li, err := indentOf(l)
		if err != nil {
			return Value{}, 0, errf(j+1, "%v", err)
		}
		if li == 0 {
			break
		}
		if ind < 0 {
			ind = li
		}
		if li < ind {
			return Value{}, 0, errf(j+1, "block scalar line indented less than its first line")
		}
		body = append(body, l[ind:])
		j++
	}
	trailing := 0
	for len(body) > 0 && body[len(body)-1] == "" {
		body = body[:len(body)-1]
		trailing++
	}
	var sb strings.Builder
	if folded {
		for k, l := range body {
			if k > 0 {
				prev := body[k-1]
				switch {
				case l == "":
					sb.WriteString("\n")
				case prev == "":
				case strings.HasPrefix(l, " ") || strings.HasPrefix(prev, " "):
					sb.WriteString("\n")
				default:
					sb.WriteString(" ")
				}
			}
			sb.WriteString(l)
		}
	} else {
		sb.WriteString(strings.Join(body, "\n"))
	}
	s := sb.String()
	if len(body) > 0 {
		switch chomp {
		case 0:
			s += "\n"
		case '+':
			s += strings.Repeat("\n", trailing+1)
		}
	}
	return Value{Kind: Scalar, Str: s}, j, nil
}

func unquote(s string, line int) (string, error) {
	q := s[0]
	var sb strings.Builder
	i := 1
	for i < len(s) {
		c := s[i]
		if q == '\'' {
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					sb.WriteByte('\'')
					i += 2
					continue
				}
				return finishQuoted(sb.String(), s[i+1:], line)
			}
			sb.WriteByte(c)
			i++
			continue
		}
		switch c {
		case '"':
			return finishQuoted(sb.String(), s[i+1:], line)
		case '\\':
			if i+1 >= len(s) {
				return "", errf(line, "dangling backslash in quoted string")
			}
			switch s[i+1] {
			case '"', '\\', '/':
				sb.WriteByte(s[i+1])
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			default:
				return "", errf(line, "unsupported escape \\%c", s[i+1])
			}
			i += 2
		default:
			sb.WriteByte(c)
			i++
		}
	}
	return "", errf(line, "unterminated quoted string")
}

func finishQuoted(val, tail string, line int) (string, error) {
	tail = strings.TrimSpace(tail)
	if tail != "" && !strings.HasPrefix(tail, "#") {
		return "", errf(line, "unexpected text after closing quote: %q", tail)
	}
	return val, nil
}

func parseFlowList(s string, line int) ([]string, error) {
	if !strings.HasSuffix(strings.TrimSpace(s), "]") {
		return nil, errf(line, "flow list must close with \"]\" on the same line")
	}
	inner := strings.TrimSpace(s)
	inner = strings.TrimSpace(inner[1 : len(inner)-1])
	if inner == "" {
		return nil, nil
	}
	var out []string
	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, errf(line, "empty item in flow list")
		}
		if part[0] == '"' || part[0] == '\'' {
			v, err := unquote(part, line)
			if err != nil {
				return nil, err
			}
			part = v
		}
		out = append(out, part)
	}
	return out, nil
}
