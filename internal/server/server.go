package server

import (
	"net/http"
	"time"
)

// Server es nuestra propia caja que junta las dos piezas clave del servidor:
// 1. httpServer: El "edificio" que abre el puerto (ej. :8080) y recibe las conexiones de internet.
// 2. mux (ServeMux): El "recepcionista" (o router) que mira qué URL pidió el usuario (ej. /health)
//    y decide a qué trabajador (Handler) pasarle el pedido.
type Server struct {
	httpServer *http.Server
	mux        *http.ServeMux
}

// New construye y configura el servidor en la dirección indicada (ej. ":8080"),
// pero todavía no lo enciende hasta que llamemos a Start().
func New(addr string) *Server {
	// 1. Creamos al recepcionista (ServeMux) con su lista de rutas vacía por ahora.
	mux := http.NewServeMux()

	// 2. Configuramos el edificio (http.Server).
	// Los "Timeouts" son tiempos máximos de seguridad: evitan que un cliente con internet
	// muy lento o malintencionado deje una conexión colgada para siempre y trabe tu servidor.
	httpServer := &http.Server{
		Addr:              addr,             // Puerto donde escuchará (ej. ":8080").
		Handler:           mux,              // Quién atiende en la puerta principal (al inicio es el recepcionista mux).
		ReadTimeout:       10 * time.Second, // Tiempo máximo para leer todo lo que envía el cliente.
		WriteTimeout:      10 * time.Second, // Tiempo máximo para enviarle nuestra respuesta.
		IdleTimeout:       60 * time.Second, // Tiempo máximo que se mantiene abierta una conexión sin usarse.
		ReadHeaderTimeout: 5 * time.Second,  // Tiempo máximo para leer las cabeceras (Headers) de la petición.
	}

	// Guardamos tanto el httpServer como el mux en nuestra estructura Server:
	// - Necesitamos mux a mano para poder agregarle rutas después con RegisterRoutes().
	// - Necesitamos httpServer para encender el servidor con Start() o envolverlo con middlewares en Use().
	return &Server{httpServer: httpServer, mux: mux}
}

// Start abre las puertas del servidor y se queda escuchando peticiones sin parar.
// ListenAndServe() es la función oficial de Go que pone el servidor en marcha.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// RegisterRoutes le enseña una nueva ruta al recepcionista (mux).
// - pattern: El método y la dirección URL, por ejemplo "GET /health".
// - handler: El encargado que responderá cuando alguien entre a esa ruta.
func (s *Server) RegisterRoutes(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// ServeHTTP hace que nuestro Server también pueda usarse directamente como un http.Handler
// si en algún momento lo necesitamos (por ejemplo, al hacer pruebas), pasándole el pedido al mux.
func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.mux.ServeHTTP(writer, request)
}

// Use coloca un "Middleware" (como un guardia de seguridad o filtro) en la entrada principal del servidor.
// Imagina que server.httpServer.Handler es quien está parado en la puerta principal (al principio es el mux).
// Al hacer middleware(server.httpServer.Handler), envolvemos a quien estaba en la puerta dentro del middleware.
// Así, cuando llega una petición: primero la recibe el middleware y luego este se la pasa al mux.
func (server *Server) Use(middleware func(http.Handler) http.Handler) {
	server.httpServer.Handler = middleware(server.httpServer.Handler)
}
