package muxlogger

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type WrappedResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewWrappedResponseWriter(w http.ResponseWriter) *WrappedResponseWriter {
	return &WrappedResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (crw *WrappedResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	crw.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := createLogFile()
		if err != nil {
			log.Printf("Error creating or opening log file: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		log.SetOutput(file)

		// get request id from header passed from the client, inject one if not present
		requestId := r.Header.Get("X-Request-ID")
		if requestId == "" {
			requestId = uuid.New().String()
			r.Header.Add("X-Request-ID", requestId)
		}

		defer closeLogFile(file)

		// wrapped writer to get response status code on exit
		writer := NewWrappedResponseWriter(w)

		// defer function used to log response status after request is served
		defer logResponseStatus(requestId, writer)

		log.Printf("%s,%s,%s,%s", requestId, r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(writer, r)
	})
}

func createLogFile() (*os.File, error) {
	// Create a filename for today's logging
	filename := strings.Join([]string{
		time.Now().Format("2006-01-02") + "_api_requests", "csv"}, ".")

	// Define the relative path to where to store the logs
	logDir := filepath.Join("api", "logs")

	// Ensure the logs directory exists; if not, create it
	err := os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	// Create file
	logPath := filepath.Join(logDir, filename)
	file, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func closeLogFile(file *os.File) {
	err := file.Close()
	if err != nil {
		log.Printf("Error closing log file %s: %v", file.Name(), err)
	}
}

func logResponseStatus(id string, w *WrappedResponseWriter) {
	log.Printf("%s,%d", id, w.statusCode)
}
