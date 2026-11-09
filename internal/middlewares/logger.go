package middlewares

import (
	"log/slog"
	"net/http"
	"time"
)

// responseRecorder envuelve la respuesta para recordar su estado HTTP.
// ResponseWriter permite escribir el estado, pero no consultarlo después.
type responseRecorder struct {
	// Es composición, no herencia: los métodos no redefinidos, como Header,
	// se promueven desde la interfaz embebida. No agrega interfaces opcionales como Flusher.
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		// Si el handler no escribe nada, net/http termina la respuesta con 200.
		statusCode: http.StatusOK,
	}
}

// WriteHeader recuerda el primer estado definitivo y lo transmite al cliente.
func (rec *responseRecorder) WriteHeader(code int) {
	if rec.wroteHeader {
		return
	}

	// Los estados informativos, como 103 Early Hints, permiten enviar después
	// el estado definitivo. 101 es especial: cambia el protocolo de la conexión.
	if code >= 100 && code < 200 && code != http.StatusSwitchingProtocols {
		rec.ResponseWriter.WriteHeader(code)
		return
	}

	// Delegamos primero: si net/http rechaza un código inválido, no lo registramos.
	rec.ResponseWriter.WriteHeader(code)
	rec.statusCode = code
	rec.wroteHeader = true
}

// Write registra el 200 implícito cuando se escribe sin un WriteHeader previo.
func (rec *responseRecorder) Write(body []byte) (int, error) {
	if !rec.wroteHeader {
		rec.statusCode = http.StatusOK
		rec.wroteHeader = true
	}
	// Dejamos que el writer original envíe el 200 implícito y detecte Content-Type.
	return rec.ResponseWriter.Write(body)
}

// RequestLogger registra método, ruta, estado y duración de cada petición completada.
// Se coloca por fuera de Recovery para registrar también los errores recuperados.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	// Nivel 1: recibimos el logger. Nivel 2: recibimos el handler que vamos a envolver.
	return func(next http.Handler) http.Handler {
		// Nivel 3: este código se ejecuta por cada petición.
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := newResponseRecorder(w)

			// Al retornar, el handler ya terminó de escribir su respuesta.
			next.ServeHTTP(rec, r)

			logger.InfoContext(r.Context(), "petición completada",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.statusCode),
				slog.Duration("duration", time.Since(start)),
				// RemoteAddr incluye IP y puerto del par conectado, que podría ser un proxy.
				slog.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}
