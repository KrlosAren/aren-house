package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
)

// Foco WiZ controlado por UDP :38899 (protocolo JSON local).
// Requiere en la app WiZ: "Allow local" SIN "Only verified controls" (ver CLAUDE.md).
type Foco struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
}

// Pilot es el estado del foco tal como lo reporta getPilot.
type Pilot struct {
	State   bool `json:"state"`
	Dimming *int `json:"dimming,omitempty"`
	R       *int `json:"r,omitempty"`
	G       *int `json:"g,omitempty"`
	B       *int `json:"b,omitempty"`
	Temp    *int `json:"temp,omitempty"`
	SceneID *int `json:"sceneId,omitempty"`
	RSSI    *int `json:"rssi,omitempty"`
}

type wizRespuesta struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

var errSinRespuesta = errors.New("sin respuesta (¿apagado en el interruptor?)")

// wizCmd envía un comando y espera la respuesta, con reintentos (UDP puede perder paquetes
// y los focos en ahorro de energía a veces no contestan al primer intento).
func wizCmd(ip, method string, params any) (json.RawMessage, error) {
	if params == nil {
		params = map[string]any{}
	}
	msg, _ := json.Marshal(map[string]any{"method": method, "params": params})

	conn, err := net.Dial("udp", net.JoinHostPort(ip, "38899"))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	buf := make([]byte, 4096)
	for intento := 0; intento < 3; intento++ {
		if _, err := conn.Write(msg); err != nil {
			return nil, err
		}
		_ = conn.SetReadDeadline(time.Now().Add(700 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			continue // timeout → reintentar
		}
		var r wizRespuesta
		if err := json.Unmarshal(buf[:n], &r); err != nil {
			return nil, fmt.Errorf("respuesta inválida: %w", err)
		}
		if r.Error != nil {
			// "Invalid params" con firmware nuevo = "Only verified controls" activado en la app
			return nil, fmt.Errorf("el foco rechazó la orden: %s", r.Error.Message)
		}
		return r.Result, nil
	}
	return nil, errSinRespuesta
}

func wizEstado(ip string) (*Pilot, error) {
	raw, err := wizCmd(ip, "getPilot", nil)
	if err != nil {
		return nil, err
	}
	var p Pilot
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// CambioFoco es lo que la PWA pide. Campos nil = no tocar.
type CambioFoco struct {
	Encendido *bool   `json:"encendido,omitempty"`
	Brillo    *int    `json:"brillo,omitempty"` // 10-100
	Color     *string `json:"color,omitempty"`  // "#rrggbb"
	Temp      *int    `json:"temp,omitempty"`   // 2200-6500 K
}

func (c CambioFoco) params() (map[string]any, error) {
	p := map[string]any{}
	if c.Encendido != nil {
		p["state"] = *c.Encendido
	}
	if c.Brillo != nil {
		if *c.Brillo < 10 || *c.Brillo > 100 {
			return nil, errors.New("brillo debe estar entre 10 y 100")
		}
		p["dimming"] = *c.Brillo
	}
	if c.Color != nil && c.Temp != nil {
		return nil, errors.New("usar color o temp, no ambos")
	}
	if c.Color != nil {
		var r, g, b int
		if _, err := fmt.Sscanf(*c.Color, "#%02x%02x%02x", &r, &g, &b); err != nil || len(*c.Color) != 7 {
			return nil, errors.New("color debe ser #rrggbb")
		}
		p["r"], p["g"], p["b"] = r, g, b
	}
	if c.Temp != nil {
		if *c.Temp < 2200 || *c.Temp > 6500 {
			return nil, errors.New("temp debe estar entre 2200 y 6500")
		}
		p["temp"] = *c.Temp
	}
	if len(p) == 0 {
		return nil, errors.New("nada que cambiar")
	}
	// Cambiar brillo/color implica encender, salvo que se pida explícitamente apagar
	if _, ok := p["state"]; !ok {
		p["state"] = true
	}
	return p, nil
}

func wizCambiar(ip string, c CambioFoco) (*Pilot, error) {
	params, err := c.params()
	if err != nil {
		return nil, err
	}
	if _, err := wizCmd(ip, "setPilot", params); err != nil {
		return nil, err
	}
	time.Sleep(300 * time.Millisecond) // el foco tarda en reflejar el cambio en getPilot
	return wizEstado(ip)
}
