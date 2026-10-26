package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

type unregistered struct {
	Event string
	Hook  string
}

func removeHookRegistrations(data []byte, projectRoot string, names map[string]bool) ([]byte, []unregistered, error) {
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

	var removed []unregistered
	keptEvents := make(object, 0, len(events))
	for _, ev := range events {
		val, drop, r := filterEvent(ev.key, ev.val, projectRoot, names)
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

func filterEvent(event string, raw json.RawMessage, projectRoot string, names map[string]bool) (json.RawMessage, bool, []unregistered) {
	entries, err := parseArray(raw)
	if err != nil {
		return raw, false, nil
	}
	var removed []unregistered
	var kept []json.RawMessage
	for _, entry := range entries {
		val, drop, r := filterEntry(event, entry, projectRoot, names)
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

func filterEntry(event string, raw json.RawMessage, projectRoot string, names map[string]bool) (json.RawMessage, bool, []unregistered) {
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
	var removed []unregistered
	var kept []json.RawMessage
	for _, h := range handlers {
		if name, ok := ownedHook(h, projectRoot, names); ok {
			removed = append(removed, unregistered{Event: event, Hook: name})
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

func ownedHook(handler json.RawMessage, projectRoot string, names map[string]bool) (string, bool) {
	var h struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(handler, &h) != nil || h.Command == "" {
		return "", false
	}
	cmd := strings.ReplaceAll(h.Command, `"`, "")
	cmd = strings.ReplaceAll(cmd, "${CLAUDE_PROJECT_DIR}", "$CLAUDE_PROJECT_DIR")
	for _, prefix := range []string{
		"$CLAUDE_PROJECT_DIR/.claude/hooks/",
		filepath.ToSlash(filepath.Join(projectRoot, ".claude", "hooks")) + "/",
		"./.claude/hooks/",
		".claude/hooks/",
	} {
		if name, ok := strings.CutPrefix(cmd, prefix); ok && names[name] {
			return name, true
		}
	}
	return "", false
}

func (in *Installer) unregisterHooks(hookNames []string) {
	if len(hookNames) == 0 {
		return
	}
	path := filepath.Join(in.ProjectRoot, ".claude", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — %v\n", path, err)
		}
		return
	}
	names := make(map[string]bool, len(hookNames))
	for _, n := range hookNames {
		names[n] = true
	}
	out, removed, err := removeHookRegistrations(data, in.ProjectRoot, names)
	if err != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — %v, hook entries left in place\n", path, err)
		return
	}
	if len(removed) == 0 {
		return
	}
	verb := "unregistered  "
	if in.DryRun {
		verb = "would unregister"
	}
	for _, r := range removed {
		fmt.Printf("%s %s (%s) in %s\n", verb, r.Hook, r.Event, path)
	}
	if in.DryRun {
		return
	}
	if err := writeFileAtomic(path, out); err != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: failed updating %s: %v\n", path, err)
	}
}

func writeFileAtomic(path string, data []byte) error {
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
