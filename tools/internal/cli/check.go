package cli

import (
	"fmt"
	"strings"

	"claude-harness/tools/internal/lint"
)

func runCheck(args []string, d Deps) error {
	ios := d.IO
	if len(args) != 0 {
		return fmt.Errorf("check takes no arguments (got %s)", strings.Join(args, " "))
	}
	harnessRoot, err := d.Locator.Locate("")
	if err != nil {
		return err
	}
	res, err := lint.Check(harnessRoot)
	if err != nil {
		return err
	}
	if len(res.Findings) > 0 {
		for _, f := range res.Findings {
			fmt.Fprintln(ios.Stderr(), "as-skill: check:", f)
		}
		return fmt.Errorf("check: %d failure(s)", len(res.Findings))
	}
	fmt.Fprintf(ios.Stdout(), "check: OK — %d manifest(s), %d SKILL.md(s)\n", res.Domains, res.Skills)
	return nil
}
