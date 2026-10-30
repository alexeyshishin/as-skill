package harness

import (
	"claude-harness/tools/internal/fsys"
	"os"
)

type MissKind int

const (
	MissEnv MissKind = iota
	MissBin
)

type Miss struct {
	Kind MissKind
	Name string
}

func (m *Manifest) GateWith(lookupEnv func(string) string, hasBin func(string) bool) (Miss, bool) {
	for _, e := range m.RequiresEnv {
		if lookupEnv(e) == "" {
			return Miss{Kind: MissEnv, Name: e}, true
		}
	}
	for _, b := range m.RequiresBin {
		if !hasBin(b) {
			return Miss{Kind: MissBin, Name: b}, true
		}
	}
	return Miss{}, false
}

func (m *Manifest) Gate() (Miss, bool) {
	return m.GateWith(os.Getenv, fsys.HasBin)
}
