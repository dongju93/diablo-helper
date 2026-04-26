package meta

var (
	Version   = "0.1.0-dev"
	Commit    = "none"
	BuildDate = "unknown"
	Author    = "dongju93"
	Repo      = "github.com/dongju93/diablo-helper"
	GoVersion = "1.26.2"
)

func Title() string {
	return "Diablo Helper v" + Version
}
