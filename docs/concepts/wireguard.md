# WireGuard

## Qué es

Protocolo VPN moderno, diseñado para ser simple, rápido y seguro. Integrado directamente en el kernel de Linux desde la versión 5.6.

## Para qué sirve

Crear túneles VPN cifrados con:
- Configuración mínima (~15 líneas)
- Alto rendimiento (kernel-space)
- Criptografía moderna sin opciones inseguras

## Por qué WireGuard sobre OpenVPN

| Aspecto | WireGuard | OpenVPN |
|---------|-----------|---------|
| Líneas de código | ~4,000 | ~100,000 |
| Ubicación | Kernel | Userspace |
| Configuración | 15 líneas | Cientos |
| Criptografía | Fija (moderna) | Configurable (puede ser insegura) |
| Reconexión | Instantánea | Lenta |
| Roaming | Automático | Manual |

## Cómo funciona

### Modelo de llaves

Cada peer tiene un par de llaves (como SSH):

```bash
# Generar llave privada
wg genkey > private.key

# Derivar llave pública
cat private.key | wg pubkey > public.key
```

Intercambias llaves públicas con los peers (nunca la privada).

### Conceptos

```
[Peer A: Mac]                              [Peer B: Gateway]

Private Key: aaa...                        Private Key: bbb...
Public Key:  AAA...                        Public Key:  BBB...

[Peer] de B en config de A:                [Peer] de A en config de B:
  PublicKey = BBB...                         PublicKey = AAA...
  Endpoint = gateway.duckdns.org:51820       (no endpoint, A inicia)
  AllowedIPs = 10.0.0.0/24                   AllowedIPs = 10.10.10.2/32
```

### AllowedIPs explicado

`AllowedIPs` define:
1. **Salida**: Qué destinos enviar por este túnel
2. **Entrada**: Qué IPs origen aceptar de este peer

```bash
# En el cliente (Mac):
AllowedIPs = 10.0.0.0/24
# → Envía tráfico a 10.0.0.x por el túnel

# En el servidor (Gateway):
AllowedIPs = 10.10.10.2/32
# → Solo acepta paquetes de este peer con IP 10.10.10.2
```

## En el homelab

### Configuración del servidor (Gateway)

```ini
# /etc/wireguard/wg0.conf

[Interface]
Address = 10.10.10.1/24          # IP del gateway en la VPN
PrivateKey = <llave_privada_gateway>
ListenPort = 51820
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT

[Peer]
# Mac de Carlos
PublicKey = <llave_publica_mac>
AllowedIPs = 10.10.10.2/32       # IP asignada al Mac
```

### Configuración del cliente (Mac)

```ini
# /etc/wireguard/wg0.conf (o en app WireGuard)

[Interface]
Address = 10.10.10.2/24
PrivateKey = <llave_privada_mac>
DNS = 10.0.0.1                   # Usar DNS del homelab

[Peer]
PublicKey = <llave_publica_gateway>
Endpoint = gateway.duckdns.org:51820
AllowedIPs = 10.0.0.0/24, 10.10.10.0/24
PersistentKeepalive = 25         # Mantener conexión viva
```

Ver: [ADR-001 WireGuard sobre OpenVPN](../decisions/001-wireguard-over-openvpn.md)

## Comandos útiles

```bash
# Iniciar interfaz
sudo wg-quick up wg0

# Detener interfaz
sudo wg-quick down wg0

# Ver estado y estadísticas
sudo wg show

# Ver estado detallado
sudo wg show wg0

# Habilitar al boot (systemd)
sudo systemctl enable wg-quick@wg0

# Verificar conectividad
ping 10.10.10.1   # Gateway VPN
ping 10.0.0.2     # Dispositivo en homelab

# Debug: ver paquetes WireGuard
sudo tcpdump -i eth0 udp port 51820
```

## Troubleshooting

| Problema | Causa probable | Solución |
|----------|---------------|----------|
| Handshake timeout | Puerto bloqueado | Verificar firewall, port forward en modem |
| No route to host | AllowedIPs incorrecto | Verificar rutas, AllowedIPs ambos lados |
| Paquetes van pero no vuelven | Falta forwarding | `net.ipv4.ip_forward=1`, reglas iptables |
| DNS no funciona | DNS no alcanzable | Verificar DNS en Interface, o usar 8.8.8.8 |

## Seguridad

- **Sin llaves = sin acceso**: Sin la llave privada correcta, el servidor ignora paquetes
- **Perfect Forward Secrecy**: Cada sesión tiene llaves efímeras
- **Sin negociación**: No hay versiones débiles que elegir
- **Silencioso**: No responde a paquetes no autenticados (invisible a escaneos)
