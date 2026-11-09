package main

import (
	"back/internal/handler"
	"back/internal/middlewares"
	"back/internal/server"
	"log"
	"log/slog"
	"os"
)

func main() {

	// 1. EL LIBRO DE REGISTROS (Logger):
	// Crea una herramienta para imprimir mensajes en la consola en formato JSON.
	// Nos sirve para ver en pantalla qué está pasando en el servidor.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// 2. CREAR EL SERVIDOR (El edificio):
	// Preparamos el servidor para que escuche en el puerto ":8080" de tu computadora
	// (por ejemplo, al entrar a http://localhost:8080). Todavía no está encendido.
	srv := server.New(":8080")

	// 3. AGREGAR UN MIDDLEWARE (El guardia en la entrada):
	// Con Use() ponemos al middleware Logger en la puerta principal del servidor.
	// Así, TODA petición que llegue pasará primero por este registrador antes
	// de llegar a cualquier ruta.
	srv.Use(middlewares.Logger(logger))

	// 4. CREAR EL ENCARGADO DE LA RUTA (El trabajador / Handler):
	// Creamos el encargado que sabe cómo responder cuando pregunten por la salud del servidor.
	healthHandler := handler.NewHealthHandler()

	// 5. REGISTRAR LAS RUTAS (El mapa para el recepcionista):
	// Le decimos al servidor:
	// - Si alguien pide "GET /health", mándalo con healthHandler.
	// - Si alguien pide "GET /ready", mándalo también con healthHandler.
	// (Desde Go 1.22 puedes escribir el método HTTP como "GET " antes de la ruta).
	srv.RegisterRoutes("GET /health", healthHandler)
	srv.RegisterRoutes("GET /ready", healthHandler)

	log.Println("Iniciando servidor en el puerto :8080")

	logger.Info("Iniciando servidor en el puerto :8080", slog.String("addr", ":8080"))

	// 6. ENCENDER EL SERVIDOR (Abrir las puertas):
	// Start() deja el programa corriendo y esperando peticiones sin cerrarse.
	// Si ocurre algún problema grave al arrancar (por ejemplo, el puerto 8080 ya está ocupado),
	// log.Fatal muestra el error y detiene el programa.
	if err := srv.Start(); err != nil {
		log.Fatal("error al iniciar o ejecutar el servidor: ", err)
	}
}
