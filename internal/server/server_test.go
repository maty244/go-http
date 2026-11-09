package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"back/internal/handler"
	"back/internal/middlewares"
	"back/internal/server"
)

// Probamos el comportamiento observable: respuesta HTTP y estado registrado.
// Esto también detecta si ServeHTTP se salta algún middleware por accidente.
func TestServerRoutesWithMiddleware(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "salud", method: http.MethodGet, path: "/health", status: http.StatusOK},
		{name: "panic recuperado", method: http.MethodGet, path: "/panic", status: http.StatusInternalServerError},
		{name: "ruta inexistente", method: http.MethodGet, path: "/missing", status: http.StatusNotFound},
		{name: "método no permitido", method: http.MethodPost, path: "/health", status: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			srv := server.New(":0", logger)
			srv.Use(middlewares.Recovery(logger))
			srv.Use(middlewares.RequestLogger(logger))
			srv.RegisterRoute("GET /health", handler.NewHealthHandler())
			srv.RegisterRoute("GET /panic", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("detalle interno de prueba")
			}))

			response := httptest.NewRecorder()
			srv.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != tt.status {
				t.Fatalf("estado = %d; esperado %d", response.Code, tt.status)
			}

			if tt.status == http.StatusOK {
				if got := response.Header().Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %q; esperado application/json", got)
				}
				var body struct {
					Status string `json:"status"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Status != "ok" {
					t.Errorf("respuesta de salud inválida: %s (error: %v)", response.Body.String(), err)
				}
			}
			if tt.status == http.StatusInternalServerError {
				if strings.Contains(response.Body.String(), "detalle interno") {
					t.Error("la respuesta expone el detalle del panic")
				}
				if !strings.Contains(logs.String(), "detalle interno de prueba") {
					t.Error("falta el detalle del panic en los logs")
				}
			}

			// Recovery puede emitir un registro antes del resumen de la petición.
			var last struct {
				Status int    `json:"status"`
				Path   string `json:"path"`
			}
			decoder := json.NewDecoder(&logs)
			for {
				err := decoder.Decode(&last)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("leer log JSON: %v", err)
				}
			}
			if last.Status != tt.status || last.Path != tt.path {
				t.Errorf("registro final = %+v; esperado estado %d y ruta %s", last, tt.status, tt.path)
			}
		})
	}
}

func TestStartReturnsErrorWhenPortIsOccupied(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := server.New(listener.Addr().String(), logger)
	if err := srv.Start(); err == nil {
		t.Fatal("Start debe devolver un error cuando el puerto ya está ocupado")
	}
}
