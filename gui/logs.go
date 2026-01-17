package gui

import (
	"image"
	"image/color"
	"time"

	"rtmp_server/internal/logger"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Colors for log levels
var (
	colorInfo  = color.NRGBA{R: 130, G: 200, B: 255, A: 255} // Light blue
	colorWarn  = color.NRGBA{R: 255, G: 200, B: 100, A: 255} // Orange
	colorError = color.NRGBA{R: 255, G: 100, B: 100, A: 255} // Red
	colorTime  = color.NRGBA{R: 150, G: 150, B: 150, A: 255} // Gray
)

// LogPanel displays real-time logs
type LogPanel struct {
	list widget.List
}

// NewLogPanel creates a new log panel
func NewLogPanel() *LogPanel {
	return &LogPanel{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

// Layout draws the log panel
func (lp *LogPanel) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	entries := logger.GetLogs()

	// Container with dark background
	return layout.Stack{}.Layout(gtx,
		// Background
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			bounds := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
			paint.FillShape(gtx.Ops, color.NRGBA{R: 15, G: 15, B: 25, A: 255},
				clip.Rect(bounds).Op())
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		// Content
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				if len(entries) == 0 {
					label := material.Body2(th, "No logs yet...")
					label.Color = colorTime
					return label.Layout(gtx)
				}

				return material.List(th, &lp.list).Layout(gtx, len(entries), func(gtx layout.Context, i int) layout.Dimensions {
					entry := entries[i]
					return lp.layoutEntry(gtx, th, entry)
				})
			})
		}),
	)
}

func (lp *LogPanel) layoutEntry(gtx layout.Context, th *material.Theme, entry logger.Entry) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEnd}.Layout(gtx,
			// Timestamp
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				timeStr := entry.Time.Format("15:04:05")
				label := material.Body2(th, timeStr)
				label.Color = colorTime
				label.Font.Weight = font.Medium
				label.TextSize = unit.Sp(12)
				return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, label.Layout)
			}),
			// Level
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				levelStr := entry.Level.String()
				label := material.Body2(th, levelStr)
				label.Font.Weight = font.Bold
				label.TextSize = unit.Sp(12)

				switch entry.Level {
				case logger.LevelWarn:
					label.Color = colorWarn
				case logger.LevelError:
					label.Color = colorError
				default:
					label.Color = colorInfo
				}

				return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, label.Layout)
			}),
			// Message
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				label := material.Body2(th, entry.Message)
				label.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
				label.TextSize = unit.Sp(12)
				label.MaxLines = 2
				return label.Layout(gtx)
			}),
		)
	})
}

// ScrollToBottom scrolls the log list to the bottom
func (lp *LogPanel) ScrollToBottom() {
	entries := logger.GetLogs()
	if len(entries) > 0 {
		lp.list.Position.First = len(entries) - 1
	}
}

// LayoutWithTitle draws the log panel with a title header
func (lp *LogPanel) LayoutWithTitle(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Title bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					bounds := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(28)))
					paint.FillShape(gtx.Ops, color.NRGBA{R: 30, G: 30, B: 45, A: 255},
						clip.Rect(bounds).Op())
					return layout.Dimensions{Size: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(28))}}
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(12), Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						label := material.Body2(th, "📋 Logs")
						label.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						label.Font.Weight = font.SemiBold
						label.Alignment = text.Start
						return label.Layout(gtx)
					})
				}),
			)
		}),
		// Log content
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return lp.Layout(gtx, th)
		}),
	)
}

// FormatTimestamp formats a time for log display
func FormatTimestamp(t time.Time) string {
	return t.Format("15:04:05")
}
