package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const shutdownTimeout = 30 * time.Second

// Server reúne el servidor HTTP, el router y el logger.
// El router decide qué handler atiende cada ruta; httpServer gestiona las conexiones.
type Server struct {
	httpServer *http.Server
	mux        *http.ServeMux
	logger     *slog.Logger
}

// New configura el servidor, pero no abre el puerto hasta llamar a Start.
// El logger debe estar inicializado.
func New(addr string, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       10 * time.Second, // Lectura de la petición, incluido el cuerpo.
		WriteTimeout:      10 * time.Second, // Escritura de la respuesta; no detiene el código del handler.
		IdleTimeout:       60 * time.Second, // Espera de otra petición en una conexión reutilizable.
		ReadHeaderTimeout: 5 * time.Second,  // Lectura de las cabeceras.
	}

	return &Server{
		httpServer: httpServer,
		mux:        mux,
		logger:     logger,
	}
}

// Start atiende peticiones hasta un error o una señal de salida.
// El apagado ordenado deja de aceptar conexiones y espera a las peticiones activas.
// Devuelve nil al cerrar correctamente; el servidor no se puede reiniciar después.
func (s *Server) Start() error {
	// 1. Escuchar señales. El canal tiene espacio para una señal pendiente.
	// SIGINT suele venir de Ctrl+C; SIGTERM es habitual en despliegues Unix/Linux.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit) // Retira la suscripción cuando Start retorna.

	// 2. Servir en otra goroutine para poder esperar señales en la actual.
	// El buffer permite enviar el error aunque Start todavía no lo esté recibiendo.
	serveErr := make(chan error, 1)
	go func() {
		// Anuncia el intento: ListenAndServe todavía puede fallar al abrir el puerto.
		s.logger.Info("iniciando servidor", slog.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	// 3. Esperar un evento. <- recibe del canal; select espera hasta que haya uno listo.
	select {
	case err := <-serveErr:
		// %w conserva el error original para poder inspeccionarlo con errors.Is/As.
		return fmt.Errorf("servir HTTP: %w", err)
	case sig := <-quit:
		s.logger.Info("iniciando apagado ordenado", slog.String("signal", sig.String()))
	}

	// 4. Crear un plazo independiente de cualquier petición HTTP.
	// cancel libera los recursos del contexto si terminamos antes de los 30 segundos.
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// 5. Cerrar la entrada y esperar. ListenAndServe retorna ErrServerClosed,
	// pero Start sigue aquí hasta que termine Shutdown o venza el plazo.
	if err := s.httpServer.Shutdown(ctx); err != nil {
		// Shutdown no fuerza las conexiones activas al vencer el plazo.
		// En esta aplicación main recibe el error y termina el proceso con os.Exit(1).
		return fmt.Errorf("apagar servidor HTTP: %w", err)
	}

	s.logger.Info("servidor detenido")
	return nil
}

// RegisterRoute asocia un patrón, como "GET /health", con un handler.
// Las rutas deben configurarse antes de iniciar el servidor.
func (s *Server) RegisterRoute(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// ServeHTTP implementa http.Handler y usa la misma cadena de middlewares que Start.
// Esto permite probar las rutas completas sin abrir el puerto de la aplicación.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.httpServer.Handler.ServeHTTP(w, r)
}

// Use envuelve el handler actual. El último middleware agregado se ejecuta primero.
// Los middlewares deben configurarse antes de iniciar el servidor.
func (s *Server) Use(middleware func(http.Handler) http.Handler) {
	s.httpServer.Handler = middleware(s.httpServer.Handler)
}
