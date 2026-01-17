# 🎬 GoStreamHLS

> A pure Go RTMP to HLS streaming server with native Windows GUI

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?style=flat&logo=windows)](https://www.microsoft.com/windows)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## ✨ Features

- **Pure Go** - No FFmpeg or external dependencies required
- **Native Windows GUI** - Modern dark theme with Gio framework
- **Multi-stream support** - Handle multiple RTMP streams simultaneously
- **SSL/HTTPS** - Built-in TLS 1.2+ support with toggle
- **Real-time monitoring** - Track streams, bitrate, and system resources
- **H.264 + AAC** - Full support for video and audio transmuxing
- **Config persistence** - Save your settings across restarts
- **CORS enabled** - Ready for web player integration

## 📸 Screenshot

![GoStreamHLS Interface](screenshot.png)

## 🚀 Quick Start

1. **Download** the latest release or build from source
2. **Run** `rtmp_server.exe`
3. **Configure** your RTMP and HTTP ports
4. **Start** the server
5. **Stream** from OBS to: `rtmp://localhost:1935/live/{stream_key}`
6. **Play** the HLS stream: `http://localhost:8080/live/{stream_key}/index.m3u8`

## 🔒 SSL/HTTPS Setup

1. Place your SSL certificate files (`cert.pem`, `key.pem`) in the app directory
2. Enable the **HTTPS toggle** in the GUI
3. Enter your **domain** name
4. Click **Start Server**
5. Access via: `https://yourdomain.com/live/{stream_key}/index.m3u8`

## 🛠️ Build from Source

```bash
# Clone the repository
git clone https://github.com/namecloudz/RTMP-to-HLS.git
cd RTMP-to-HLS

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
│   ├── rtmp.go             # RTMP server (gortmplib)
│   ├── hls.go              # HTTP/HTTPS HLS server
│   └── manager.go          # Multi-stream manager
└── internal/
    ├── config/             # Configuration persistence
    ├── logger/             # Thread-safe log buffer
    └── monitor/            # System resource monitoring
```

## ⚙️ Configuration

Settings are saved to `config.json`:

```json
{
  "http_port": "8080",
  "rtmp_port": "1935",
  "ssl_enabled": false,
  "ssl_domain": "",
  "ssl_cert": "cert.pem",
  "ssl_key": "key.pem"
}
```

## 🎥 OBS Settings

1. Go to **Settings** → **Stream**
2. Set Service to **Custom**
3. Server: `rtmp://localhost:1935/live`
4. Stream Key: Choose any name (e.g., `mystream`)

**Recommended Output Settings:**
- Video Encoder: x264 or NVENC
- Audio: AAC, 48kHz, Stereo
- Keyframe Interval: 2 seconds

## 📡 Playback

**VLC:**
```
Media → Open Network Stream → http://localhost:8080/live/mystream/index.m3u8
```

**Web (hls.js):**
```html
<script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
<video id="video" controls></video>
<script>
  var video = document.getElementById('video');
  var hls = new Hls();
  hls.loadSource('http://localhost:8080/live/mystream/index.m3u8');
  hls.attachMedia(video);
</script>
```

## 🔧 API Endpoints

| Endpoint | Description |
|----------|-------------|
| `/live/{key}/index.m3u8` | HLS playlist |
| `/live/{key}/*.ts` | Media segments |
| `/api/streams` | JSON list of active streams |
| `/health` | Health check |

## 🔧 Technical Details

- **RTMP Handling**: [gortmplib](https://github.com/bluenviron/gortmplib)
- **HLS Muxing**: [gohlslib](https://github.com/bluenviron/gohlslib)
- **GUI Framework**: [Gio](https://gioui.org/)
- **Codec Support**: H.264 video, AAC audio (transmux only)

## 📝 License

MIT License - feel free to use in your projects!

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

---

Made with ❤️ in Go
