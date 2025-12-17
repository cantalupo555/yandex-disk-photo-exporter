# Yandex Disk Photo Exporter

A Go-based automation tool to bulk download photos from Yandex Disk, organized by date.

## Features

- 📸 Automatically downloads all photos from your Yandex Disk
- 📅 Processes photos organized by date
- 🔄 Intelligent scrolling to avoid reprocessing
- 🔐 Uses your existing browser profile (preserves login)
- ⚙️ Configurable batch size and download directory
- 📊 Progress logging with clear status messages

## Prerequisites

- **Go 1.25+** installed
- **Chromium** or **Google Chrome** browser
- Logged into your Yandex account in the browser

## Installation

```bash
# Clone the repository
git clone https://github.com/cantalupo555/yandex-disk-photo-exporter.git
cd yandex-disk-photo-exporter

# Install dependencies
go mod download

# Build the binary
go build -o yandex-disk-photo-exporter
```

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

### Available Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-profile` | `~/snap/chromium/common/chromium` | Path to browser profile directory |
| `-batch` | `10` | Number of dates to process per batch |
| `-exec` | `chromium` | Browser executable path |
| `-download` | `~/Downloads` | Directory to save downloaded files |

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
