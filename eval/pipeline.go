package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	subjectTimeout = 30 * time.Minute
	judgeTimeout   = 15 * time.Minute
)

type Pipeline struct {
	repoRoot     string
	runLabel     string
	runDir       string
	toolDir      string
	workspaceDir string
	subjectModel string
	judgeModel   string
}

func NewPipeline(repoRoot, label, subjectModel, judgeModel string) *Pipeline {
	return &Pipeline{
		repoRoot:     repoRoot,
		runLabel:     label,
		subjectModel: subjectModel,
		judgeModel:   judgeModel,
	}
}

func (p *Pipeline) Run(prompt string, dryRun bool) error {
	fmt.Printf("=== eval run: %s ===\n", p.runLabel)
	fmt.Printf("subject model: %s\n", p.subjectModel)
	fmt.Printf("judge model:   %s\n", p.judgeModel)

	if err := p.stage(); err != nil {
		return fmt.Errorf("stage: %w", err)
	}
	fmt.Printf("[stage] done → %s\n", p.runDir)

	if dryRun {
		fmt.Println("[dry-run] skipping LLM calls")
		return nil
	}

	if err := p.runSubject(prompt); err != nil {
		return fmt.Errorf("subject: %w", err)
	}
	fmt.Println("[subject] done")

	if err := p.runJudge(prompt); err != nil {
		return fmt.Errorf("judge: %w", err)
	}
	fmt.Println("[judge] done")

	if err := p.surface(); err != nil {
		return fmt.Errorf("surface: %w", err)
	}

	return nil
}

func (p *Pipeline) stage() error {
	p.runDir = filepath.Join(p.repoRoot, "eval", "runs", p.runLabel)
	p.toolDir = filepath.Join(p.runDir, "tool")
	p.workspaceDir = filepath.Join(p.runDir, "workspace")

	for _, dir := range []string{p.runDir, p.toolDir, p.workspaceDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	if err := p.buildAndCopyBinary(); err != nil {
		return err
	}

	for _, doc := range []string{"README.md", "DOCUMENTATION.md"} {
		if err := copyFile(filepath.Join(p.repoRoot, doc), filepath.Join(p.toolDir, doc)); err != nil {
			return fmt.Errorf("copy %s: %w", doc, err)
		}
	}

	agentSrc := filepath.Join(p.repoRoot, "eval", "agents")
	subjectAgentsDir := filepath.Join(p.workspaceDir, ".opencode", "agents")
	if err := os.MkdirAll(subjectAgentsDir, 0755); err != nil {
		return err
	}
	if err := copyFile(
		filepath.Join(agentSrc, "eval-subject.md"),
		filepath.Join(subjectAgentsDir, "eval-subject.md"),
	); err != nil {
		return fmt.Errorf("stage subject agent: %w", err)
	}

	judgeAgentsDir := filepath.Join(p.runDir, ".opencode", "agents")
	if err := os.MkdirAll(judgeAgentsDir, 0755); err != nil {
		return err
	}
	if err := copyFile(
		filepath.Join(agentSrc, "eval-judge.md"),
		filepath.Join(judgeAgentsDir, "eval-judge.md"),
	); err != nil {
		return fmt.Errorf("stage judge agent: %w", err)
	}

	return nil
}

func (p *Pipeline) buildAndCopyBinary() error {
	cmd := exec.Command("go", "build", "-o", filepath.Join(p.toolDir, "drift"), "./cmd/drift")
	cmd.Dir = p.repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	return os.Chmod(filepath.Join(p.toolDir, "drift"), 0755)
}

func (p *Pipeline) runSubject(prompt string) error {
	subjectOut, err := os.Create(filepath.Join(p.runDir, "subject.jsonl"))
	if err != nil {
		return err
	}
	defer subjectOut.Close()

	fullPrompt := buildSubjectPrompt(prompt)

	return p.runOpencode(&opencodeArgs{
		agent:   "eval-subject",
		model:   p.subjectModel,
		dir:     p.workspaceDir,
		title:   "subject",
		prompt:  fullPrompt,
		stdout:  subjectOut,
		timeout: subjectTimeout,
	})
}

func (p *Pipeline) runJudge(originalPrompt string) error {
	judgeOut, err := os.Create(filepath.Join(p.runDir, "judge.jsonl"))
	if err != nil {
		return err
	}
	defer judgeOut.Close()

	judgePrompt := buildJudgePrompt(originalPrompt, p.workspaceDir, p.runDir)

	return p.runOpencode(&opencodeArgs{
		agent:   "eval-judge",
		model:   p.judgeModel,
		dir:     p.runDir,
		title:   "judge",
		prompt:  judgePrompt,
		stdout:  judgeOut,
		timeout: judgeTimeout,
	})
}

func (p *Pipeline) surface() error {
	reportPath := filepath.Join(p.runDir, "report.md")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("read report.md: %w", err)
	}
	fmt.Println("\n=== REPORT ===")
	fmt.Println(string(data))

	logPath := filepath.Join(p.repoRoot, "eval", "runs", "log.csv")
	row := fmt.Sprintf("%s,%q,%s,%s,%s\n",
		p.runLabel,
		"see report",
		p.runDir,
		p.subjectModel,
		p.judgeModel,
	)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log.csv: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(row)
	return err
}

type opencodeArgs struct {
	agent   string
	model   string
	dir     string
	title   string
	prompt  string
	stdout  io.Writer
	timeout time.Duration
}

func (p *Pipeline) runOpencode(a *opencodeArgs) error {
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "opencode", "run",
		"--agent", a.agent,
		"--model", a.model,
		"--auto",
		"--format", "json",
		"--dir", a.dir,
		"--title", a.title,
		a.prompt,
	)
	cmd.Stdout = a.stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[%s] running opencode (agent=%s model=%s dir=%s timeout=%v)\n",
		a.title, a.agent, a.model, a.dir, a.timeout)
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("opencode timed out after %v", a.timeout)
		}
		return fmt.Errorf("opencode run: %w", err)
	}
	return nil
}

func buildSubjectPrompt(task string) string {
	return fmt.Sprintf(`You are being evaluated on your ability to use a spec-drift tool called "driftpin".

## Your environment

- Your working directory is an EMPTY project directory.
- A pre-built `+"`drift`"+` binary and its documentation live in `+"`../tool/`"+`. Read `+"`../tool/README.md`"+` and `+"`../tool/DOCUMENTATION.md`"+` to understand the tool.
- The `+"`drift`"+` binary is at `+"`../tool/drift`"+`. You will need to make it executable or invoke it with a full path.
- You have NO access to the driftpin source code — only the binary and docs.

## Your task

%s

## What you must do

1. Read the driftpin documentation in `+"`../tool/`"+`.
2. Complete the task above — build the project, write the code, make it work.
3. Use driftpin properly and end-to-end throughout:
   - Run `+"`drift init`"+` in your project.
   - Create spec files (`+"`*.pin.xml`"+`) that describe what your code does.
   - Place `+"`D! id=<markerid>`"+` markers in your code at the locations that implement each spec.
   - Run `+"`drift link <marker> <module.spec>`"+` to connect markers to specs.
   - Run `+"`drift todo`"+` and make sure it reports "No changes detected." (meaning specs and code are in sync).
4. Write a file called `+"`self-debrief.md`"+` in your project root with these EXACT sections:
   - **What worked well**: What was easy or intuitive about using driftpin.
   - **What was confusing**: What was hard to understand or figure out.
   - **Errors encountered**: Any errors you hit and how you resolved them (or didn't).
   - **Missing documentation**: Things you needed to know that weren't in the docs.
   - **Suggestions for the tool authors**: Concrete improvements that would make driftpin easier for an LLM to use cold.

## Important

- Work autonomously. Do not ask questions. Make your best judgment.
- Finish with a working project that has specs, markers, links, and a clean `+"`drift todo`"+`.
- Your `+"`self-debrief.md`"+` is critical — it will be read by a judge LLM evaluating your work. Be thorough and honest.
`, task)
}

func buildJudgePrompt(originalTask, workspaceDir, runDir string) string {
	return fmt.Sprintf(`You are the JUDGE in an LLM-as-judge evaluation of a spec-drift tool called "driftpin".

## Context

A subject LLM was given a task and asked to use driftpin (a spec-drift tool) end-to-end while completing it. You must evaluate how well the subject used driftpin and how well the tool served the subject.

## Artifacts to inspect

1. **The original task prompt:**
   %s

2. **The subject's workspace** (its completed project): `+"`%s`"+`
   - Check `+"`main.pin.xml`"+` — is it present? Well-structured? Does it use the module/import system?
   - Run `+"`%s/drift todo`"+` — does it report "No changes detected."? (Clean = good)
   - Check `+"`*.pin.xml`"+` files — are specs meaningful or boilerplate?
   - Check markers (`+"`D! id=...`"+`) in code — are they placed at meaningful locations?
   - Check links — are markers linked to specs?
   - Read `+"`self-debrief.md`"+` — the subject's own feedback (this is the user LLM speaking to you).

3. **The subject's transcript** (JSONL of its session): `+"`%s/subject.jsonl`"+`
   - Sample it for confusion, tool misuse, or errors. You don't need to read every line — focus on moments where the subject struggled.

## What you must produce

Write a file called `+"`report.md`"+` in the run directory (your current working directory) with these EXACT sections:

### 1. Scorecard

Rate each item PASS or FAIL with a one-line note:
- Built the project (task completed)
- Created `+"`main.pin.xml`"+` (entry point exists)
- Used module/import system correctly
- Created meaningful specs (not boilerplate)
- Placed markers at meaningful code locations
- Linked markers to specs (`+"`drift link`"+`)
- `+"`drift todo`"+` reports clean (no drift)

### 2. Qualitative Assessment

3-5 paragraphs covering:
- How well did the subject understand and use driftpin?
- What patterns of confusion or success did you see?
- Did the subject's `+"`self-debrief.md`"+` reveal any UX problems?
- Was the documentation sufficient for cold use?

### 3. Tool-Improvement Recommendations

A PRIORITIZED list of concrete, actionable improvements to driftpin, ordered by impact:
1. [High/Medium/Low] <recommendation> — <reasoning>
2. ...

These recommendations will be triaged into the tool's development plan, so be specific and practical.

## Constraints

- You may read any file in the workspace or run directory.
- You may run bash commands (e.g., `+"`drift todo`"+`) to verify the subject's work.
- You may ONLY write to `+"`report.md`"+` — do not modify any other file.
- Be rigorous and fair. Don't inflate scores.
`, originalTask, workspaceDir, filepath.Join(workspaceDir, "..", "tool"), runDir)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func init() {
	// Suppress unused import warning for strings if we expand later.
	_ = strings.TrimSpace
}
