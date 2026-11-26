package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

type Frigate struct {
	URL    string // http://frigate.cameras.svc.cluster.local:5000
	Camara string // casona
	http   *http.Client
}

type Evento struct {
	ID        string  `json:"id"`
	Etiqueta  string  `json:"etiqueta"`
	Inicio    float64 `json:"inicio"`
	Score     float64 `json:"score"`
	TieneFoto bool    `json:"tiene_foto"`
}

type EstadoCamara struct {
	// Armada viene de detection_enabled en /api/stats: el estado REAL.
	// (MQTT detect/state repite el payload recibido aunque no se aplique; ver CLAUDE.md)
	Armada           bool    `json:"armada"`
	FPS              float64 `json:"fps"`
	Calidad          string  `json:"calidad"` // connection_quality de Frigate
	ReconexionesHora int     `json:"reconexiones_hora"`
	UltimoEvento     *Evento `json:"ultimo_evento"`
}

func (f *Frigate) getJSON(ruta string, dst any) error {
	resp, err := f.http.Get(f.URL + ruta)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("frigate %s: HTTP %d", ruta, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func (f *Frigate) Estado() (*EstadoCamara, error) {
	var stats struct {
		Cameras map[string]struct {
			CameraFPS         float64 `json:"camera_fps"`
			DetectionEnabled  bool    `json:"detection_enabled"`
			ConnectionQuality string  `json:"connection_quality"`
			ReconnectsHour    int     `json:"reconnects_last_hour"`
		} `json:"cameras"`
	}
	if err := f.getJSON("/api/stats", &stats); err != nil {
		return nil, err
	}
	c, ok := stats.Cameras[f.Camara]
	if !ok {
		return nil, fmt.Errorf("cámara %q no existe en Frigate", f.Camara)
	}
	e := &EstadoCamara{Armada: c.DetectionEnabled, FPS: c.CameraFPS, Calidad: c.ConnectionQuality, ReconexionesHora: c.ReconnectsHour}

	var eventos []struct {
		ID          string  `json:"id"`
		Label       string  `json:"label"`
		StartTime   float64 `json:"start_time"`
		HasSnapshot bool    `json:"has_snapshot"`
		Data        struct {
			Score float64 `json:"score"`
		} `json:"data"`
	}
	q := url.Values{"camera": {f.Camara}, "limit": {"1"}}
	if err := f.getJSON("/api/events?"+q.Encode(), &eventos); err == nil && len(eventos) > 0 {
		ev := eventos[0]
		e.UltimoEvento = &Evento{ID: ev.ID, Etiqueta: ev.Label, Inicio: ev.StartTime, Score: ev.Data.Score, TieneFoto: ev.HasSnapshot}
	}
	return e, nil
}

// Armar / desarmar. Frigate solo acepta "ON"/"OFF" EXACTOS en mayúsculas:
// cualquier otra cosa responde success sin hacer nada. Por eso el valor sale de un bool.
func (f *Frigate) SetDeteccion(activa bool) error {
	valor := "OFF"
	if activa {
		valor = "ON"
	}
	body, _ := json.Marshal(map[string]string{"value": valor})
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/camera/%s/set/detect", f.URL, f.Camara), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("frigate: HTTP %d: %s", resp.StatusCode, b)
	}
	return nil
}

var idEvento = regexp.MustCompile(`^[0-9]+\.[0-9]+-[a-z0-9]+$`)

// Proxy de imágenes (Frigate no tiene auth y no se expone directo a la PWA).
func (f *Frigate) proxyImagen(w http.ResponseWriter, ruta string) {
	cli := &http.Client{Timeout: 8 * time.Second}
	resp, err := cli.Get(f.URL + ruta)
	if err != nil {
		http.Error(w, "frigate no responde", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "imagen no disponible", resp.StatusCode)
		return
	}
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.Copy(w, resp.Body)
}
