# 🎬 GoStreamHLS

> A pure Go RTMP to HLS streaming server with native Windows GUI

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?style=flat&logo=windows)](https://www.microsoft.com/windows)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## ✨ Features

- **Pure Go** - No FFmpeg or external dependencies required
- **Native Windows GUI** - Built with Gio for a modern, responsive interface
- **Multi-stream support** - Handle multiple RTMP streams simultaneously
- **Real-time monitoring** - Track streams, bitrate, and system resources
- **H.264 + AAC** - Full support for video and audio transmuxing
- **Config persistence** - Save your port settings across restarts

## 📸 Screenshot

![GoStreamHLS Interface](screenshot.png)

## 🚀 Quick Start

1. **Download** the latest release or build from source
2. **Run** `rtmp_server.exe`
3. **Configure** your RTMP and HTTP ports
4. **Start** the server
5. **Stream** from OBS/other software to: `rtmp://localhost:1935/live/{stream_key}`
6. **Play** the HLS stream: `http://localhost:8080/live/{stream_key}/index.m3u8`

## 🛠️ Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/gostreamhls.git
cd gostreamhls

# Download dependencies
go mod tidy

# Build (with GUI, no console)
go build -ldflags "-H windowsgui" -o rtmp_server.exe .

# Build (with console for debugging)
go build -o rtmp_server_debug.exe .
```

## 📦 Project Structure

```
gostreamhls/
├── main.go                 # Entry point
├── gui/
│   ├── app.go              # Main GUI application
│   ├── dashboard.go        # Stream dashboard panel
│   └── logs.go             # Log viewer panel
├── server/
│   ├── rtmp.go             # RTMP server using gortmplib
│   ├── hls.go              # HLS HTTP server
│   └── manager.go          # Multi-stream manager
└── internal/
    ├── config/             # Configuration persistence
    ├── logger/             # Thread-safe log buffer
    └── monitor/            # System resource monitoring
```

## ⚙️ Configuration

Settings are automatically saved to `config.json`:

```json
{
  "http_port": "8080",
  "rtmp_port": "1935"
}
```

## 🎥 OBS Settings

1. Go to **Settings** → **Stream**
2. Set Service to **Custom**
3. Server: `rtmp://localhost:1935/live`
4. Stream Key: Choose any name (e.g., `mystream`)

## 📡 Playback

Use any HLS-compatible player:
- **VLC**: Open Network Stream → `http://localhost:8080/live/mystream/index.m3u8`
- **Web**: Use hls.js or Video.js with the HLS URL

## 🔧 Technical Details

- **RTMP Handling**: [gortmplib](https://github.com/bluenviron/gortmplib)
- **HLS Muxing**: [gohlslib](https://github.com/bluenviron/gohlslib)
- **GUI Framework**: [Gio](https://gioui.org/)
- **Codec Support**: H.264 video, AAC audio (transmux only, no transcoding)

## 📝 License

MIT License - feel free to use in your projects!

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

---

Made with ❤️ in Go
