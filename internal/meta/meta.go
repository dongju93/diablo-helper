package meta

var (
	Version   = "0.3.0"
	Commit    = "none"
	BuildDate = "unknown"
	Author    = "dongju93"
	Repo      = "https://github.com/dongju93/diablo-helper"
	GoVersion = "1.26.2"
)

func Title() string {
	return "Diablo Helper v" + Version
}
