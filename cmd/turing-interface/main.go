package main

import (
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	pyroscope "github.com/grafana/pyroscope-go"

	"github.com/alexwbaule/turing-screen/internal/ui"
)

func startProfiling(appName, pprofAddr string) {
	go func() {
		log.Printf("[profiling] pprof listening on http://%s/debug/pprof/", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Printf("[profiling] pprof error: %v", err)
		}
	}()

	if addr := os.Getenv("PYROSCOPE_ADDR"); addr != "" {
		_, err := pyroscope.Start(pyroscope.Config{
			ApplicationName: appName,
			ServerAddress:   addr,
			Logger:          pyroscope.StandardLogger,
			ProfileTypes: []pyroscope.ProfileType{
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,
				pyroscope.ProfileGoroutines,
			},
		})
		if err != nil {
			log.Printf("[profiling] pyroscope start error: %v", err)
		} else {
			log.Printf("[profiling] pyroscope pushing to %s as '%s'", addr, appName)
		}
	}
}

const ipcSocket = "/tmp/turing-interface.sock"

func main() {
	// If another instance is already running, signal it to show its window and exit.
	conn, err := net.DialTimeout("unix", ipcSocket, 300*time.Millisecond)
	if err == nil {
		conn.Write([]byte("show")) //nolint:errcheck
		conn.Close()
		os.Exit(0)
	}

	// First instance: claim the socket before starting the app.
	os.Remove(ipcSocket)
	ln, err := net.Listen("unix", ipcSocket)
	if err != nil {
		log.Printf("[single-instance] socket listen failed: %v (continuing without single-instance enforcement)", err)
	}

	startProfiling("turing-interface", "localhost:6061")
	editorApp := ui.NewEditorApp()

	if ln != nil {
		go func() {
			defer ln.Close()
			defer os.Remove(ipcSocket)
			for {
				c, err := ln.Accept()
				if err != nil {
					return
				}
				c.Close()
				editorApp.ShowWindow()
			}
		}()
	}

	editorApp.Run()

	// Clean up socket on normal exit (Run() blocks until the app quits).
	if ln != nil {
		ln.Close()
		os.Remove(ipcSocket)
	}
}
