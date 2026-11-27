package main

import (
	"net/http"
	"net/url"
)

// Protección contra peticiones de OTROS sitios usando el navegador del usuario (CSRF).
//
// Tailscale decide QUÉ DISPOSITIVO llega a casa, pero el navegador de ese dispositivo
// puede abrir cualquier página: sin esto, una web maliciosa podría hacer un POST a
// /api/camara/deteccion (desarmar la cámara) o abrir el WebSocket del vivo y LEER el video
// (los WebSockets no están sujetos a CORS: "Cross-Site WebSocket Hijacking").
//
//   - POST/PUT/DELETE: http.CrossOriginProtection (Go 1.25), vía Sec-Fetch-Site / Origin.
//   - WebSocket del vivo (es un GET, que CrossOriginProtection deja pasar): mismoOrigen().
//
// Clientes que no son navegador (Atajos de iOS, curl) no mandan Sec-Fetch-Site ni Origin
// y se aceptan: no pueden ser víctimas de CSRF.

func protegerOrigen(h http.Handler) http.Handler {
	return http.NewCrossOriginProtection().Handler(h)
}

// mismoOrigen replica el criterio de CrossOriginProtection para un GET (WebSocket).
func mismoOrigen(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-origin", "none":
		return true
	case "":
		// navegador viejo o cliente que no es navegador: decidir por Origin
	default: // "cross-site", "same-site"
		return false
	}
	origen := r.Header.Get("Origin")
	if origen == "" {
		return true // sin Origin = no es una petición iniciada por una página web
	}
	u, err := url.Parse(origen)
	return err == nil && u.Host == r.Host
}
