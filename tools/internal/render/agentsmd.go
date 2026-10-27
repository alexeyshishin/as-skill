package render

import "bytes"

const (
	BlockBegin = "<!-- as-skill:rules:begin -->"
	BlockEnd   = "<!-- as-skill:rules:end -->"
)

func findBlock(b []byte) (int, int) {
	s := bytes.Index(b, []byte(BlockBegin))
	if s < 0 {
		return -1, -1
	}
	e := bytes.Index(b[s:], []byte(BlockEnd))
	if e < 0 {
		return -1, -1
	}
	return s, s + e + len(BlockEnd)
}

func UpsertBlock(existing []byte, body string) []byte {
	block := BlockBegin + "\n" + body
	if len(body) > 0 && body[len(body)-1] != '\n' {
		block += "\n"
	}
	block += BlockEnd
	s, e := findBlock(existing)
	if s >= 0 {
		out := make([]byte, 0, len(existing)+len(block))
		out = append(out, existing[:s]...)
		out = append(out, block...)
		return append(out, existing[e:]...)
	}
	out := append([]byte(nil), existing...)
	if len(out) > 0 {
		if out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		out = append(out, '\n')
	}
	out = append(out, block...)
	return append(out, '\n')
}

func RemoveBlock(existing []byte) ([]byte, bool) {
	s, e := findBlock(existing)
	if s < 0 {
		return existing, false
	}
	pre, post := existing[:s], existing[e:]
	if string(post) == "\n" {
		pre, post = trimAdded(pre), nil
	}
	out := make([]byte, 0, len(pre)+len(post))
	out = append(out, pre...)
	return append(out, post...), true
}

func trimAdded(pre []byte) []byte {
	if bytes.HasSuffix(pre, []byte("\n\n")) {
		return pre[:len(pre)-1]
	}
	return pre
}
