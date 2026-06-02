package initializer

import (
	"fmt"

	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
)

// initStatic sends the full background image for static theme mode.
func (i *Initializer) initStatic(background device.ImageBackground) error {
	i.log.Info("init: static image mode")

	if _, err := i.sender.Execute(i.cmdPayload.SendStaticBitmap(background)); err != nil {
		return fmt.Errorf("SEND_STATIC_BITMAP failed: %w", err)
	}

	i.log.Info("init: static background sent successfully")
	return nil
}
