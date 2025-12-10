# VPN (Virtual Private Network)

## Qué es

Tecnología que crea un túnel cifrado entre dos puntos a través de una red no confiable (como internet). Permite que dispositivos remotos se comporten como si estuvieran en la misma red local.

## Para qué sirve

- **Acceso remoto**: Conectar a tu red de casa desde cualquier lugar
- **Privacidad**: Cifrar tráfico en redes WiFi públicas
- **Bypass geográfico**: Aparecer como si estuvieras en otro país
- **Conectar oficinas**: Unir redes de diferentes ubicaciones

## Cómo funciona

```
[Tu laptop]                                          [Red del homelab]
    |                                                      |
    |  1. Paquete original                                 |
    |     Src: 10.10.10.2 → Dst: 10.0.0.2                 |
    |                                                      |
    |  2. VPN encapsula y cifra                           |
    |     [Cifrado: paquete original]                     |
    |     Src: IP_publica → Dst: gateway_homelab          |
    |                                                      |
    |=================== INTERNET =========================|
    |                                                      |
    |  3. Gateway descifra                                 |
    |     Extrae paquete original                          |
    |     Lo envía a 10.0.0.2                              |
    |                                                      |
```

### Encapsulación

```
Paquete original:
┌─────────────────────────────────────────┐
│ IP Header │ TCP Header │ Datos HTTP    │
│ 10.0.0.2  │   :80      │ GET /index... │
└─────────────────────────────────────────┘

Paquete VPN:
┌───────────────────────────────────────────────────────────┐
│ IP Header  │ UDP Header │ VPN Header │ [CIFRADO]         │
│ IP_Gateway │   :51820   │ WireGuard  │ Paquete original  │
└───────────────────────────────────────────────────────────┘
```

## Tipos de VPN

| Tipo | Uso | Ejemplo |
|------|-----|---------|
| **Remote Access** | Un usuario → red | WireGuard a homelab |
| **Site-to-Site** | Red → Red | Conectar dos oficinas |
| **Client-to-Client** | Usuario → Usuario | Mesh VPN (Tailscale) |

## Protocolos VPN comunes

| Protocolo | Velocidad | Seguridad | Complejidad |
|-----------|-----------|-----------|-------------|
| WireGuard | Muy alta | Muy alta | Baja |
| OpenVPN | Media | Alta | Alta |
| IPSec/IKEv2 | Alta | Alta | Muy alta |
| PPTP | Alta | Baja (obsoleto) | Baja |

## En el homelab

Usamos **WireGuard** para acceder a la red 10.0.0.0/24 desde fuera:

```
[Mac remoto]                    [Gateway RPi]              [Homelab]
10.10.10.2 ════════════════════> 10.10.10.1 ──────────────> 10.0.0.x
    │        túnel WireGuard        │                          │
    │          (cifrado)            │                          │
    └───────────────────────────────┴──────────────────────────┘
                                Red VPN: 10.10.10.0/24
```

Ver:
- [ADR-001 WireGuard sobre OpenVPN](../decisions/001-wireguard-over-openvpn.md)
- [WireGuard](wireguard.md)

## Conceptos clave

- **Túnel**: Conexión cifrada entre dos puntos
- **Peer**: Cada extremo de la VPN
- **Endpoint**: IP:puerto donde contactar al peer
- **AllowedIPs**: Qué tráfico enviar por el túnel
- **Keepalive**: Paquetes periódicos para mantener conexión viva

## Comandos útiles

```bash
# Ver estado de interfaces VPN
ip link show type wireguard

# Ver conexiones activas
sudo wg show

# Probar conectividad a través del túnel
ping 10.0.0.1  # Gateway del homelab

# Ver rutas agregadas por VPN
ip route | grep wg
```
