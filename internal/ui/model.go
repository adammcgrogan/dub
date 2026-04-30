package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Key bindings ──────────────────────────────────────────────────────────────

// Individual key bindings. Each binding declares both the physical key(s) and
// the help text shown in the footer bar.
var (
	keyDownload = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "download"))
	keyHelp     = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help"))
	keyQuit     = key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "quit"))
	keyCancel   = key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel"))
)

// inputKeyMap lists the keys that are active on the URL input screen.
type inputKeyMap struct{}

func (inputKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keyDownload, keyHelp, keyQuit}
}
func (inputKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{(inputKeyMap{}).ShortHelp()}
}

// downloadingKeyMap lists the keys that are active during an active download.
type downloadingKeyMap struct{}

func (downloadingKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keyCancel}
}
func (downloadingKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{(downloadingKeyMap{}).ShortHelp()}
}

// finishedKeyMap lists the keys that are active after a download completes.
type finishedKeyMap struct{}

func (finishedKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keyQuit}
}
func (finishedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{(finishedKeyMap{}).ShortHelp()}
}

// ── Application phase ─────────────────────────────────────────────────────────

// applicationPhase represents which stage of the lifecycle the app is in.
type applicationPhase int

const (
	// phaseAwaitingURL shows the URL input prompt.
	phaseAwaitingURL applicationPhase = iota
	// phaseDownloading shows the live progress table while yt-dlp runs.
	phaseDownloading
	// phaseFinished is set once all tracks have been processed.
	phaseFinished
)

// ── Track ─────────────────────────────────────────────────────────────────────

// track holds the display state for a single audio file in the progress table.
type track struct {
	// title is the human-readable track name shown in the table.
	title string
	// percentComplete is the download progress as a decimal string, e.g. "42.5".
	percentComplete string
	// status is one of "Downloading", "Done", or "Cached".
	status string
	// playlistIndex is this track's position within the playlist, e.g. "3".
	playlistIndex string
	// playlistSize is the total number of tracks in the playlist, e.g. "12".
	playlistSize string
}

// ── Model ─────────────────────────────────────────────────────────────────────

// defaultTerminalWidth is used before the first tea.WindowSizeMsg arrives.
const defaultTerminalWidth = 80

// model is the central state container for the Bubble Tea application.
// It implements tea.Model via Init, Update, and View.
type model struct {
	// phase tracks which stage of the application lifecycle is active.
	phase applicationPhase
	// urlInput is the text field where the user enters a SoundCloud URL.
	urlInput textinput.Model
	// spinner animates while waiting for yt-dlp to produce its first output.
	spinner spinner.Model
	// helpBar renders the keybinding hints at the bottom of the screen.
	helpBar help.Model
	// downloadError holds any fatal error that terminated the download.
	downloadError error
	// tracks is the ordered list of audio files seen in this session.
	tracks []track
	// playlistPosition is the index of the track currently being downloaded.
	playlistPosition string
	// playlistSize is the total number of tracks reported by yt-dlp.
	playlistSize string
	// completionMessage is shown after all downloads finish successfully.
	completionMessage string
	// showingHelp controls whether the keybinding overlay is visible.
	showingHelp bool
	// terminalWidth and terminalHeight are updated on every resize event.
	terminalWidth  int
	terminalHeight int
	// downloadCtx is passed to the yt-dlp subprocess so it can be cancelled.
	downloadCtx context.Context
	// cancelDownload cancels downloadCtx to abort an in-progress download.
	cancelDownload context.CancelFunc
}

// InitialModel constructs the starting model with a focused URL input and a
// ready spinner. ctx and cancelDownload are wired to the yt-dlp subprocess so
// that pressing Esc/Ctrl-C cleanly terminates it.
func InitialModel(downloadCtx context.Context, cancelDownload context.CancelFunc) model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://soundcloud.com/artist/track-or-playlist"
	urlInput.Focus()
	urlInput.PromptStyle = colorDown
	urlInput.TextStyle = lipgloss.NewStyle().Foreground(clrBright)
	urlInput.CursorStyle = colorDown
	urlInput.CharLimit = 1024
	urlInput.Width = defaultTerminalWidth - 8

	spinnerWidget := spinner.New()
	spinnerWidget.Spinner = spinner.MiniDot
	spinnerWidget.Style = colorDown

	helpWidget := help.New()
	helpWidget.Styles = help.Styles{
		ShortKey:       helpKeyStyle,
		ShortDesc:      helpDescStyle,
		ShortSeparator: footerStyle,
		Ellipsis:       footerStyle,
		FullKey:        helpKeyStyle,
		FullDesc:       helpDescStyle,
		FullSeparator:  footerStyle,
	}

	return model{
		phase:          phaseAwaitingURL,
		urlInput:       urlInput,
		spinner:        spinnerWidget,
		helpBar:        helpWidget,
		tracks:         []track{},
		terminalWidth:  defaultTerminalWidth,
		downloadCtx:    downloadCtx,
		cancelDownload: cancelDownload,
	}
}

// ── Init ──────────────────────────────────────────────────────────────────────

// Init starts the cursor blinking in the URL input field.
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// ── Update ────────────────────────────────────────────────────────────────────

// Update is the central message handler. It receives events from Bubble Tea
// (key presses, window resizes) as well as custom messages sent by the yt-dlp
// background worker, and returns the updated model along with any follow-up
// commands.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	// Resize: keep column widths and the input box width in sync with the terminal.
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		m.helpBar.Width = msg.Width
		m.urlInput.Width = max(40, msg.Width-8)
		return m, nil

	case tea.KeyMsg:
		// "?" opens the help overlay from any state; any subsequent key closes it.
		if msg.String() == "?" && !m.showingHelp {
			m.showingHelp = true
			return m, nil
		}
		if m.showingHelp {
			m.showingHelp = false
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.cancelDownload != nil {
				m.cancelDownload()
			}
			return m, tea.Quit
		case tea.KeyEnter:
			return m.handleURLSubmitted()
		}

	// yt-dlp is starting a new track download.
	case trackNameMsg:
		m.markLastTrackComplete()
		m.appendTrack(string(msg), "Downloading", "0.0")
		return m, nil

	// yt-dlp skipped a track that was already present on disk.
	case trackCachedMsg:
		m.markLastTrackComplete()
		m.appendTrack(string(msg), "Cached", "100")
		return m, nil

	// yt-dlp reported a new download percentage for the current track.
	case progressMsg:
		if len(m.tracks) > 0 {
			m.tracks[len(m.tracks)-1].percentComplete = string(msg)
		}
		return m, nil

	// yt-dlp reported its position within the playlist (e.g. 3 of 12).
	case playlistProgressMsg:
		m.playlistPosition = msg.current
		m.playlistSize = msg.total
		return m, nil

	// All tracks have been processed successfully.
	case downloadFinishedMsg:
		m.markLastTrackComplete()
		m.completionMessage = string(msg)
		m.phase = phaseFinished
		return m, tea.Quit

	// A fatal error occurred (only shown when no tracks downloaded at all).
	case errMsg:
		if m.downloadCtx.Err() != nil {
			// The user pressed Esc/Ctrl-C — exit silently.
			return m, tea.Quit
		}
		m.markLastTrackComplete()
		m.downloadError = msg.err
		m.phase = phaseFinished
		return m, tea.Quit
	}

	// Forward remaining messages to the active sub-component.
	if m.phase == phaseAwaitingURL {
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}

	if m.phase == phaseDownloading {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleURLSubmitted is called when the user presses Enter on the input screen.
// It strips query parameters from the URL before passing it to the downloader.
func (m model) handleURLSubmitted() (tea.Model, tea.Cmd) {
	if m.phase != phaseAwaitingURL {
		return m, nil
	}

	rawURL := strings.TrimSpace(m.urlInput.Value())

	// Remove any query string (e.g. ?si=...) that SoundCloud appends when
	// copying a link from the browser — yt-dlp handles the clean URL better.
	if queryStart := strings.Index(rawURL, "?"); queryStart != -1 {
		rawURL = rawURL[:queryStart]
	}

	if rawURL == "" {
		return m, nil
	}

	m.phase = phaseDownloading
	return m, tea.Batch(m.spinner.Tick, downloadTrack(m.downloadCtx, rawURL))
}

// markLastTrackComplete transitions the most recent track from "Downloading"
// to "Done". It is called whenever a new track starts or the session ends, so
// the previous entry always shows its final state.
func (m *model) markLastTrackComplete() {
	if len(m.tracks) == 0 {
		return
	}
	last := &m.tracks[len(m.tracks)-1]
	if last.status == "Downloading" {
		last.status = "Done"
		last.percentComplete = "100"
	}
}

// appendTrack adds a new entry to the progress table. playlistPosition and
// playlistSize are captured from the current model state at the moment the
// track is added, so each row correctly reflects its position in the playlist.
func (m *model) appendTrack(title, status, percentComplete string) {
	m.tracks = append(m.tracks, track{
		title:           title,
		percentComplete: percentComplete,
		status:          status,
		playlistIndex:   m.playlistPosition,
		playlistSize:    m.playlistSize,
	})
}

// ── View ──────────────────────────────────────────────────────────────────────

// View renders the full terminal UI from the current model state.
// It is called by Bubble Tea after every Update.
func (m model) View() string {
	header := m.renderHeader()
	footer := m.renderFooter()

	var body string
	switch {
	case m.showingHelp:
		body = m.renderHelpOverlay()

	case m.downloadError != nil && len(m.tracks) == 0:
		// Fatal error with no successful downloads — show only the error.
		body = "\n" + errorStyle.Render("  Error: "+m.downloadError.Error()) + "\n"

	case m.phase == phaseAwaitingURL:
		body = m.renderURLInput()

	case m.phase == phaseDownloading && len(m.tracks) == 0:
		// yt-dlp has started but hasn't announced any tracks yet.
		body = m.renderFetchingState()

	default:
		body = "\n" + m.renderProgressTable() + "\n"
		if m.downloadError != nil {
			// Some tracks failed but others succeeded — show table plus warning.
			body += errorStyle.Render("  Some tracks could not be downloaded.") + "\n"
		} else if m.phase == phaseFinished {
			body += m.renderCompletionBanner()
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// renderHeader renders the application title and a full-width separator line.
func (m model) renderHeader() string {
	title := titleStyle.Render("🎧  dub")
	tagline := dimStyle.Render("A terminal tool for ripping SoundCloud tracks and playlists to MP3.")
	separator := sepStyle.Render(strings.Repeat("─", max(22, m.terminalWidth-1)))
	return lipgloss.JoinVertical(lipgloss.Left, title, tagline, separator)
}

// renderURLInput renders the labelled text input where the user pastes a URL.
func (m model) renderURLInput() string {
	label := inputLabelStyle.Render("Paste a SoundCloud track or playlist URL:")
	inputBox := inputBoxStyle.Render(m.urlInput.View())
	return lipgloss.JoinVertical(lipgloss.Left, "\n"+label, "", inputBox)
}

// renderFetchingState renders a spinner with a message while dub waits for
// yt-dlp to announce the first track. This avoids a blank screen on playlists
// that take a moment to enumerate.
func (m model) renderFetchingState() string {
	message := normalStyle.Render("Fetching track info & metadata…")
	return fmt.Sprintf("\n  %s  %s\n", m.spinner.View(), message)
}

// renderProgressTable renders the bordered table of tracks, their statuses,
// and their download progress. Column widths are calculated from the current
// terminal width so the layout stays correct after a resize.
func (m model) renderProgressTable() string {
	const statusColWidth, progressColWidth = 26, 16
	// Subtract 4 to account for the rounded border (1 each side) and
	// the padding (1 each side) applied by the table border style.
	nameColWidth := max(20, m.terminalWidth-4-statusColWidth-progressColWidth)

	statusCol   := lipgloss.NewStyle().Width(statusColWidth)
	nameCol     := lipgloss.NewStyle().Width(nameColWidth)
	progressCol := lipgloss.NewStyle().Width(progressColWidth)

	tableHeader := headerStyle.Render(
		statusCol.Render("STATUS") +
			nameCol.Render("TRACK NAME") +
			progressCol.Render("PROGRESS"),
	)
	divider := dimStyle.Render(strings.Repeat("─", statusColWidth+nameColWidth+progressColWidth))

	rows := []string{tableHeader, divider}
	for _, t := range m.tracks {
		rows = append(rows, m.renderTrackRow(t, statusCol, nameCol, progressCol))
	}

	borderStyle := tableBorderStyle
	if m.phase == phaseDownloading {
		borderStyle = tableBorderActiveStyle
	}
	return borderStyle.Render(strings.Join(rows, "\n"))
}

// renderTrackRow renders a single row in the progress table.
// colum styles are passed in so the caller controls the layout widths.
func (m model) renderTrackRow(t track, statusCol, nameCol, progressCol lipgloss.Style) string {
	playlistBadge := ""
	if t.playlistSize != "" {
		playlistBadge = dimStyle.Render(fmt.Sprintf(" (%s/%s)", t.playlistIndex, t.playlistSize))
	}

	var statusText string
	switch t.status {
	case "Downloading":
		statusText = colorDown.Render(m.spinner.View()+" Downloading") + playlistBadge
	case "Done":
		statusText = successStyle.Render("✔ Done") + playlistBadge
	case "Cached":
		statusText = colorCached.Render("◎ Cached") + playlistBadge
	}

	// Truncate long track names to fit within the name column.
	displayTitle := t.title
	maxRunes := nameCol.GetWidth() - 2
	if runes := []rune(displayTitle); len(runes) > maxRunes {
		displayTitle = string(runes[:maxRunes-1]) + "…"
	}

	return statusCol.Render(statusText) +
		nameCol.Render(displayTitle) +
		progressCol.Render(renderProgressBar(t.percentComplete))
}

// renderCompletionBanner renders the success message and output folder path
// shown after all tracks have finished downloading.
func (m model) renderCompletionBanner() string {
	checkmark := successStyle.Render("✔  All done!")
	outputPath := donePathStyle.Render("   " + m.completionMessage)
	return lipgloss.JoinVertical(lipgloss.Left, checkmark, outputPath, "")
}

// renderHelpOverlay renders a bordered panel listing all keybindings, grouped
// by application phase. It is shown when the user presses "?".
func (m model) renderHelpOverlay() string {
	type binding struct{ key, description string }
	sections := []struct {
		heading  string
		bindings []binding
	}{
		{"Input", []binding{
			{"enter", "start download"},
			{"?", "toggle this panel"},
			{"esc / ctrl+c", "quit"},
		}},
		{"During download", []binding{
			{"esc / ctrl+c", "cancel & quit"},
		}},
	}

	lines := []string{helpTitleStyle.Render("Keybindings"), ""}
	for _, section := range sections {
		lines = append(lines, helpGroupStyle.Render(strings.ToUpper(section.heading)))
		for _, b := range section.bindings {
			keyLabel := helpKeyStyle.Render(fmt.Sprintf("  %-18s", b.key))
			lines = append(lines, keyLabel+helpDescStyle.Render(b.description))
		}
		lines = append(lines, "")
	}
	lines = append(lines, dimStyle.Render("press any key to close"))

	return "\n" + helpOverlayStyle.Render(strings.Join(lines, "\n"))
}

// renderFooter renders the keybinding hint bar at the bottom of the screen.
// The set of keys shown changes depending on the current application phase.
func (m model) renderFooter() string {
	var keymap help.KeyMap
	switch m.phase {
	case phaseAwaitingURL:
		keymap = inputKeyMap{}
	case phaseDownloading:
		keymap = downloadingKeyMap{}
	default:
		keymap = finishedKeyMap{}
	}
	return footerStyle.Render("\n" + m.helpBar.View(keymap))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// progressBarWidth is the number of block characters used to represent
// the filled and empty portions of the inline progress bar.
const progressBarWidth = 8

// renderProgressBar returns a fixed-width string combining a Unicode block
// bar (e.g. "████░░░░") with a right-aligned percentage label (e.g. " 42%").
// percentStr is expected to be a decimal string as produced by yt-dlp.
func renderProgressBar(percentStr string) string {
	percentage, _ := strconv.ParseFloat(percentStr, 64)

	filledBlocks := int(percentage / 100 * progressBarWidth)
	if filledBlocks > progressBarWidth {
		filledBlocks = progressBarWidth
	}
	emptyBlocks := progressBarWidth - filledBlocks

	bar := barFilledStyle.Render(strings.Repeat("█", filledBlocks)) +
		barEmptyStyle.Render(strings.Repeat("░", emptyBlocks))

	return bar + " " + barPctStyle.Render(fmt.Sprintf("%3.0f%%", percentage))
}
