# 002. Segmentación de red con Raspberry Pi como Gateway

**Fecha:** 2025-12-06
**Estado:** Aceptado

## Contexto

El homelab necesita estar aislado de la red principal de casa para:
- Experimentar sin afectar dispositivos de la familia
- Aprender sobre networking en un entorno real
- Tener control sobre el tráfico del homelab

Opciones consideradas:
- **VLAN en el router/modem**: Requiere equipo con soporte, el modem del ISP no lo tiene
- **Router dedicado**: Costo adicional, menos aprendizaje
- **Raspberry Pi como gateway**: Usa hardware existente, máximo aprendizaje

## Decisión

Usar una **Raspberry Pi con dos interfaces de red** como gateway/router entre la red del modem (192.168.1.0/24) y la red del homelab (10.0.0.0/24).

```
Modem ──[USB-Ethernet]── RPi Gateway ──[eth0]── Switch TP-Link SG105PE ──┬── RPi 2 (PoE)
192.168.1.x              192.168.1.84            │                       ├── RPi 3 (PoE)
                         10.0.0.1                │                       └── (expansión)
                                                 │                           10.0.0.x
                                                 │
                                            PoE+ para nodos
                                            (gateway con fuente separada)
```

### Hardware de red
- **Switch**: TP-Link SG105PE (5 puertos Gigabit, 4 con PoE+)
- **PoE**: Alimenta los nodos worker, no el gateway (fuente dedicada)
- **VLANs**: No utilizadas actualmente (segmentación via gateway)

## Consecuencias

### Positivas
- **Aprendizaje real**: Configurar routing, NAT, forwarding a mano
- **Control total**: Puedo ver y manipular todo el tráfico
- **Costo cero**: Usa una RPi que ya tengo
- **Flexibilidad**: Puedo agregar firewall, VPN, DNS, DHCP en el mismo equipo

### Negativas
- **Single point of failure**: Si el gateway cae, el homelab pierde internet
- **Performance**: Una RPi no es un router dedicado (suficiente para mi uso)
- **Complejidad**: Más cosas que configurar y mantener

### Configuración clave
- IP forwarding habilitado: `net.ipv4.ip_forward = 1`
- NAT con iptables: `MASQUERADE` en la interfaz hacia el modem
- Rutas estáticas en el Mac para alcanzar 10.0.0.0/24 via VPN
