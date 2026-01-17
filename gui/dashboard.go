package gui

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"rtmp_server/server"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// Colors
var (
	colorLive    = color.NRGBA{R: 50, G: 205, B: 50, A: 255}   // Green
	colorOffline = color.NRGBA{R: 128, G: 128, B: 128, A: 255} // Gray
	colorCard    = color.NRGBA{R: 35, G: 35, B: 55, A: 255}    // Dark card bg
	colorText    = color.NRGBA{R: 230, G: 230, B: 230, A: 255} // Light text
	colorSubtext = color.NRGBA{R: 150, G: 150, B: 170, A: 255} // Muted text
	colorAccent  = color.NRGBA{R: 100, G: 150, B: 255, A: 255} // Blue accent
)

// Dashboard displays stream status
type Dashboard struct {
	manager     *server.Manager
	httpAddr    string
	lastRefresh time.Time
}

// NewDashboard creates a new dashboard
func NewDashboard(manager *server.Manager, httpAddr string) *Dashboard {
	return &Dashboard{
		manager:  manager,
		httpAddr: httpAddr,
	}
}

// Layout draws the dashboard
func (d *Dashboard) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	streams := d.manager.GetAllStreams()

	if len(streams) == 0 {
		return d.layoutEmpty(gtx, th)
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Title
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.H6(th, fmt.Sprintf("📡 Active Streams (%d)", len(streams)))
				label.Color = colorText
				return label.Layout(gtx)
			})
		}),
		// Stream cards
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				childrenFromStreams(gtx, th, streams, d.httpAddr)...,
			)
		}),
	)
}

func childrenFromStreams(gtx layout.Context, th *material.Theme, streams []server.StreamInfo, httpAddr string) []layout.FlexChild {
	children := make([]layout.FlexChild, len(streams))
	for i, stream := range streams {
		s := stream // Capture
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layoutStreamCard(gtx, th, s, httpAddr)
		})
	}
	return children
}

func layoutStreamCard(gtx layout.Context, th *material.Theme, stream server.StreamInfo, httpAddr string) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Card background
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				bounds := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(80)))
				rr := gtx.Dp(unit.Dp(8))
				paint.FillShape(gtx.Ops, colorCard,
					clip.UniformRRect(bounds, rr).Op(gtx.Ops))
				return layout.Dimensions{Size: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(80))}}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
						// Left side: status and name
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								// Stream name with status indicator
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										// Status dot
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											size := gtx.Dp(unit.Dp(10))
											bounds := image.Rect(0, 0, size, size)
											statusColor := colorLive
											if !stream.Active {
												statusColor = colorOffline
											}
											paint.FillShape(gtx.Ops, statusColor,
												clip.Ellipse(bounds).Op(gtx.Ops))
											return layout.Dimensions{Size: image.Point{X: size, Y: size}}
										}),
										layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
										// Stream key
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											label := material.Body1(th, stream.Key)
											label.Color = colorText
											label.Font.Weight = font.SemiBold
											return label.Layout(gtx)
										}),
									)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
								// Duration
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									duration := time.Since(stream.StartTime)
									label := material.Body2(th, "⏱ "+server.FormatDuration(duration))
									label.Color = colorSubtext
									return label.Layout(gtx)
								}),
							)
						}),
						// Right side: bitrate and URL
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Alignment: layout.End}.Layout(gtx,
								// Bitrate
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									label := material.Body1(th, "📊 "+server.FormatBitrate(stream.Bitrate))
									label.Color = colorAccent
									label.Font.Weight = font.Medium
									return label.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
								// HLS URL
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									url := fmt.Sprintf("http://%s/live/%s/index.m3u8", httpAddr, stream.Key)
									label := material.Caption(th, url)
									label.Color = colorSubtext
									return label.Layout(gtx)
								}),
							)
						}),
					)
				})
			}),
		)
	})
}

func (d *Dashboard) layoutEmpty(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := material.H5(th, "📺 No Active Streams")
				label.Color = colorSubtext
				return label.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := material.Body2(th, "Push an RTMP stream to start broadcasting")
				label.Color = colorSubtext
				return label.Layout(gtx)
			}),
		)
	})
}
