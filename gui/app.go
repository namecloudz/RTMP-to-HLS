package gui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"time"

	"rtmp_server/internal/config"
	"rtmp_server/internal/logger"
	"rtmp_server/internal/monitor"
	"rtmp_server/server"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Premium dark theme colors
var (
	bgColor         = color.NRGBA{R: 15, G: 15, B: 25, A: 255}    // Deep navy
	cardColor       = color.NRGBA{R: 25, G: 28, B: 42, A: 255}    // Card bg
	cardBorderColor = color.NRGBA{R: 45, G: 50, B: 70, A: 255}    // Subtle border
	accentColor     = color.NRGBA{R: 88, G: 166, B: 255, A: 255}  // Blue accent
	successColor    = color.NRGBA{R: 72, G: 207, B: 133, A: 255}  // Green
	dangerColor     = color.NRGBA{R: 255, G: 107, B: 107, A: 255} // Red
	warningColor    = color.NRGBA{R: 255, G: 193, B: 7, A: 255}   // Yellow
	textColor       = color.NRGBA{R: 240, G: 242, B: 250, A: 255} // Light text
	textMuted       = color.NRGBA{R: 130, G: 140, B: 165, A: 255} // Muted
	inputBgColor    = color.NRGBA{R: 35, G: 40, B: 58, A: 255}    // Input bg
)

// App is the main application
type App struct {
	window    *app.Window
	theme     *material.Theme
	manager   *server.Manager
	rtmp      *server.RTMPServer
	http      *server.HTTPServer
	dashboard *Dashboard
	logPanel  *LogPanel

	// Widgets
	startBtn      widget.Clickable
	mainList      widget.List
	httpPortInput widget.Editor
	rtmpPortInput widget.Editor

	// SSL widgets
	sslToggle     widget.Bool
	domainInput   widget.Editor
	certPathInput widget.Editor
	keyPathInput  widget.Editor

	// State
	running    bool
	rtmpAddr   string
	httpAddr   string
	sslEnabled bool
}

// NewApp creates a new application
func NewApp() *App {
	a := &App{
		window:   new(app.Window),
		theme:    material.NewTheme(),
		logPanel: NewLogPanel(),
		mainList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}

	// Load saved config
	cfg := config.Load()

	// Initialize port inputs with saved values
	a.httpPortInput.SetText(cfg.HTTPPort)
	a.httpPortInput.SingleLine = true

	a.rtmpPortInput.SetText(cfg.RTMPPort)
	a.rtmpPortInput.SingleLine = true

	// Initialize SSL settings
	a.sslToggle.Value = cfg.SSLEnabled
	a.sslEnabled = cfg.SSLEnabled
	a.domainInput.SetText(cfg.SSLDomain)
	a.domainInput.SingleLine = true
	a.certPathInput.SetText(cfg.SSLCert)
	a.certPathInput.SingleLine = true
	a.keyPathInput.SetText(cfg.SSLKey)
	a.keyPathInput.SingleLine = true

	// Configure theme
	a.theme.Palette.Bg = bgColor
	a.theme.Palette.Fg = textColor

	return a
}

// Run starts the application
func (a *App) Run() error {
	a.window.Option(
		app.Title("🎬 Name RTMP to HLS Streaming Server"),
		app.Size(unit.Dp(1000), unit.Dp(700)),
	)

	// Start refresh ticker
	go a.refreshLoop()

	return a.eventLoop()
}

func (a *App) refreshLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		monitor.UpdateStats()
		a.window.Invalidate()
	}
}

func (a *App) eventLoop() error {
	var ops op.Ops

	for {
		switch e := a.window.Event().(type) {
		case app.DestroyEvent:
			a.stop()
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			a.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (a *App) layout(gtx layout.Context) layout.Dimensions {
	// Fill background with gradient-like color
	paint.Fill(gtx.Ops, bgColor)

	return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(gtx,
			// Header with branding
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutHeader(gtx)
			}),
			// Spacer
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			// Status cards row
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutStatusCards(gtx)
			}),
			// Spacer
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			// Config and controls
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutConfigSection(gtx)
			}),
			// Spacer
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			// SSL Config section
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutSSLSection(gtx)
			}),
			// Spacer
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			// Streams and logs
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return a.layoutMainContent(gtx)
			}),
		)
	})
}

func (a *App) layoutHeader(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
		// Left: Logo and title
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := material.H4(a.theme, "🎬 RTMP → HLS")
					label.Color = textColor
					label.Font.Weight = font.Bold
					return label.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := material.Body2(a.theme, "Streaming Server")
					label.Color = textMuted
					return label.Layout(gtx)
				}),
			)
		}),
		// Right: System stats
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			stats := monitor.GetStats()
			statsText := fmt.Sprintf("💾 %.1f MB  |  🔄 %d goroutines  |  ⏱ %s",
				stats.MemAllocMB, stats.NumGoroutines, monitor.FormatUptime(stats.Uptime))
			label := material.Caption(a.theme, statsText)
			label.Color = textMuted
			return label.Layout(gtx)
		}),
	)
}

func (a *App) layoutStatusCards(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly}.Layout(gtx,
		// Server Status Card
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			status := "🔴 Offline"
			statusColor := dangerColor
			if a.running {
				status = "🟢 Online"
				statusColor = successColor
			}
			return a.layoutCard(gtx, "Server Status", status, statusColor)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
		// Active Streams Card
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			count := "0"
			if a.manager != nil {
				count = fmt.Sprintf("%d", a.manager.StreamCount())
			}
			return a.layoutCard(gtx, "Active Streams", count+" stream(s)", accentColor)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
		// Uptime Card
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			uptime := "—"
			if a.running {
				stats := monitor.GetStats()
				uptime = monitor.FormatUptime(stats.Uptime)
			}
			return a.layoutCard(gtx, "Server Uptime", uptime, warningColor)
		}),
	)
}

func (a *App) layoutCard(gtx layout.Context, title, value string, valueColor color.NRGBA) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		// Card background
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			bounds := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(85)))
			rr := gtx.Dp(unit.Dp(12))
			paint.FillShape(gtx.Ops, cardColor, clip.UniformRRect(bounds, rr).Op(gtx.Ops))
			return layout.Dimensions{Size: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(85))}}
		}),
		// Card content
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.Caption(a.theme, title)
						label.Color = textMuted
						return label.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.H6(a.theme, value)
						label.Color = valueColor
						label.Font.Weight = font.SemiBold
						return label.Layout(gtx)
					}),
				)
			})
		}),
	)
}

func (a *App) layoutConfigSection(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		// Background
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			bounds := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(80)))
			rr := gtx.Dp(unit.Dp(12))
			paint.FillShape(gtx.Ops, cardColor, clip.UniformRRect(bounds, rr).Op(gtx.Ops))
			return layout.Dimensions{Size: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(80))}}
		}),
		// Content
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
					// Port inputs
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.layoutPortInput(gtx, "RTMP Port", &a.rtmpPortInput, !a.running)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(32)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.layoutPortInput(gtx, "HTTP Port", &a.httpPortInput, !a.running)
							}),
						)
					}),
					// Start/Stop button
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutStartButton(gtx)
					}),
				)
			})
		}),
	)
}

func (a *App) layoutSSLSection(gtx layout.Context) layout.Dimensions {
	// Update sslEnabled from toggle
	if a.sslToggle.Update(gtx) {
		a.sslEnabled = a.sslToggle.Value
	}

	return layout.Stack{}.Layout(gtx,
		// Background
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			bounds := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(70)))
			rr := gtx.Dp(unit.Dp(12))
			paint.FillShape(gtx.Ops, cardColor, clip.UniformRRect(bounds, rr).Op(gtx.Ops))
			return layout.Dimensions{Size: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(70))}}
		}),
		// Content
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceEvenly}.Layout(gtx,
					// SSL Toggle
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								icon := "🔓"
								if a.sslEnabled {
									icon = "🔒"
								}
								label := material.Body2(a.theme, icon+" HTTPS")
								label.Color = textMuted
								return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, label.Layout)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								sw := material.Switch(a.theme, &a.sslToggle, "Enable SSL")
								if a.running {
									return layout.Dimensions{}
								}
								return sw.Layout(gtx)
							}),
						)
					}),
					// Domain input
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutSSLInput(gtx, "Domain", &a.domainInput, "example.com", !a.running && a.sslEnabled)
					}),
					// Cert path
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutSSLInput(gtx, "Cert", &a.certPathInput, "cert.pem", !a.running && a.sslEnabled)
					}),
					// Key path
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutSSLInput(gtx, "Key", &a.keyPathInput, "key.pem", !a.running && a.sslEnabled)
					}),
				)
			})
		}),
	)
}

func (a *App) layoutSSLInput(gtx layout.Context, label string, editor *widget.Editor, hint string, enabled bool) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Caption(a.theme, label)
			lbl.Color = textMuted
			return layout.Inset{Right: unit.Dp(6)}.Layout(gtx, lbl.Layout)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					bgCol := inputBgColor
					if !enabled {
						bgCol = color.NRGBA{R: 25, G: 28, B: 40, A: 255}
					}
					bounds := image.Rect(0, 0, gtx.Dp(unit.Dp(120)), gtx.Dp(unit.Dp(32)))
					rr := gtx.Dp(unit.Dp(6))
					paint.FillShape(gtx.Ops, bgCol, clip.UniformRRect(bounds, rr).Op(gtx.Ops))
					return layout.Dimensions{Size: image.Point{X: gtx.Dp(unit.Dp(120)), Y: gtx.Dp(unit.Dp(32))}}
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(104))
						ed := material.Editor(a.theme, editor, hint)
						if enabled {
							ed.Color = textColor
						} else {
							ed.Color = textMuted
						}
						ed.HintColor = textMuted
						return ed.Layout(gtx)
					})
				}),
			)
		}),
	)
}

func (a *App) layoutPortInput(gtx layout.Context, label string, editor *widget.Editor, enabled bool) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(a.theme, label)
			lbl.Color = textMuted
			return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, lbl.Layout)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					bgCol := inputBgColor
					if !enabled {
						bgCol = color.NRGBA{R: 30, G: 32, B: 45, A: 255}
					}
					bounds := image.Rect(0, 0, gtx.Dp(unit.Dp(90)), gtx.Dp(unit.Dp(40)))
					rr := gtx.Dp(unit.Dp(8))
					paint.FillShape(gtx.Ops, bgCol, clip.UniformRRect(bounds, rr).Op(gtx.Ops))
					return layout.Dimensions{Size: image.Point{X: gtx.Dp(unit.Dp(90)), Y: gtx.Dp(unit.Dp(40))}}
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
						ed := material.Editor(a.theme, editor, "")
						ed.Color = textColor
						ed.HintColor = textMuted
						return ed.Layout(gtx)
					})
				}),
			)
		}),
	)
}

func (a *App) layoutStartButton(gtx layout.Context) layout.Dimensions {
	// Handle click
	if a.startBtn.Clicked(gtx) {
		if a.running {
			a.stop()
		} else {
			a.start()
		}
	}

	btnText := "▶  Start Server"
	btnColor := successColor
	if a.running {
		btnText = "⏹  Stop Server"
		btnColor = dangerColor
	}

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			bounds := image.Rect(0, 0, gtx.Dp(unit.Dp(150)), gtx.Dp(unit.Dp(45)))
			rr := gtx.Dp(unit.Dp(8))
			paint.FillShape(gtx.Ops, btnColor, clip.UniformRRect(bounds, rr).Op(gtx.Ops))
			return layout.Dimensions{Size: image.Point{X: gtx.Dp(unit.Dp(150)), Y: gtx.Dp(unit.Dp(45))}}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min = image.Point{X: gtx.Dp(unit.Dp(150)), Y: gtx.Dp(unit.Dp(45))}
				return a.startBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						label := material.Body1(a.theme, btnText)
						label.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
						label.Font.Weight = font.SemiBold
						return label.Layout(gtx)
					})
				})
			})
		}),
	)
}

func (a *App) layoutMainContent(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly}.Layout(gtx,
		// Left: Active Streams
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.layoutStreamsPanel(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
		// Right: Logs
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.layoutLogsPanel(gtx)
		}),
	)
}

func (a *App) layoutStreamsPanel(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		// Background
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			rr := gtx.Dp(unit.Dp(12))
			paint.FillShape(gtx.Ops, cardColor, clip.UniformRRect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y), rr).Op(gtx.Ops))
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		// Content
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					// Title
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.H6(a.theme, "📺 Active Streams")
						label.Color = textColor
						label.Font.Weight = font.Medium
						return label.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					// Streams list or placeholder
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if a.dashboard != nil {
							return a.dashboard.Layout(gtx, a.theme)
						}
						// Placeholder
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							msg := "Start server to see active streams"
							if a.running && a.manager != nil && a.manager.StreamCount() == 0 {
								msg = "Waiting for RTMP streams...\n\nStream URL: rtmp://localhost" + a.rtmpAddr + "/live/{key}"
							}
							label := material.Body2(a.theme, msg)
							label.Color = textMuted
							return label.Layout(gtx)
						})
					}),
				)
			})
		}),
	)
}

func (a *App) layoutLogsPanel(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		// Background
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			rr := gtx.Dp(unit.Dp(12))
			paint.FillShape(gtx.Ops, cardColor, clip.UniformRRect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y), rr).Op(gtx.Ops))
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		// Content
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					// Title
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.H6(a.theme, "📋 Server Logs")
						label.Color = textColor
						label.Font.Weight = font.Medium
						return label.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					// Logs content
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.logPanel.Layout(gtx, a.theme)
					}),
				)
			})
		}),
	)
}

func (a *App) start() {
	if a.running {
		return
	}

	// Get ports from inputs
	httpPort := strings.TrimSpace(a.httpPortInput.Text())
	rtmpPort := strings.TrimSpace(a.rtmpPortInput.Text())

	if httpPort == "" {
		httpPort = "8080"
	}
	if rtmpPort == "" {
		rtmpPort = "1935"
	}

	// Get SSL settings
	sslDomain := strings.TrimSpace(a.domainInput.Text())
	sslCert := strings.TrimSpace(a.certPathInput.Text())
	sslKey := strings.TrimSpace(a.keyPathInput.Text())

	if sslCert == "" {
		sslCert = "cert.pem"
	}
	if sslKey == "" {
		sslKey = "key.pem"
	}

	a.rtmpAddr = ":" + rtmpPort
	a.httpAddr = "0.0.0.0:" + httpPort

	// Save config for next time (including SSL settings)
	config.Save(config.Config{
		HTTPPort:   httpPort,
		RTMPPort:   rtmpPort,
		SSLEnabled: a.sslEnabled,
		SSLDomain:  sslDomain,
		SSLCert:    sslCert,
		SSLKey:     sslKey,
	})

	// Create new servers with configured ports
	a.manager = server.NewManager("./hls")
	a.rtmp = server.NewRTMPServer(a.rtmpAddr, a.manager)
	a.http = server.NewHTTPServer(a.httpAddr, a.manager)

	// Set dashboard display URL
	displayHost := "localhost:" + httpPort
	if sslDomain != "" && a.sslEnabled {
		displayHost = sslDomain
	}
	a.dashboard = NewDashboard(a.manager, displayHost)

	logger.Info("Starting streaming server...")

	if err := a.rtmp.Start(); err != nil {
		logger.Error("Failed to start RTMP server: %v", err)
		return
	}

	// Start HTTP server with or without SSL
	var httpErr error
	if a.sslEnabled {
		httpErr = a.http.StartWithTLS(sslCert, sslKey)
	} else {
		httpErr = a.http.Start()
	}

	if httpErr != nil {
		logger.Error("Failed to start HTTP server: %v", httpErr)
		a.rtmp.Stop()
		return
	}

	a.running = true
	logger.Info("✅ Server started successfully")
	logger.Info("📡 RTMP URL: rtmp://localhost%s/live/{stream_key}", a.rtmpAddr)
	if a.sslEnabled {
		logger.Info("🔒 HLS URL:  https://%s/live/{stream_key}/index.m3u8", displayHost)
	} else {
		logger.Info("🎬 HLS URL:  http://localhost:%s/live/{stream_key}/index.m3u8", httpPort)
	}
}

func (a *App) stop() {
	if !a.running {
		return
	}

	logger.Info("Stopping server...")
	a.http.Stop()
	a.rtmp.Stop()
	a.running = false
	a.dashboard = nil
	logger.Info("⏹  Server stopped")
}

// Main entry point
func Main() {
	go func() {
		a := NewApp()
		if err := a.Run(); err != nil {
			logger.Error("Application error: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}
