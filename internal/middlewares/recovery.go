package middlewares

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery captura panics del siguiente handler en la misma goroutine.
// Registra el detalle y responde con 500 si la respuesta aún no había comenzado.
// No sustituye el manejo de errores normales con if err != nil ni deshace cambios.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	// Esta función recuerda el logger (closure) y envuelve al siguiente handler.
	return func(next http.Handler) http.Handler {
		// El cuerpo de HandlerFunc se ejecuta una vez por petición.
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Dejamos listo el defer ANTES de llamar a next. Se ejecutará al salir,
			// tanto en una salida normal como cuando un panic interrumpa las llamadas.
			defer func() {
				// recover debe llamarse directamente desde la función diferida.
				// Su valor puede ser un string u otro tipo: no necesariamente un error.
				recovered := recover()
				if recovered == nil {
					return
				}

				// net/http usa este valor especial para abortar una respuesta.
				// Lo dejamos llegar al servidor en lugar de convertirlo en un 500.
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}

				// El stack muestra archivos, líneas y llamadas de esta goroutine.
				// El detalle queda en los logs; el cliente recibe un mensaje genérico.
				logger.ErrorContext(r.Context(), "panic recuperado",
					slog.Any("error", recovered),
					slog.String("stack", string(debug.Stack())),
					slog.String("path", r.URL.Path),
				)

				// Si ya se envió parte de la respuesta, no se puede cambiar su estado
				// ni deshacer lo enviado. Este middleware no almacena el cuerpo en memoria.
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}()

			// Si next hace panic, el defer lo captura; no se reanuda la línea que falló.
			next.ServeHTTP(w, r)
		})
	}
}
