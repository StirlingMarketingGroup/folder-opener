// Folder Opener is a small local companion server that opens folders in the
// system's file browser on behalf of web apps running in the browser.
//
// It listens on localhost only and exposes two endpoints:
//
//	GET  /status  -> {"status":"running","version":"..."}
//	POST /open    -> {"path":"/absolute/path"} opens the folder (or reveals a file)
//
// It follows the same local-companion pattern as Dazzle
// (github.com/StirlingMarketingGroup/dazzle), our ZPL print server.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

// version is stamped at build time via -ldflags "-X main.version=v1.2.3".
var version = "dev"

const defaultPort = 29101

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "autostart":
			autostartCLI(os.Args[2:])
			return
		case "version", "-v", "--version":
			fmt.Println(version)
			return
		case "help", "-h", "--help":
			fmt.Print(usage)
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
			os.Exit(2)
		}
	}

	port := defaultPort
	if v := os.Getenv("FOLDER_OPENER_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			log.Fatalf("invalid FOLDER_OPENER_PORT %q", v)
		}
		port = p
	}

	// Binding fails fast if another instance already owns the port, which
	// doubles as our single-instance guard.
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		log.Fatalf("failed to bind 127.0.0.1:%d (already running?): %v", port, err)
	}
	log.Printf("Folder Opener %s listening on 127.0.0.1:%d", version, port)

	server := &http.Server{Handler: newHandler()}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// The tray owns the process lifetime; Quit shuts the server down.
	runTray(port, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	})
}

const usage = `Folder Opener — local "open folder in file browser" companion server

Usage:
  folder-opener                       run the server (with tray icon)
  folder-opener autostart enable      start automatically at login
  folder-opener autostart disable     stop starting at login
  folder-opener autostart status      report whether autostart is enabled
  folder-opener version               print the version

Environment:
  FOLDER_OPENER_PORT   port to listen on (default 29101, localhost only)
`
