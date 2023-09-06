package sensors

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	edevice "github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/shirou/gopsutil/v3/net"
	"time"
)

type NetStat struct {
	log     *logger.Logger
	jobs    chan<- command.Command
	builder *local.Builder
	p       *command.UpdatePayload
	names   edevice.Net
	wifi    lastValues
	wired   lastValues
}

type lastValues struct {
	sent uint64
	recv uint64
}

func NewDNetStat(l *logger.Logger, j chan<- command.Command, b *local.Builder, p *command.UpdatePayload, m edevice.Net) *NetStat {
	return &NetStat{
		log:     l.With("runner", "net_stats"),
		jobs:    j,
		builder: b,
		p:       p,
		names:   m,
		wifi: lastValues{
			sent: 0,
			recv: 0,
		},
		wired: lastValues{
			sent: 0,
			recv: 0,
		},
	}
}

func (g *NetStat) RunNetStat(ctx context.Context, e *theme.Network) error {
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	err := g.getNetStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("Stopping RunNetStat")
			return ctx.Err()
		case <-ticker.C:

		}
		err := g.getNetStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *NetStat) getNetStat(ctx context.Context, e *theme.Network) error {
	var payloads []*command.UpdatePayload

	netIos, err := net.IOCountersWithContext(ctx, true)
	if err != nil {
		return err
	}

	for _, netIo := range netIos {
		if e.Wired != nil {
			if netIo.Name == g.names.Wired {
				btr := netIo.BytesRecv
				bts := netIo.BytesSent
				recvtx := btr - g.wired.recv
				senttx := bts - g.wired.sent

				if e.Wired.Download != nil && e.Wired.Download.Text.Show {
					v := (recvtx / uint64(e.Interval.Seconds())) * 8
					if recvtx == 0 {
						v = recvtx
					}
					img, x, y := BuildTextUint(g.builder, v, utils.Bits, e.Wired.Download.Text)
					payloads = append(payloads, g.p.SendPayload(img, x, y))
				}
				if e.Wired.Downloaded != nil && e.Wired.Downloaded.Text.Show {
					img, x, y := BuildTextUint(g.builder, btr, utils.IBytes, e.Wired.Downloaded.Text)
					payloads = append(payloads, g.p.SendPayload(img, x, y))
				}
				if e.Wired.Upload != nil && e.Wired.Upload.Text.Show {
					v := (senttx / uint64(e.Interval.Seconds())) * 8
					if senttx == 0 {
						v = recvtx
					}
					img, x, y := BuildTextUint(g.builder, v, utils.Bits, e.Wired.Upload.Text)
					payloads = append(payloads, g.p.SendPayload(img, x, y))
				}
				if e.Wired.Uploaded != nil && e.Wired.Uploaded.Text.Show {
					img, x, y := BuildTextUint(g.builder, bts, utils.IBytes, e.Wired.Uploaded.Text)
					payloads = append(payloads, g.p.SendPayload(img, x, y))
				}
				g.wired.recv = btr
				g.wired.sent = bts
			}
		}
		if e.Wifi != nil {
			if netIo.Name == g.names.Wifi {
				btr := netIo.BytesRecv
				bts := netIo.BytesSent
				recvtx := btr - g.wifi.recv
				senttx := bts - g.wifi.sent

				if e.Wifi.Download != nil && e.Wifi.Download.Text.Show {
					v := (recvtx / uint64(e.Interval.Seconds())) * 8
					if recvtx == 0 {
						v = recvtx
					}
					img, x, y := BuildTextUint(g.builder, v, utils.Bits, e.Wifi.Download.Text)
					payloads = append(payloads, g.p.SendPayload(img, x, y))
				}
				if e.Wifi.Downloaded != nil && e.Wifi.Downloaded.Text.Show {
					img, x, y := BuildTextUint(g.builder, btr, utils.IBytes, e.Wifi.Downloaded.Text)
					payloads = append(payloads, g.p.SendPayload(img, x, y))
				}
				if e.Wifi.Upload != nil && e.Wifi.Upload.Text.Show {
					v := (senttx / uint64(e.Interval.Seconds())) * 8
					if senttx == 0 {
						v = recvtx
					}
					img, x, y := BuildTextUint(g.builder, v, utils.Bits, e.Wifi.Upload.Text)
					payloads = append(payloads, g.p.SendPayload(img, x, y))
				}
				if e.Wifi.Uploaded != nil && e.Wifi.Uploaded.Text.Show {
					img, x, y := BuildTextUint(g.builder, bts, utils.IBytes, e.Wifi.Uploaded.Text)
					payloads = append(payloads, g.p.SendPayload(img, x, y))
				}
				g.wifi.recv = btr
				g.wifi.sent = bts
			}
		}
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("Stopping getNetStat")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}
