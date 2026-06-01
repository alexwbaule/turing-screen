package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/alexwbaule/gg"
	"github.com/alexwbaule/serial"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// =====================================================================
// Enums from PR #348
// =====================================================================

type Orientation int

const (
	PORTRAIT          Orientation = 0
	REVERSE_PORTRAIT  Orientation = 1
	LANDSCAPE         Orientation = 2
	REVERSE_LANDSCAPE Orientation = 3
)

type Command []byte

var (
	HELLO                   Command = []byte{0x01, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0xc5, 0xd3}
	OPTIONS                 Command = []byte{0x7d, 0xef, 0x69, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x2d}
	SET_BRIGHTNESS          Command = []byte{0x7b, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}
	STOP_VIDEO              Command = []byte{0x79, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01}
	STOP_MEDIA              Command = []byte{0x96, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01}
	QUERY_STATUS            Command = []byte{0xcf, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01}
	START_DISPLAY_BITMAP    Command = []byte{0x2c}
	PRE_UPDATE_BITMAP       Command = []byte{0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01}
	UPDATE_BITMAP           Command = []byte{0xcc, 0xef, 0x69, 0x00}
	START_VIDEO             Command = []byte{0x78, 0xef, 0x69, 0x00, 0x00, 0x00}
	INIT_VIDEO_OVERLAY      Command = []byte{0xd0, 0xef, 0x69, 0x00, 0x00, 0x00}
	LIST_FILES              Command = []byte{0x65, 0xef, 0x69, 0x00, 0x00, 0x00}
	UPLOAD_FILE             Command = []byte{0x6f, 0xef, 0x69, 0x00, 0x00, 0x00}
	DELETE_FILE             Command = []byte{0x66, 0xef, 0x69, 0x00, 0x00, 0x00}
	GET_FILE_SIZE           Command = []byte{0x6e, 0xef, 0x69, 0x00, 0x00, 0x00}
	DISPLAY_BITMAP          Command = []byte{0xc8, 0xef, 0x69, 0x00, 0x17, 0x70}
	DISPLAY_BITMAP_ON_VIDEO Command = []byte{0xca, 0xef, 0x69, 0x00, 0x17, 0x70}
	SEND_PAYLOAD            Command = []byte{0xFF}
	STARTMODE_DEFAULT       Command = []byte{0x00}
	NO_FLIP                 Command = []byte{0x00}
	FLIP_180                Command = []byte{0x01}
)

func cmdName(c Command) string {
	if bytes.Equal(c, SEND_PAYLOAD) {
		return "SEND_PAYLOAD"
	}
	if bytes.Equal(c, HELLO) {
		return "HELLO"
	}
	if bytes.Equal(c, OPTIONS) {
		return "OPTIONS"
	}
	if bytes.Equal(c, SET_BRIGHTNESS) {
		return "SET_BRIGHTNESS"
	}
	if bytes.Equal(c, STOP_VIDEO) {
		return "STOP_VIDEO"
	}
	if bytes.Equal(c, QUERY_STATUS) {
		return "QUERY_STATUS"
	}
	if bytes.Equal(c, START_DISPLAY_BITMAP) {
		return "START_DISPLAY_BITMAP"
	}
	if bytes.Equal(c, PRE_UPDATE_BITMAP) {
		return "PRE_UPDATE_BITMAP"
	}
	if bytes.Equal(c, UPDATE_BITMAP) {
		return "UPDATE_BITMAP"
	}
	if bytes.Equal(c, START_VIDEO) {
		return "START_VIDEO"
	}
	if bytes.Equal(c, INIT_VIDEO_OVERLAY) {
		return "INIT_VIDEO_OVERLAY"
	}
	if bytes.Equal(c, LIST_FILES) {
		return "LIST_FILES"
	}
	if bytes.Equal(c, UPLOAD_FILE) {
		return "UPLOAD_FILE"
	}
	if bytes.Equal(c, GET_FILE_SIZE) {
		return "GET_FILE_SIZE"
	}
	if bytes.Equal(c, DISPLAY_BITMAP) {
		return "DISPLAY_BITMAP"
	}
	if bytes.Equal(c, DISPLAY_BITMAP_ON_VIDEO) {
		return "DISPLAY_BITMAP_ON_VIDEO"
	}
	return "UNKNOWN"
}

type Padding []byte

var (
	PADDING_NULL                 Padding = []byte{0x00}
	PADDING_START_DISPLAY_BITMAP Padding = []byte{0x2c}
)

type SleepInterval []byte

var SLEEP_INTERVAL_OFF SleepInterval = []byte{0x00}

var CountStart = 0

// =====================================================================
// Configuration
// =====================================================================

var (
	COM_PORT          = "AUTO"
	VIDEO_DEVICE_PATH = "/root/video/earth.mp4"
	VIDEO_LOCAL_PATH  = "FromApp/5inchENG/TURZX-V3.1.0-5inchENG/video/800480/earth.mp4"
	DISPLAY_WIDTH     = 480
	DISPLAY_HEIGHT    = 800
)

// =====================================================================
// LcdCommRevC — exact replica from PR #348
// =====================================================================

type LcdCommRevC struct {
	ComPort              string
	DisplayWidth         int
	DisplayHeight        int
	Orientation          Orientation
	LcdSerial            *serial.Port
	VideoOverlay         *image.NRGBA
	PreviousVideoOverlay *image.NRGBA
	fontCache            map[string]font.Face
	imageCache           map[string]image.Image
}

func NewLcdCommRevC(comPort string, displayWidth int, displayHeight int) *LcdCommRevC {
	lcd := &LcdCommRevC{
		ComPort:       comPort,
		DisplayWidth:  displayWidth,
		DisplayHeight: displayHeight,
		Orientation:   LANDSCAPE,
		fontCache:     make(map[string]font.Face),
		imageCache:    make(map[string]image.Image),
	}
	lcd.openSerial()
	return lcd
}

func (l *LcdCommRevC) getWidth() int {
	if l.Orientation == PORTRAIT || l.Orientation == REVERSE_PORTRAIT {
		return l.DisplayWidth
	}
	return l.DisplayHeight
}

func (l *LcdCommRevC) getHeight() int {
	if l.Orientation == PORTRAIT || l.Orientation == REVERSE_PORTRAIT {
		return l.DisplayHeight
	}
	return l.DisplayWidth
}

// ---- Serial (from lcd_comm.py) ----

func (l *LcdCommRevC) openSerial() {
	if l.ComPort == "AUTO" {
		l.ComPort = autoDetectComPort()
		if l.ComPort == "" {
			fmt.Println("Cannot find COM port automatically")
			os.Exit(1)
		}
	}
	fmt.Printf("Opening serial port %s\n", l.ComPort)
	initial := true
	mode := &serial.Config{
		Name:        l.ComPort,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		InitialDTR:  &initial,
		InitialRTS:  &initial,
		ReadTimeout: 1 * time.Second,
	}
	port, err := serial.OpenPort(mode)
	if err != nil {
		log.Fatalf("Failed to open port: %v", err)
	}
	// Python's timeout=1
	l.LcdSerial = port
}

func (l *LcdCommRevC) closeSerial() {
	if l.LcdSerial != nil {
		l.LcdSerial.Close()
	}
}

func (l *LcdCommRevC) WriteData(byteBuffer []byte) {
	l.LcdSerial.Write(byteBuffer)
}

func (l *LcdCommRevC) ReadData(readSize int) []byte {
	buf := make([]byte, readSize)
	n, _ := l.LcdSerial.Read(buf)
	return buf[:n]
}

func autoDetectComPort() string {
	ports, err := serial.GetDetailedPortsList()
	if err == nil {
		for _, port := range ports {
			if port.SerialNumber == "20080411" {
				return port.Name
			} else if port.SerialNumber == "USB7INCH" {
				return port.Name
			}
		}
	}
	return ""
}

// ---- _send_command — verbatim from PR #348 ----

func (l *LcdCommRevC) sendCommand(cmd Command, payload []byte, padding Padding, bypassQueue bool, readsize int) []byte {
	var message []byte
	if !bytes.Equal(cmd, SEND_PAYLOAD) {
		message = append(message, cmd...)
	}

	if padding == nil {
		padding = PADDING_NULL
	}

	if payload != nil {
		message = append(message, payload...)
	}

	msgSize := float64(len(message))
	if math.Mod(msgSize/250.0, 1.0) != 0 {
		padSize := int(250.0*math.Ceil(msgSize/250.0) - msgSize)
		padBytes := make([]byte, padSize)
		for i := 0; i < padSize; i++ {
			padBytes[i] = padding[0]
		}
		message = append(message, padBytes...)
	}

	previewLen := len(message)
	if previewLen > 20 {
		previewLen = 20
	}
	hexPreview := hex.EncodeToString(message[:previewLen])
	name := cmdName(cmd)
	fmt.Printf("  TX [%s] %d bytes: %s...\n", name, len(message), hexPreview)

	l.WriteData(message)
	if readsize > 0 {
		resp := l.ReadData(readsize)
		text := strings.TrimRight(string(resp), "\x00")
		fmt.Printf("  RX [%s] %d bytes: \"%s\"\n", name, len(resp), text)
		return resp
	}
	return nil
}

// ---- _hello — verbatim from PR #348 ----

func (l *LcdCommRevC) hello() {
	l.sendCommand(HELLO, nil, nil, true, 0)
	resp := l.ReadData(23)
	l.LcdSerial.ResetInputBuffer()
	fmt.Printf("  HELLO response: %v\n", resp)
}

func (l *LcdCommRevC) InitializeComm() {
	l.hello()
}

// ---- SetOrientation — verbatim from PR #348 ----

func (l *LcdCommRevC) SetOrientation(orientation Orientation) {
	l.Orientation = orientation
	b := make([]byte, 0)
	b = append(b, STARTMODE_DEFAULT...)
	b = append(b, PADDING_NULL...)
	b = append(b, NO_FLIP...)
	b = append(b, SLEEP_INTERVAL_OFF...)
	l.sendCommand(OPTIONS, b, nil, true, 0)

	names := []string{"PORTRAIT", "REVERSE_PORTRAIT", "LANDSCAPE", "REVERSE_LANDSCAPE"}
	fmt.Printf("  SetOrientation: %s (width=%d, height=%d)\n", names[orientation], l.getWidth(), l.getHeight())
}

// ---- SetBrightness — verbatim from PR #348 ----

func (l *LcdCommRevC) SetBrightness(level float64) {
	convertedLevel := byte((level / 100.0) * 255.0)
	l.sendCommand(SET_BRIGHTNESS, []byte{convertedLevel}, nil, true, 0)
}

// ---- StopVideo ----

func (l *LcdCommRevC) StopVideo() {
	l.sendCommand(STOP_VIDEO, nil, nil, false, 0)
}

// ---- _generate_full_image — verbatim from PR #348 ----

func generateFullImage(img image.Image, orientation Orientation) []byte {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	var rotated *image.NRGBA
	switch orientation {
	case PORTRAIT:
		rotated = image.NewNRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rotated.Set(y, w-1-x, img.At(x, y))
			}
		}
	case REVERSE_PORTRAIT:
		rotated = image.NewNRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rotated.Set(h-1-y, x, img.At(x, y))
			}
		}
	case REVERSE_LANDSCAPE:
		rotated = image.NewNRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rotated.Set(w-1-x, h-1-y, img.At(x, y))
			}
		}
	default:
		rotated = image.NewNRGBA(bounds)
		draw.Draw(rotated, bounds, img, bounds.Min, draw.Src)
	}

	w, h = rotated.Bounds().Dx(), rotated.Bounds().Dy()
	var hexData []byte
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := rotated.NRGBAAt(x, y)
			hexData = append(hexData, c.B, c.G, c.R, c.A)
		}
	}

	var result []byte
	for i := 0; i < len(hexData); i += 249 {
		end := i + 249
		if end > len(hexData) {
			end = len(hexData)
		}
		if i > 0 {
			result = append(result, 0x00)
		}
		result = append(result, hexData[i:end]...)
	}
	return result
}

// ---- DisplayPILImage — verbatim from PR #348 ----

func (l *LcdCommRevC) DisplayPILImage(img image.Image, x, y int) {
	if x == 0 && y == 0 && img.Bounds().Dx() == l.getWidth() && img.Bounds().Dy() == l.getHeight() {
		l.sendCommand(PRE_UPDATE_BITMAP, nil, nil, false, 0)
		l.sendCommand(START_DISPLAY_BITMAP, nil, PADDING_START_DISPLAY_BITMAP, false, 0)
		l.sendCommand(DISPLAY_BITMAP, nil, nil, false, 0)
		l.sendCommand(SEND_PAYLOAD, generateFullImage(img, l.Orientation), nil, false, 1024)
		l.sendCommand(QUERY_STATUS, nil, nil, false, 1024)
	}
}

// ---- Clear — verbatim from PR #348 ----

func (l *LcdCommRevC) Clear(bg image.Image) {
	backupOrientation := l.Orientation
	l.SetOrientation(LANDSCAPE)
	blank := image.NewNRGBA(image.Rect(0, 0, l.getWidth(), l.getHeight()))
	draw.Draw(blank, blank.Bounds(), bg, image.Point{}, draw.Src)
	l.DisplayPILImage(blank, 0, 0)
	l.SetOrientation(backupOrientation)
}

// ---- File operations — verbatim from PR #348 ----

func (l *LcdCommRevC) GetFileSize(filePath string) int {
	pyd := []byte{byte(len(filePath))}
	pyd = append(pyd, PADDING_NULL[0], PADDING_NULL[0], PADDING_NULL[0])
	pyd = append(pyd, []byte(filePath)...)
	l.sendCommand(GET_FILE_SIZE, pyd, nil, true, 0)
	reply := l.ReadData(1024)
	replyStr := strings.Trim(string(reply), "\x00")
	var size int
	fmt.Sscanf(replyStr, "%d", &size)
	return size
}

func (l *LcdCommRevC) readInChunks(file *os.File, chunkSize int) <-chan []byte {
	ch := make(chan []byte)
	go func() {
		defer close(ch)
		for {
			buf := make([]byte, chunkSize)
			n, err := file.Read(buf)
			if n > 0 {
				ch <- buf[:n]
			}
			if err != nil {
				break
			}
		}
	}()
	return ch
}

func (l *LcdCommRevC) UploadFile(localPath string, destinationPath string) {
	pyd := []byte{byte(len(destinationPath))}
	pyd = append(pyd, PADDING_NULL[0], PADDING_NULL[0], PADDING_NULL[0])
	pyd = append(pyd, []byte(destinationPath)...)

	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return
	}
	fileSizeBytes := int32(fileInfo.Size())

	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, uint32(fileSizeBytes))
	pyd = append(pyd, sizeBuf...)

	l.sendCommand(UPLOAD_FILE, pyd, nil, false, 1024)

	videoFile, err := os.Open(localPath)
	if err != nil {
		return
	}
	defer videoFile.Close()

	for packet := range l.readInChunks(videoFile, 249) {
		l.sendCommand(SEND_PAYLOAD, packet, nil, false, 0)
	}

	time.Sleep(1 * time.Second)
	l.LcdSerial.ResetInputBuffer()
}

func (l *LcdCommRevC) ListFiles(dirPath string) string {
	pyd := []byte{byte(len(dirPath))}
	pyd = append(pyd, PADDING_NULL[0], PADDING_NULL[0], PADDING_NULL[0])
	pyd = append(pyd, []byte(dirPath)...)
	l.sendCommand(LIST_FILES, pyd, nil, true, 0)
	reply := l.ReadData(10240)
	return strings.Trim(string(reply), "\x00")
}

// ---- StartVideo — verbatim from PR #348 ----

func (l *LcdCommRevC) StartVideo(videoPath string) {
	pyd := []byte{byte(len(videoPath))}
	pyd = append(pyd, PADDING_NULL[0], PADDING_NULL[0], PADDING_NULL[0])
	pyd = append(pyd, []byte(videoPath)...)
	l.sendCommand(START_VIDEO, pyd, nil, false, 0)
	fmt.Printf("  StartVideo: %s\n", videoPath)
}

// ---- InitializeVideoOverlay — verbatim from PR #348 ----

func (l *LcdCommRevC) InitializeVideoOverlay(backgroundImage image.Image) {
	fmt.Println("\n=== InitializeVideoOverlay ===")

	l.sendCommand(PRE_UPDATE_BITMAP, nil, nil, false, 0)
	l.sendCommand(START_DISPLAY_BITMAP, nil, PADDING_START_DISPLAY_BITMAP, false, 0)
	l.sendCommand(DISPLAY_BITMAP_ON_VIDEO, nil, nil, false, 0)

	rect := image.Rect(0, 0, l.getWidth(), l.getHeight())
	if backgroundImage != nil {
		img := image.NewNRGBA(rect)
		// Equivalent to LANCZOS resize
		xdraw.CatmullRom.Scale(img, rect, backgroundImage, backgroundImage.Bounds(), draw.Over, nil)
		l.VideoOverlay = img
		fmt.Printf("  Using background image (%dx%d)\n", img.Bounds().Dx(), img.Bounds().Dy())
	} else {
		l.VideoOverlay = image.NewNRGBA(rect)
		fmt.Println("  Using transparent overlay")
	}

	l.PreviousVideoOverlay = image.NewNRGBA(rect)
	draw.Draw(l.PreviousVideoOverlay, rect, l.VideoOverlay, image.Point{}, draw.Src)

	l.sendCommand(SEND_PAYLOAD, generateFullImage(l.VideoOverlay, l.Orientation), nil, false, 0)

	visiblePixels := []byte{0xef, 0x69}
	packetSize := []byte{byte(len(visiblePixels))}

	l.sendCommand(INIT_VIDEO_OVERLAY, packetSize, nil, false, 0)
	l.sendCommand(SEND_PAYLOAD, visiblePixels, nil, false, 0)

	time.Sleep(1 * time.Second)
	l.LcdSerial.ResetInputBuffer()
	l.sendCommand(QUERY_STATUS, nil, nil, false, 1024)

	fmt.Println("=== InitializeVideoOverlay done ===\n")
}

// ---- _get_visible_segments — verbatim from PR #348 ----

func getVisibleSegments(img *image.NRGBA, y int, imageWidth int) [][]int {
	var visibleSegments [][]int
	i := 0
	for i < imageWidth {
		c := img.NRGBAAt(i, y)
		if c.A > 0 {
			visibleSegment := []int{i, 1}
			j := i + 1
			for j < imageWidth && img.NRGBAAt(j, y).A > 0 {
				visibleSegment[1] = visibleSegment[1] + 1
				j++
			}
			i = j
			visibleSegments = append(visibleSegments, visibleSegment)
		}
		i++
	}
	return visibleSegments
}

// ---- ResfreshVideoOverlay — verbatim from PR #348 ----

func (l *LcdCommRevC) ResfreshVideoOverlay() {
	fmt.Println("\n--- ResfreshVideoOverlay ---")

	w, h := l.getWidth(), l.getHeight()
	updateImage := image.NewNRGBA(image.Rect(0, 0, w, h))

	// np.any(previous != current, axis=-1) diff logic
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			c1 := l.PreviousVideoOverlay.NRGBAAt(px, py)
			c2 := l.VideoOverlay.NRGBAAt(px, py)
			if c1 != c2 {
				updateImage.SetNRGBA(px, py, c2)
			}
		}
	}

	var imgRawData []string
	var visiblePixels []string

	for hy := 0; hy < l.VideoOverlay.Bounds().Dy(); hy++ {
		updatedPixelsSegments := getVisibleSegments(updateImage, hy, w)
		visibleSegments := getVisibleSegments(l.VideoOverlay, hy, w)

		for _, segment := range updatedPixelsSegments {
			x := segment[0]
			segmentWidth := segment[1]
			imgRawData = append(imgRawData, fmt.Sprintf("%06x%04x", hy*l.DisplayHeight+x, segmentWidth))

			for wOffset := 0; wOffset < segmentWidth; wOffset++ {
				c := l.VideoOverlay.NRGBAAt(x+wOffset, hy)
				alphaByte := int(float64(c.A) / 255.0 * 15.0)
				b := int(float64(c.B)/255.0*15.0)<<4 | ((alphaByte & 0xC) >> 2)
				g := int(float64(c.G)/255.0*15.0)<<4 | (alphaByte & 0x3)
				r := int(float64(c.R)/255.0*15.0) << 4
				imgRawData = append(imgRawData, fmt.Sprintf("%02x%02x%02x", b, g, r))
			}
		}

		for _, segment := range visibleSegments {
			x := segment[0]
			segmentWidth := segment[1]
			visiblePixels = append(visiblePixels, fmt.Sprintf("%06x%04x", hy*l.DisplayHeight+x, segmentWidth))
		}
	}

	imageMsg := strings.Join(imgRawData, "")
	imageMsg = imageMsg + strings.Join(visiblePixels, "")
	visiblePixelsMsg := strings.Join(visiblePixels, "")
	visiblePixelsSize := len(visiblePixelsMsg) / 2

	var imgPayload []byte
	var imageSizeHex string

	if len(imageMsg) > 500 {
		var chunks []string
		for i := 0; i < len(imageMsg); i += 498 {
			end := i + 498
			if end > len(imageMsg) {
				end = len(imageMsg)
			}
			chunks = append(chunks, imageMsg[i:end])
		}
		imageMsgTemp := strings.Join(chunks, "00")
		imgPayloadBytes, _ := hex.DecodeString(imageMsgTemp)
		imgPayload = append(imgPayload, imgPayloadBytes...)

		mod := len(imgPayload) % 250
		if len(imgPayload) > 250 && (mod == 0 || mod == 248 || mod == 249) {
			imgPayload = append(imgPayload, []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0xef, 0x69}...)
			imageSizeHex = fmt.Sprintf("%06x", (len(imageMsg)/2)+7)
			visiblePixelsSize = visiblePixelsSize + 5
		} else {
			imgPayload = append(imgPayload, []byte{0xef, 0x69}...)
			imageSizeHex = fmt.Sprintf("%06x", (len(imageMsg)/2)+2)
		}
	} else {
		if len(imageMsg) > 0 {
			imgPayloadBytes, _ := hex.DecodeString(imageMsg)
			imgPayload = append(imgPayload, imgPayloadBytes...)
		}
		imgPayload = append(imgPayload, []byte{0xef, 0x69}...)
		imageSizeHex = fmt.Sprintf("%06x", (len(imageMsg)/2)+2)
	}

	fmt.Printf("  image_size=0x%s visible_pixels_size=%d payload=%dB\n", imageSizeHex, visiblePixelsSize, len(imgPayload))

	updateImagePayload := []byte{}
	updateImagePayload = append(updateImagePayload, UPDATE_BITMAP...)
	imageSizeBytes, _ := hex.DecodeString(imageSizeHex)
	updateImagePayload = append(updateImagePayload, imageSizeBytes...)
	updateImagePayload = append(updateImagePayload, PADDING_NULL[0], PADDING_NULL[0], PADDING_NULL[0])

	countBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(countBuf, uint32(CountStart))
	updateImagePayload = append(updateImagePayload, countBuf...)

	vpBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(vpBuf, uint32(visiblePixelsSize))
	updateImagePayload = append(updateImagePayload, vpBuf...)

	CountStart = CountStart + 1
	draw.Draw(l.PreviousVideoOverlay, l.PreviousVideoOverlay.Bounds(), l.VideoOverlay, image.Point{}, draw.Src)

	l.sendCommand(SEND_PAYLOAD, updateImagePayload, nil, false, 0)
	l.sendCommand(SEND_PAYLOAD, imgPayload, nil, false, 0)

	fmt.Printf("--- ResfreshVideoOverlay done (count=%d) ---\n\n", CountStart)
}

// ---- DrawPILImageOnVideo — verbatim from PR #348 ----

func (l *LcdCommRevC) DrawPILImageOnVideo(img image.Image, x, y int) {
	draw.Draw(l.VideoOverlay, img.Bounds().Add(image.Pt(x, y)), img, image.Point{}, draw.Src)
}

func (l *LcdCommRevC) DrawTextOnVideo(text string, x, y int, fontName string, fontSize int, fontColor color.Color, backgroundColor color.Color, align string, anchor string) {
	textImage, left, top := l.DrawText(text, x, y, fontName, fontSize, fontColor, backgroundColor, "", align, anchor)
	l.DrawPILImageOnVideo(textImage, left, top)
}

func (l *LcdCommRevC) DrawProgressBarOnVideo(x, y, width, height int, minValue, maxValue, value int, barColor color.Color, barOutline bool, backgroundColor color.Color) {
	progressBarImage := l.DrawProgressBar(x, y, width, height, minValue, maxValue, value, barColor, barOutline, backgroundColor, "")
	l.DrawPILImageOnVideo(progressBarImage, x, y)
}

func (l *LcdCommRevC) DrawRadialProgressBarOnVideo(xc, yc, radius, barWidth, minValue, maxValue, angleStart, angleEnd, angleSep, angleSteps int, clockwise bool, value int, text string, withText bool, fontName string, fontSize int, fontColor color.Color, barColor color.Color, backgroundColor color.Color) {
	barImage := l.DrawRadialProgressBar(xc, yc, radius, barWidth, minValue, maxValue, angleStart, angleEnd, angleSep, angleSteps, clockwise, value, text, withText, fontName, fontSize, fontColor, barColor, backgroundColor, "")
	l.DrawPILImageOnVideo(barImage, xc-radius, yc-radius)
}

func (l *LcdCommRevC) DrawText(text string, x, y int, fontName string, fontSize int, fontColor color.Color, backgroundColor color.Color, backgroundImage string, align string, anchor string) (image.Image, int, int) {

	if backgroundColor == nil {
		backgroundColor = color.RGBA{0, 0, 0, 0} // Mudei de branco para transparente
	}
	if fontColor == nil {
		fontColor = color.RGBA{0, 0, 0, 255}
	}

	var textImage *image.NRGBA
	if backgroundImage == "" {
		textImage = image.NewNRGBA(image.Rect(0, 0, l.getWidth(), l.getHeight()))
		draw.Draw(textImage, textImage.Bounds(), &image.Uniform{backgroundColor}, image.Point{}, draw.Src)
	} else {
		src := l.openImage(backgroundImage)
		textImage = image.NewNRGBA(src.Bounds())
		draw.Draw(textImage, textImage.Bounds(), src, image.Point{}, draw.Src)
	}

	cacheKey := fmt.Sprintf("%s_%d", fontName, fontSize)
	if _, ok := l.fontCache[cacheKey]; !ok {
		fontBytes, _ := os.ReadFile("./res/fonts/" + fontName)
		f, _ := opentype.Parse(fontBytes)
		face, _ := opentype.NewFace(f, &opentype.FaceOptions{
			Size: float64(fontSize),
			DPI:  72,
		})
		l.fontCache[cacheKey] = face
	}
	face := l.fontCache[cacheKey]

	dc := gg.NewContextForImage(textImage)
	dc.SetFontFace(face)
	dc.SetColor(fontColor)

	w, h := dc.MeasureString(text)
	// Simplified textbbox matching - anchor is approximated
	ax, ay := 0.0, 0.0
	if align == "center" {
		ax = 0.5
	} else if align == "right" {
		ax = 1.0
	}

	dc.DrawStringAnchored(text, float64(x), float64(y)+h, ax, ay)

	left := int(math.Floor(float64(x) - w*ax))
	top := int(math.Floor(float64(y)))
	right := int(math.Ceil(float64(left) + w))
	bottom := int(math.Ceil(float64(top) + h*1.5)) // Approximate descent

	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	if right > l.getWidth() {
		right = l.getWidth()
	}
	if bottom > l.getHeight() {
		bottom = l.getHeight()
	}

	cropped := textImage.SubImage(image.Rect(left, top, right, bottom))
	resImg := image.NewNRGBA(image.Rect(0, 0, right-left, bottom-top))
	draw.Draw(resImg, resImg.Bounds(), cropped, image.Pt(left, top), draw.Src)

	return resImg, left, top
}

func (l *LcdCommRevC) DisplayText(text string, x, y int, fontName string, fontSize int, fontColor color.Color, backgroundColor color.Color, backgroundImage string, align string, anchor string) {
	textImage, left, top := l.DrawText(text, x, y, fontName, fontSize, fontColor, backgroundColor, backgroundImage, align, anchor)
	l.DisplayPILImage(textImage, left, top)
}

func (l *LcdCommRevC) DrawProgressBar(x, y, width, height, minValue, maxValue, value int, barColor color.Color, barOutline bool, backgroundColor color.Color, backgroundImage string) image.Image {
	if backgroundColor == nil {
		backgroundColor = color.RGBA{0, 0, 0, 0} // Transparente
	}
	if barColor == nil {
		barColor = color.RGBA{0, 0, 0, 255} // Preto
	}
	if value < minValue {
		value = minValue
	} else if value > maxValue {
		value = maxValue
	}

	var barImage *image.NRGBA
	if backgroundImage == "" {
		barImage = image.NewNRGBA(image.Rect(0, 0, width, height))
		draw.Draw(barImage, barImage.Bounds(), &image.Uniform{backgroundColor}, image.Point{}, draw.Src)
	} else {
		src := l.openImage(backgroundImage)
		barImage = image.NewNRGBA(image.Rect(0, 0, width, height))
		draw.Draw(barImage, barImage.Bounds(), src, image.Pt(x, y), draw.Src)
	}

	barFilledWidth := float64(value-minValue)/float64(maxValue-minValue)*float64(width) - 1
	if barFilledWidth < 0 {
		barFilledWidth = 0
	}

	dc := gg.NewContextForImage(barImage)
	dc.SetColor(barColor)
	dc.DrawRectangle(0, 0, barFilledWidth, float64(height-1))
	dc.Fill()

	if barOutline {
		dc.DrawRectangle(0, 0, float64(width-1), float64(height-1))
		dc.Stroke()
	}

	return dc.Image()
}

func (l *LcdCommRevC) DisplayProgressBar(x, y, width, height, minValue, maxValue, value int, barColor color.Color, barOutline bool, backgroundColor color.Color, backgroundImage string) {
	progressBarImage := l.DrawProgressBar(x, y, width, height, minValue, maxValue, value, barColor, barOutline, backgroundColor, backgroundImage)
	l.DisplayPILImage(progressBarImage, x, y)
}

func (l *LcdCommRevC) DrawRadialProgressBar(xc, yc, radius, barWidth, minValue, maxValue, angleStart, angleEnd, angleSep, angleSteps int, clockwise bool, value int, text string, withText bool, fontName string, fontSize int, fontColor color.Color, barColor color.Color, backgroundColor color.Color, backgroundImage string) image.Image {
	if backgroundColor == nil {
		backgroundColor = color.RGBA{0, 0, 0, 0} // Transparente
	}
	if barColor == nil {
		barColor = color.RGBA{0, 0, 0, 255} // Preto
	}
	if fontColor == nil {
		fontColor = color.RGBA{0, 0, 0, 255} // Preto
	}
	if angleStart%361 == angleEnd%361 {
		if clockwise {
			angleStart += 0 // integer approx for 0.1
		} else {
			angleEnd += 0
		}
	}

	if value < minValue {
		value = minValue
	} else if value > maxValue {
		value = maxValue
	}

	diameter := 2 * radius
	var barImage *image.NRGBA
	if backgroundImage == "" {
		barImage = image.NewNRGBA(image.Rect(0, 0, diameter, diameter))
		draw.Draw(barImage, barImage.Bounds(), &image.Uniform{backgroundColor}, image.Point{}, draw.Src)
	} else {
		src := l.openImage(backgroundImage)
		barImage = image.NewNRGBA(image.Rect(0, 0, diameter, diameter))
		draw.Draw(barImage, barImage.Bounds(), src, image.Pt(xc-radius, yc-radius), draw.Src)
	}

	pct := float64(value-minValue) / float64(maxValue-minValue)

	// Equivalência de graus para radianos
	rad := func(deg float64) float64 { return deg * math.Pi / 180.0 }

	dc := gg.NewContextForImage(barImage)
	dc.SetColor(barColor)
	dc.SetLineWidth(float64(barWidth))
	dc.SetLineCap(gg.LineCapButt) // Pontas secas, igual à PIL

	// O gg desenha pelo centro. Para bater com o BoundingBox da PIL,
	// descontamos metade da largura da linha no raio.
	rCenter := float64(radius)
	arcRadius := rCenter - float64(barWidth)/2.0

	angleStartMod := float64(angleStart % 361)
	angleEndMod := float64(angleEnd % 361)

	if clockwise {
		ecart := 0.0
		if angleEndMod < angleStartMod {
			ecart = 360 - angleStartMod + angleEndMod
		} else {
			ecart = angleEndMod - angleStartMod
		}

		if angleSep == 0 {
			aE := angleStartMod + pct*ecart
			aS := angleStartMod
			if angleEndMod < angleStartMod {
				aS = angleStartMod
				aE = angleStartMod + pct*ecart
			}
			dc.DrawArc(rCenter, rCenter, arcRadius, rad(aS), rad(aE))
			dc.Stroke()
		} else {
			aE := angleStartMod + pct*ecart
			angleComplet := ecart / float64(angleSteps)
			etapes := int((aE - angleStartMod) / angleComplet)
			for i := 0; i < etapes; i++ {
				s := angleStartMod + float64(i)*angleComplet
				e := angleStartMod + float64(i+1)*angleComplet - float64(angleSep)
				dc.DrawArc(rCenter, rCenter, arcRadius, rad(s), rad(e))
				dc.Stroke()
			}
			s := angleStartMod + float64(etapes)*angleComplet
			dc.DrawArc(rCenter, rCenter, arcRadius, rad(s), rad(aE))
			dc.Stroke()
		}
	} else {
		ecart := 0.0
		if angleEndMod < angleStartMod {
			ecart = angleStartMod - angleEndMod
		} else {
			ecart = 360 - angleEndMod + angleStartMod
		}

		if angleSep == 0 {
			aE := angleStartMod
			aS := angleStartMod - pct*ecart
			if angleEndMod < angleStartMod {
				aE = angleStartMod
				aS = angleStartMod - pct*ecart
			}
			dc.DrawArc(rCenter, rCenter, arcRadius, rad(aS), rad(aE))
			dc.Stroke()
		} else {
			aS := angleStartMod - pct*ecart
			angleComplet := ecart / float64(angleSteps)
			etapes := int((angleStartMod - aS) / angleComplet)
			for i := 0; i < etapes; i++ {
				e := angleStartMod - float64(i)*angleComplet
				s := angleStartMod - float64(i+1)*angleComplet + float64(angleSep)
				dc.DrawArc(rCenter, rCenter, arcRadius, rad(s), rad(e))
				dc.Stroke()
			}
			e := angleStartMod - float64(etapes)*angleComplet
			dc.DrawArc(rCenter, rCenter, arcRadius, rad(aS), rad(e))
			dc.Stroke()
		}
	}

	if withText {
		if text == "" {
			text = fmt.Sprintf("%d%%", int(pct*100+0.5))
		}

		fontBytes, _ := os.ReadFile("./res/fonts/" + fontName)
		f, _ := opentype.Parse(fontBytes)
		face, _ := opentype.NewFace(f, &opentype.FaceOptions{Size: float64(fontSize), DPI: 72})

		dc.SetFontFace(face)
		dc.SetColor(fontColor)
		dc.DrawStringAnchored(text, float64(radius), float64(radius), 0.5, 0.5)
	}

	return dc.Image()
}

func (l *LcdCommRevC) DisplayRadialProgressBar(xc, yc, radius, barWidth, minValue, maxValue, angleStart, angleEnd, angleSep, angleSteps int, clockwise bool, value int, text string, withText bool, fontName string, fontSize int, fontColor color.Color, barColor color.Color, backgroundColor color.Color, backgroundImage string) {
	barImage := l.DrawRadialProgressBar(xc, yc, radius, barWidth, minValue, maxValue, angleStart, angleEnd, angleSep, angleSteps, clockwise, value, text, withText, fontName, fontSize, fontColor, barColor, backgroundColor, backgroundImage)
	l.DisplayPILImage(barImage, xc-radius, yc-radius)
}

func (l *LcdCommRevC) openImage(bitmapPath string) image.Image {
	if _, ok := l.imageCache[bitmapPath]; !ok {
		file, err := os.Open(bitmapPath)
		if err == nil {
			img, _, _ := image.Decode(file)
			file.Close()
			l.imageCache[bitmapPath] = img
		}
	}
	// return a copy-like structure by re-drawing into NRGBA
	src := l.imageCache[bitmapPath]
	if src == nil {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1))
	}
	cpy := image.NewNRGBA(src.Bounds())
	draw.Draw(cpy, cpy.Bounds(), src, src.Bounds().Min, draw.Src)
	return cpy
}

// =====================================================================
// Main — exact replica of test_new_features.py from PR #348
// =====================================================================

func main() {
	bgPtr := flag.String("background", "", "Path to a background image (PNG/JPG) to use as the initial overlay instead of a transparent image")
	flag.Parse()

	var background image.Image
	if *bgPtr != "" {
		file, err := os.Open(*bgPtr)
		if err == nil {
			background, _, _ = image.Decode(file)
			file.Close()
			fmt.Printf("Loaded background image: %s (%dx%d)\n", *bgPtr, background.Bounds().Dx(), background.Bounds().Dy())
		}
	}

	fmt.Println("Selected Hardware Revision C (Turing Smart Screen 5\")")

	lcdComm := NewLcdCommRevC(COM_PORT, DISPLAY_WIDTH, DISPLAY_HEIGHT)

	orientation := LANDSCAPE
	lcdComm.SetOrientation(orientation)

	lcdComm.InitializeComm()

	videoSize := lcdComm.GetFileSize(VIDEO_DEVICE_PATH)
	fmt.Printf("Video on device: %d bytes\n", videoSize)

	if videoSize == 0 && VIDEO_LOCAL_PATH != "" {
		if _, err := os.Stat(VIDEO_LOCAL_PATH); err == nil {
			fmt.Println("Uploading video ...")
			lcdComm.UploadFile(VIDEO_LOCAL_PATH, VIDEO_DEVICE_PATH)
			fmt.Println("Upload done")
			fmt.Println(lcdComm.GetFileSize(VIDEO_DEVICE_PATH))
		}
	}

	lcdComm.StopVideo()
	lcdComm.Clear(background)

	lcdComm.StartVideo(VIDEO_DEVICE_PATH)

	lcdComm.InitializeVideoOverlay(background)
	lcdComm.ResfreshVideoOverlay()

	stop := false
	barValue := 0

	for !stop {
		start := time.Now()

		tStr := time.Now().Format("15:04:05.000000")
		lcdComm.DrawTextOnVideo(tStr, 160, 2, "roboto/Roboto-Bold.ttf", 20, color.RGBA{255, 0, 0, 255}, nil, "left", "")

		lcdComm.DrawProgressBarOnVideo(10, 40, 140, 30, 0, 100, barValue, color.RGBA{255, 255, 0, 255}, true, nil)

		lcdComm.DrawProgressBarOnVideo(160, 40, 140, 30, 0, 19, barValue%20, color.RGBA{0, 255, 0, 255}, false, nil)

		lcdComm.DrawRadialProgressBarOnVideo(98, 260, 25, 4, 0, 100, 0, 0, 0, 10, true, barValue, "", true, "roboto/Roboto-Black.ttf", 20, color.RGBA{255, 255, 255, 255}, color.RGBA{0, 255, 0, 255}, nil)

		tempText := fmt.Sprintf("%d°C", 10*(barValue/10))
		lcdComm.DrawRadialProgressBarOnVideo(222, 260, 40, 13, 0, 100, 405, 135, 5, 10, false, barValue, tempText, true, "geforce/GeForce-Bold.ttf", 20, color.RGBA{255, 255, 0, 255}, color.RGBA{255, 255, 0, 255}, nil)

		lcdComm.ResfreshVideoOverlay()

		barValue = (barValue + 2) % 101
		end := time.Since(start).Seconds()
		fmt.Printf("refresh done (took %.3f s)\n", end)
	}

	lcdComm.closeSerial()
}
