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

type Op struct {
	Type      OpType
	Src, Dst  string
	Mode      lockfile.Mode
	Domain    string
	Kind      lockfile.Kind
	Name      string
	Platforms []platform.ID
	Exclude   []string
	Item      bool
	rm        *removeUnit
	unit      string
	unitKind  lockfile.Kind
}

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
	git       gitCache
	rm        *removeState
}
