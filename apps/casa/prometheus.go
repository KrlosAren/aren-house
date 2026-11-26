package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
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
