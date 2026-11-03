package prompt

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func splitPath(value string) (string, string) {
	cut := strings.LastIndexAny(value, "/"+string(os.PathSeparator))
	return value[:cut+1], value[cut+1:]
}

type ranked struct {
	path string
	rank int
}

func dirMatches(value string) []string {
	if value == "" {
		return nil
	}
	if value == "~" {
		return []string{"~/"}
	}
	dirPart, base := splitPath(value)
	readDir := dirPart
	if readDir == "" {
		readDir = "."
	} else if strings.HasPrefix(readDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		readDir = home + readDir[1:]
	}
	entries, err := os.ReadDir(readDir)
	if err != nil {
		return nil
	}
	lowBase := strings.ToLower(base)
	var found []ranked
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(base, ".") {
			continue
		}
		lowName := strings.ToLower(name)
		if !strings.Contains(lowName, lowBase) {
			continue
		}
		if !e.IsDir() {
			if e.Type()&os.ModeSymlink == 0 {
				continue
			}
			info, err := os.Stat(readDir + string(os.PathSeparator) + name)
			if err != nil || !info.IsDir() {
				continue
			}
		}
		rank := 2
		switch {
		case lowName == lowBase:
			rank = 0
		case strings.HasPrefix(lowName, lowBase):
			rank = 1
		}
		found = append(found, ranked{dirPart + name + "/", rank})
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].rank != found[j].rank {
			return found[i].rank < found[j].rank
		}
		return found[i].path < found[j].path
	})
	if len(found) == 0 {
		return nil
	}
	out := make([]string, len(found))
	for i, f := range found {
		out[i] = f.path
	}
	return out
}

func commonPrefix(items []string) string {
	if len(items) == 0 {
		return ""
	}
	prefix := []rune(items[0])
	for _, it := range items[1:] {
		r := []rune(it)
		n := 0
		for n < len(prefix) && n < len(r) && prefix[n] == r[n] {
			n++
		}
		prefix = prefix[:n]
	}
	return string(prefix)
}

func tabResult(value string) (string, []string) {
	matches := dirMatches(value)
	switch len(matches) {
	case 0:
		return value, nil
	case 1:
		return matches[0], matches
	}
	dirPart, base := splitPath(value)
	lowBase := strings.ToLower(base)
	names := make([]string, len(matches))
	for i, m := range matches {
		_, name := splitPath(strings.TrimSuffix(m, "/"))
		if !strings.HasPrefix(strings.ToLower(name), lowBase) {
			return value, matches
		}
		names[i] = name
	}
	lcp := commonPrefix(names)
	if len([]rune(lcp)) > len([]rune(base)) {
		return dirPart + lcp, matches
	}
	return value, matches
}

func wrapSel(sel, delta, total int) int {
	if total <= 0 {
		return -1
	}
	if delta > 0 {
		if sel+1 >= total {
			return 0
		}
		return sel + 1
	}
	if sel <= 0 {
		return total - 1
	}
	return sel - 1
}

func scrollOff(off, sel, total, rows int) int {
	if total <= rows || rows < 2 {
		return 0
	}
	window := rows - 1
	if sel >= 0 {
		if sel < off {
			off = sel
		}
		if sel >= off+window {
			off = sel - window + 1
		}
	}
	return min(max(off, 0), total-window)
}

func renderMatches(matches []string, sel, off, rows int) string {
	total := len(matches)
	if total < 2 || rows < 2 {
		return ""
	}
	sel = min(max(sel, -1), total-1)
	start, end, hidden := 0, total, 0
	if total > rows {
		window := rows - 1
		start = min(max(off, 0), total-window)
		end = start + window
		hidden = total - window
	}
	lines := make([]string, 0, rows)
	for i := start; i < end; i++ {
		_, name := splitPath(strings.TrimSuffix(matches[i], "/"))
		marker := "  "
		if i == sel {
			marker = "> "
		}
		lines = append(lines, marker+name+"/")
	}
	if hidden > 0 {
		lines = append(lines, fmt.Sprintf("  … +%d more", hidden))
	}
	return strings.Join(lines, "\n")
}
