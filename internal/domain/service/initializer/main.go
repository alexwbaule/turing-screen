package initializer

import (
	"fmt"
	"image"
	"os"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/config"
	"github.com/alexwbaule/turing-screen/internal/application/theme"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	themeEntity "github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/video"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
)

// Initializer lida com a sequência de inicialização síncrona do dispositivo.
type Initializer struct {
	sender     serial.SerialSender
	log        *logger.Logger
	cfg        *config.Config
	theme      *theme.Theme
	cmdDevice  *command.Device
	cmdMedia   *command.Media
	cmdOption  *command.Option
	cmdStorage *command.Storage
	cmdBright  *command.Brightness
	cmdPayload *command.Payload
}

// New cria uma nova instância do Initializer.
func New(
	sender serial.SerialSender,
	log *logger.Logger,
	cfg *config.Config,
	theme *theme.Theme,
	cmdDevice *command.Device,
	cmdMedia *command.Media,
	cmdOption *command.Option,
	cmdStorage *command.Storage,
	cmdBright *command.Brightness,
	cmdPayload *command.Payload,
) *Initializer {
	return &Initializer{
		sender:     sender,
		log:        log,
		cfg:        cfg,
		theme:      theme,
		cmdDevice:  cmdDevice,
		cmdMedia:   cmdMedia,
		cmdOption:  cmdOption,
		cmdStorage: cmdStorage,
		cmdBright:  cmdBright,
		cmdPayload: cmdPayload,
	}
}

// Run executes the entire synchronous initialization sequence.
func (i *Initializer) Run(background device.ImageBackground) error {
	i.log.Info("starting synchronous device initialization sequence...")

	// Passo 1: Handshake
	if _, err := i.sender.Execute(i.cmdDevice.Hello()); err != nil {
		return fmt.Errorf("failed on HELLO command: %w", err)
	}
	i.log.Info("step 1/5: HELLO command successful")

	// Passo 2: Orientação
	i.cmdOption.SetOptions(command.Default, command.NoFlip, command.Disabled)
	if _, err := i.sender.Execute(i.cmdOption); err != nil {
		return fmt.Errorf("failed on OPTIONS command: %w", err)
	}
	i.log.Info("step 2/5: OPTIONS command successful")

	// Passo 3: Parar qualquer mídia e definir o brilho
	if _, err := i.sender.Execute(i.cmdMedia.StopVideo()); err != nil {
		return fmt.Errorf("failed on STOP_VIDEO command: %w", err)
	}
	if _, err := i.sender.Execute(i.cmdMedia.StopMedia()); err != nil {
		return fmt.Errorf("failed on STOP_MEDIA command: %w", err)
	}
	brightness := i.cfg.GetDeviceDisplay().Brightness
	if _, err := i.sender.Execute(i.cmdBright.SetBrightness(brightness)); err != nil {
		return fmt.Errorf("failed on SET_BRIGHTNESS command: %w", err)
	}
	i.log.Info("step 3/5: media stopped and brightness set")

	// Passo 4 & 5: Lógica de Vídeo ou Imagem Estática
	videoDisplay := i.theme.GetVideoPlay()
	if videoDisplay != nil {
		// MODO VÍDEO
		i.log.Info("step 4/5: video mode detected, handling video initialization...")
		if err := i.handleVideoInitialization(videoDisplay); err != nil {
			return err
		}

		// Clear display before starting video (matching reference: StopVideo → Clear → StartVideo → InitOverlay)
		i.log.Info("clearing display before video start...")
		if _, err := i.sender.Execute(i.cmdMedia.StopVideo()); err != nil {
			return fmt.Errorf("failed on STOP_VIDEO (before clear) command: %w", err)
		}
		i.clearDisplay()

		// Start the video
		if _, err := i.sender.Execute(i.cmdMedia.StartVideo(videoDisplay.Path)); err != nil {
			return fmt.Errorf("failed on START_VIDEO command: %w", err)
		}
		i.log.Info("video started")

		// Initialize video overlay (full flow matching reference InitializeVideoOverlay)
		i.log.Info("step 5/5: initializing video overlay...")
		display := i.cfg.GetDeviceDisplay()
		w, h := display.Width, display.Height
		if display.Reverse {
			w, h = h, w
		}
		orientation := i.theme.GetDisplay().Orientation

		// Scale background image to display dimensions
		var scaled *image.NRGBA
		if videoDisplay.BackgroundImage != nil {
			scaled = device.NewScaledNRGBA(videoDisplay.BackgroundImage, w, h)
		} else {
			scaled = device.NewBlank(w, h)
		}

		// Build BGRA payload (rotation + BGRA + 249+1 chunking)
		bgraPayload := video.BuildInitPayload(scaled, orientation)

		// Create overlay buffer with initial background
		overlay := video.NewOverlayBuffer(i.log, display)
		overlay.SetInitial(scaled)

		// Send InitVideoOverlay command (pre-encoded packets)
		initCmd := command.NewInitVideoOverlay(i.log, bgraPayload)
		if _, err := i.sender.Execute(initCmd); err != nil {
			return fmt.Errorf("failed on INIT_VIDEO_OVERLAY command: %w", err)
		}
		i.log.Info("video overlay initialized")

		// Initial refresh (matching reference: ResfreshVideoOverlay called right after InitializeVideoOverlay)
		refreshCmd := overlay.Refresh()
		if _, err := i.sender.Execute(refreshCmd); err != nil {
			return fmt.Errorf("failed on initial overlay refresh: %w", err)
		}
		i.log.Info("initial overlay refresh sent")

		// Wait before querying status (matching reference: time.Sleep + QUERY_STATUS)
		time.Sleep(1 * time.Second)

		// Query device status
		if _, err := i.sender.Execute(i.cmdMedia.QueryStatus()); err != nil {
			i.log.Warnf("final query status check failed, but continuing: %v", err)
		}

	} else {
		// MODO IMAGEM ESTÁTICA
		i.log.Info("step 4/5: static image mode detected")
		if _, err := i.sender.Execute(i.cmdPayload.SendPayload(background, false)); err != nil {
			return fmt.Errorf("failed on static image SEND_PAYLOAD command: %w", err)
		}
		i.log.Info("step 5/5: static background image sent successfully")
	}

	i.log.Info("synchronous device initialization completed successfully!")
	return nil
}

// clearDisplay sends a blank white image to clear the display,
// matching the reference Clear() function.
func (i *Initializer) clearDisplay() {
	display := i.cfg.GetDeviceDisplay()
	w, h := display.Width, display.Height
	if display.Reverse {
		w, h = h, w
	}
	blankImg := device.NewImageProcess(device.NewBlank(w, h))
	if _, err := i.sender.Execute(i.cmdPayload.SendPayload(blankImg, false)); err != nil {
		i.log.Warnf("failed on clear display command: %v", err)
	}
}

func (i *Initializer) handleVideoInitialization(videoDisplay *themeEntity.DinamicImage) error {
	i.log.Info("checking for video file on device...")

	// Checa o tamanho do arquivo no dispositivo
	cmd := i.cmdStorage.GetFileInfo(videoDisplay.Path)
	resp, err := i.sender.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to execute GetFileInfo: %w", err)
	}

	videoSize, err := utils.ParseFileSize(resp)
	if err != nil {
		return fmt.Errorf("failed to parse file size from device response: %w", err)
	}
	i.log.Infof("device video file size: %d", videoSize)

	// Se o arquivo não existe, faz o upload
	if videoSize == 0 {
		i.log.Info("video file not found on device, starting upload process.")

		fileInfo, err := os.Stat(videoDisplay.Path)
		if err != nil {
			return fmt.Errorf("failed to get local video file info '%s': %w", videoDisplay.Path, err)
		}
		localSize := fileInfo.Size()
		i.log.Infof("local video file '%s' found, size: %d. enqueuing create file command.", videoDisplay.Path, localSize)

		// Envia comando para criar o arquivo
		createCmd := i.cmdStorage.CreateFile(videoDisplay.Path, localSize)
		if _, err := i.sender.Execute(createCmd); err != nil {
			return fmt.Errorf("failed on CREATE_FILE command: %w", err)
		}
		i.log.Info("CREATE_FILE command successful, proceeding with upload.")

		// Lê o arquivo local e envia o payload
		videoBytes, err := os.ReadFile(videoDisplay.Path)
		if err != nil {
			return fmt.Errorf("failed to read local video file for upload: %w", err)
		}

		uploadCmd := i.cmdStorage.UploadFile(videoBytes)
		if _, err := i.sender.Execute(uploadCmd); err != nil {
			return fmt.Errorf("failed on UPLOAD_FILE command: %w", err)
		}
		i.log.Info("UPLOAD_FILE command successful.")
	}

	return nil
}
