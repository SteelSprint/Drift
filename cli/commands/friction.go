package commands

import (
	"encoding/json"
	"os"
	"time"

	"drift/internal/fileio"
)

// Friction rate-limit constants (see cli.reset_friction_block).
const (
	frictionWindow   = 30 * time.Second
	frictionBurst    = 3
	frictionPruneAge = 60 * time.Second
	frictionFileName = "friction.json"
)

// frictionOverrideSquawk is written to stderr when a reset proceeds under
// --dangerously-override-friction. Not in the JSON changeSummary.Message —
// that field is the caller-supplied lead-in ("Closure HASH resolved.").
const frictionOverrideSquawk = "WARNING: bypassing friction rate limit at user request. I sure hope you know what you're doing.\n"

// frictionWarningField is surfaced in the JSON changeSummary output as the
// `warning` field when a reset proceeded under override, so programmatic
// consumers can detect the bypass without parsing stderr.
const frictionWarningField = "bypass-friction"

// frictionFile is the on-disk telemetry shape: a list of Unix-second
// timestamps of recent successful non-dry-run resets. Missing/empty/malformed
// file is treated as zero timestamps (permissive — see R6).
type frictionFile struct {
	Resets []int64 `json:"resets"`
}

// D! id=frblk range-start
// checkFriction loads the friction telemetry file via the Session and returns
// nil if the rate limit is not exceeded. If ≥ frictionBurst resets have
// occurred within frictionWindow, it returns a non-nil error whose message
// names the friction principle and the intended workflow. The message MUST
// NOT advertise the --dangerously-override-friction flag (see R1).
//
// Missing, empty, or malformed friction.json is treated as zero prior resets
// (permissive — telemetry is not authoritative). The Session lock is already
// held by the caller for the duration of the CLI invocation, so the read is
// serialized with any concurrent reset.
func checkFriction(sess *fileio.Session) error {
	data, err := sess.Read(frictionFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		// Malformed/unreadable telemetry — permissive: do not block.
		return nil
	}
	var ff frictionFile
	if err := json.Unmarshal(data, &ff); err != nil {
		return nil
	}
	now := time.Now().Unix()
	cutoff := now - int64(frictionWindow/time.Second)
	recent := 0
	for _, ts := range ff.Resets {
		if ts >= cutoff {
			recent++
		}
	}
	if recent < frictionBurst {
		return nil
	}
	return errFrictionBlocked
}

// D! id=frblk range-end

// D! id=frrec range-start
// recordReset appends the current Unix-second timestamp to the friction
// telemetry file via the Session, pruning entries older than frictionPruneAge.
// Called only after a successful non-dry-run reset.
//
// If the existing file is missing or malformed, recording starts fresh with
// just the new timestamp. Pruning bounds the file to roughly
// frictionBurst entries over frictionPruneAge.
func recordReset(sess *fileio.Session) error {
	data, err := sess.Read(frictionFileName)
	var ff frictionFile
	if err == nil {
		_ = json.Unmarshal(data, &ff)
	}
	now := time.Now().Unix()
	pruneCutoff := now - int64(frictionPruneAge/time.Second)
	var kept []int64
	for _, ts := range ff.Resets {
		if ts >= pruneCutoff {
			kept = append(kept, ts)
		}
	}
	kept = append(kept, now)
	ff.Resets = kept
	out, mErr := json.Marshal(ff)
	if mErr != nil {
		return mErr
	}
	return sess.Write(frictionFileName, out)
}

// D! id=frrec range-end

var errFrictionBlocked = simpleError("rate limit: 3 closures resolved in the last 30s. Drift's friction principle expects per-closure review — the intended workflow is `drift todo` → `drift diff --all` → `drift reset <hash>` one closure at a time, with each closure reviewed before it is reset.")

// simpleError wraps a string as an error without exposing the friction
// internals to callers that might try to unwrap or inspect it.
type simpleError string

func (s simpleError) Error() string { return string(s) }
