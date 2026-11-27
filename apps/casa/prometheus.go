package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

type Prometheus struct {
	URL  string // http://prometheus.monitoring.svc.cluster.local:9090
	http *http.Client
}

type Nodo struct {
	Nombre      string   `json:"nombre"`
	Arriba      bool     `json:"arriba"`
	Temperatura *float64 `json:"temperatura,omitempty"` // °C
	CPU         *float64 `json:"cpu,omitempty"`         // % uso (5m)
}

type EstadoHomelab struct {
	Nodos      []Nodo   `json:"nodos"`
	PingCamara *bool    `json:"ping_camara,omitempty"`
	RTTCamara  *float64 `json:"rtt_camara_ms,omitempty"`
}

// consulta devuelve {valor del label `por`: valor} para una query instantánea.
func (p *Prometheus) consulta(q, por string) (map[string]float64, error) {
	resp, err := p.http.Get(p.URL + "/api/v1/query?query=" + url.QueryEscape(q))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  [2]any            `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.Status != "success" {
		return nil, fmt.Errorf("prometheus: %s", r.Status)
	}
	out := map[string]float64{}
	for _, s := range r.Data.Result {
		v, _ := strconv.ParseFloat(fmt.Sprint(s.Value[1]), 64)
		out[s.Metric[por]] = v
	}
	return out, nil
}

// rango devuelve {epoch: valor} de una query_range (una sola serie).
func (p *Prometheus) rango(q string, desde, hasta time.Time, paso time.Duration) (map[int64]float64, error) {
	v := url.Values{
		"query": {q},
		"start": {strconv.FormatInt(desde.Unix(), 10)},
		"end":   {strconv.FormatInt(hasta.Unix(), 10)},
		"step":  {strconv.Itoa(int(paso.Seconds()))},
	}
	resp, err := p.http.Get(p.URL + "/api/v1/query_range?" + v.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Values [][2]any `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.Status != "success" {
		return nil, fmt.Errorf("prometheus: %s", r.Status)
	}
	out := map[int64]float64{}
	if len(r.Data.Result) > 0 {
		for _, par := range r.Data.Result[0].Values {
			t, _ := par[0].(float64)
			val, _ := strconv.ParseFloat(fmt.Sprint(par[1]), 64)
			out[int64(t)] = val
		}
	}
	return out, nil
}

// Tramo de la barra de salud. Estado: "ok" | "lenta" (pérdida de ping, video OK) | "caida" | "sin_datos"
type Tramo struct {
	Inicio int64  `json:"inicio"`
	Estado string `json:"estado"`
}

type SaludCamara struct {
	Horas          int     `json:"horas"`
	PasoMin        int     `json:"paso_min"`
	Tramos         []Tramo `json:"tramos"`
	Disponibilidad float64 `json:"disponibilidad"` // % del tiempo con video (fps >= 1)
	Cortes         int     `json:"cortes"`         // veces que pasó de con video → sin video
}

// Salud de las últimas `horas`, en tramos de `paso`. Un tramo es "caida" si en algún
// momento del tramo el video bajó de 1 fps: los cortes duran 1-3 min y con un promedio
// se perderían.
func (p *Prometheus) Salud(camara string, horas int, paso time.Duration) (*SaludCamara, error) {
	hasta := time.Now().Truncate(paso)
	desde := hasta.Add(-time.Duration(horas) * time.Hour)
	v := fmt.Sprintf("[%ds]", int(paso.Seconds()))

	fps, err := p.rango(`min_over_time(frigate_camera_fps{camera_name="`+camara+`"}`+v+`)`, desde, hasta, paso)
	if err != nil {
		return nil, err
	}
	ping, _ := p.rango(`min_over_time(probe_success{job="blackbox-icmp-camara",camera_name="`+camara+`"}`+v+`)`, desde, hasta, paso)

	s := &SaludCamara{Horas: horas, PasoMin: int(paso.Minutes())}
	conDatos, caidos, antes := 0, 0, ""
	for t := desde.Add(paso); !t.After(hasta); t = t.Add(paso) {
		ts := t.Unix()
		estado := "sin_datos"
		if f, ok := fps[ts]; ok {
			conDatos++
			switch {
			case f < 1:
				estado = "caida"
				caidos++
				if antes != "caida" {
					s.Cortes++
				}
			case ping[ts] < 1:
				if _, hayPing := ping[ts]; hayPing {
					estado = "lenta"
				} else {
					estado = "ok"
				}
			default:
				estado = "ok"
			}
		}
		s.Tramos = append(s.Tramos, Tramo{Inicio: t.Add(-paso).Unix(), Estado: estado})
		antes = estado
	}
	// Disponibilidad exacta (muestras cada 30 s), no por tramos: un corte de 1 min no debe
	// contar como 5 min caídos
	disp, err := p.consulta(fmt.Sprintf(
		`100 * avg_over_time((frigate_camera_fps{camera_name=%q} >= bool 1)[%dh:30s])`, camara, horas), "camera_name")
	if d, ok := disp[camara]; err == nil && ok {
		s.Disponibilidad = d
	} else if conDatos > 0 { // respaldo: aproximación por tramos
		s.Disponibilidad = 100 * float64(conDatos-caidos) / float64(conDatos)
	}
	return s, nil
}

func (p *Prometheus) Estado() (*EstadoHomelab, error) {
	up, err := p.consulta(`up{job="node-exporter"}`, "node")
	if err != nil {
		return nil, err
	}
	// Mismos chips que las alertas: cpu-thermal (Pi) y coretemp (x86)
	temps, _ := p.consulta(`max by (node) (node_hwmon_temp_celsius{chip=~".*thermal.*|platform_coretemp.*"})`, "node")
	cpus, _ := p.consulta(`100 * (1 - avg by (node) (rate(node_cpu_seconds_total{mode="idle"}[5m])))`, "node")

	e := &EstadoHomelab{}
	for nombre, v := range up {
		n := Nodo{Nombre: nombre, Arriba: v == 1}
		if t, ok := temps[nombre]; ok {
			n.Temperatura = &t
		}
		if c, ok := cpus[nombre]; ok {
			n.CPU = &c
		}
		e.Nodos = append(e.Nodos, n)
	}
	sort.Slice(e.Nodos, func(i, j int) bool { return e.Nodos[i].Nombre < e.Nodos[j].Nombre })

	if ping, err := p.consulta(`probe_success{job="blackbox-icmp-camara"}`, "camera_name"); err == nil {
		if v, ok := ping["casona"]; ok {
			b := v == 1
			e.PingCamara = &b
		}
	}
	if rtt, err := p.consulta(`probe_icmp_duration_seconds{job="blackbox-icmp-camara",phase="rtt"} * 1000`, "camera_name"); err == nil {
		if v, ok := rtt["casona"]; ok {
			e.RTTCamara = &v
		}
	}
	return e, nil
}
