# Configuración de Firewall

Guía para configurar firewall en el gateway usando UFW e iptables.

## Estado Actual

El firewall está configurado via Ansible con el playbook `firewall.yml`:

- **Gateway**: UFW que permite todo desde LAN (necesario para netboot), reglas específicas para VPN y WAN
- **Nodos**: UFW con reglas mínimas (SSH desde LAN/VPN, todo desde gateway)

```bash
# Aplicar configuración
ansible-playbook playbooks/firewall.yml

# Ver estado
sudo ufw status verbose
```

Ver [ADR-005: UFW Firewall](../decisions/005-ufw-firewall.md) para decisiones de diseño.

## Reglas Necesarias por Servicio

### Puertos que debe aceptar el gateway

| Puerto | Protocolo | Servicio | Desde |
|--------|-----------|----------|-------|
| 22 | TCP | SSH | 10.0.0.0/24, 10.0.1.0/24 |
| 53 | TCP/UDP | DNS | 10.0.0.0/24 |
| 67-68 | UDP | DHCP | 10.0.0.0/24 |
| 69 | UDP | TFTP | 10.0.0.0/24 |
| 80 | TCP | HTTP (Traefik) | 10.0.0.0/24 |
| 111 | TCP/UDP | RPC (NFS) | 10.0.0.0/24 |
| 443 | TCP | HTTPS (Traefik) | 10.0.0.0/24 |
| 2049 | TCP/UDP | NFS | 10.0.0.0/24 |
| 6443 | TCP | k3s API Server | 10.0.0.0/24, Tailscale |
| 8472 | UDP | Flannel VXLAN | 10.0.0.0/24 |
| 9100 | TCP | node_exporter | 10.0.0.0/24 |
| 10250 | TCP | kubelet | 10.0.0.0/24 |
| 51820 | UDP | WireGuard | 0.0.0.0/0 (WAN) |

### Tráfico FORWARD necesario

| Origen | Destino | Propósito |
|--------|---------|-----------|
| 10.0.0.0/24 | Internet | Nodos acceden a internet |
| 10.0.1.0/24 | 10.0.0.0/24 | VPN accede a red interna |
| 10.0.1.0/24 | Internet | VPN accede a internet |
| tailscale0 | * | Tráfico entrante desde Tailscale |
| * | tailscale0 | Tráfico saliente hacia Tailscale |

## Configuración con UFW

### Instalación

```bash
sudo apt install ufw
```

### Habilitar forwarding en UFW

Editar `/etc/ufw/sysctl.conf`:
```bash
net/ipv4/ip_forward=1
```

### Configurar NAT en UFW

Editar `/etc/ufw/before.rules`, agregar al inicio (antes de `*filter`):
```bash
# NAT para red interna
*nat
:POSTROUTING ACCEPT [0:0]
-A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
COMMIT
```

### Reglas básicas

```bash
# Política por defecto
sudo ufw default deny incoming
sudo ufw default allow outgoing

# SSH (desde redes internas y VPN)
sudo ufw allow from 10.0.0.0/24 to any port 22 proto tcp
sudo ufw allow from 10.0.1.0/24 to any port 22 proto tcp

# DNS
sudo ufw allow from 10.0.0.0/24 to any port 53

# DHCP (se maneja diferente, es broadcast)
# UFW no puede filtrar DHCP fácilmente, se permite implícitamente

# TFTP
sudo ufw allow from 10.0.0.0/24 to any port 69 proto udp

# NFS
sudo ufw allow from 10.0.0.0/24 to any port 111
sudo ufw allow from 10.0.0.0/24 to any port 2049

# WireGuard (desde cualquier lugar en WAN)
sudo ufw allow 51820/udp

# Habilitar
sudo ufw enable
```

### Verificar estado

```bash
sudo ufw status verbose
sudo ufw status numbered
```

## Configuración con iptables (Manual)

Si prefieres iptables directamente sin UFW:

### Script de configuración

Crear `/etc/iptables/rules.sh`:
```bash
#!/bin/bash

# Limpiar reglas existentes
iptables -F
iptables -X
iptables -t nat -F

# Políticas por defecto
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# Permitir loopback
iptables -A INPUT -i lo -j ACCEPT

# Permitir conexiones establecidas
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT

# SSH desde redes internas
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -s 10.0.1.0/24 -p tcp --dport 22 -j ACCEPT

# DNS desde red interna
iptables -A INPUT -s 10.0.0.0/24 -p udp --dport 53 -j ACCEPT
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 53 -j ACCEPT

# DHCP desde red interna
iptables -A INPUT -i eth0 -p udp --dport 67:68 -j ACCEPT

# TFTP desde red interna
iptables -A INPUT -s 10.0.0.0/24 -p udp --dport 69 -j ACCEPT

# NFS desde red interna
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 111 -j ACCEPT
iptables -A INPUT -s 10.0.0.0/24 -p udp --dport 111 -j ACCEPT
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 2049 -j ACCEPT
iptables -A INPUT -s 10.0.0.0/24 -p udp --dport 2049 -j ACCEPT

# WireGuard desde WAN
iptables -A INPUT -p udp --dport 51820 -j ACCEPT

# FORWARD: Red interna a internet
iptables -A FORWARD -i eth0 -o enx00e04c683da2 -j ACCEPT

# FORWARD: VPN a red interna
iptables -A FORWARD -i wg0 -o eth0 -j ACCEPT
iptables -A FORWARD -i eth0 -o wg0 -j ACCEPT

# FORWARD: VPN a internet
iptables -A FORWARD -i wg0 -o enx00e04c683da2 -j ACCEPT

# NAT para salida a internet
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
iptables -t nat -A POSTROUTING -s 10.0.1.0/24 -o enx00e04c683da2 -j MASQUERADE
```

### Hacer reglas persistentes

```bash
# Instalar iptables-persistent
sudo apt install iptables-persistent

# Guardar reglas actuales
sudo netfilter-persistent save

# Las reglas se guardan en:
# /etc/iptables/rules.v4
# /etc/iptables/rules.v6
```

## Verificación

### Ver reglas activas

```bash
# Ver todas las reglas
sudo iptables -L -n -v

# Ver reglas NAT
sudo iptables -t nat -L -n -v

# Ver reglas con números de línea
sudo iptables -L -n --line-numbers
```

### Probar conectividad

```bash
# Desde un nodo (10.0.0.x)
ping 8.8.8.8           # Internet
ping 10.0.0.1          # Gateway
nslookup google.com    # DNS

# Desde VPN (10.0.1.x)
ping 10.0.0.2          # Nodo interno
ssh admin@10.0.0.2     # SSH a nodo
```

## Troubleshooting

### Los nodos no tienen internet después de habilitar firewall

1. Verificar regla FORWARD:
   ```bash
   sudo iptables -L FORWARD -n -v
   ```

2. Verificar NAT:
   ```bash
   sudo iptables -t nat -L POSTROUTING -n -v
   ```

3. Verificar IP forwarding:
   ```bash
   cat /proc/sys/net/ipv4/ip_forward  # Debe ser 1
   ```

### No puedo hacer SSH al gateway

1. Verificar que la regla existe:
   ```bash
   sudo iptables -L INPUT -n | grep 22
   ```

2. Verificar desde qué IP estás conectando:
   ```bash
   # Debe ser 10.0.0.x o 10.0.1.x
   ```

### WireGuard no conecta

1. Verificar que el puerto está abierto:
   ```bash
   sudo iptables -L INPUT -n | grep 51820
   ```

2. Verificar que el servicio está corriendo:
   ```bash
   sudo wg show
   ```

### Los nodos no bootean (TFTP/NFS)

1. Verificar puertos TFTP y NFS:
   ```bash
   sudo iptables -L INPUT -n | grep -E "69|2049|111"
   ```

2. Temporalmente deshabilitar firewall para probar:
   ```bash
   sudo ufw disable
   # o
   sudo iptables -P INPUT ACCEPT
   sudo iptables -P FORWARD ACCEPT
   ```

## Logs

### Habilitar logging de paquetes rechazados

```bash
# Agregar antes de las reglas DROP
iptables -A INPUT -j LOG --log-prefix "IPT-INPUT-DROP: " --log-level 4
iptables -A FORWARD -j LOG --log-prefix "IPT-FORWARD-DROP: " --log-level 4
```

### Ver logs

```bash
sudo tail -f /var/log/syslog | grep IPT
# o
sudo journalctl -f | grep IPT
```

## Notas sobre k3s y Tailscale

### k3s

El playbook `k3s.yml` agrega automáticamente reglas iptables para FORWARD de Tailscale y las persiste con `iptables-persistent`. Los puertos críticos de k3s son:

- **6443/TCP** - API Server (kubectl se conecta aquí)
- **8472/UDP** - Flannel VXLAN (comunicación entre pods de distintos nodos)
- **10250/TCP** - kubelet (métricas y logs de pods)

El playbook `firewall.yml` permite todo el tráfico desde la LAN en eth0 (`ufw allow in on eth0 from 10.0.0.0/24`), lo que cubre estos puertos para comunicación entre nodos.

### Tailscale

Tailscale maneja su propio tunnel y no necesita puertos explícitos en UFW. Sin embargo, el playbook `k3s.yml` agrega reglas FORWARD para la interfaz `tailscale0` para permitir que el tráfico de kubectl via Tailscale llegue al API Server.

## Próximos Pasos

1. ~~Crear rol Ansible para firewall~~ (completado: `playbooks/firewall.yml`)
2. ~~Agregar rate limiting para protección contra fuerza bruta~~ (completado: `ufw limit` en SSH)
3. ~~Revisar reglas HTTP/HTTPS para servicios web~~ (completado: Traefik via MetalLB)
4. Considerar fail2ban para SSH
5. Agregar logging de paquetes rechazados
