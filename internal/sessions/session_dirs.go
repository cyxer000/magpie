package sessions

import (
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"

	"github.com/yetone/magpie/internal/settings"
)

// settings.json's sessionDirs name the folders of agents magpie doesn't
// know whose sessions are kept in a format it reads. format names the
// reader; today only "opencode", for any OpenCode fork that keeps OpenCode's
// tables in a data folder of its own (<dir>/opencode.db, else
// <dir>/storage). Each folder's sessions are listed and counted under the
// entry's own agent id, with no command to resume them: magpie doesn't
// know the agent's.

// dirAgents are the agent ids sessionDirs last read, for the lookups by
// agent made for every file (parserFor), which can't read settings.json
// each time. Each listing reads them again.
var dirAgents atomic.Pointer[map[string]bool]

// ownSessionAgents are the agents whose sessions magpie reads by itself: a
// sessionDirs folder can't take one's id (it is left out), or that agent's
// own sessions would be read as OpenCode's.
var ownSessionAgents = []string{"agy", "alma", "claude", "claude-desktop", "cline", "codebuddy", "codex", "cursor",
	"droid", "dsh", "gemini", "grok", "hermes", "kimi", "minimax-code", "omp", "opencode", "pi", "qoder", "qoder-cn",
	"workbuddy", "zcode"}

// sessionDirs are settings.json's sessionDirs in a format magpie reads,
// "~" expanded.
func sessionDirs() []settings.SessionDir {
	var out []settings.SessionDir
	set := map[string]bool{}
	for _, d := range settings.Shared().SessionDirs {
		if d.Format != "opencode" || d.Agent == "" || d.Dir == "" {
			continue
		}
		if slices.Contains(ownSessionAgents, d.Agent) {
			continue
		}
		d.Dir = expandHome(d.Dir)
		out = append(out, d)
		set[d.Agent] = true
	}
	dirAgents.Store(&set)
	return out
}

// OpenCodeLike says whether an agent's sessions are OpenCode's tables:
// OpenCode's own, or those of a sessionDirs folder in its format.
func OpenCodeLike(agent string) bool {
	if agent == "opencode" {
		return true
	}
	m := dirAgents.Load()
	if m == nil {
		sessionDirs()
		m = dirAgents.Load()
	}
	return (*m)[agent]
}

// ocKind is agent as its sessions are read: "opencode" for every agent
// whose sessions are OpenCode's tables.
func ocKind(agent string) string {
	if OpenCodeLike(agent) {
		return "opencode"
	}
	return agent
}

// ocDirFiles are the sessions in an OpenCode data folder: its database,
// else its JSON files.
func ocDirFiles(agent, dir string) []file {
	if db := filepath.Join(dir, "opencode.db"); fileExists(db) {
		return openCodeDBFiles(agent, db)
	}
	return openCodeJSONFiles(agent, filepath.Join(dir, "storage"))
}

// ocFilesOf are an OpenCode-like agent's sessions.
func ocFilesOf(agent string) []file {
	if agent == "opencode" {
		return openCodeFiles()
	}
	for _, d := range sessionDirs() {
		if d.Agent == agent {
			return ocDirFiles(agent, d.Dir)
		}
	}
	return nil
}

// sessionDirFiles are the sessions in every sessionDirs folder.
func sessionDirFiles() []file {
	var out []file
	for _, d := range sessionDirs() {
		out = append(out, ocDirFiles(d.Agent, d.Dir)...)
	}
	return out
}

// sessionDirCallFiles are those sessions as call sources.
func sessionDirCallFiles() []file {
	var out []file
	for _, d := range sessionDirs() {
		out = append(out, ocCallFilesIn(d.Agent, filepath.Join(d.Dir, "opencode.db"), filepath.Join(d.Dir, "storage"))...)
	}
	return out
}

// sessionDirsThere are the sessionDirs folders that are there.
func sessionDirsThere() []string {
	var out []string
	for _, d := range sessionDirs() {
		if fi, err := os.Stat(d.Dir); err == nil && fi.IsDir() {
			out = append(out, d.Dir)
		}
	}
	return out
}
