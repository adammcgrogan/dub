package ui

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Program is the active Bubble Tea program. It must be set by main immediately
// after tea.NewProgram so that parseYtDlpOutput can forward subprocess output
// as UI messages via Program.Send.
var Program *tea.Program

// Compiled regular expressions for parsing yt-dlp's stdout line by line.
// Pre-compiling them once avoids redundant work inside the hot scan loop.
var (
	// rxPlaylistProgress matches lines like "[download] Downloading item 3 of 12"
	rxPlaylistProgress = regexp.MustCompile(`\[download\] Downloading (?:video|item) (\d+) of (\d+)`)

	// rxDestination matches the line yt-dlp prints when it opens an output file,
	// e.g. "[download] Destination: /Users/.../Track Name.webm"
	rxDestination = regexp.MustCompile(`\[download\] Destination: (.*)`)

	// rxAlreadyDownloaded matches the skip notice yt-dlp prints when a file
	// already exists on disk, e.g. "[download] Track Name.mp3 has already been downloaded"
	rxAlreadyDownloaded = regexp.MustCompile(`\[download\] (.*) has already been downloaded`)

	// rxDownloadPercent matches the periodic progress lines yt-dlp emits,
	// e.g. "[download]  42.5% of ..."
	rxDownloadPercent = regexp.MustCompile(`\[download\]\s+([\d\.]+)%`)
)

// Compiled regular expressions used to sanitise playlist titles for use as
// folder names on the host filesystem.
var (
	// rxUnsafeChars strips any character that is not a word character, a space,
	// or a hyphen — removing punctuation that could cause filesystem issues.
	rxUnsafeChars = regexp.MustCompile(`[^\w\s-]`)

	// rxWhitespaceRuns collapses runs of whitespace or underscores into a single
	// underscore, producing clean snake_case folder names.
	rxWhitespaceRuns = regexp.MustCompile(`[\s_]+`)
)

// downloadTrack returns a Bubble Tea command that runs yt-dlp in the background
// and streams its output back to the UI as a sequence of typed messages.
// The download is associated with ctx so that cancelling the context (via Esc
// or Ctrl-C) sends SIGTERM to the subprocess.
func downloadTrack(ctx context.Context, soundCloudURL string) tea.Cmd {
	return func() tea.Msg {
		downloadDir, ytDlpPath, err := prepareDownloadEnvironment(ctx, soundCloudURL)
		if err != nil {
			return errMsg{err}
		}

		ytDlpCmd := exec.CommandContext(ctx, ytDlpPath,
			"--newline",          // one progress line per yt-dlp output tick
			"--hls-prefer-native", // use Go's native HLS downloader when possible

			// Quality — extract the best available audio and convert to MP3.
			"--format", "bestaudio/best",
			"--extract-audio",
			"--audio-format", "mp3",
			"--audio-quality", "0", // 0 = highest VBR quality

			// Metadata — embed the track's title, artist, and cover art into each file.
			"--embed-metadata",
			"--embed-thumbnail",
			"--convert-thumbnails", "jpg",

			"-o", filepath.Join(downloadDir, "%(title)s.%(ext)s"),
			soundCloudURL,
		)

		stdoutPipe, err := ytDlpCmd.StdoutPipe()
		if err != nil {
			return errMsg{fmt.Errorf("could not connect to yt-dlp output stream: %v", err)}
		}

		// Capture stderr separately so we can surface yt-dlp's error message if
		// the subprocess exits with a non-zero code.
		var errorOutput bytes.Buffer
		ytDlpCmd.Stderr = &errorOutput

		if err := ytDlpCmd.Start(); err != nil {
			return errMsg{fmt.Errorf("could not start yt-dlp — make sure it is in the same folder as this app")}
		}

		// Parse yt-dlp's stdout line by line, forwarding each recognised event to
		// the UI. The returned count tells us how many tracks were processed.
		tracksProcessed := parseYtDlpOutput(bufio.NewScanner(stdoutPipe))

		if err := ytDlpCmd.Wait(); err != nil && tracksProcessed == 0 {
			// yt-dlp failed before downloading anything — surface the real reason.
			detail := strings.TrimSpace(errorOutput.String())
			if detail == "" {
				detail = err.Error()
			}
			return errMsg{fmt.Errorf("yt-dlp reported an error: %s", detail)}
		}

		// If at least one track was processed, report success even if yt-dlp
		// exited with a non-zero code (which happens when some playlist items
		// are unavailable, geo-blocked, or deleted).
		return downloadFinishedMsg(fmt.Sprintf("Tracks saved to: %s", downloadDir))
	}
}

// prepareDownloadEnvironment resolves the yt-dlp binary path, fetches the
// content title for use in the folder name, creates the timestamped output
// directory inside the user's Downloads folder, and returns the directory path
// and yt-dlp binary path.
func prepareDownloadEnvironment(ctx context.Context, soundCloudURL string) (downloadDir, ytDlpPath string, err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("could not locate your home directory: %v", err)
	}

	// yt-dlp must be bundled next to the dub executable.
	selfExecutablePath, err := os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("could not determine the application install path: %v", err)
	}
	ytDlpPath = filepath.Join(filepath.Dir(selfExecutablePath), "yt-dlp")

	contentTitle := fetchContentTitle(ctx, ytDlpPath, soundCloudURL)
	timestamp := time.Now().Format("2006-01-02_15-04-05")

	var folderName string
	if contentTitle != "" {
		folderName = fmt.Sprintf("scdown_%s_%s", timestamp, contentTitle)
	} else {
		folderName = fmt.Sprintf("scdown_%s", timestamp)
	}
	downloadDir = filepath.Join(homeDir, "Downloads", folderName)

	if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("could not create the download folder at %q: %v", downloadDir, err)
	}

	return downloadDir, ytDlpPath, nil
}

// fetchContentTitle asks yt-dlp for the playlist or track title without
// downloading any audio. It is used to build a human-readable folder name.
// If the title cannot be determined for any reason, an empty string is returned
// and the caller falls back to a timestamp-only folder name.
func fetchContentTitle(ctx context.Context, ytDlpPath, soundCloudURL string) string {
	metadataCmd := exec.CommandContext(ctx, ytDlpPath,
		"--flat-playlist",         // enumerate playlist without downloading
		"--no-warnings",           // suppress noisy informational output
		"--playlist-items", "1",   // only look at the first item (fast)
		"--print", "%(playlist_title,title)s", // playlist title, falling back to track title
		soundCloudURL,
	)

	rawOutput, err := metadataCmd.Output()
	if err != nil {
		return ""
	}

	// Output may contain multiple lines if yt-dlp emits warnings despite
	// --no-warnings; we only want the first line.
	title := strings.TrimSpace(strings.SplitN(string(rawOutput), "\n", 2)[0])

	// yt-dlp uses "NA" or "None" when a field has no value.
	if title == "" || title == "NA" || title == "None" {
		return ""
	}

	return sanitizeFolderName(title)
}

// sanitizeFolderName converts a raw playlist or track title into a string
// safe for use as a filesystem folder name. It strips special characters,
// trims whitespace, collapses runs of spaces and underscores into a single
// underscore, and enforces a maximum length of 50 characters.
func sanitizeFolderName(rawTitle string) string {
	cleaned := rxUnsafeChars.ReplaceAllString(rawTitle, "")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = rxWhitespaceRuns.ReplaceAllString(cleaned, "_")
	if len(cleaned) > 50 {
		cleaned = strings.TrimRight(cleaned[:50], "_")
	}
	return cleaned
}

// parseYtDlpOutput reads yt-dlp's combined stdout line by line and dispatches
// typed messages to the Bubble Tea program for each recognised event. It
// returns the total number of tracks seen (new downloads plus cache hits), which
// the caller uses to distinguish a partial success from a total failure.
func parseYtDlpOutput(scanner *bufio.Scanner) int {
	var tracksProcessed int

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case Program == nil:
			// Program hasn't been set yet; skip sending to avoid a nil panic.

		case rxPlaylistProgress.MatchString(line):
			match := rxPlaylistProgress.FindStringSubmatch(line)
			Program.Send(playlistProgressMsg{current: match[1], total: match[2]})

		case rxDestination.MatchString(line):
			match := rxDestination.FindStringSubmatch(line)
			trackTitle := strings.TrimSuffix(filepath.Base(match[1]), filepath.Ext(match[1]))
			Program.Send(trackNameMsg(trackTitle))
			tracksProcessed++

		case rxAlreadyDownloaded.MatchString(line):
			match := rxAlreadyDownloaded.FindStringSubmatch(line)
			trackTitle := strings.TrimSuffix(filepath.Base(match[1]), filepath.Ext(match[1]))
			Program.Send(trackCachedMsg(trackTitle))
			tracksProcessed++

		case rxDownloadPercent.MatchString(line):
			match := rxDownloadPercent.FindStringSubmatch(line)
			Program.Send(progressMsg(match[1]))
		}
	}

	return tracksProcessed
}
