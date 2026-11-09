# go-http

Servidor HTTP de estudio escrito con la biblioteca estándar de Go. Incluye rutas de salud, registros estructurados, recuperación de panics y apagado ordenado.

## Ejecutar y comprobar

Desde la raíz del proyecto:

```powershell
go run ./cmd
```

En otra terminal:

```powershell
curl.exe -i http://localhost:8080/health
curl.exe -i http://localhost:8080/ready
curl.exe -i http://localhost:8080/panic
```

Las dos primeras rutas responden con JSON y estado 200. `/panic` provoca un panic de prueba: el cliente recibe 500 y el detalle queda en los logs. Después puedes consultar `/health` otra vez.

Para probar Ctrl+C directamente sobre el ejecutable en Windows:

```powershell
go build -o "$env:TEMP\go-http-study.exe" ./cmd
& "$env:TEMP\go-http-study.exe"
```

Presiona Ctrl+C en esa terminal. El servidor deja de aceptar conexiones y espera hasta 30 segundos a las peticiones activas. Si no queda trabajo, sale antes.

```powershell
go test ./...
go vet ./...
```

Los tests verifican respuestas, estados registrados, recuperación de panics, cabeceras informativas y el error de puerto ocupado. No simulan las señales de apagado del sistema operativo.

## Orden de lectura

1. [cmd/main.go](cmd/main.go): conecta las piezas. Configura, registra rutas e inicia el servidor.
2. [internal/handler/health.go](internal/handler/health.go): muestra una petición sencilla que produce JSON.
3. [internal/logger/logger.go](internal/logger/logger.go): elige nivel, formato y destino de los registros.
4. [internal/middlewares/logger.go](internal/middlewares/logger.go): envuelve la respuesta y mide la petición.
5. [internal/middlewares/recovery.go](internal/middlewares/recovery.go): relaciona `defer`, `panic` y `recover`.
6. [internal/server/server.go](internal/server/server.go): configura HTTP y coordina el apagado.
7. [internal/server/server_test.go](internal/server/server_test.go): comprueba la cadena completa sin abrir el puerto de la aplicación.

Los comentarios del código explican las decisiones y los conceptos junto a su uso. Esta guía reúne los ejemplos para poder leer las funciones sin interrupciones largas.

## Nombres y estilo

| Antes | Ahora | Motivo |
| --- | --- | --- |
| `DefaultConfing` | `DefaultConfig` | Corrige la escritura de “configuración” en el identificador. |
| `ProductionConfing` | `ProductionConfig` | Usa el mismo nombre `Config` que el tipo. |
| `RegisterRoutes` | `RegisterRoute` | Cada llamada registra una sola ruta. |
| `middlewares.Logger` | `middlewares.RequestLogger` | Distingue el registro de peticiones de la construcción del logger. |
| `err := recover()` | `recovered := recover()` | El valor de un panic no tiene por qué implementar `error`. |

Usamos `w` y `r` en handlers, `s` como receptor de `Server` y nombres más descriptivos donde el significado no es tan inmediato. Los identificadores están en inglés y las explicaciones en español. Los nombres exportados empiezan con mayúscula; los internos, con minúscula.

Estas decisiones siguen las convenciones de nombres y formato de [Effective Go](https://go.dev/doc/effective_go#names). Los comentarios de declaraciones empiezan por su nombre, según [Go Doc Comments](https://go.dev/doc/comment). La cantidad de explicación se adapta aquí al objetivo de aprendizaje.

## Cómo leer un middleware

```go
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler
```

Hay tres momentos:

1. `Recovery(appLogger)` recibe el logger y devuelve una función. Esa función recuerda el logger: es un cierre o *closure*.
2. `srv.Use(...)` le entrega el handler actual y guarda el nuevo envoltorio.
3. Con cada petición, el `http.HandlerFunc` ejecuta su cuerpo y llama a `next.ServeHTTP`.

En este proyecto la entrada sigue este recorrido:

```text
Petición -> RequestLogger -> Recovery -> router -> handler
```

`Use` envuelve lo que ya existe, así que el último agregado entra primero. Recovery transforma un panic normal en un 500 y retorna; entonces RequestLogger puede registrar ese estado. Recovery no cubre al RequestLogger exterior.

`responseRecorder` usa composición: contiene un `http.ResponseWriter`. Sus métodos `WriteHeader` y `Write` guardan el estado y delegan la escritura. Los estados informativos, como 103, permiten enviar después un estado definitivo. El envoltorio también debe conservar la detección automática de `Content-Type` al escribir. Estos comportamientos pertenecen al contrato de [http.ResponseWriter](https://pkg.go.dev/net/http#ResponseWriter).

## Logger: opciones, construcción y uso

```go
cfg := logger.DefaultConfig()
cfg.Level = slog.LevelDebug
appLogger := logger.New(cfg)
appLogger.Debug("configuración cargada", slog.String("addr", ":8080"))
```

`Config` guarda las opciones; `New` construye el logger; `Debug` registra un evento si el nivel lo permite. Para JSON, usa `logger.ProductionConfig()`. No se selecciona automáticamente al desplegar.

El paquete `logger` y la variable `appLogger` tienen nombres distintos para no ocultarse. El handler de `slog` procesa registros; el handler de `net/http` procesa peticiones.

## Recovery y apagado: dos mecanismos distintos

Recovery actúa sobre una petición que hace panic. El `defer` queda preparado antes de llamar a `next`; `recover` se ejecuta dentro de ese defer y detiene la propagación. No vuelve a ejecutar la línea que falló, no revierte cambios y no captura panics de otras goroutines. El valor especial `http.ErrAbortHandler` se propaga para que `net/http` aborte la respuesta.

El apagado ordenado actúa sobre el servidor completo:

```text
Ctrl+C -> señal en quit -> select despierta -> Shutdown -> retorno a main
```

`ListenAndServe` trabaja en otra goroutine porque bloquea mientras atiende conexiones. Los canales comunican un fallo o una señal a `Start`. El contexto de apagado fija un límite a la espera, y `defer cancel()` libera sus recursos.

`Shutdown` cierra la entrada y espera a las conexiones activas; si vence el plazo, devuelve un error y no las fuerza a cerrar por sí mismo. Aquí `main` recibe ese error y termina el proceso. Las conexiones tomadas mediante `Hijack`, como ciertos WebSockets, y las tareas independientes requieren su propia coordinación. Véase [http.Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown).

## Ejercicios y próximos pasos

1. Cambia INFO por DEBUG y observa el origen del registro: archivo, función y línea.
2. Prueba un `POST /health` y una ruta inexistente. Compara los estados 405 y 404 con los logs.
3. Lee un test y predice la respuesta antes de ejecutarlo. Luego altera el estado esperado para entender el mensaje de fallo.
4. Como siguiente mejora de diseño, pasa un `context.Context` a `Start` y gestiona las señales en `main`. Esto permitiría probar el apagado sin enviar señales reales al proceso.
5. Cuando agregues una base de datos, define qué debe comprobar `/ready`. Actualmente es un alias de `/health` y no verifica dependencias.

Antes de desplegar, retira la ruta de demostración `/panic` y adapta dirección, tiempos y configuración a las necesidades reales. El registrador actual cubre HTTP básico: para streaming o WebSockets habría que conservar también las interfaces opcionales del writer. Si una respuesta ya comenzó, Recovery no puede cambiar su estado ni retirar los bytes enviados.
