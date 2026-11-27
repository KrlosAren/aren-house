package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// Video en vivo: proxy WebSocket hacia go2rtc (/api/ws?src=...), que envía H.264 por MSE.
//
//   - go2rtc ya recibe el stream para Frigate: cada espectador es otra salida del mismo
//     stream, NO otra conexión a la Tapo (que acepta una sola).
//   - Por el proxy la PWA queda en el mismo origen (mismo HTTPS de Tailscale) y solo se
//     permiten los streams de la lista, no cualquier `src` de go2rtc.
type Go2RTC struct {
	URL     *url.URL          // http://go2rtc.cameras.svc.cluster.local:1984
	Streams map[string]string // calidad → stream de go2rtc
}

func (g *Go2RTC) vivo(w http.ResponseWriter, r *http.Request) {
	calidad := r.URL.Query().Get("calidad")
	src, ok := g.Streams[calidad]
	if !ok {
		calidad, src = "sub", g.Streams["sub"]
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(g.URL)
			pr.Out.URL.Path = "/api/ws"
			pr.Out.URL.RawQuery = url.Values{"src": {src}}.Encode()
			// go2rtc rechaza WebSockets cuyo Origin no coincide con su Host;
			// el origen real (casa.<tailnet>) ya lo controla Tailscale.
			pr.Out.Header.Del("Origin")
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("vivo %s: go2rtc no responde: %v", calidad, err)
			http.Error(w, "go2rtc no responde", http.StatusBadGateway)
		},
	}

	inicio := time.Now()
	log.Printf("vivo %s (%s): abierto desde %s", calidad, src, r.RemoteAddr)
	proxy.ServeHTTP(w, r) // bloquea mientras dure el WebSocket
	log.Printf("vivo %s: cerrado tras %s", calidad, time.Since(inicio).Round(time.Second))
}
