package hooktest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var scripts = []string{"test-gate.sh", "test-gate-record.sh", "prettier-format.sh", "test-gate-lib.sh"}

func hooksSrc(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../../domains/code/hooks")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func need(t *testing.T) {
	t.Helper()
	for _, b := range []string{"bash", "jq", "grep"} {
		if _, err := exec.LookPath(b); err != nil {
			t.Skipf("%s missing", b)
		}
	}
}

func project(t *testing.T, dot string) string {
	t.Helper()
	need(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hd := filepath.Join(root, dot, "hooks")
	if err := os.MkdirAll(hd, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, s := range scripts {
		if err := os.Symlink(filepath.Join(hooksSrc(t), s), filepath.Join(hd, s)); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func run(t *testing.T, root, dot, script, stdin string) (string, int) {
	t.Helper()
	cmd := exec.Command("bash", filepath.Join(root, dot, "hooks", script))
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return out.String(), code
}

func silent(t *testing.T, root, dot, script, stdin string) {
	t.Helper()
	out, code := run(t, root, dot, script, stdin)
	if out != "" || code != 0 {
		t.Fatalf("%s: want silent exit 0, got code=%d out=%q", script, code, out)
	}
}

func transcript(t *testing.T, root string, cmds ...string) string {
	t.Helper()
	var b strings.Builder
	line := func(name string, input map[string]string) {
		m := map[string]any{"type": "assistant", "message": map[string]any{"content": []any{
			map[string]any{"type": "tool_use", "name": name, "input": input}}}}
		j, _ := json.Marshal(m)
		b.Write(j)
		b.WriteString("\n")
	}
	line("Edit", map[string]string{"file_path": filepath.Join(root, "a.go")})
	for _, c := range cmds {
		line("Bash", map[string]string{"command": c})
	}
	p := filepath.Join(root, "transcript.jsonl")
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestClaudeStop(t *testing.T) {
	root := project(t, ".claude")
	tr := transcript(t, root)
	payload := fmt.Sprintf(`{"session_id":"s1","transcript_path":%q}`, tr)
	out, code := run(t, root, ".claude", "test-gate.sh", payload)
	if code != 0 {
		t.Fatalf("code %d", code)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("bad json %q: %v", out, err)
	}
	if got["decision"] != "block" {
		t.Fatalf("no block: %s", out)
	}
	hso := got["hookSpecificOutput"].(map[string]any)
	if hso["hookEventName"] != "Stop" || !strings.Contains(hso["additionalContext"].(string), "test-gate 1/2") {
		t.Fatalf("bad hso: %s", out)
	}
	out, _ = run(t, root, ".claude", "test-gate.sh", payload)
	if !strings.Contains(out, "test-gate 2/2") {
		t.Fatalf("second block missing: %q", out)
	}
	silent(t, root, ".claude", "test-gate.sh", payload)
}

func TestClaudeStopSilent(t *testing.T) {
	root := project(t, ".claude")
	tr := transcript(t, root, "go test ./...")
	silent(t, root, ".claude", "test-gate.sh", fmt.Sprintf(`{"session_id":"s2","transcript_path":%q}`, tr))
	tr2 := transcript(t, root)
	silent(t, root, ".claude", "test-gate.sh", fmt.Sprintf(`{"session_id":"s3","stop_hook_active":true,"transcript_path":%q}`, tr2))
}

func patch(path string) string {
	return fmt.Sprintf(`{"session_id":"c1","hook_event_name":"PostToolUse","tool_name":"apply_patch","tool_input":{"command":"*** Begin Patch\n*** Update File: %s\n@@\n-a\n+b\n*** End Patch"}}`, path)
}

func TestCodexSyntheticFlow(t *testing.T) {
	root := project(t, ".codex")
	silent(t, root, ".codex", "test-gate-record.sh", patch("a.go"))
	out, _ := run(t, root, ".codex", "test-gate.sh", `{"session_id":"c1","hook_event_name":"Stop"}`)
	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil || got["decision"] != "block" || got["reason"] == "" {
		t.Fatalf("want codex block, got %q (%v)", out, err)
	}
	if strings.Contains(out, "hookSpecificOutput") {
		t.Fatalf("unexpected hso: %s", out)
	}
	silent(t, root, ".codex", "test-gate.sh", `{"session_id":"c1","stop_hook_active":true}`)

	root2 := project(t, ".codex")
	silent(t, root2, ".codex", "test-gate-record.sh", patch("a.go"))
	silent(t, root2, ".codex", "test-gate-record.sh", `{"session_id":"c1","tool_name":"shell","tool_input":{"command":["bash","-lc","go test ./..."]}}`)
	silent(t, root2, ".codex", "test-gate.sh", `{"session_id":"c1","hook_event_name":"Stop"}`)
	if _, err := os.Stat(filepath.Join(root2, ".codex", "sessions", "c1.state")); err != nil {
		t.Fatal(err)
	}
}

func TestCursorSyntheticFlow(t *testing.T) {
	root := project(t, ".cursor")
	edit := fmt.Sprintf(`{"conversation_id":"k1","hook_event_name":"afterFileEdit","file_path":%q,"edits":[]}`, filepath.Join(root, "a.go"))
	silent(t, root, ".cursor", "test-gate-record.sh", edit)
	silent(t, root, ".cursor", "test-gate.sh", `{"conversation_id":"k1","status":"aborted","loop_count":0}`)
	stop := `{"conversation_id":"k1","status":"completed","loop_count":0}`
	for i := 0; i < 2; i++ {
		out, _ := run(t, root, ".cursor", "test-gate.sh", stop)
		var got map[string]string
		if err := json.Unmarshal([]byte(out), &got); err != nil || got["followup_message"] == "" || len(got) != 1 {
			t.Fatalf("iter %d: want followup_message only, got %q", i, out)
		}
	}
	silent(t, root, ".cursor", "test-gate.sh", stop)

	root2 := project(t, ".cursor")
	silent(t, root2, ".cursor", "test-gate-record.sh", fmt.Sprintf(`{"conversation_id":"k1","file_path":%q}`, filepath.Join(root2, "a.go")))
	silent(t, root2, ".cursor", "test-gate-record.sh", `{"conversation_id":"k1","hook_event_name":"afterShellExecution","command":"go test ./..."}`)
	silent(t, root2, ".cursor", "test-gate.sh", stop)
}

func TestMalformedAndEmptyFailOpen(t *testing.T) {
	for _, dot := range []string{".claude", ".codex", ".cursor"} {
		root := project(t, dot)
		for _, s := range []string{"test-gate.sh", "test-gate-record.sh", "prettier-format.sh"} {
			for _, in := range []string{"", "not json{", "[]", `{"session_id":5}`} {
				silent(t, root, dot, s, in)
			}
		}
	}
}

func TestPrettierSkippedWithoutConfig(t *testing.T) {
	for _, dot := range []string{".claude", ".codex", ".cursor"} {
		root := project(t, dot)
		f := filepath.Join(root, "a.js")
		if err := os.WriteFile(f, []byte("var  a=1"), 0o644); err != nil {
			t.Fatal(err)
		}
		in := fmt.Sprintf(`{"session_id":"p","conversation_id":"p","tool_name":"Write","tool_input":{"file_path":%q},"file_path":%q}`, f, f)
		silent(t, root, dot, "prettier-format.sh", in)
	}
}

func TestRecordPrintsNothing(t *testing.T) {
	root := project(t, ".codex")
	out, code := run(t, root, ".codex", "test-gate-record.sh", patch("a.go"))
	if out != "" || code != 0 {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func mbProject(t *testing.T, dot string, index bool) string {
	t.Helper()
	need(t)
	root := t.TempDir()
	hd := filepath.Join(root, dot, "hooks")
	if err := os.MkdirAll(hd, 0o755); err != nil {
		t.Fatal(err)
	}
	src, err := filepath.Abs("../../../domains/core/hooks/memory-bank-load.sh")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(src, filepath.Join(hd, "memory-bank-load.sh")); err != nil {
		t.Fatal(err)
	}
	if index {
		mb := filepath.Join(root, ".memory-bank")
		if err := os.MkdirAll(mb, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(mb, "index.md"), []byte("# idx\nhello\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestMemoryBankLoadOutputShapes(t *testing.T) {
	for _, dot := range []string{".claude", ".codex"} {
		root := mbProject(t, dot, true)
		out, code := run(t, root, dot, "memory-bank-load.sh", "{}")
		if code != 0 {
			t.Fatalf("%s: exit %d", dot, code)
		}
		var v struct {
			H struct {
				A string `json:"additionalContext"`
			} `json:"hookSpecificOutput"`
		}
		if err := json.Unmarshal([]byte(out), &v); err != nil || !strings.Contains(v.H.A, "hello") {
			t.Fatalf("%s: bad output %q (%v)", dot, out, err)
		}
	}
	root := mbProject(t, ".cursor", true)
	out, code := run(t, root, ".cursor", "memory-bank-load.sh", "{}")
	if code != 0 {
		t.Fatalf("cursor: exit %d", code)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("cursor: bad json %q", out)
	}
	if _, ok := m["hookSpecificOutput"]; ok || len(m) != 1 {
		t.Fatalf("cursor: unexpected keys in %q", out)
	}
	var ac string
	if err := json.Unmarshal(m["additional_context"], &ac); err != nil || !strings.Contains(ac, "hello") {
		t.Fatalf("cursor: bad additional_context %q", out)
	}
}

func TestMemoryBankLoadNoIndex(t *testing.T) {
	for _, dot := range []string{".claude", ".codex", ".cursor"} {
		root := mbProject(t, dot, false)
		silent(t, root, dot, "memory-bank-load.sh", "{}")
	}
}
