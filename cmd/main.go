package main

import (
	"log/slog"
	"net/http"
	"os"

	"back/internal/handler"
	"back/internal/logger"
	"back/internal/middlewares"
	"back/internal/server"
)

func main() {
	// 1. Configurar el logger. El paquete logger construye la instancia appLogger.
	// DefaultConfig devuelve las opciones; New construye el logger con ellas.
	appLogger := logger.New(logger.DefaultConfig())

	// 2. Preparar el servidor. New todavía no abre el puerto.
	srv := server.New(":8080", appLogger)

	// 3. Envolver el router. El último middleware agregado se ejecuta primero:
	// RequestLogger -> Recovery -> router -> handler de la ruta.
	// Así, RequestLogger también registra el 500 que escribe Recovery ante un panic.
	srv.Use(middlewares.Recovery(appLogger))
	srv.Use(middlewares.RequestLogger(appLogger))

	// 4. Registrar las rutas. Cada llamada agrega un patrón y su handler.
	healthHandler := handler.NewHealthHandler()
	srv.RegisterRoute("GET /health", healthHandler)
	// Por ahora /ready responde lo mismo que /health: no comprueba dependencias.
	srv.RegisterRoute("GET /ready", healthHandler)

	// Ruta de estudio: permite observar el recovery y luego volver a probar /health.
	// Debe retirarse antes de publicar una aplicación real.
	srv.RegisterRoute("GET /panic", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("esto es una prueba de recovery")
	}))

	// 5. Atender peticiones hasta recibir una señal de salida o un error.
	// Start espera el apagado ordenado antes de volver. nil significa cierre correcto.
	if err := srv.Start(); err != nil {
		// Registramos el error aquí una sola vez. El código 1 indica fallo al sistema.
		// os.Exit no ejecuta defer pendientes; los de Start ya se ejecutaron al retornar.
		appLogger.Error("el servidor terminó con un error", slog.Any("error", err))
		os.Exit(1)
	}
}
