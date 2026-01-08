# Configuración de Tailscale en el Homelab

## ¿Por qué Tailscale?

El ISP (Entel) usa **CGNAT** (Carrier-Grade NAT), lo que significa que la IP pública es compartida entre múltiples clientes. Esto impide que el port forwarding funcione para WireGuard.

### El problema: CGNAT
```
Internet                              ISP                              Tu casa
    │
    │     IP pública                 CGNAT                      IP privada
    │    200.111.224.99             del ISP                    100.78.117.90
    │          │                       │                            │
    │          ▼                       ▼                            ▼
    │   ┌─────────────┐         ┌─────────────┐              ┌─────────────┐
    │   │   Router    │         │   Router    │              │   Tu Modem  │
    │   │    ISP      │────────►│    CGNAT    │─────────────►│             │
    │   └─────────────┘         └─────────────┘              └─────────────┘
    │                                 │
    │                                 ├──► Cliente 1 (100.78.117.90) ← TÚ
    │                                 ├──► Cliente 2 (100.78.117.91)
    │                                 └──► Cliente N (100.78.xxx.xx)
    │
    │  Puerto 51820 → ¿A cuál cliente? ❌ No sabe
```

### La solución: Tailscale

Tailscale usa técnicas de NAT traversal y servidores relay para establecer conexiones sin necesidad de abrir puertos.
```
Tu Mac                 Servidores Tailscale              Tu homelab
    │                         │                              │
    │   1. "Estoy aquí"       │       2. "Estoy aquí"        │
    ├────────────────────────►│◄─────────────────────────────┤
    │                         │                              │
    │   3. Tailscale coordina la conexión                   │
    │                         │                              │
    │◄═══════════════════════════════════════════════════════╡
    │         4. Conexión establecida ✅                     │
```

---

## Arquitectura actual
```
                         INTERNET
                             │
                             │ CGNAT (no port forwarding)
                             │
                    ┌────────┴────────┐
                    │  Tailscale Cloud │
                    │   (coordina)     │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
      ┌──────────────┐              ┌──────────────┐
      │   Tu Mac     │              │  rp1-master  │
      │ 100.70.50.39 │              │ 100.94.94.49 │
      │  (Tailscale) │              │  (Tailscale) │
      └──────────────┘              │              │
                                    │ Subnet Router│
                                    │ 10.0.0.0/24  │
                                    └──────┬───────┘
                                           │
                                    ┌──────┴──────┐
                                    │             │
                                    ▼             ▼
                              ┌──────────┐ ┌──────────┐
                              │   rp2    │ │   rp3    │
                              │ 10.0.0.2 │ │ 10.0.0.3 │
                              └──────────┘ └──────────┘
```

---

## Dispositivos configurados

| Dispositivo | IP Tailscale | IP Local | Rol |
|-------------|--------------|----------|-----|
| Tu Mac | 100.70.50.39 | - | Cliente |
| rp1-master | 100.94.94.49 | 10.0.0.1 | Subnet Router |
| rp2-node | - | 10.0.0.2 | Via subnet |
| rp3-node | - | 10.0.0.3 | Via subnet |

---

## Instalación

### En el Gateway (rp1-master)
```bash
# Instalar Tailscale
curl -fsSL https://tailscale.com/install.sh | sh

# Habilitar IP forwarding
echo 'net.ipv4.ip_forward = 1' | sudo tee -a /etc/sysctl.conf
echo 'net.ipv6.conf.all.forwarding = 1' | sudo tee -a /etc/sysctl.conf
sudo sysctl -p

# Iniciar con subnet routing
sudo tailscale up --advertise-routes=10.0.0.0/24
```

### Aprobar subnet en consola web

1. Ir a [https://login.tailscale.com/admin/machines](https://login.tailscale.com/admin/machines)
2. Buscar **rp1-master**
3. Clic en **"..."** → **"Edit route settings"**
4. Activar la subred `10.0.0.0/24`

### En tu Mac
```bash
# Instalar con Homebrew
brew install tailscale

# O descargar desde: https://tailscale.com/download/mac
```

Abrir la app y conectar con tu cuenta.

---

## Uso diario

### Conectar al homelab
```bash
# Verificar estado
tailscale status

# SSH al gateway (via Tailscale)
ssh admin@100.94.94.49

# SSH a los nodos (via subnet)
ssh admin@10.0.0.2
ssh admin@10.0.0.3

# También funciona con nombres .homelab.local si el DNS está configurado
ssh admin@rp1.homelab.local
```

### Verificar conectividad
```bash
# Desde tu Mac
ping 100.94.94.49    # Gateway via Tailscale
ping 10.0.0.2        # Nodo via subnet
```

---

## Comandos útiles

### En cualquier dispositivo
```bash
# Ver estado y dispositivos conectados
tailscale status

# Ver IP de Tailscale
tailscale ip

# Ver información detallada
tailscale status --json | jq

# Desconectar
tailscale down

# Reconectar
tailscale up
```

### En el gateway (subnet router)
```bash
# Ver rutas anunciadas
tailscale status

# Re-anunciar rutas si es necesario
sudo tailscale up --advertise-routes=10.0.0.0/24

# Ver logs
sudo journalctl -u tailscaled -f
```

---

## Comparación: WireGuard manual vs Tailscale

| Aspecto | WireGuard manual | Tailscale |
|---------|------------------|-----------|
| Funciona con CGNAT | ❌ No | ✅ Sí |
| Configuración | Manual | Automática |
| Necesita abrir puertos | Sí | No |
| Encriptación | WireGuard | WireGuard |
| Costo | Gratis | Gratis (hasta 100 dispositivos) |
| Control total | ✅ Sí | Parcial |
| Dependencia externa | No | Sí (servidores Tailscale) |

---

## Troubleshooting

### No puedo conectar a los nodos (10.0.0.x)

1. Verificar que subnet está aprobada en consola web
2. Verificar IP forwarding en gateway:
```bash
   ssh admin@100.94.94.49 "sysctl net.ipv4.ip_forward"
   # Debe mostrar: net.ipv4.ip_forward = 1
```
3. Verificar que Tailscale está corriendo:
```bash
   ssh admin@100.94.94.49 "tailscale status"
```

### Conexión lenta

Tailscale puede estar usando relay (DERP) en lugar de conexión directa:
```bash
# Ver tipo de conexión
tailscale status

# Si muestra "relay", el tráfico pasa por servidores de Tailscale
# Esto es normal con CGNAT
```

### Dispositivo no aparece

1. Verificar que Tailscale está corriendo en ambos dispositivos
2. Verificar que ambos usan la misma cuenta
3. Reiniciar Tailscale:
```bash
   sudo systemctl restart tailscaled
```

---

## Seguridad

### ¿Es seguro Tailscale?

- **Encriptación**: Usa WireGuard (mismo protocolo que configuraste antes)
- **End-to-end**: Tailscale no puede ver tu tráfico
- **Código abierto**: Cliente es auditable
- **Zero trust**: Cada dispositivo se autentica individualmente

### Buenas prácticas

1. Habilitar **MFA** en tu cuenta de Tailscale
2. Revisar dispositivos autorizados periódicamente
3. Usar **ACLs** si necesitas restringir acceso entre dispositivos

---

## DuckDNS (mantener activo)

Aunque Tailscale resuelve el acceso remoto, DuckDNS sigue siendo útil para:
- Tener un dominio memorable (`aren-homelab.duckdns.org`)
- Servicios futuros que no usen Tailscale
- Cloudflare Tunnel (si lo configuras después)

El cron sigue actualizando la IP cada 5 minutos en el gateway.
