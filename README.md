# dub

**A terminal tool for ripping SoundCloud tracks and playlists to MP3.**

> ⚖️ Please read the [Legal Disclaimer](#️-legal-disclaimer) before use.

---

## Overview

dub is a fast, minimal terminal UI for downloading SoundCloud tracks and playlists as high-quality MP3s. Paste a URL, press Enter, and watch it work — no browser extensions, no GUI.

**What it does:**
- Downloads tracks and full playlists
- Converts audio to MP3 at the highest available quality
- Embeds cover art and metadata (artist, title, etc.) into every file
- Saves everything to a timestamped, named folder in your Downloads
- Skips tracks that were already downloaded in a previous session

---

## Requirements

### Go
dub is written in Go. Download it from [go.dev](https://go.dev/dl/) — **version 1.21 or newer** is required.

Verify your installation:
```bash
go version
```

### FFmpeg
FFmpeg converts the downloaded audio to MP3 and embeds cover art.

| Platform | Command |
|----------|---------|
| macOS | `brew install ffmpeg` |
| Linux (Debian/Ubuntu) | `sudo apt install ffmpeg` |
| Windows | `winget install ffmpeg` |

### AtomicParsley *(macOS only)*
Required for embedding cover art on macOS:
```bash
brew install atomicparsley
```

---

## Installation

### Option 1 — Download a pre-built release *(recommended)*

Go to the [Releases page](https://github.com/adammcgrogan/dub/releases) and download the archive for your platform. Each archive contains two files:

| File | Purpose |
|------|---------|
| `dub` / `dub.exe` | The main application |
| `yt-dlp` / `yt-dlp.exe` | The download engine |

> **Important:** Keep both files in the same folder. dub looks for `yt-dlp` next to itself at startup.

### Option 2 — Build from source

```bash
# 1. Clone the repository
git clone https://github.com/adammcgrogan/dub.git
cd dub

# 2. Build the binary
go build -o dub .

# 3. Download yt-dlp and place it in the same folder as your dub binary

# macOS / Linux:
curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o yt-dlp
chmod +x yt-dlp

# Windows — download yt-dlp.exe from:
# https://github.com/yt-dlp/yt-dlp/releases/latest
```

---

## Getting Started

### macOS & Linux

```bash
# First time only — grant execute permissions and remove macOS quarantine
chmod +x dub yt-dlp
xattr -cr .

# Run dub
./dub
```

### Windows

Double-click `dub.exe`.

---

## Usage

When dub starts, you'll see a text prompt:

```
🎧  soundcloud-ripper
──────────────────────────────────────────────────────────────

Paste a SoundCloud track or playlist URL:

╭──────────────────────────────────────────────────────────────╮
│ › _                                                          │
╰──────────────────────────────────────────────────────────────╯

enter download  · ? help  · esc quit
```

1. **Paste a URL** from SoundCloud — either a single track or a full playlist
2. **Press Enter** — dub fetches metadata, then starts downloading
3. **Watch the table** — each track appears with a live progress bar as it downloads
4. **Done** — your MP3s are waiting in `~/Downloads/scdown_[date]_[playlist-name]/`

### Keybindings

| Key | Action |
|-----|--------|
| `enter` | Start the download |
| `?` | Show / hide the keybinding panel |
| `esc` or `ctrl+c` | Quit, or cancel a running download |

### Notes

- Query strings are automatically stripped from URLs (e.g. `?si=...` added by the SoundCloud app)
- Tracks that were already downloaded in a previous session are shown as `◎ Cached` and skipped
- If some tracks in a playlist are unavailable (deleted, geo-blocked, or private), dub skips them and continues with the rest

---

## Output

Files are saved to your `~/Downloads` folder in a timestamped subfolder named after the playlist or track:

```
~/Downloads/
└── scdown_2026-04-30_15-04-05_My_Playlist/
    ├── Track One.mp3
    ├── Track Two.mp3
    └── Track Three.mp3
```

Each file includes embedded cover art and metadata.

---

## ⚖️ Legal Disclaimer

**This tool is intended solely for personal use with audio content you legally own or have explicit permission to download.**

- Do **not** use dub to download copyrighted material without the authorisation of the rights holder.
- Downloading copyrighted content without permission may violate copyright law in your country and SoundCloud's [Terms of Use](https://soundcloud.com/terms-of-use).
- The developer of dub does not condone, endorse, or accept any responsibility for unlawful use of this software.
- **By using this software, you assume full legal responsibility for ensuring your use complies with all applicable laws and the terms of service of any platform you access.**

