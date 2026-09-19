package commands

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"drift/cli/output"
	"drift/orchestrator"
)

// D! id=ccfgcov range-start

// PathTarget is a per-path coverage override from .drift/coverage.xml.
// Files whose path starts with the pattern resolve to this target instead
// of the global one; the most specific (longest) matching pattern wins.
type PathTarget struct {
	Pattern string
	Target  int
}

// CoverageConfig is the parsed form of .drift/coverage.xml — optional,
// committed, team policy. It carries no behavior; EvaluateCoverageCheck
// applies it to a CoverageReport.
type CoverageConfig struct {
	Target  int
	Paths   []PathTarget
	present bool
}

// Present reports whether a coverage.xml file exists. A missing file means
// "no gate" — coverage behaves exactly as before this feature existed.
func (c CoverageConfig) Present() bool { return c.present }

type coverageXML struct {
	Target *targetXML `xml:"target"`
	Paths  []pathXML  `xml:"path"`
}

type targetXML struct {
	Lines *string `xml:"lines,attr"`
}

type pathXML struct {
	Pattern *string `xml:"pattern,attr"`
	Target  *string `xml:"target,attr"`
}

// coverageFileName is the committed project policy file inside .drift/.
const coverageFileName = "coverage.xml"

// ParseCoverageConfig parses the bytes of .drift/coverage.xml. Malformed XML,
// a missing or out-of-range target, or a path entry missing its pattern or
// target is an error naming the problem — never a silent fallback. The
// error text names coverage.xml so the user knows which file to fix.
func ParseCoverageConfig(data []byte) (CoverageConfig, error) {
	var raw coverageXML
	if err := xml.Unmarshal(data, &raw); err != nil {
		return CoverageConfig{}, fmt.Errorf("coverage.xml: %s", err)
	}

	cfg := CoverageConfig{present: true}

	if raw.Target == nil || raw.Target.Lines == nil {
		return CoverageConfig{}, fmt.Errorf("coverage.xml: <target lines=\"N\"/> is required (N is 0-100)")
	}
	lines, err := strconv.Atoi(strings.TrimSpace(*raw.Target.Lines))
	if err != nil {
		return CoverageConfig{}, fmt.Errorf("coverage.xml: <target lines> must be a number, got %q", *raw.Target.Lines)
	}
	if lines < 0 || lines > 100 {
		return CoverageConfig{}, fmt.Errorf("coverage.xml: <target lines> must be 0-100, got %d", lines)
	}
	cfg.Target = lines

	for _, p := range raw.Paths {
		if p.Pattern == nil || *p.Pattern == "" {
			return CoverageConfig{}, fmt.Errorf("coverage.xml: <path> requires a non-empty pattern attribute")
		}
		if p.Target == nil {
			return CoverageConfig{}, fmt.Errorf("coverage.xml: <path pattern=%q> requires a target attribute", *p.Pattern)
		}
		t, err := strconv.Atoi(strings.TrimSpace(*p.Target))
		if err != nil {
			return CoverageConfig{}, fmt.Errorf("coverage.xml: <path target> must be a number, got %q", *p.Target)
		}
		if t < 0 || t > 100 {
			return CoverageConfig{}, fmt.Errorf("coverage.xml: <path target> must be 0-100, got %d", t)
		}
		cfg.Paths = append(cfg.Paths, PathTarget{Pattern: *p.Pattern, Target: t})
	}

	return cfg, nil
}

// CoverageFailure is defined in cli/output (see result.go); this package
// produces the values.

// EvaluateCoverageCheck applies a config to a coverage report and returns
// the failing groups. A group fails when its covered percentage is below
// its target. Only meaningful when cfg.Present().
func EvaluateCoverageCheck(cfg CoverageConfig, report orchestrator.CoverageReport) []output.CoverageFailure {
	type group struct{ covered, total int }
	groups := map[string]*group{"": {}}
	for _, pt := range cfg.Paths {
		groups[pt.Pattern] = &group{}
	}
	for _, f := range report.Files {
		key := ""
		bestLen := -1
		norm := filepath.ToSlash(f.Path)
		for _, pt := range cfg.Paths {
			if strings.HasPrefix(norm, pt.Pattern) && len(pt.Pattern) > bestLen {
				key, bestLen = pt.Pattern, len(pt.Pattern)
			}
		}
		g := groups[key]
		g.covered += f.CoveredAny
		g.total += f.TotalLines
	}

	targets := map[string]int{"": cfg.Target}
	for _, pt := range cfg.Paths {
		targets[pt.Pattern] = pt.Target
	}

	var failures []output.CoverageFailure
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		g := groups[k]
		target := targets[k]
		pct := 0
		if g.total > 0 {
			pct = g.covered * 100 / g.total
		}
		if pct < target {
			name := k
			if name == "" {
				name = "(global)"
			}
			failures = append(failures, output.CoverageFailure{Group: name, Actual: pct, Target: target})
		}
	}
	return failures
}

// D! id=ccfgcov range-end
