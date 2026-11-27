package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

type EventoLista struct {
	Evento
	Fin       *float64 `json:"fin"` // nil = evento en curso
	TieneClip bool     `json:"tiene_clip"`
}

// GET /api/camara/eventos?limite=20
func (a *App) eventos(w http.ResponseWriter, r *http.Request) {
	limite := 20
	if v, err := strconv.Atoi(r.URL.Query().Get("limite")); err == nil && v > 0 && v <= 50 {
		limite = v
	}
	var crudos []struct {
		ID          string   `json:"id"`
		Label       string   `json:"label"`
		StartTime   float64  `json:"start_time"`
		EndTime     *float64 `json:"end_time"`
		HasSnapshot bool     `json:"has_snapshot"`
		HasClip     bool     `json:"has_clip"`
		Data        struct {
			Score float64 `json:"score"`
		} `json:"data"`
	}
	q := url.Values{"camera": {a.frigate.Camara}, "limit": {strconv.Itoa(limite)}}
	// Más margen que el cliente compartido (4 s, pensado para el polling de /api/estado)
	lento := &Frigate{URL: a.frigate.URL, Camara: a.frigate.Camara, http: &http.Client{Timeout: 12 * time.Second}}
	if err := lento.getJSON("/api/events?"+q.Encode(), &crudos); err != nil {
		fallo(w, http.StatusBadGateway, err)
		return
	}
	lista := make([]EventoLista, 0, len(crudos))
	for _, e := range crudos {
		lista = append(lista, EventoLista{
			Evento:    Evento{ID: e.ID, Etiqueta: e.Label, Inicio: e.StartTime, Score: e.Data.Score, TieneFoto: e.HasSnapshot},
			Fin:       e.EndTime,
			TieneClip: e.HasClip,
		})
	}
	responder(w, http.StatusOK, lista)
}

func (a *App) miniatura(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !idEvento.MatchString(id) {
		fallo(w, http.StatusBadRequest, errors.New("id de evento inválido"))
		return
	}
	a.frigate.proxyImagen(w, "/api/events/"+id+"/thumbnail.jpg")
}

// ------------------------------------------------------------------ clips
//
// Frigate arma el mp4 al vuelo y responde 200 aunque se pida un Range. Safari (iOS) NO
// reproduce un <video> si el servidor no contesta 206 a los Range → casa baja el clip
// una vez a disco (emptyDir) y lo sirve con http.ServeContent, que implementa Range.

type CacheClips struct {
	Dir    string
	Max    int // clips guardados (los más viejos se borran)
	mu     sync.Mutex
	client *http.Client
}

func (c *CacheClips) obtener(frigateURL, id string) (string, error) {
	c.mu.Lock() // descarga serializada: es un uso de una sola persona
	defer c.mu.Unlock()

	ruta := filepath.Join(c.Dir, id+".mp4")
	if _, err := os.Stat(ruta); err == nil {
		return ruta, nil
	}
	resp, err := c.client.Get(fmt.Sprintf("%s/api/events/%s/clip.mp4", frigateURL, id))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("frigate: HTTP %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp(c.Dir, "descarga-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name()) // no-op si el rename funcionó
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), ruta); err != nil { // atómico: nunca se sirve un clip a medias
		return "", err
	}
	c.podar()
	return ruta, nil
}

func (c *CacheClips) podar() {
	archivos, _ := filepath.Glob(filepath.Join(c.Dir, "*.mp4"))
	if len(archivos) <= c.Max {
		return
	}
	sort.Slice(archivos, func(i, j int) bool {
		fi, _ := os.Stat(archivos[i])
		fj, _ := os.Stat(archivos[j])
		return fi.ModTime().Before(fj.ModTime())
	})
	for _, f := range archivos[:len(archivos)-c.Max] {
		_ = os.Remove(f)
	}
}

func (a *App) clip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !idEvento.MatchString(id) {
		fallo(w, http.StatusBadRequest, errors.New("id de evento inválido"))
		return
	}
	inicio := time.Now()
	ruta, err := a.clips.obtener(a.frigate.URL, id)
	if err != nil {
		log.Printf("clip %s: %v", id, err)
		fallo(w, http.StatusBadGateway, err)
		return
	}
	f, err := os.Open(ruta)
	if err != nil {
		fallo(w, http.StatusInternalServerError, err)
		return
	}
	defer f.Close()
	info, _ := f.Stat()
	if r.Header.Get("Range") == "" || r.Header.Get("Range") == "bytes=0-1" {
		log.Printf("clip %s: %d KB (listo en %s)", id, info.Size()/1024, time.Since(inicio).Round(time.Millisecond))
	}
	w.Header().Set("Cache-Control", "private, max-age=3600") // el clip de un evento terminado no cambia
	http.ServeContent(w, r, id+".mp4", info.ModTime(), f)
}
