package handler

import (
	"encoding/json"
	"net/http"
)

// HealthHandler es el "trabajador" encargado de responder si el servidor sigue vivo y funcionando.
// Está vacío (struct{}) porque por ahora no necesita guardar ningún dato adentro (como una base de datos).
type HealthHandler struct{}

// NewHealthHandler construye y devuelve este encargado para poder asignarlo a una ruta en main.go.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// healthResponse es el molde de los datos que le vamos a responder al usuario.
// - Los nombres empiezan con Mayúscula (Status, Version) porque en Go solo así el paquete "json" tiene permiso de leerlos.
// - Las etiquetas `json:"status"` le dicen a Go: "cuando conviertas esto a texto JSON, escríbelo en minúscula".
type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ServeHTTP es LA FUNCIÓN MÁS IMPORTANTE de un Handler en Go.
// Regla de oro de net/http:
// Cualquier estructura que tenga un método llamado exactamente:
//
//	ServeHTTP(writer http.ResponseWriter, request *http.Request)
//
// automáticamente se convierte en un "http.Handler" válido para Go y puede atender rutas.
//
// Siempre recibe dos cosas:
//  1. writer  (http.ResponseWriter) -> La "hoja en blanco" donde escribimos lo que vamos a responderle al cliente.
//  2. request (*http.Request)       -> El "sobre con el pedido" que nos mandó el cliente (qué URL pidió, qué método usó, etc.).
func (handle *HealthHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// ¿Qué viene dentro de "request"? Todo lo que envió el cliente.
	// Por ejemplo, si alguien entra a: http://localhost:8080/health?nombre=Ana
	//   request.Method                    -> "GET" (el verbo HTTP)
	//   request.URL.Path                  -> "/health" (la ruta)
	//   request.URL.Query().Get("nombre") -> "Ana" (parámetros en la URL)
	//   request.Header.Get("Accept")      -> cabeceras que mandó el navegador o cliente
	//
	// Aunque aquí no leemos nada de "request" porque siempre contestamos lo mismo,
	// Go nos obliga a ponerlo en los paréntesis para cumplir con la regla de ServeHTTP.

	// 1. Preparamos los datos que queremos enviar de vuelta.
	resp := healthResponse{Status: "ok", Version: "1.0.0"}

	// 2. Configuramos y enviamos la respuesta usando "writer" EN ESTE ORDEN EXACTO:
	//    Paso A - Cabeceras (Header): Avisamos al cliente qué tipo de contenido le vamos a mandar (JSON).
	//    Paso B - Código de Estado (WriteHeader): Enviamos el código HTTP 200 (StatusOK = todo salió bien).
	//    ¡OJO! Siempre debes configurar los Headers ANTES de llamar a WriteHeader o de escribir el cuerpo.
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK) // Envía el código HTTP 200.

	//    Paso C - Cuerpo de la respuesta (Body):
	//    json.NewEncoder(writer).Encode(resp) transforma nuestra estructura "resp" en texto JSON:
	//    {"status":"ok","version":"1.0.0"} y lo envía directamente al cliente a través de "writer".
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		return
	}
}
