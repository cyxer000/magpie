package sessions

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/settings"
)

// An OpenCode fork keeps OpenCode's tables in a data folder of its own:
// named in sessionDirs, its sessions are listed, counted on Usage and read
// as transcripts under that entry's agent id, not OpenCode's, and with no
// command magpie would guess to resume them.
func TestSessionDirsOpenCodeFormat(t *testing.T) {
	inZone(t, 0)
	claude, _ := setup(t)
	dir := filepath.Join(filepath.Dir(claude), "fork-cli")
	os.MkdirAll(dir, 0o755)
	ocMakeDB(t, filepath.Join(dir, "opencode.db"))
	if err := settings.Save(settings.Settings{SessionDirs: []settings.SessionDir{{Agent: "fork-cli", Format: "opencode", Dir: dir}}}); err != nil {
		t.Fatal(err)
	}

	ss := List(0)
	fork := find(t, ss, "fork-cli", ocMain)
	for _, s := range ss {
		if s.Agent == "opencode" {
			t.Fatalf("the fork's session listed as OpenCode's too: %+v", s)
		}
	}
	if fork.Cwd != "/work/oc" || fork.Title != "Flaky test hunt" || fork.Resume != "" {
		t.Fatalf("fork-cli: %+v", fork)
	}
	if o := model(fork, "claude-opus-5-5"); o.Tokens != (Tokens{40, 60, 700, 300, 0}) {
		t.Fatalf("the subagent's opus: %+v", o)
	}
	if got, _ := usageOn(statsAt(0, statsNow), "2026-09-26", "/work/oc"); got != (Tokens{1340, 240, 7700, 300, 0}) {
		t.Fatalf("fork-cli on the 26th: %+v", got)
	}

	n := 0
	for _, c := range Calls(time.Time{}) {
		if c.Agent == "opencode" {
			t.Fatalf("a fork-cli call counted as OpenCode's: %+v", c)
		}
		if c.Agent == "fork-cli" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("want fork-cli's 2 own calls (the gateway's left out), got %d", n)
	}

	tr, err := TranscriptOf(fork)
	if err != nil || len(tr.Parts) == 0 {
		t.Fatalf("transcript: %v %+v", err, tr)
	}
	if d := Dirs(); d[len(d)-1] != dir {
		t.Fatalf("the folder isn't among the dirs read: %v", d)
	}
}

// A sessionDirs entry can't take the id of an agent magpie reads itself:
// Claude Code's sessions stay Claude Code's.
func TestSessionDirsKeepOwnAgents(t *testing.T) {
	claude, _ := setup(t)
	dir := filepath.Join(filepath.Dir(claude), "other")
	os.MkdirAll(dir, 0o755)
	ocMakeDB(t, filepath.Join(dir, "opencode.db"))
	if err := settings.Save(settings.Settings{SessionDirs: []settings.SessionDir{{Agent: "claude", Format: "opencode", Dir: dir}}}); err != nil {
		t.Fatal(err)
	}
	claudes := 0
	for _, s := range List(0) {
		if s.ID == ocMain {
			t.Fatalf("an entry named claude was read: %+v", s)
		}
		if s.Agent == "claude" && len(s.Models) > 0 {
			claudes++
		}
	}
	if claudes == 0 {
		t.Fatal("Claude Code's own sessions went")
	}
}
