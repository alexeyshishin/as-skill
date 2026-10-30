package cli

import (
	"reflect"
	"testing"
)

func TestParseAddArgsTable(t *testing.T) {
	cases := []struct {
		name    string
		verb    string
		args    []string
		want    addOpts
		wantErr string
	}{
		{name: "interspersed", verb: "add", args: []string{"git", "-y", "core", "-p", "x"}, want: addOpts{verb: "add", names: []string{"git", "core"}, project: "x", projectSet: true, yes: true}},
		{name: "equals long", verb: "add", args: []string{"--project=x", "--agent=codex", "git"}, want: addOpts{verb: "add", names: []string{"git"}, project: "x", projectSet: true, agents: []string{"codex"}}},
		{name: "repeat agents raw", verb: "add", args: []string{"-a", "claude,codex", "--platform", "cursor", "-a", "all"}, want: addOpts{verb: "add", project: ".", agents: []string{"claude,codex", "cursor", "all"}}},
		{name: "hidden flags", verb: "add", args: []string{"--force", "--harness-root", "h"}, want: addOpts{verb: "add", project: ".", force: true, harnessRoot: "h"}},
		{name: "help short", verb: "add", args: []string{"-h"}, want: addOpts{verb: "add", project: ".", help: true}},
		{name: "help long remove", verb: "remove", args: []string{"--help"}, want: addOpts{verb: "remove", project: ".", help: true}},
		{name: "dry run copy", verb: "add", args: []string{"--dry-run", "--copy", "x"}, want: addOpts{verb: "add", names: []string{"x"}, project: ".", dryRun: true, copy: true}},
		{name: "empty project value", verb: "add", args: []string{"--project="}, want: addOpts{verb: "add", project: "", projectSet: true}},
		{name: "value may start with dash", verb: "add", args: []string{"-p", "-x"}, want: addOpts{verb: "add", project: "-x", projectSet: true}},
		{name: "copy on remove", verb: "remove", args: []string{"--copy"}, wantErr: "--copy applies to add only"},
		{name: "unknown long", verb: "add", args: []string{"--bogus"}, wantErr: `unknown flag "--bogus"`},
		{name: "unknown long equals keeps raw arg", verb: "add", args: []string{"--bogus=1"}, wantErr: `unknown flag "--bogus=1"`},
		{name: "unknown short", verb: "add", args: []string{"-z"}, wantErr: `unknown flag "-z"`},
		{name: "no abbreviation", verb: "add", args: []string{"--proj", "x"}, wantErr: `unknown flag "--proj"`},
		{name: "bare dash positional", verb: "add", args: []string{"-"}, want: addOpts{verb: "add", names: []string{"-"}, project: "."}},
		{name: "bundled unknown short", verb: "add", args: []string{"-yx"}, wantErr: `unknown flag "-x"`},
		{name: "invalid bool", verb: "add", args: []string{"--dry-run=maybe"}, wantErr: `invalid argument "maybe" for "--dry-run" flag: strconv.ParseBool: parsing "maybe": invalid syntax`},
		{name: "missing short value", verb: "add", args: []string{"git", "-p"}, wantErr: "-p needs a value"},
		{name: "missing long value", verb: "add", args: []string{"--agent"}, wantErr: "--agent needs a value"},
		{name: "missing harness value", verb: "add", args: []string{"--harness-root"}, wantErr: "--harness-root needs a value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseAddArgs(c.verb, c.args)
			if c.wantErr != "" {
				if err == nil || err.Error() != c.wantErr {
					t.Fatalf("err = %v, want %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %+v\nwant %+v", got, c.want)
			}
		})
	}
}

func TestParseInstallArgsTable(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantPos []string
		want    installOpts
		wantErr string
	}{
		{name: "defaults", args: []string{"code"}, wantPos: []string{"code"}, want: installOpts{project: ".", harnessRoot: "."}},
		{name: "interspersed", args: []string{"--copy", "code", "-p", "x", "git"}, wantPos: []string{"code", "git"}, want: installOpts{project: "x", harnessRoot: ".", copy: true}},
		{name: "domain lists", args: []string{"-d", "a, b,,c", "--domain=d"}, want: installOpts{project: ".", harnessRoot: ".", domains: []string{"a", "b", "c", "d"}}},
		{name: "harness root set", args: []string{"--harness-root", "h"}, want: installOpts{project: ".", harnessRoot: "h", harnessSet: true}},
		{name: "platform aliases", args: []string{"--platform=codex", "--agent", "cursor", "-a", "claude"}, want: installOpts{project: ".", harnessRoot: ".", platforms: []string{"codex", "cursor", "claude"}}},
		{name: "dry run force", args: []string{"--dry-run", "--force"}, want: installOpts{project: ".", harnessRoot: ".", dryRun: true, force: true}},
		{name: "agent all", args: []string{"-a", "all"}, want: installOpts{project: ".", harnessRoot: ".", platforms: []string{"all"}}},
		{name: "agent list raw", args: []string{"-a", "claude,codex"}, want: installOpts{project: ".", harnessRoot: ".", platforms: []string{"claude,codex"}}},
		{name: "yes is unknown", args: []string{"-y"}, wantErr: `unknown flag "-y"`},
		{name: "help is unknown", args: []string{"--help"}, wantErr: `unknown flag "--help"`},
		{name: "short help is unknown", args: []string{"x", "-h"}, wantErr: `unknown flag "-h"`},
		{name: "unknown equals", args: []string{"--nope=1"}, wantErr: `unknown flag "--nope=1"`},
		{name: "missing value", args: []string{"-d"}, wantErr: "-d needs a value"},
		{name: "missing long value", args: []string{"--project"}, wantErr: "--project needs a value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pos, got, err := parseInstallArgs(c.args)
			if c.wantErr != "" {
				if err == nil || err.Error() != c.wantErr {
					t.Fatalf("err = %v, want %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(pos, c.wantPos) || !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %v %+v\nwant %v %+v", pos, got, c.wantPos, c.want)
			}
		})
	}
}

func TestParseStatusArgsTable(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		want    statusOpts
		wantErr string
	}{
		{name: "defaults", want: statusOpts{project: "."}},
		{name: "all flags", args: []string{"-p", "x", "--harness-root=h", "-a", "codex", "--platform", "cursor"}, want: statusOpts{project: "x", harnessRoot: "h", platforms: []string{"codex", "cursor"}}},
		{name: "agent all", args: []string{"-a", "all"}, want: statusOpts{project: ".", platforms: []string{"all"}}},
		{name: "agent list raw", args: []string{"-a", "claude,codex"}, want: statusOpts{project: ".", platforms: []string{"claude,codex"}}},
		{name: "positional rejected", args: []string{"x"}, wantErr: `unknown flag "x"`},
		{name: "unknown", args: []string{"--dry-run"}, wantErr: `unknown flag "--dry-run"`},
		{name: "help unknown", args: []string{"-h"}, wantErr: `unknown flag "-h"`},
		{name: "missing value", args: []string{"--agent"}, wantErr: "--agent needs a value"},
		{name: "missing short value", args: []string{"-p"}, wantErr: "-p needs a value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseStatusArgs(c.args)
			if c.wantErr != "" {
				if err == nil || err.Error() != c.wantErr {
					t.Fatalf("err = %v, want %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %+v\nwant %+v", got, c.want)
			}
		})
	}
}

func TestParseListArgsTable(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantProject string
		wantHarness string
		wantPos     []string
		wantErr     string
	}{
		{name: "defaults", wantProject: "."},
		{name: "interspersed", args: []string{"skills", "-p", "x", "--harness-root", "h"}, wantProject: "x", wantHarness: "h", wantPos: []string{"skills"}},
		{name: "equals", args: []string{"--project=x", "domains"}, wantProject: "x", wantPos: []string{"domains"}},
		{name: "unknown", args: []string{"-y"}, wantErr: `unknown flag "-y"`},
		{name: "unknown equals", args: []string{"--x=1"}, wantErr: `unknown flag "--x=1"`},
		{name: "help unknown", args: []string{"--help"}, wantErr: `unknown flag "--help"`},
		{name: "missing value", args: []string{"-p"}, wantErr: "-p needs a value"},
		{name: "missing long value", args: []string{"--harness-root"}, wantErr: "--harness-root needs a value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, h, pos, err := parseListArgs(c.args)
			if c.wantErr != "" {
				if err == nil || err.Error() != c.wantErr {
					t.Fatalf("err = %v, want %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p != c.wantProject || h != c.wantHarness || !reflect.DeepEqual(pos, c.wantPos) {
				t.Fatalf("got %q %q %v", p, h, pos)
			}
		})
	}
}
