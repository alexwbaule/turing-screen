package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application"
	appTheme "github.com/alexwbaule/turing-screen/internal/application/theme"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/domain/service/video"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
)

// =====================================================================
// Test: InitializeVideoOverlay + Draw + RefreshVideoOverlay
// using OUR implementations (serial, command, video overlay)
//
// Flow mirrors turing-test/main.go exactly:
//   1. Open serial
//   2. HELLO
//   3. SetOrientation (LANDSCAPE)
//   4. GetFileSize / Upload if needed
//   5. StopVideo
//   6. Clear display
//   7. StartVideo
//   8. InitializeVideoOverlay
//   9. Loop: Draw text/bars/progress -> RefreshVideoOverlay
// =====================================================================

func main() {
	portPtr := flag.String("port", "AUTO", "Serial port (or AUTO)")
	flag.Parse()

	app := application.NewApplication()

	statsTheme, err := appTheme.NewTheme(app.Config, app.Log)
	if err != nil {
		os.Exit(-1)
	}

	img := statsTheme.GetStaticImages()

	tttt := theme.BackgroundStyle{
		BackgroundColor:     nil,
		BackgroundImagePath: img["background"].Path,
		BackgroundImage:     img["background"].BackgroundImage,
	}

	builder := renderer.NewBuilder(app.Log, app.Config.GetDeviceDisplay(), statsTheme.GetDisplay())

	fmt.Println("=== turing-my: Video Overlay Test using OUR implementations ===")

	s, err := serial.NewSerial(*portPtr, app.Log)
	if err != nil {
		log.Fatalf("Failed to open serial: %v", err)
	}
	defer s.Close()

	exec := func(cmd command.Command) {
		_, err := s.Execute(cmd)
		if err != nil {
			log.Fatalf("Command %s failed: %v", cmd.GetName(), err)
		}
		fmt.Printf("  [OK] %s\n", cmd.GetName())
	}

	// --- Step 1: HELLO ---
	fmt.Println("\n--- HELLO ---")
	cmdDevice := command.NewDevice(app.Log)
	exec(cmdDevice.Hello())

	// --- Step 2: SetOrientation (LANDSCAPE) ---
	fmt.Println("\n--- OPTIONS (LANDSCAPE) ---")
	cmdOption := command.NewOption(app.Log)
	exec(cmdOption.SetOptions(command.Default, command.NoFlip, command.Disabled))

	// --- Step 3: Check video file size / Upload if needed ---
	fmt.Println("\n--- VIDEO FILE CHECK ---")
	cmdStorage := command.NewStorage(app.Log)

	getFileCmd := cmdStorage.GetFileInfo(statsTheme.GetVideoPlay().DevicePath)
	_, err = s.Execute(getFileCmd)
	if err != nil {
		log.Fatalf("GetFileInfo failed: %v", err)
	}
	videoSize := cmdStorage.GetVideoSize()
	fmt.Printf("  Video on device: %d bytes\n", videoSize)

	// --- Step 4: StopVideo ---
	fmt.Println("\n--- STOP_VIDEO ---")
	cmdMedia := command.NewMedia(app.Log)
	exec(cmdMedia.StopVideo())

	// --- Step 5: Clear display ---
	fmt.Println("\n--- CLEAR DISPLAY ---")
	clearImg := image.NewNRGBA(image.Rect(0, 0, 800, 480))
	draw.Draw(clearImg, clearImg.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	clearPayload := video.BuildInitPayload(clearImg, statsTheme.GetDisplay().Orientation)
	clearCmd := command.NewInitVideoOverlay(app.Log, clearPayload)
	exec(clearCmd)

	// --- Step 6: StartVideo ---
	fmt.Println("\n--- START_VIDEO ---")
	exec(cmdMedia.StartVideo(statsTheme.GetVideoPlay().DevicePath))

	// --- Step 7: InitializeVideoOverlay ---
	fmt.Println("\n--- INITIALIZE VIDEO OVERLAY ---")

	var initImg *image.NRGBA
	if statsTheme.GetVideoPlay().ForegroundImage != nil {
		initImg = convertToNRGBA(statsTheme.GetVideoPlay().ForegroundImage)
		fmt.Printf("  Loaded background: %s\n", statsTheme.GetVideoPlay().Path)
	}
	if initImg == nil {
		initImg = image.NewNRGBA(image.Rect(0, 0, 800, 480))
	}

	bgraPayload := video.BuildInitPayload(initImg, statsTheme.GetDisplay().Orientation)

	//display := &device.Display{Width: 480, Height: 800, Reverse: false}
	overlay := video.NewOverlayBuffer(app.Log, app.Config.GetDeviceDisplay(), statsTheme.GetDisplay().Orientation)
	overlay.SetInitial(initImg)

	initCmd := command.NewInitVideoOverlay(app.Log, bgraPayload)
	exec(initCmd)

	// --- Step 8: Initial Refresh (matching reference) ---
	fmt.Println("\n--- INITIAL REFRESH ---")
	refreshCmd := overlay.Refresh()
	exec(refreshCmd)

	time.Sleep(1 * time.Second)

	// Query status
	fmt.Println("\n--- QUERY STATUS ---")
	_, err = s.Execute(cmdMedia.QueryStatus())
	if err != nil {
		fmt.Printf("  QueryStatus warning: %v\n", err)
	}

	txt := buildText(&tttt)
	bar := buildProgressBar(&tttt)
	rbar := buildRadialBar(&tttt)

	fmt.Println("\n=== STARTING MAIN LOOP ===")

	barValue := 0
	for {
		start := time.Now()

		// --- Draw Text with time ---
		tStr := time.Now().Format("15:04:05.000000")
		tdx, tx, ty := builder.DrawText(tStr, txt, 15)
		overlay.Draw(tdx, tx, ty)

		// --- Draw progress bar (yellow, 0-100) ---
		barW := float64(barValue)/100.0*140.0 - 1
		if barW < 0 {
			barW = 0
		}
		bdc := builder.DrawProgressBar(barW, bar)
		overlay.Draw(bdc, bar.X, bar.Y)

		// --- Draw progress bar (green, 0-19) ---
		greenBar := image.NewNRGBA(image.Rect(0, 0, 140, 30))
		val2 := barValue % 20
		barW2 := float64(val2)/19.0*140.0 - 1
		if barW2 < 0 {
			barW2 = 0
		}
		greenColor := color.NRGBA{0, 255, 0, 255}
		for py := 0; py < 30; py++ {
			for px := 0; px < int(barW2); px++ {
				greenBar.SetNRGBA(px, py, greenColor)
			}
		}
		overlay.Draw(greenBar, 160, 40)

		// --- Draw radial progress bar (green, small) ---
		radialBar := builder.DrawRadialProgressBar(float64(barValue), rbar)
		overlay.Draw(radialBar, rbar.X, rbar.Y)

		// --- REFRESH (one single refresh per cycle, matching reference) ---
		refreshCmd := overlay.Refresh()
		_, err := s.Execute(refreshCmd)
		if err != nil {
			log.Fatalf("Refresh failed: %v", err)
		}

		barValue = (barValue + 2) % 101
		elapsed := time.Since(start).Seconds()
		fmt.Printf("refresh done (took %.3f s)\n", elapsed)
		time.Sleep(100 * time.Millisecond)
	}
}

func convertToNRGBA(src image.Image) *image.NRGBA {
	// 1. Create a blank NRGBA image canvas with matching boundaries
	bounds := src.Bounds()
	nrgbaDst := image.NewNRGBA(bounds)

	// 2. Draw the source image onto the new NRGBA destination canvas
	draw.Draw(nrgbaDst, bounds, src, bounds.Min, draw.Src)

	return nrgbaDst
}

func buildText(background *theme.BackgroundStyle) *theme.Text {
	return &theme.Text{
		Show:     true,
		ShowUnit: true,
		Format:   theme.LONG,
		Layout: theme.Layout{
			X: 40,
			Y: 180,
		},
		TextStyle: theme.TextStyle{
			Font:      utils.DefaultFont,
			FontColor: color.White,
			Align:     theme.LEFT,
		},
		BackgroundStyle: theme.BackgroundStyle{
			BackgroundColor:     color.Transparent,
			BackgroundImagePath: background.BackgroundImagePath,
			BackgroundImage:     background.BackgroundImage,
		},
		Size: 20,
	}
}

func buildProgressBar(background *theme.BackgroundStyle) *theme.Graph {
	return &theme.Graph{
		Show: true,
		Layout: theme.Layout{
			X:      10,
			Y:      40,
			Width:  139,
			Height: 29,
		},
		MinValue:   0,
		MaxValue:   100,
		BarColor:   color.White,
		BarOutline: true,
		BackgroundStyle: theme.BackgroundStyle{
			BackgroundImagePath: background.BackgroundImagePath,
			BackgroundImage:     background.BackgroundImage,
		},
	}
}

func buildRadialBar(background *theme.BackgroundStyle) *theme.Radial {
	return &theme.Radial{
		Layout: theme.Layout{
			X:      240,
			Y:      360,
			Width:  200,
			Height: 200,
		},
		BackgroundStyle: theme.BackgroundStyle{
			BackgroundImagePath: background.BackgroundImagePath,
			BackgroundImage:     background.BackgroundImage,
		},
		TextStyle: theme.TextStyle{
			Font:      utils.DefaultFont,
			FontColor: color.White,
			Align:     theme.LEFT,
		},
		Show:       true,
		Radius:     25,
		MinValue:   0,
		MaxValue:   100,
		AngleStart: 0,
		AngleEnd:   180,
		AngleSteps: 5,
		AngleSep:   5,
		Clockwise:  false,
		BarColor:   color.RGBA{0, 128, 255, 255},
		ShowText:   false,
		ShowUnit:   true,
	}
}
