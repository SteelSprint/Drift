package commands

import (
	"drift/cli/output"
	"drift/internal/fileio"
	"drift/orchestrator"
)

// D! id=ocmd range-start
// Command is the interface every subcommand implements. The dispatcher
// (cli.RunWithRender) looks up a Command by name in cli.Registry, constructs a
// Context, calls Run, and renders the returned Result via the active
// Presenter. Commands produce data (Result); they never format strings — that
// is the Presenter's job.
type Command interface {
	Run(ctx Context) (output.Result, int)
	Meta() Meta
}

// Context carries everything a command needs at dispatch time. Args are
// positional (global flags will be stripped here in Landing 4). Dir is the
// project root. Orch is the orchestrator wired to the project's state store,
// scanner, and baseline store. Sess is the fileio.Session begun at the start
// of the CLI invocation; commands must route all orchestrator calls through
// it (see fileio.session).
type Context struct {
	Args []string
	Dir  string
	Orch *orchestrator.Orchestrator
	Sess *fileio.Session
}

// LockMode declares a command's state-access needs. The dispatcher reads it
// before Run and decides whether — and how — to open the fileio.Session:
//
//	LockNone     the command never touches .drift/ (help, skill, version).
//	             No session is opened; Context.Sess is nil.
//	LockRequire  the command reads or writes state of an initialized project
//	             (todo, list, show, coverage, diff, link, unlink, reset,
//	             config). The dispatcher locks via fileio.Begin; an
//	             uninitialized project fails with exit code 2 and a "run
//	             'drift init' first" message. No .drift/ is created.
//	LockInit     only init. The dispatcher locks via fileio.BeginCreate, which
//	             creates .drift/ — init is the single command that may.
type LockMode int

const (
	LockNone LockMode = iota
	LockRequire
	LockInit
)

// Meta describes a command's user-facing metadata. The dispatcher derives
// help text, flag validation, and the command table from Meta — this is the
// single source of truth that replaces the former help.txt,
// subcommandHelpTexts, and recognizedFlags maps.
type Meta struct {
	Name  string   // e.g. "todo"
	Short string   // one-line description for the command table in `drift help`
	Usage string   // multi-line usage text for `drift <cmd> --help`
	Flags []string // recognized long flags beyond --help (e.g. "--verbose", "--all")
	Lock  LockMode // state-access mode; see LockMode. Zero value is LockNone.
}

// Version is the build version string, set by main.go via ldflags before any
// command dispatches. VersionCommand reads this at Run time.
var Version = "dev"

// D! id=ocmd range-end
