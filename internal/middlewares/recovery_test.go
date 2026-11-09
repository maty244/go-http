package middlewares_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"back/internal/middlewares"
)

func TestRecoveryPreservesAbortHandler(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	handler := middlewares.Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	response := httptest.NewRecorder()

	// El panic especial debe llegar a net/http sin transformarse en respuesta 500.
	defer func() {
		if recovered := recover(); recovered != http.ErrAbortHandler {
			t.Errorf("panic = %v; esperado http.ErrAbortHandler", recovered)
		}
		if response.Body.Len() != 0 || logs.Len() != 0 {
			t.Error("abortar la petición no debe escribir una respuesta ni registrar un panic común")
		}
	}()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
}
