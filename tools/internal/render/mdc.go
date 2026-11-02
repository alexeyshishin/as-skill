package render

import (
	"strings"

	"claude-harness/tools/internal/lockfile"
)

func MDCStub(stem, rel, body string, mode lockfile.Mode) []byte {
	if mode == lockfile.ModeLink {
		body = "Follow the rule in `" + rel + "`.\n"
	} else if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return []byte("---\ndescription: " + stem + " rule\nalwaysApply: true\n---\n" + body)
}
