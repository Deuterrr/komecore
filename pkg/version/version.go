package version

import "fmt"

// Build-time variables injected via -ldflags
var (
	// Version is the current semantic release tag.
	Version = "v1.0.0"
	// GitCommit is the short git commit hash of the build.
	GitCommit = "none"
	// BuildDate is the ISO-8601 UTC timestamp of the build.
	BuildDate = "unknown"
)

// Info encapsulates the compiled version metadata.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
}

// Get returns the structured version metadata.
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
	}
}

// String returns a formatted version string.
func String() string {
	if GitCommit == "none" || GitCommit == "" {
		return Version
	}
	return fmt.Sprintf("%s (%s, built %s)", Version, GitCommit, BuildDate)
}
