package installer

import (
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

type OpType string

const (
	OpLink   OpType = "link"
	OpCopy   OpType = "copy"
	OpRender OpType = "render"
	OpBlock  OpType = "block"
	OpRemove OpType = "remove"
	OpHook   OpType = "hook"
)

type Stream string

const (
	Stdout Stream = "stdout"
	Stderr Stream = "stderr"
)

// Op is one placement unit. Name == "" means the whole Src tree at Dst;
// Exclude lists skills left out of a skills tree.
type Op struct {
	Type      OpType
	Src, Dst  string
	Mode      lockfile.Mode
	Domain    string
	Kind      lockfile.Kind
	Name      string
	Platforms []platform.ID
	Exclude   []string
	// Item: placed through placeOne (lockfile entry carries Platforms), not place().
	Item bool
	// unit/unitKind: the domain (or skill) and kind this op was planned for; error wording only.
	unit     string
	unitKind lockfile.Kind
}

// Warning is printed before Ops[Before] (Before == len(Ops): after the last op).
// WriteOnly warnings are skipped on a dry run.
type Warning struct {
	Text      string
	Stream    Stream
	Before    int
	WriteOnly bool
}

type Conflict = ConflictError

type Plan struct {
	Ops       []Op
	Conflicts []Conflict
	Warnings  []Warning
	// git: git-ignore answers shared by the planner and Apply for this plan.
	git gitCache
}
