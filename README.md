# Yandex Disk Photo Exporter

A Go-based automation tool to bulk download photos from Yandex Disk, organized by date.

## Features

- 📸 Automatically downloads all photos from your Yandex Disk
- 📅 Processes photos organized by date
- 🗓️ **Filter by date range** - download only photos from a specific period
- 🔄 Intelligent scrolling to avoid reprocessing
- 🔐 Uses your existing browser profile (preserves login)
- ⚙️ Configurable batch size and download directory
- 📊 Progress logging with clear status messages

## Prerequisites

- **Chromium-based browser** (Chrome, Chromium, Edge, Vivaldi, Opera, or Brave)
- Logged into your Yandex account in the browser

## Installation

### Option 1: Download (Recommended)

Download the latest release for your platform from the [Releases](https://github.com/cantalupo555/yandex-disk-photo-exporter/releases) page.

| Platform | Architecture | File |
|----------|--------------|------|
| **Linux** | x64 (Intel/AMD) | `.deb`, `.rpm`, or binary |
| **Linux** | ARM64 | `.deb`, `.rpm`, or binary |
| **Windows** | x64 (Intel/AMD) | `.exe` |
| **Windows** | ARM64 | `.exe` |
| **macOS** | Intel | `.zip` |
| **macOS** | Apple Silicon (M1/M2/M3) | `.zip` |

#### Linux (Debian/Ubuntu)
```bash
# Download and install .deb package
sudo dpkg -i yandex-disk-photo-exporter_*_linux_amd64.deb
```

#### Linux (Fedora/RHEL)
```bash
# Download and install .rpm package
sudo rpm -i yandex-disk-photo-exporter_*_linux_amd64.rpm
```

#### Windows
1. Download the `.exe` file
2. Run directly or add to PATH

#### macOS
```bash
# Extract and run
unzip yandex-disk-photo-exporter_*_macOS_arm64.zip
./yandex-disk-photo-exporter
```

### Option 2: Build from Source

Requires **Go 1.25+** installed.

```bash
# Clone the repository
git clone https://github.com/cantalupo555/yandex-disk-photo-exporter.git
cd yandex-disk-photo-exporter

# Install dependencies
go mod download

# Build the binary
go build -o yandex-disk-photo-exporter
```

## Browser Compatibility

This tool works with any **Chromium-based browser**. The application automatically detects installed browsers in the following priority order:

| Priority | Browser | Notes |
|----------|---------|-------|
| 1st | **Google Chrome** | Recommended for best compatibility |
| 2nd | **Chromium** | Open-source alternative |
| 3rd | **Microsoft Edge** | Chromium-based (Windows/macOS/Linux) |
| 4th | **Vivaldi** | Power-user browser |
| 5th | **Opera** | Feature-rich browser |
| 6th | **Brave** | Privacy-focused browser |

### Auto-Detection

The tool automatically finds your browser. No configuration needed in most cases:

```bash
# Just run - browser is auto-detected
./yandex-disk-photo-exporter
# Output: ✓ Auto-detected browser: /usr/bin/google-chrome
```

### Using a Specific Browser

If you have multiple browsers or want to use a specific one:

```bash
# Use Brave on Linux
./yandex-disk-photo-exporter -exec /usr/bin/brave-browser

# Use Vivaldi on Windows
./yandex-disk-photo-exporter -exec "C:\Users\YourUser\AppData\Local\Vivaldi\Application\vivaldi.exe"

# Use Opera on macOS
./yandex-disk-photo-exporter -exec "/Applications/Opera.app/Contents/MacOS/Opera"
```

> **Note:** Any Chromium-based browser should work. If your browser isn't auto-detected, use the `-exec` flag with the full path.

## Usage

### Basic Usage

```bash
# Run with default settings
./yandex-disk-photo-exporter
```

### With Custom Options

```bash
# Specify custom download directory
./yandex-disk-photo-exporter -download ~/Pictures/YandexPhotos

# Use Google Chrome instead of Chromium
./yandex-disk-photo-exporter -exec google-chrome

# Custom browser profile path
./yandex-disk-photo-exporter -profile ~/.config/google-chrome

# Combine options
./yandex-disk-photo-exporter \
  -exec google-chrome \
  -profile ~/.config/google-chrome \
  -download ~/Pictures/YandexBackup \
  -batch 20
```

### Download Photos from a Specific Date Range

```bash
# Download photos from 2023 only
./yandex-disk-photo-exporter --from 2023-01-01 --to 2023-12-31

# Download photos from a specific month
./yandex-disk-photo-exporter --from 2024-06-01 --to 2024-06-30

# Download photos from a start date until today
./yandex-disk-photo-exporter --from 2024-01-01

# Download photos up until a specific date
./yandex-disk-photo-exporter --to 2023-12-31
```

### Available Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-profile` | OS-specific* | Path to browser profile directory |
| `-batch` | `10` | Number of dates to process per batch |
| `-exec` | Auto-detect | Browser executable path (auto-detected if not specified) |
| `-download` | `~/Downloads` | Directory to save downloaded files |
| `-from` | - | Start date for filtering (format: `YYYY-MM-DD`) |
| `-to` | - | End date for filtering (format: `YYYY-MM-DD`) |
| `-version` | - | Show version and exit |

*Default profile paths by OS:
- **Linux:** `~/snap/chromium/common/chromium` or `~/.config/chromium`
- **macOS:** `~/Library/Application Support/yandex-exporter-profile`
- **Windows:** `~\.yandex-exporter-profile`

## How It Works

1. **Opens the browser** with your existing profile (to use saved login)
2. **Navigates** to Yandex Disk Photos page
3. **Detects login status** - waits 60 seconds if login is required
4. **For each date group visible**:
   - Hovers to reveal the checkbox
   - Selects all photos for that date
   - Clicks the Download button
   - Deselects and scrolls to the next group
5. **Repeats** until no more photos are found

## Important Notes

⚠️ **Before running:**
- Make sure you're logged into Yandex Disk in your browser
- Close any existing browser windows using the same profile
- Ensure sufficient disk space for downloads

⚠️ **During execution:**
- Don't interact with the browser window
- The script handles scrolling and clicking automatically
- Press `Ctrl+C` to stop at any time

## Troubleshooting

### Login required message appears
The script detected you're not logged in. Log in manually within the 60-second window.

### Browser doesn't open
- Check if the browser executable path is correct
- Try specifying the full path: `-exec /usr/bin/chromium-browser`

### Downloads not appearing
- Verify the download directory exists
- Check browser download settings
- Some files may take time to download (large archives)

### Script stops unexpectedly
- Check if Yandex Disk page layout changed
- Ensure stable internet connection
- Try increasing wait times by modifying source

## Development

```bash
# Run directly with Go
go run main.go

# Run with flags
go run main.go -download ~/Pictures/Test -batch 5
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Disclaimer

This tool is for personal use to backup your own photos. Please respect Yandex's Terms of Service and use responsibly.
