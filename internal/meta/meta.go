package meta

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
	Author    = "dongju93"
	Repo      = "github.com/dongju93/diablo-helper"
	GoVersion = "unknown"
)

func Title() string {
	return "Diablo Helper v" + Version
}
