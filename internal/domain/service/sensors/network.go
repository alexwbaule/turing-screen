package sensors

import (
	"context"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	edevice "github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
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
		log:     l,
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

	err := g.getNetStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			//g.log.Infof("Stopping RunNetStat job...")
			return context.Canceled
		}
		err := g.getNetStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *NetStat) getNetStat(ctx context.Context, e *theme.Network) error {
	//g.log.Debugf("Network: [%#v]", e)
	select {
	case <-ctx.Done():
		//g.log.Infof("Stopping getNetStat job...")
		return context.Canceled
	default:
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
						text := e.Wired.Download.Text
						//g.log.Debugf("Text: [%#v]", text)

						v := (recvtx / uint64(e.Interval.Seconds())) * 8
						if recvtx == 0 {
							v = recvtx
						}
						value := fmt.Sprintf("%s/s", utils.Bits(v))
						//g.log.Infof("NetIo Wired Download: %s", value)
						img := g.builder.DrawText(value, text)
						imgUpdt := device.NewImageProcess(img)
						g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
					}
					if e.Wired.Downloaded != nil && e.Wired.Downloaded.Text.Show {
						text := e.Wired.Downloaded.Text
						//g.log.Debugf("Text: [%#v]", text)

						value := fmt.Sprintf("%s", utils.IBytes(btr))
						//g.log.Infof("NetIo Wired Downloaded: %s", value)
						img := g.builder.DrawText(value, text)
						imgUpdt := device.NewImageProcess(img)
						g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
					}
					if e.Wired.Upload != nil && e.Wired.Upload.Text.Show {
						text := e.Wired.Upload.Text
						//g.log.Debugf("Text: [%#v]", text)

						v := (senttx / uint64(e.Interval.Seconds())) * 8
						if senttx == 0 {
							v = recvtx
						}
						value := fmt.Sprintf("%s/s", utils.Bits(v))
						//g.log.Infof("NetIo Wired Upload: %s", value)
						img := g.builder.DrawText(value, text)
						imgUpdt := device.NewImageProcess(img)
						g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
					}
					if e.Wired.Uploaded != nil && e.Wired.Uploaded.Text.Show {
						text := e.Wired.Uploaded.Text
						//g.log.Debugf("Text: [%#v]", text)

						value := fmt.Sprintf("%s", utils.IBytes(bts))
						//g.log.Infof("NetIo Wired Uploaded: %s", value)
						img := g.builder.DrawText(value, text)
						imgUpdt := device.NewImageProcess(img)
						g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
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
						text := e.Wifi.Download.Text
						//g.log.Debugf("Text: [%#v]", text)

						v := (recvtx / uint64(e.Interval.Seconds())) * 8
						if recvtx == 0 {
							v = recvtx
						}
						value := fmt.Sprintf("%s/s", utils.Bits(v))
						//g.log.Infof("NetIo Wifi Download: %s", value)
						img := g.builder.DrawText(value, text)
						imgUpdt := device.NewImageProcess(img)
						g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
					}
					if e.Wifi.Downloaded != nil && e.Wifi.Downloaded.Text.Show {
						text := e.Wifi.Downloaded.Text
						//g.log.Debugf("Text: [%#v]", text)

						value := fmt.Sprintf("%s", utils.IBytes(btr))
						//g.log.Infof("NetIo Wifi Downloaded: %s", value)
						img := g.builder.DrawText(value, text)
						imgUpdt := device.NewImageProcess(img)
						g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
					}
					if e.Wifi.Upload != nil && e.Wifi.Upload.Text.Show {
						text := e.Wifi.Upload.Text
						//g.log.Debugf("Text: [%#v]", text)

						v := (senttx / uint64(e.Interval.Seconds())) * 8
						if senttx == 0 {
							v = recvtx
						}
						value := fmt.Sprintf("%s/s", utils.Bits(v))
						//g.log.Infof("NetIo Wifi Upload: %s", value)
						img := g.builder.DrawText(value, text)
						imgUpdt := device.NewImageProcess(img)
						g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
					}
					if e.Wifi.Uploaded != nil && e.Wifi.Uploaded.Text.Show {
						text := e.Wifi.Uploaded.Text
						//g.log.Debugf("Text: [%#v]", text)

						value := fmt.Sprintf("%s", utils.IBytes(bts))
						//g.log.Infof("NetIo Wifi Uploaded: %s", value)
						img := g.builder.DrawText(value, text)
						imgUpdt := device.NewImageProcess(img)
						g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
					}
					g.wifi.recv = btr
					g.wifi.sent = bts
				}
			}
		}
	}
	return nil
}
