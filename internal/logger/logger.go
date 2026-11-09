// Package logger construye loggers con una configuración compartida por la aplicación.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// Config define el nivel, el formato y el destino de los registros.
// Crear esta configuración no crea el logger ni escribe mensajes.
type Config struct {
	// Level es el mínimo que se registra: DEBUG < INFO < WARN < ERROR.
	// Con INFO, por ejemplo, los mensajes DEBUG se descartan.
	Level slog.Level

	// JSON selecciona JSON cuando es true y texto con clave=valor cuando es false.
	JSON bool

	// Output acepta cualquier io.Writer, como un archivo, un buffer o os.Stdout.
	// Si es nil, New usa la salida estándar.
	Output io.Writer
}

// DefaultConfig devuelve opciones para desarrollo: nivel INFO y texto en la salida estándar.
func DefaultConfig() Config {
	return Config{
		Level:  slog.LevelInfo,
		JSON:   false,
		Output: os.Stdout,
	}
}

// ProductionConfig usa JSON para facilitar el procesamiento de los registros.
// La aplicación debe elegirla explícitamente; no detecta el entorno automáticamente.
func ProductionConfig() Config {
	cfg := DefaultConfig()
	cfg.JSON = true
	return cfg
}

// New construye un logger; todavía no emite ningún mensaje.
// El logger recibe eventos y su slog.Handler los filtra, formatea y escribe.
// Este handler de registros es distinto de http.Handler, que atiende peticiones.
func New(cfg Config) *slog.Logger {
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level: cfg.Level,
		// Decisión de este proyecto: en DEBUG incluimos función, archivo y línea.
		AddSource: cfg.Level == slog.LevelDebug,
	}

	// Ambos formatos implementan slog.Handler; el resto usa la misma interfaz.
	var handler slog.Handler
	if cfg.JSON {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	return slog.New(handler)
}
