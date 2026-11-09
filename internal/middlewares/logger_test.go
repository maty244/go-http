package middlewares_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"back/internal/middlewares"
)

// El canal permite esperar al log sin compartir un buffer mutable entre goroutines.
type logChannel chan []byte

func (logs logChannel) Write(data []byte) (int, error) {
	logs <- append([]byte(nil), data...)
	return len(data), nil
}

func TestRequestLoggerPreservesHTTPResponse(t *testing.T) {
	tests := []struct {
		name     string
		statuses []int
		body     string
		wantCode int
	}{
		{name: "respuesta vacía", wantCode: http.StatusOK},
		{name: "200 implícito y tipo de contenido", body: "<html>hola</html>", wantCode: http.StatusOK},
		{name: "primer estado definitivo", statuses: []int{201, 500}, body: "creado", wantCode: http.StatusCreated},
		{name: "informativos antes del definitivo", statuses: []int{103, 103, 201}, body: "creado", wantCode: http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := make(logChannel, 1)
			logger := slog.New(slog.NewJSONHandler(logs, nil))
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for _, code := range tt.statuses {
					w.WriteHeader(code)
				}
				if tt.body != "" {
					_, _ = io.WriteString(w, tt.body)
				}
			})
			// Usamos un servidor real de prueba: ResponseRecorder no simula los 1xx.
			srv := httptest.NewServer(middlewares.RequestLogger(logger)(handler))
			t.Cleanup(srv.Close)
			client := srv.Client()
			client.Timeout = 5 * time.Second
			response, err := client.Get(srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tt.wantCode || string(body) != tt.body {
				t.Errorf("respuesta = %d %q; esperada %d %q", response.StatusCode, body, tt.wantCode, tt.body)
			}
			if tt.body != "" {
				if got, want := response.Header.Get("Content-Type"), http.DetectContentType([]byte(tt.body)); got != want {
					t.Errorf("Content-Type = %q; esperado %q", got, want)
				}
			}

			select {
			case data := <-logs:
				var record struct {
					Status int `json:"status"`
				}
				if err := json.Unmarshal(data, &record); err != nil {
					t.Fatal(err)
				}
				if record.Status != tt.wantCode {
					t.Errorf("estado registrado = %d; esperado %d", record.Status, tt.wantCode)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("no se registró la petición")
			}
		})
	}
}
