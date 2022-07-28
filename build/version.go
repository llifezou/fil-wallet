package build

var CurrentCommit string

var version = "v0.1.1"

func Version() string {
	return version + CurrentCommit
}
