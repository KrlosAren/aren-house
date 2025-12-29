# ADR-004: IP Forwarding y NAT para Gateway

## Estado

Aceptado

## Contexto

El gateway (rp1-master) necesita enrutar tráfico entre múltiples redes:

- **LAN Homelab** (10.0.0.0/24) → Internet (via modem 192.168.100.x)
- **VPN** (10.0.1.0/24) → LAN Homelab (10.0.0.0/24)
- **VPN** (10.0.1.0/24) → Internet

Por defecto, Linux no reenvía paquetes entre interfaces. Los nodos en 10.0.0.0/24 no pueden acceder a internet porque sus paquetes llegan al gateway pero no salen por la interfaz WAN.

## Decisión

### 1. Habilitar IP Forwarding

Usamos `sysctl` para habilitar el reenvío de paquetes IPv4:

```bash
net.ipv4.ip_forward = 1
```

Configurado en `/etc/sysctl.d/99-ip-forward.conf` via Ansible (rol wireguard).

### 2. NAT con iptables MASQUERADE

Usamos MASQUERADE para traducir direcciones de origen:

```bash
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
```

Esto permite que los nodos internos usen la IP del gateway para salir a internet.

### 3. Reglas FORWARD para WireGuard

WireGuard agrega reglas FORWARD dinámicamente via PostUp/PostDown:

```bash
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -A FORWARD -o wg0 -j ACCEPT
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -D FORWARD -o wg0 -j ACCEPT
```

## Alternativas Consideradas

### UFW (Uncomplicated Firewall)
- **Pro**: Más fácil de gestionar
- **Contra**: Capa de abstracción sobre iptables, menos control
- **Decisión**: Planificado para fase de seguridad (ver roadmap)

### nftables
- **Pro**: Sucesor moderno de iptables
- **Contra**: Menos documentación, curva de aprendizaje mayor
- **Decisión**: No adoptado, iptables es suficiente para el caso de uso

### Firewalld
- **Pro**: Gestión de zonas, recarga dinámica
- **Contra**: Overhead para homelab simple
- **Decisión**: No adoptado

## Consecuencias

### Positivas
- Nodos internos tienen acceso a internet transparente
- Clientes VPN acceden a red interna y a internet
- Configuración simple y directa

### Negativas
- Las reglas iptables no persisten tras reinicio (excepto las de WireGuard via PostUp)
- La regla MASQUERADE debe aplicarse manualmente o agregar a scripts de inicio
- Sin firewall activo, todo el tráfico pasa (seguridad pendiente)

### Deuda Técnica
- Implementar persistencia de reglas iptables
- Agregar firewall (ufw) con reglas restrictivas
- Documentar todas las reglas necesarias

## Referencias

- [Kernel IP Forwarding](https://www.kernel.org/doc/Documentation/networking/ip-sysctl.txt)
- [iptables NAT HOWTO](https://www.netfilter.org/documentation/HOWTO/NAT-HOWTO.html)
- [WireGuard PostUp/PostDown](https://www.wireguard.com/quickstart/)
