# NAT (Network Address Translation)

## Qué es

Técnica que traduce direcciones IP privadas a públicas (y viceversa), permitiendo que múltiples dispositivos compartan una única IP pública para acceder a internet.

## Para qué sirve

- **Conservar IPs públicas**: Millones de dispositivos, pocas IPs IPv4
- **Seguridad básica**: Dispositivos internos no son accesibles directamente
- **Flexibilidad**: Puedes usar cualquier rango privado internamente

```
[Red interna]              [NAT/Router]              [Internet]
                                |
PC1: 10.0.0.10 ─────┐          |
                    ├──> 192.168.1.84 ──────> google.com
PC2: 10.0.0.11 ─────┘          |                (IP pública)
                                |
     (IPs privadas)        (traduce)           (IP pública)
```

## Rangos de IP privadas (RFC 1918)

| Rango | Máscara | Cantidad de IPs |
|-------|---------|-----------------|
| 10.0.0.0 - 10.255.255.255 | /8 | 16 millones |
| 172.16.0.0 - 172.31.255.255 | /12 | 1 millón |
| 192.168.0.0 - 192.168.255.255 | /16 | 65,536 |

Estas IPs nunca se enrutan en internet; requieren NAT para salir.

## Tipos de NAT

### SNAT (Source NAT) / Masquerade

Traduce IP origen al salir. Usado para acceso a internet.

```
Paquete sale:
  Origen: 10.0.0.10:54321 → Destino: 142.250.80.14:443
         ↓ NAT
  Origen: 192.168.1.84:12345 → Destino: 142.250.80.14:443

Respuesta regresa:
  Origen: 142.250.80.14:443 → Destino: 192.168.1.84:12345
         ↓ NAT (tabla de conexiones)
  Origen: 142.250.80.14:443 → Destino: 10.0.0.10:54321
```

### DNAT (Destination NAT) / Port Forwarding

Traduce IP destino al entrar. Usado para exponer servicios.

```
Internet quiere acceder a servidor web interno:

  Origen: 8.8.8.8:54321 → Destino: 192.168.1.84:80
         ↓ DNAT
  Origen: 8.8.8.8:54321 → Destino: 10.0.0.10:80
```

## Cómo funciona (tabla de conexiones)

```
# El router mantiene una tabla:
Proto | IP Interna    | Puerto | IP Externa     | Puerto | Destino
TCP   | 10.0.0.10     | 54321  | 192.168.1.84   | 12345  | 142.250.80.14:443
TCP   | 10.0.0.11     | 33456  | 192.168.1.84   | 12346  | 151.101.1.140:443

# Así sabe a quién enviar las respuestas
```

## En el homelab

El gateway (RPi) hace NAT para que la red 10.0.0.0/24 acceda a internet via la red 192.168.1.0/24:

```bash
# Habilitar forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward

# Persistente en /etc/sysctl.conf
net.ipv4.ip_forward = 1

# Regla iptables MASQUERADE
iptables -t nat -A POSTROUTING -o usb0 -j MASQUERADE
# usb0 = interfaz hacia el modem (192.168.1.x)

# Permitir tráfico forward
iptables -A FORWARD -i eth0 -o usb0 -j ACCEPT
iptables -A FORWARD -i usb0 -o eth0 -m state --state RELATED,ESTABLISHED -j ACCEPT
```

Ver: [ADR-002 Segmentación de red](../decisions/002-network-segmentation.md)

## Comandos útiles

```bash
# Ver reglas NAT
sudo iptables -t nat -L -v -n

# Ver tabla de conexiones activas
sudo conntrack -L

# Contar conexiones NAT
sudo conntrack -C

# Ver estadísticas de forwarding
cat /proc/net/stat/nf_conntrack

# Agregar port forwarding (ej: SSH a rp2)
sudo iptables -t nat -A PREROUTING -i usb0 -p tcp --dport 2222 -j DNAT --to-destination 10.0.0.2:22
```

## Limitaciones

- **Conexiones entrantes**: Requieren port forwarding explícito
- **Algunos protocolos**: FTP activo, SIP, etc. tienen problemas con NAT
- **Performance**: Cada paquete debe ser traducido
- **IPv6**: NAT es menos común, hay suficientes IPs para todos
