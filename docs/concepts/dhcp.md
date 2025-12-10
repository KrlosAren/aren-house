# DHCP (Dynamic Host Configuration Protocol)

## Qué es

Protocolo de red que asigna automáticamente direcciones IP y otros parámetros de configuración a los dispositivos que se conectan a la red.

## Para qué sirve

Sin DHCP, tendrías que configurar manualmente la IP, máscara de red, gateway y DNS en cada dispositivo. DHCP automatiza esto:

1. Dispositivo se conecta a la red
2. Pide una IP al servidor DHCP (broadcast)
3. Servidor responde con IP + configuración
4. Dispositivo usa esa IP durante el tiempo del "lease"

## Cómo funciona

El proceso DORA:

```
[Dispositivo]                    [Servidor DHCP]
     |                                  |
     |-- DISCOVER (broadcast) --------->|  "¿Hay algún servidor DHCP?"
     |                                  |
     |<--------- OFFER ----------------|  "Sí, te ofrezco 10.0.0.105"
     |                                  |
     |-- REQUEST ---------------------->|  "Ok, quiero esa IP"
     |                                  |
     |<--------- ACK ------------------|  "Confirmado, es tuya por 24h"
```

### Conceptos clave

- **Lease**: Tiempo que un dispositivo "alquila" una IP (ej: 24 horas)
- **Scope/Range**: Rango de IPs que el servidor puede asignar (ej: 10.0.0.100-200)
- **Reservation**: IP fija asignada siempre al mismo dispositivo (por MAC address)

## Puertos

| Puerto | Protocolo | Uso |
|--------|-----------|-----|
| 67/udp | DHCP Server | Recibe peticiones |
| 68/udp | DHCP Client | Recibe respuestas |

## En el homelab

Usamos **dnsmasq** como servidor DHCP en el gateway (10.0.0.1):

```bash
# /etc/dnsmasq.conf
dhcp-range=10.0.0.100,10.0.0.200,24h    # Rango dinámico
dhcp-host=AA:BB:CC:DD:EE:FF,rp2,10.0.0.2  # IP fija para rp2
```

Ver: [ADR-003 dnsmasq](../decisions/003-dnsmasq-dhcp-dns-tftp.md)

## Comandos útiles

```bash
# Ver leases activos (en el servidor)
cat /var/lib/misc/dnsmasq.leases

# Renovar IP (en cliente Linux)
sudo dhclient -r && sudo dhclient

# Ver IP asignada
ip addr show

# Ver configuración DHCP recibida
cat /var/lib/dhcp/dhclient.leases
```
