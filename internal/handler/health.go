package handler

import (
	"encoding/json"
	"net/http"
)

// HealthHandler indica que la aplicación puede responder peticiones HTTP.
// No comprueba bases de datos ni otros servicios. No necesita guardar estado.
type HealthHandler struct{}

// NewHealthHandler crea el handler que registramos en las rutas de salud.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// healthResponse define el cuerpo JSON. Los campos exportados (mayúscula inicial)
// permiten que encoding/json los lea; las etiquetas eligen sus nombres en el JSON.
type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ServeHTTP implementa http.Handler sin necesidad de declarar "implements".
// w escribe la respuesta y r contiene la petición; son nombres habituales en net/http.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response := healthResponse{Status: "ok", Version: "1.0.0"}

	// Las cabeceras van antes del cuerpo. Al escribir sin WriteHeader, Go envía 200 OK.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Con estos campos string no esperamos un fallo de serialización, pero puede
		// fallar la escritura al cliente. No intentamos enviar otra respuesta encima.
		return
	}
}
