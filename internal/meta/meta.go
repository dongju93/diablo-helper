// Package meta exposes build-time application metadata.
package meta

var (
	// Version is the semantic application version shown in the UI and binary metadata.
	Version = "0.3.1"
	// Commit is the source revision stamped into release builds.
	Commit = "none"
	// BuildDate is the UTC build timestamp stamped into release builds.
	BuildDate = "unknown"
	// Author is the project author name used in version metadata.
	Author = "dongju93"
	// Repo is the canonical source repository URL used in metadata.
	Repo = "https://github.com/dongju93/diablo-helper"
	// GoVersion is the Go toolchain version stamped into release builds.
	GoVersion = "1.26.2"
)

// Title returns the versioned application title used by the UI.
func Title() string {
	return "Diablo Helper v" + Version
}
