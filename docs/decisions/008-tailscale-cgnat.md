# 008. Tailscale vs WireGuard Directo

**Fecha:** 2025-12-15
**Estado:** Aceptado

## Contexto

Necesitamos acceso remoto al homelab desde cualquier ubicación (casa, trabajo, móvil).

Inicialmente configuramos WireGuard directo (ver [ADR-001](001-wireguard-over-openvpn.md)) con:
- Servidor en rp1-master (puerto 51820)
- DuckDNS para DNS dinámico
- Port forwarding en el modem

**Problema descubierto**: El ISP (Entel Chile) usa **CGNAT**.

```bash
# IP del modem (WAN)
100.78.117.90  ← IP privada de CGNAT (rango 100.64.0.0/10)

# IP pública real
curl ifconfig.me → 200.111.224.99  ← Compartida con otros clientes
```

El port forwarding no funciona porque la IP pública es compartida.

## Opciones

### Opción A: Pedir IP pública al ISP
- Llamar a Entel, pedir IP dedicada
- Puede tener costo mensual adicional
- Mantiene WireGuard configurado

### Opción B: VPS como relay
- Servidor externo con IP pública
- WireGuard en VPS, túnel al homelab
- Costo ~$5/mes

### Opción C: Tailscale (elegida)
- Usa NAT traversal / relay automático
- Funciona con CGNAT
- Gratis hasta 100 dispositivos
- Usa WireGuard internamente

## Decisión

Usar **Tailscale** para acceso remoto.

## Consecuencias

### Positivas
- **Funciona con CGNAT**: No requiere port forwarding
- **Gratis**: Sin costos adicionales
- **Simple**: Configuración mínima
- **Seguro**: Usa WireGuard internamente
- **Subnet routing**: Acceso a toda la red 10.0.0.0/24 desde un solo nodo

### Negativas
- **Dependencia externa**: Servidores de Tailscale para coordinación
- **Menos control**: No podemos inspeccionar el tráfico de coordinación
- **Latencia potencial**: Si usa relay en vez de conexión directa

## Implementación

```bash
# En rp1-master (gateway)
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up --advertise-routes=10.0.0.0/24

# Aprobar subnet en admin.tailscale.com
```

## Configuración Actual

```
Tu Mac (100.70.50.39) ◄──► Tailscale Cloud ◄──► rp1-master (100.94.94.49)
                                                      │
                                                      ▼ subnet router
                                                 10.0.0.0/24
                                                 ├── rp1: 10.0.0.1
                                                 ├── rp2: 10.0.0.2
                                                 └── rp3: 10.0.0.3
```

## Estado de WireGuard

WireGuard manual queda como **backup** si en el futuro:
- Conseguimos IP pública dedicada
- Cambiamos de ISP sin CGNAT

## Referencias

- [tailscale-setup.md](../guides/tailscale-setup.md)
- [ADR-009: CGNAT Workaround](009-cgnat-workaround.md)
