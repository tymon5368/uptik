package main

// Build metadata injected at compile-time via -ldflags
var (
	// Version is the current semantic version of the UpTik application
	Version = "1.0.0"

	// Commit is the git commit hash of the build
	Commit = "none"

	// BuildDate is the timestamp when the binary was built
	BuildDate = "unknown"
)
