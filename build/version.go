package build

var CurrentCommit string

var version = "v0.2.0"

func Version() string {
	return version + CurrentCommit
}
