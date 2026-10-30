package hookcfg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/platform"
)

type member struct {
	key string
	val json.RawMessage
}

type object []member

func parseObject(raw []byte) (object, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("not a JSON object")
	}
	var obj object
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
		obj = append(obj, member{key: keyTok.(string), val: val})
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return obj, nil
}

func (o object) index(key string) int {
	for i, m := range o {
		if m.key == key {
			return i
		}
	}
	return -1
}

func (o object) marshal() []byte {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, m := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(quote(m.key))
		b.WriteByte(':')
		b.Write(m.val)
	}
	b.WriteByte('}')
	return b.Bytes()
}

func quote(s string) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return bytes.TrimRight(b.Bytes(), "\n")
}

func parseArray(raw []byte) ([]json.RawMessage, error) {
	var a []json.RawMessage
	err := json.Unmarshal(raw, &a)
	return a, err
}

func marshalArray(a []json.RawMessage) []byte {
	return append(append([]byte{'['}, bytes.Join(rawSlices(a), []byte{','})...), ']')
}

func rawSlices(a []json.RawMessage) [][]byte {
	out := make([][]byte, len(a))
	for i, r := range a {
		out[i] = r
	}
	return out
}

type Unregistered struct {
	Event string
	Hook  string
}

func removeHookRegistrations(data []byte, projectRoot string, names map[string]bool) ([]byte, []Unregistered, error) {
	return RemoveRegistrations(data, projectRoot, names, platform.Claude)
}

func RemoveRegistrations(data []byte, projectRoot string, names map[string]bool, p platform.ID) ([]byte, []Unregistered, error) {
	dirRel, flat := hookDirRel(p), p == platform.Cursor
	if !json.Valid(data) {
		return nil, nil, errors.New("not valid JSON")
	}
	top, err := parseObject(data)
	if err != nil {
		return nil, nil, err
	}
	hi := top.index("hooks")
	if hi < 0 {
		return data, nil, nil
	}
	events, err := parseObject(top[hi].val)
	if err != nil {
		return data, nil, nil
	}

	var removed []Unregistered
	keptEvents := make(object, 0, len(events))
	for _, ev := range events {
		val, drop, r := filterEvent(ev.key, ev.val, projectRoot, names, dirRel, flat)
		removed = append(removed, r...)
		if drop {
			continue
		}
		keptEvents = append(keptEvents, member{key: ev.key, val: val})
	}
	if len(removed) == 0 {
		return data, nil, nil
	}

	if len(keptEvents) == 0 {
		top = append(top[:hi:hi], top[hi+1:]...)
	} else {
		top[hi].val = keptEvents.marshal()
	}
	var out bytes.Buffer
	if err := json.Indent(&out, top.marshal(), "", "  "); err != nil {
		return nil, nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), removed, nil
}

func filterEvent(event string, raw json.RawMessage, projectRoot string, names map[string]bool, dirRel string, flat bool) (json.RawMessage, bool, []Unregistered) {
	entries, err := parseArray(raw)
	if err != nil {
		return raw, false, nil
	}
	var removed []Unregistered
	var kept []json.RawMessage
	for _, entry := range entries {
		var val json.RawMessage
		var drop bool
		var r []Unregistered
		if flat {
			val, drop, r = filterFlatEntry(event, entry, projectRoot, names, dirRel)
		} else {
			val, drop, r = filterEntry(event, entry, projectRoot, names, dirRel)
		}
		removed = append(removed, r...)
		if !drop {
			kept = append(kept, val)
		}
	}
	if len(removed) == 0 {
		return raw, false, nil
	}
	if len(kept) == 0 {
		return nil, true, removed
	}
	return marshalArray(kept), false, removed
}

func filterEntry(event string, raw json.RawMessage, projectRoot string, names map[string]bool, dirRel string) (json.RawMessage, bool, []Unregistered) {
	obj, err := parseObject(raw)
	if err != nil {
		return raw, false, nil
	}
	hi := obj.index("hooks")
	if hi < 0 {
		return raw, false, nil
	}
	handlers, err := parseArray(obj[hi].val)
	if err != nil {
		return raw, false, nil
	}
	var removed []Unregistered
	var kept []json.RawMessage
	for _, h := range handlers {
		if name, ok := ownedHookIn(h, projectRoot, names, dirRel); ok {
			removed = append(removed, Unregistered{Event: event, Hook: name})
			continue
		}
		kept = append(kept, h)
	}
	if len(removed) == 0 {
		return raw, false, nil
	}
	if len(kept) == 0 {
		return nil, true, removed
	}
	obj[hi].val = marshalArray(kept)
	return obj.marshal(), false, removed
}

func filterFlatEntry(event string, raw json.RawMessage, projectRoot string, names map[string]bool, dirRel string) (json.RawMessage, bool, []Unregistered) {
	if name, ok := ownedHookIn(raw, projectRoot, names, dirRel); ok {
		return nil, true, []Unregistered{{Event: event, Hook: name}}
	}
	return raw, false, nil
}

func hookDirRel(p platform.ID) string {
	spec, ok := platform.Get(p)
	if !ok || spec.Hooks.Dir == "" {
		return claudeHooksDir()
	}
	return spec.Hooks.Dir
}

func claudeHooksDir() string {
	spec, _ := platform.Get(platform.Claude)
	return spec.Hooks.Dir
}

func ownedHook(handler json.RawMessage, projectRoot string, names map[string]bool) (string, bool) {
	return ownedHookIn(handler, projectRoot, names, claudeHooksDir())
}

func ownedHookIn(handler json.RawMessage, projectRoot string, names map[string]bool, dirRel string) (string, bool) {
	var h struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(handler, &h) != nil || h.Command == "" {
		return "", false
	}
	cmd := strings.ReplaceAll(h.Command, `"`, "")
	cmd = strings.ReplaceAll(cmd, "${CLAUDE_PROJECT_DIR}", "$CLAUDE_PROJECT_DIR")
	for _, prefix := range []string{
		"$CLAUDE_PROJECT_DIR/" + dirRel + "/",
		filepath.ToSlash(filepath.Join(projectRoot, dirRel)) + "/",
		"./" + dirRel + "/",
		dirRel + "/",
	} {
		if name, ok := strings.CutPrefix(cmd, prefix); ok && names[name] {
			return name, true
		}
	}
	return "", false
}

type HookReg struct {
	Name, Event, Matcher string
}

func HookTargets(path string, p platform.ID) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	for _, line := range strings.Split(string(data), "\n") {
		rest, found := strings.CutPrefix(line, "# platforms: ")
		if !found {
			continue
		}
		for _, f := range strings.FieldsFunc(rest, func(r rune) bool { return r == ',' || r == ' ' }) {
			if f == string(p) {
				return true
			}
		}
		return false
	}
	return true
}

func ReadRegisterHeader(path string) (event, matcher string, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		rest, found := strings.CutPrefix(line, "# register: ")
		if !found {
			continue
		}
		f := strings.Fields(rest)
		if len(f) == 0 {
			return "", "", false
		}
		if len(f) > 1 {
			matcher = f[1]
		}
		return f[0], matcher, true
	}
	return "", "", false
}

func hookCommand(p platform.ID, name string) string {
	if p == platform.Claude {
		return `"$CLAUDE_PROJECT_DIR"/` + hookDirRel(p) + "/" + name
	}
	return "./" + hookDirRel(p) + "/" + name
}

func commandsOf(entry json.RawMessage) []string {
	obj, err := parseObject(entry)
	if err != nil {
		return nil
	}
	var out []string
	if i := obj.index("command"); i >= 0 {
		var c string
		if json.Unmarshal(obj[i].val, &c) == nil {
			out = append(out, c)
		}
	}
	if i := obj.index("hooks"); i >= 0 {
		hs, err := parseArray(obj[i].val)
		if err == nil {
			for _, h := range hs {
				out = append(out, commandsOf(h)...)
			}
		}
	}
	return out
}

func isRegistered(arr []json.RawMessage, dirRel, name string) bool {
	needle := "/" + dirRel + "/" + name
	for _, e := range arr {
		for _, c := range commandsOf(e) {
			if strings.Contains(c, needle) {
				return true
			}
		}
	}
	return false
}

type RegResult struct {
	HookReg
	Added bool
}

func AddRegistrations(data []byte, p platform.ID, regs []HookReg, also []byte) ([]byte, []RegResult, error) {
	spec, ok := platform.Get(p)
	if !ok || spec.Hooks.Schema == platform.HooksNone {
		return nil, nil, fmt.Errorf("hooks unsupported on %s", p)
	}
	dirRel := hookDirRel(p)
	if len(bytes.TrimSpace(data)) == 0 {
		data = []byte("{}")
	}
	if !json.Valid(data) {
		return nil, nil, errors.New("not valid JSON")
	}
	top, err := parseObject(data)
	if err != nil {
		return nil, nil, err
	}
	hi := top.index("hooks")
	events := object{}
	if hi >= 0 {
		events, err = parseObject(top[hi].val)
		if err != nil {
			return nil, nil, errors.New("unexpected shape of .hooks")
		}
	}
	var alsoTop object
	if len(also) > 0 && json.Valid(also) {
		alsoTop, _ = parseObject(also)
	}
	var results []RegResult
	changed := false
	for _, r := range regs {
		mapped := []string{r.Event}
		if len(spec.Hooks.EventMap) > 0 {
			mapped = spec.Hooks.EventMap[r.Event]
		}
		if len(mapped) == 0 {
			return nil, nil, fmt.Errorf("event %s of %s has no mapping on %s", r.Event, r.Name, p)
		}
		added := false
		for _, ev := range mapped {
			var arr []json.RawMessage
			ei := events.index(ev)
			if ei >= 0 {
				arr, err = parseArray(events[ei].val)
				if err != nil {
					return nil, nil, fmt.Errorf("unexpected shape of .hooks.%s", ev)
				}
			}
			if isRegistered(arr, dirRel, r.Name) {
				continue
			}
			if hj := alsoTop.index("hooks"); hj >= 0 {
				if oe, err := parseObject(alsoTop[hj].val); err == nil {
					if li := oe.index(ev); li >= 0 {
						if la, err := parseArray(oe[li].val); err == nil && isRegistered(la, dirRel, r.Name) {
							continue
						}
					}
				}
			}
			cmd := quote(hookCommand(p, r.Name))
			var entry string
			if spec.Hooks.Schema == platform.HooksCursor {
				extra := ""
				if ev == "stop" {
					extra = `,"loop_limit":2`
				}
				entry = `{"command":` + string(cmd) + extra + `}`
			} else {
				m := ""
				if r.Matcher != "" && p == platform.Claude {
					m = `"matcher":` + string(quote(r.Matcher)) + `,`
				}
				entry = `{` + m + `"hooks":[{"type":"command","command":` + string(cmd) + `}]}`
			}
			arr = append(arr, json.RawMessage(entry))
			if ei >= 0 {
				events[ei].val = marshalArray(arr)
			} else {
				events = append(events, member{key: ev, val: marshalArray(arr)})
			}
			added, changed = true, true
		}
		results = append(results, RegResult{HookReg: r, Added: added})
	}
	if !changed {
		return data, results, nil
	}
	if hi >= 0 {
		top[hi].val = events.marshal()
	} else {
		top = append(top, member{key: "hooks", val: events.marshal()})
	}
	if spec.Hooks.Schema == platform.HooksCursor && top.index("version") < 0 {
		top = append(object{{key: "version", val: json.RawMessage("1")}}, top...)
	}
	var out bytes.Buffer
	if err := json.Indent(&out, top.marshal(), "", "  "); err != nil {
		return nil, nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), results, nil
}

func HookFiles(dir string, names []string, p platform.ID) ([]HookReg, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	want := map[string]bool{}
	for _, n := range names {
		if n == "all" {
			want = nil
			break
		}
		if !strings.HasSuffix(n, ".sh") {
			n += ".sh"
		}
		want[n] = true
	}
	var regs []HookReg
	found := map[string]bool{}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".sh") || (want != nil && !want[n]) {
			continue
		}
		found[n] = true
		if !HookTargets(filepath.Join(dir, n), p) {
			continue
		}
		ev, m, ok := ReadRegisterHeader(filepath.Join(dir, n))
		if !ok {
			continue
		}
		regs = append(regs, HookReg{Name: n, Event: ev, Matcher: m})
	}
	for n := range want {
		if !found[n] {
			return nil, fmt.Errorf("hook %q not found in %s", n, dir)
		}
	}
	return regs, nil
}

func WriteFileAtomic(path string, data []byte) error {
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), filepath.Base(target)+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, info.Mode().Perm()); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
