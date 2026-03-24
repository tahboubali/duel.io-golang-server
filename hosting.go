package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := recorder.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

func (recorder *statusRecorder) Flush() {
	flusher, ok := recorder.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func (s *Server) NewHandler() (http.Handler, error) {
	root, err := webRoot()
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.Dir(root))
	mux := http.NewServeMux()

	mux.HandleFunc("/connect", s.handleConnect)
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/panel.html" {
			http.ServeFile(w, r, filepath.Join(root, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	}))

	return logRequests(mux), nil
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade ws connection for %s: %v", r.RemoteAddr, err)
		return
	}
	s.wg.Add(1)
	go s.handleConn(&Conn{Conn: conn})
}

func webRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("could not resolve hosting.go path")
	}
	return filepath.Join(filepath.Dir(filename), "web"), nil
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf("[web] %s %s -> %d", r.Method, r.URL.Path, recorder.status)
	})
}
