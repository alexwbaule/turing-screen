package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"nhooyr.io/websocket"
)

// Controller provides actions the API can trigger on the daemon.
type Controller interface {
	GetStatus() StatusResponse
	SetModeEditor() error
	SetModeNormal() error
	SetBrightness(value int) error
	RestartDevice() error
	RebootDevice() error
	ResetUSB() error
	TurnOff() error
	PreviewImage(imgData []byte) error
	ApplyTheme(name string) error
	GetThemeList() []string
	GetCurrentTheme() string
	GetSensorValues() map[string]interface{}
	GetStorageInfo() (StorageInfo, error)
	GetStorageFiles(path string) ([]string, error)
	UploadFile(name string, data []byte) error
	DeleteFile(path string) error
	PlayVideo(path string) error
	StopVideo() error
}

type StatusResponse struct {
	Mode       string `json:"mode"`
	Theme      string `json:"theme"`
	Firmware   string `json:"firmware"`
	Uptime     string `json:"uptime"`
	APIVersion string `json:"api_version"`
}

type StorageInfo struct {
	Total int64 `json:"total"`
	Used  int64 `json:"used"`
	Free  int64 `json:"free"`
}

// Message is the generic WebSocket message format.
type Message struct {
	ID      string          `json:"id"`
	Action  string          `json:"action"`
	Status  string          `json:"status,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type Server struct {
	log        *logger.Logger
	controller Controller
	port       int
	server     *http.Server
	clients    map[*websocket.Conn]bool
	clientsMu  sync.Mutex
}

func NewServer(log *logger.Logger, controller Controller, port int) *Server {
	return &Server{
		log:        log,
		controller: controller,
		port:       port,
		clients:    make(map[*websocket.Conn]bool),
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.handleWebSocket(w, r, ctx)
	})

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	s.log.Infof("WebSocket server starting on ws://localhost:%d/ws", s.port)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(shutdownCtx)
	}()

	// Start status broadcast goroutine
	go s.broadcastStatusLoop(ctx)

	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("WebSocket server error: %w", err)
	}
	return nil
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		s.log.Warnf("WebSocket accept failed: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")

	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	s.log.Info("WebSocket client connected")

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			s.log.Debugf("WebSocket read error: %v", err)
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			s.sendError(ctx, conn, "", "invalid JSON: "+err.Error())
			continue
		}

		response := s.handleMessage(msg)
		respData, _ := json.Marshal(response)
		conn.Write(ctx, websocket.MessageText, respData)
	}
}

func (s *Server) handleMessage(msg Message) Message {
	switch msg.Action {

	// --- Status & Control ---
	case "status.get":
		return s.respond(msg.ID, msg.Action, s.controller.GetStatus())

	case "mode.editor":
		if err := s.controller.SetModeEditor(); err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respond(msg.ID, msg.Action, map[string]string{"mode": "editor"})

	case "mode.normal":
		if err := s.controller.SetModeNormal(); err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respond(msg.ID, msg.Action, map[string]string{"mode": "normal"})

	// --- Device ---
	case "device.brightness":
		var p struct {
			Value int `json:"value"`
		}
		json.Unmarshal(msg.Payload, &p)
		if err := s.controller.SetBrightness(p.Value); err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respondOK(msg.ID, msg.Action)

	case "device.restart":
		s.controller.RestartDevice()
		return s.respondOK(msg.ID, msg.Action)

	case "device.reboot":
		s.controller.RebootDevice()
		return s.respondOK(msg.ID, msg.Action)

	case "device.reset":
		if err := s.controller.ResetUSB(); err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respondOK(msg.ID, msg.Action)

	case "device.turnoff":
		s.controller.TurnOff()
		return s.respondOK(msg.ID, msg.Action)

	// --- Theme ---
	case "theme.current":
		return s.respond(msg.ID, msg.Action, map[string]string{"theme": s.controller.GetCurrentTheme()})

	case "theme.list":
		return s.respond(msg.ID, msg.Action, map[string]interface{}{"themes": s.controller.GetThemeList()})

	case "theme.apply":
		var p struct {
			Name string `json:"name"`
		}
		json.Unmarshal(msg.Payload, &p)
		if p.Name == "" {
			return s.respondError(msg.ID, msg.Action, "name is required")
		}
		if err := s.controller.ApplyTheme(p.Name); err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respond(msg.ID, msg.Action, map[string]string{"status": "ok", "theme": p.Name})

	case "theme.preview":
		var p struct {
			Image string `json:"image"`
		}
		json.Unmarshal(msg.Payload, &p)
		imgData, err := base64.StdEncoding.DecodeString(p.Image)
		if err != nil {
			return s.respondError(msg.ID, msg.Action, "invalid base64")
		}
		if err := s.controller.PreviewImage(imgData); err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respondOK(msg.ID, msg.Action)

	case "theme.video.start":
		var p struct {
			Path string `json:"path"`
		}
		json.Unmarshal(msg.Payload, &p)
		if err := s.controller.PlayVideo(p.Path); err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respondOK(msg.ID, msg.Action)

	case "theme.video.stop":
		if err := s.controller.StopVideo(); err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respondOK(msg.ID, msg.Action)

	// --- Storage ---
	case "storage.info":
		info, err := s.controller.GetStorageInfo()
		if err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respond(msg.ID, msg.Action, info)

	case "storage.files":
		var p struct {
			Path string `json:"path"`
		}
		json.Unmarshal(msg.Payload, &p)
		if p.Path == "" {
			p.Path = "/root/video/"
		}
		files, err := s.controller.GetStorageFiles(p.Path)
		if err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respond(msg.ID, msg.Action, map[string]interface{}{"files": files})

	case "storage.upload":
		var p struct {
			Name string `json:"name"`
			Data string `json:"data"`
		}
		json.Unmarshal(msg.Payload, &p)
		data, err := base64.StdEncoding.DecodeString(p.Data)
		if err != nil {
			return s.respondError(msg.ID, msg.Action, "invalid base64")
		}
		if err := s.controller.UploadFile(p.Name, data); err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respondOK(msg.ID, msg.Action)

	case "storage.delete":
		var p struct {
			Path string `json:"path"`
		}
		json.Unmarshal(msg.Payload, &p)
		if err := s.controller.DeleteFile(p.Path); err != nil {
			return s.respondError(msg.ID, msg.Action, err.Error())
		}
		return s.respondOK(msg.ID, msg.Action)

	// --- Sensors ---
	case "sensors.values":
		return s.respond(msg.ID, msg.Action, s.controller.GetSensorValues())

	default:
		return s.respondError(msg.ID, msg.Action, "unknown action: "+msg.Action)
	}
}

// broadcastStatusLoop sends status events to all connected clients every 2 seconds.
func (s *Server) broadcastStatusLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			status := s.controller.GetStatus()
			s.broadcast(ctx, "event.status", status)
		}
	}
}

// broadcast sends a push event to all connected clients.
func (s *Server) broadcast(ctx context.Context, action string, payload interface{}) {
	payloadData, _ := json.Marshal(payload)
	msg := Message{
		Action:  action,
		Status:  "ok",
		Payload: payloadData,
	}
	data, _ := json.Marshal(msg)

	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	for conn := range s.clients {
		conn.Write(ctx, websocket.MessageText, data)
	}
}

// --- Response helpers ---

func (s *Server) respond(id, action string, payload interface{}) Message {
	data, _ := json.Marshal(payload)
	return Message{ID: id, Action: action, Status: "ok", Payload: data}
}

func (s *Server) respondOK(id, action string) Message {
	data, _ := json.Marshal(map[string]string{"status": "ok"})
	return Message{ID: id, Action: action, Status: "ok", Payload: data}
}

func (s *Server) respondError(id, action, errMsg string) Message {
	return Message{ID: id, Action: action, Status: "error", Error: errMsg}
}

func (s *Server) sendError(ctx context.Context, conn *websocket.Conn, id, errMsg string) {
	msg := Message{ID: id, Status: "error", Error: errMsg}
	data, _ := json.Marshal(msg)
	conn.Write(ctx, websocket.MessageText, data)
}
