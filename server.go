package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type statusResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type openRequest struct {
	Path string `json:"path"`
}

type openResponse struct {
	Path string `json:"path"`
	// Action is "opened" for a directory, "revealed" for a file selected in
	// its parent folder.
	Action string `json:"action"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", statusHandler)
	mux.HandleFunc("POST /open", openHandler)
	return corsMiddleware(mux)
}

// corsMiddleware allows any web origin to call us, matching Dazzle's
// permissive CORS: the server is bound to 127.0.0.1 and only opens the file
// browser, so the origin doesn't matter. It also answers Chrome's Private
// Network Access preflight so pages on public origins can reach localhost.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			if r.Header.Get("Access-Control-Request-Private-Network") == "true" {
				h.Set("Access-Control-Allow-Private-Network", "true")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func statusHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{Status: "running", Version: version})
}

func openHandler(w http.ResponseWriter, r *http.Request) {
	var req openRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid JSON body: " + err.Error(),
			Code:  "bad_request",
		})
		return
	}

	action, err := openPath(req.Path)
	if err != nil {
		log.Printf("open %q: %v", req.Path, err)
		status, code := http.StatusInternalServerError, "internal"
		switch {
		case errors.Is(err, errNotFound):
			status, code = http.StatusNotFound, "not_found"
		case errors.Is(err, errBadPath):
			status, code = http.StatusBadRequest, "bad_request"
		}
		writeJSON(w, status, errorResponse{Error: err.Error(), Code: code})
		return
	}

	log.Printf("open %q: %s", req.Path, action)
	writeJSON(w, http.StatusOK, openResponse{Path: req.Path, Action: action})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
