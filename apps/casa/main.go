// casa: API + PWA para ver y controlar los dispositivos de la casa (cámara, focos, homelab).
// Acceso solo por Tailscale Service svc:casa (sin auth propia, igual que Frigate/ntfy; ADR-017).
package main

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed all:web/dist
var webDist embed.FS

type App struct {
	frigate *Frigate
	prom    *Prometheus
	go2rtc  *Go2RTC
	focos   []Foco
}

func env(clave, defecto string) string {
	if v := os.Getenv(clave); v != "" {
		return v
	}
	return defecto
}

// FOCOS="living:Living:192.168.1.165,habitacion:Habitación principal:192.168.1.166"
func parseFocos(s string) ([]Foco, error) {
	var focos []Foco
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item == "" {
			continue
		}
		p := strings.Split(item, ":")
		if len(p) != 3 {
			return nil, errors.New("FOCOS: formato id:nombre:ip")
		}
		focos = append(focos, Foco{ID: p[0], Nombre: p[1], IP: p[2]})
	}
	return focos, nil
}

func responder(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fallo(w http.ResponseWriter, status int, err error) {
	responder(w, status, map[string]string{"error": err.Error()})
}

// ------------------------------------------------------------------ /api/estado

type EstadoFoco struct {
	Foco
	Alcanzable bool   `json:"alcanzable"`
	Error      string `json:"error,omitempty"`
	*Pilot
}

type Estado struct {
	Hora         time.Time      `json:"hora"`
	Camara       *EstadoCamara  `json:"camara,omitempty"`
	CamaraError  string         `json:"camara_error,omitempty"`
	Focos        []EstadoFoco   `json:"focos"`
	Homelab      *EstadoHomelab `json:"homelab,omitempty"`
	HomelabError string         `json:"homelab_error,omitempty"`
}

// Cada fuente se consulta en paralelo y falla por separado: si Prometheus no responde,
// la cámara y los focos se siguen viendo.
func (a *App) estado(w http.ResponseWriter, r *http.Request) {
	e := Estado{Hora: time.Now(), Focos: make([]EstadoFoco, len(a.focos))}
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if c, err := a.frigate.Estado(); err != nil {
			e.CamaraError = err.Error()
		} else {
			e.Camara = c
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if h, err := a.prom.Estado(); err != nil {
			e.HomelabError = err.Error()
		} else {
			e.Homelab = h
		}
	}()

	for i, f := range a.focos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ef := EstadoFoco{Foco: f}
			if p, err := wizEstado(f.IP); err != nil {
				ef.Error = err.Error()
			} else {
				ef.Alcanzable, ef.Pilot = true, p
			}
			e.Focos[i] = ef
		}()
	}

	wg.Wait()
	responder(w, http.StatusOK, e)
}

// ------------------------------------------------------------------ cámara

func (a *App) deteccion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Activa *bool `json:"activa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Activa == nil {
		fallo(w, http.StatusBadRequest, errors.New(`body esperado: {"activa": true|false}`))
		return
	}
	if err := a.frigate.SetDeteccion(*body.Activa); err != nil {
		fallo(w, http.StatusBadGateway, err)
		return
	}
	log.Printf("cámara: detección → %v (desde %s)", *body.Activa, r.RemoteAddr)
	// /api/stats tarda ~15 s en reflejarlo: la PWA espera hasta verlo confirmado
	responder(w, http.StatusAccepted, map[string]bool{"activa": *body.Activa})
}

func (a *App) imagenCamara(w http.ResponseWriter, r *http.Request) {
	a.frigate.proxyImagen(w, "/api/"+a.frigate.Camara+"/latest.jpg?h=480")
}

func (a *App) fotoEvento(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !idEvento.MatchString(id) {
		fallo(w, http.StatusBadRequest, errors.New("id de evento inválido"))
		return
	}
	a.frigate.proxyImagen(w, "/api/events/"+id+"/snapshot.jpg?bbox=1")
}

// ------------------------------------------------------------------ focos

func (a *App) cambiarFoco(w http.ResponseWriter, r *http.Request) {
	var foco *Foco
	for i := range a.focos {
		if a.focos[i].ID == r.PathValue("id") {
			foco = &a.focos[i]
		}
	}
	if foco == nil {
		fallo(w, http.StatusNotFound, errors.New("foco no existe"))
		return
	}
	var c CambioFoco
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		fallo(w, http.StatusBadRequest, err)
		return
	}
	p, err := wizCambiar(foco.IP, c)
	if err != nil {
		status := http.StatusBadGateway
		if !errors.Is(err, errSinRespuesta) && !strings.HasPrefix(err.Error(), "el foco") {
			status = http.StatusBadRequest // validación
		}
		fallo(w, status, err)
		return
	}
	cambio, _ := json.Marshal(c) // %+v mostraría punteros, no valores
	log.Printf("foco %s: %s (desde %s)", foco.ID, cambio, r.RemoteAddr)
	responder(w, http.StatusOK, EstadoFoco{Foco: *foco, Alcanzable: true, Pilot: p})
}

// ------------------------------------------------------------------ main

func main() {
	// Go no conoce .webmanifest (y distroless no trae /etc/mime.types)
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")

	focos, err := parseFocos(env("FOCOS", ""))
	if err != nil {
		log.Fatal(err)
	}
	cli := &http.Client{Timeout: 4 * time.Second}
	camara := env("CAMARA", "casona")
	go2rtcURL, err := url.Parse(env("GO2RTC_URL", "http://go2rtc.cameras.svc.cluster.local:1984"))
	if err != nil {
		log.Fatalf("GO2RTC_URL: %v", err)
	}
	app := &App{
		frigate: &Frigate{URL: env("FRIGATE_URL", "http://frigate.cameras.svc.cluster.local:5000"), Camara: camara, http: cli},
		prom:    &Prometheus{URL: env("PROMETHEUS_URL", "http://prometheus.monitoring.svc.cluster.local:9090"), http: cli},
		// sub = 720p (~330 kbps), hd = 3MP (~1,2 Mbps); mismos nombres que en go2rtc.yaml
		go2rtc: &Go2RTC{URL: go2rtcURL, Streams: map[string]string{"sub": camara + "_sub", "hd": camara}},
		focos:  focos,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("GET /api/estado", app.estado)
	mux.HandleFunc("POST /api/camara/deteccion", app.deteccion)
	mux.HandleFunc("GET /api/camara/imagen", app.imagenCamara)
	mux.HandleFunc("GET /api/camara/eventos/{id}/foto", app.fotoEvento)
	mux.HandleFunc("GET /api/camara/vivo", app.go2rtc.vivo) // WebSocket (MSE)
	mux.HandleFunc("POST /api/focos/{id}", app.cambiarFoco)

	// PWA: archivos estáticos; rutas desconocidas → index.html (SPA)
	dist, _ := fs.Sub(webDist, "web/dist")
	archivos := http.FileServerFS(dist)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		ruta := strings.TrimPrefix(r.URL.Path, "/")
		if ruta == "" || ruta == "sw.js" || ruta == "index.html" {
			w.Header().Set("Cache-Control", "no-cache") // que la PWA detecte versiones nuevas
		}
		if _, err := fs.Stat(dist, ruta); ruta != "" && err != nil {
			r.URL.Path = "/"
		}
		archivos.ServeHTTP(w, r)
	})

	addr := ":" + env("PORT", "8080")
	log.Printf("casa escuchando en %s — focos: %d, frigate: %s", addr, len(focos), app.frigate.URL)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
