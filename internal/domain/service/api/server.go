package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
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

type Server struct {
	log        *logger.Logger
	controller Controller
	port       int
	server     *http.Server
}

func NewServer(log *logger.Logger, controller Controller, port int) *Server {
	return &Server{
		log:        log,
		controller: controller,
		port:       port,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Status & Control
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("POST /mode/editor", s.handleModeEditor)
	mux.HandleFunc("POST /mode/normal", s.handleModeNormal)

	// Device
	mux.HandleFunc("POST /device/brightness", s.handleBrightness)
	mux.HandleFunc("POST /device/restart", s.handleRestart)
	mux.HandleFunc("POST /device/reboot", s.handleReboot)
	mux.HandleFunc("POST /device/reset", s.handleReset)
	mux.HandleFunc("POST /device/turnoff", s.handleTurnOff)

	// Theme
	mux.HandleFunc("GET /theme/current", s.handleThemeCurrent)
	mux.HandleFunc("GET /theme/list", s.handleThemeList)
	mux.HandleFunc("POST /theme/preview", s.handleThemePreview)
	mux.HandleFunc("POST /theme/apply", s.handleThemeApply)
	mux.HandleFunc("POST /theme/video/start", s.handleVideoStart)
	mux.HandleFunc("POST /theme/video/stop", s.handleVideoStop)

	// Storage
	mux.HandleFunc("GET /storage/info", s.handleStorageInfo)
	mux.HandleFunc("GET /storage/files", s.handleStorageFiles)
	mux.HandleFunc("POST /storage/upload", s.handleStorageUpload)
	mux.HandleFunc("DELETE /storage/file", s.handleStorageDelete)

	// Sensors
	mux.HandleFunc("GET /sensors/values", s.handleSensorValues)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	s.log.Infof("API server starting on port %d", s.port)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(shutdownCtx)
	}()

	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("API server error: %w", err)
	}
	return nil
}

// --- Handlers ---

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, s.controller.GetStatus())
}

func (s *Server) handleModeEditor(w http.ResponseWriter, r *http.Request) {
	if err := s.controller.SetModeEditor(); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"mode": "editor"})
}

func (s *Server) handleModeNormal(w http.ResponseWriter, r *http.Request) {
	if err := s.controller.SetModeNormal(); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"mode": "normal"})
}

func (s *Server) handleBrightness(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value int `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := s.controller.SetBrightness(req.Value); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "ok"})
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	if err := s.controller.RestartDevice(); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "restarted"})
}

func (s *Server) handleReboot(w http.ResponseWriter, r *http.Request) {
	if err := s.controller.RebootDevice(); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "rebooted"})
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if err := s.controller.ResetUSB(); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "reset"})
}

func (s *Server) handleTurnOff(w http.ResponseWriter, r *http.Request) {
	if err := s.controller.TurnOff(); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "off"})
}

func (s *Server) handleThemeCurrent(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, map[string]string{"theme": s.controller.GetCurrentTheme()})
}

func (s *Server) handleThemeList(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, map[string]interface{}{"themes": s.controller.GetThemeList()})
}

func (s *Server) handleThemePreview(w http.ResponseWriter, r *http.Request) {
	data, err := readBody(r, 10*1024*1024) // 10MB max
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.controller.PreviewImage(data); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "ok"})
}

func (s *Server) handleThemeApply(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		s.errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := s.controller.ApplyTheme(req.Name); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "applied", "theme": req.Name})
}

func (s *Server) handleStorageInfo(w http.ResponseWriter, r *http.Request) {
	info, err := s.controller.GetStorageInfo()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, info)
}

func (s *Server) handleStorageFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/root/video/"
	}
	files, err := s.controller.GetStorageFiles(path)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]interface{}{"files": files})
}

func (s *Server) handleStorageUpload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(20 * 1024 * 1024) // 20MB max
	file, header, err := r.FormFile("file")
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "file field required")
		return
	}
	defer file.Close()

	data := make([]byte, header.Size)
	if _, err := file.Read(data); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	if err := s.controller.UploadFile(header.Filename, data); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "uploaded", "file": header.Filename})
}

func (s *Server) handleStorageDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := s.controller.DeleteFile(req.Path); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "deleted"})
}

func (s *Server) handleSensorValues(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, s.controller.GetSensorValues())
}

func (s *Server) handleVideoStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := s.controller.PlayVideo(req.Path); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "playing"})
}

func (s *Server) handleVideoStop(w http.ResponseWriter, r *http.Request) {
	if err := s.controller.StopVideo(); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, map[string]string{"status": "stopped"})
}

// --- Helpers ---

func (s *Server) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) errorResponse(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func readBody(r *http.Request, maxSize int64) ([]byte, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxSize)
	data := make([]byte, 0, 1024)
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return data, nil
}
