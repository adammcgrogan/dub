// Package ui implements the terminal user interface for dub, built on the
// Bubble Tea framework. It handles URL input, real-time download progress
// display, and communication with the yt-dlp subprocess.
package ui

// trackNameMsg is dispatched when yt-dlp begins downloading a new track.
// The value is the sanitised track title extracted from the destination path.
type trackNameMsg string

// trackCachedMsg is dispatched when yt-dlp skips a track because it was
// already downloaded in a previous session.
// The value is the track title.
type trackCachedMsg string

// progressMsg carries the current download percentage for the active track.
// The value is a decimal string such as "42.5".
type progressMsg string

// playlistProgressMsg is dispatched whenever yt-dlp reports its position
// within a playlist, e.g. "Downloading item 3 of 12".
type playlistProgressMsg struct{ current, total string }

// downloadFinishedMsg is dispatched when all tracks have been processed.
// The value is a human-readable summary that includes the output folder path.
type downloadFinishedMsg string

// errMsg wraps any error that should terminate the download and display
// a diagnostic message to the user.
type errMsg struct{ err error }
