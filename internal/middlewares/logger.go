package middlewares

import (
	"log/slog"
	"net/http"
	"time"
)

// ¿POR QUÉ EXISTE responseRecorder? (El truco del "papel carbón"):
// En Go, cuando un Handler envía un código de estado (por ejemplo, writer.WriteHeader(200)),
// el http.ResponseWriter original lo manda directo al cliente pero NO tiene ninguna variable
// donde nosotros podamos leer después "¿qué código se envió?".
//
// Entonces creamos "responseRecorder": es un envoltorio alrededor del ResponseWriter original
// que funciona como un "papel carbón": deja pasar todo hacia el cliente, pero antes anota en
// su libreta (statusCode) qué número de estado se envió (200, 404, 500, etc.) para que el Logger lo sepa.
type responseRecorder struct {
	// Al poner http.ResponseWriter aquí adentro sin nombre, nuestro responseRecorder
	// hereda automáticamente todas sus capacidades y puede hacerse pasar por un ResponseWriter normal.
	http.ResponseWriter
	statusCode  int  // Aquí anotamos el código HTTP (ej. 200) para leerlo al final en el log.
	wroteHeader bool // Nos avisa si ya se envió el código una vez, para no enviarlo repetido por error.
}

// newResponseRecorder crea nuestro "papel carbón" envolviendo al writer original.
// Por defecto le ponemos 200 (StatusOK), porque en Go si un handler nunca llama a WriteHeader,
// el estado automático siempre es 200.
func newResponseRecorder(writer http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: writer,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader intercepta el momento en que el handler dice: writer.WriteHeader(200).
// 1. Revisa si ya se había enviado antes (wroteHeader).
// 2. Guarda una copia del código en rr.statusCode.
// 3. Se lo pasa al ResponseWriter verdadero para que llegue al cliente.
func (rr *responseRecorder) WriteHeader(code int) {
	if rr.wroteHeader {
		return
	}

	rr.statusCode = code
	rr.wroteHeader = true
	rr.ResponseWriter.WriteHeader(code)
}

// Write intercepta cuando el handler escribe el contenido final (por ejemplo, el JSON).
// Si el handler se olvidó de llamar a WriteHeader antes de escribir los datos,
// nosotros marcamos el 200 OK primero y luego enviamos los datos al cliente.
func (rr *responseRecorder) Write(body []byte) (int, error) {
	if !rr.wroteHeader {
		rr.WriteHeader(http.StatusOK)
	}
	return rr.ResponseWriter.Write(body)
}

// ¿QUÉ ES UN MIDDLEWARE? (El guardia de seguridad en la entrada):
// Un Middleware es un intermediario que se para en la puerta ANTES que tus handlers.
// Cuando llega un cliente:
//
//	[Cliente pide GET /health]
//	        │
//	        ▼
//	1. Entra al Middleware Logger (prende un cronómetro y prepara el "papel carbón" rec)
//	        │
//	        ▼
//	2. Llama a next.ServeHTTP(rec, request) -> Deja pasar el pedido al Recepcionista (mux) y al HealthHandler
//	        │
//	        ▼
//	3. HealthHandler responde {"status":"ok"} con código 200 (y "rec" anota ese 200)
//	        │
//	        ▼
//	4. Vuelve al Middleware Logger (detiene el cronómetro e imprime en consola qué pasó y cuánto tardó)
//
// ¿Por qué hay 3 funciones una dentro de otra? (No te asustes, es una receta estándar en Go):
//   - Nivel 1: Logger(logger) -> Recibe tu herramienta de logs cuando arranca main.go.
//   - Nivel 2: func(next http.Handler) -> Recibe a quién debe envolver/proteger (el recepcionista mux).
//   - Nivel 3: http.HandlerFunc(...) -> Es el guardia en vivo: este código sí se ejecuta en CADA petición.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	// NIVEL 2: Esta función se ejecuta una sola vez cuando haces srv.Use(...) en main.go.
	// "next" es el siguiente paso en la fila (en tu proyecto, es el recepcionista mux).
	return func(next http.Handler) http.Handler {
		// NIVEL 3: http.HandlerFunc convierte una función común en un http.Handler oficial.
		// Todo lo que está aquí adentro se ejecuta CADA VEZ que un usuario entra a cualquier URL.
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// --- PASO A: ANTES DE ATENDER EL PEDIDO ---
			// 1. Miramos el reloj para saber a qué hora exacta entró la petición.
			start := time.Now()
			// 2. Envolvemos el writer original con nuestro "papel carbón" (rec) para espiar qué código responde.
			rec := newResponseRecorder(writer)

			// --- PASO B: DEJAR PASAR EL PEDIDO ---
			// Le pasamos el pedido al siguiente encargado (next).
			// El programa se queda esperando en esta línea hasta que el HealthHandler termine de responderle al cliente.
			next.ServeHTTP(rec, request)

			// --- PASO C: DESPUÉS DE ATENDER EL PEDIDO ---
			// Ya le respondimos al cliente. Ahora el guardia anota el resumen en la consola:
			logger.InfoContext(request.Context(), "request completado",
				slog.String("method", request.Method),          // Qué verbo usó (ej. "GET").
				slog.String("path", request.URL.Path),          // A qué ruta entró (ej. "/health").
				slog.Int("status", rec.statusCode),             // Qué código devolvió el handler (ej. 200).
				slog.Duration("duration", time.Since(start)),   // Cuánto tiempo tardó desde "start" hasta ahora.
				slog.String("remote_addr", request.RemoteAddr), // Desde qué dirección IP nos llamaron.
			)
		})
	}
}
