# Troubleshooting de Red

Guía sistemática para diagnosticar problemas de conectividad en el homelab.

## Matriz de Conectividad

### Qué debe poder alcanzar qué

| Origen | Destino | Protocolo | Puerto | Propósito |
|--------|---------|-----------|--------|-----------|
| Nodos (10.0.0.x) | Gateway (10.0.0.1) | ICMP | - | Ping |
| Nodos | Gateway | UDP | 53 | DNS |
| Nodos | Gateway | TCP/UDP | 2049 | NFS |
| Nodos | Internet | TCP/UDP | * | Navegación, apt |
| VPN (10.0.1.x) | Gateway | ICMP | - | Ping |
| VPN | Nodos | TCP | 22 | SSH |
| VPN | Internet | TCP/UDP | * | Navegación |
| Mac (VPN) | Gateway | UDP | 51820 | Túnel WireGuard |

## Diagnóstico por Capas

### Capa 1: Física

```bash
# Ver estado de interfaces
ip link show

# Verificar cable conectado
ethtool eth0 | grep "Link detected"

# Ver estadísticas de errores
ip -s link show eth0
```

**Problemas comunes**:
- Cable desconectado
- Puerto del switch dañado
- Adaptador USB-Ethernet no reconocido

### Capa 2: Enlace (MAC)

```bash
# Ver tabla ARP
ip neigh show

# Ver MAC de la interfaz
ip link show eth0 | grep ether

# Verificar que switch ve el dispositivo
# (en el switch TP-Link via web interface)
```

**Problemas comunes**:
- MAC incorrecta en dnsmasq
- Switch no reenvía tráfico
- Conflicto de MAC

### Capa 3: Red (IP)

```bash
# Ver configuración IP
ip addr show

# Ver tabla de rutas
ip route show

# Verificar gateway
ip route get 8.8.8.8

# Ping al gateway
ping -c 3 10.0.0.1
```

**Problemas comunes**:
- IP no asignada (DHCP falla)
- Ruta por defecto incorrecta
- IP duplicada

### Capa 4+: Transporte/Aplicación

```bash
# Verificar puertos abiertos
ss -tulnp

# Probar puerto específico
nc -zv 10.0.0.1 22

# Probar DNS
nslookup google.com 10.0.0.1

# Probar HTTP
curl -I https://google.com
```

## Escenarios de Troubleshooting

### Escenario 1: Nodo sin IP

**Síntomas**: El nodo no tiene IP en eth0

**Diagnóstico**:
```bash
# En el nodo (si puedes acceder)
ip addr show eth0

# En el gateway - ver leases DHCP
cat /var/lib/misc/dnsmasq.leases

# En el gateway - ver logs DHCP
sudo tail -f /var/log/dnsmasq.log
```

**Causas y soluciones**:

1. **DHCP no responde**
   ```bash
   # Verificar dnsmasq
   systemctl status dnsmasq
   sudo journalctl -u dnsmasq --since "5 min ago"
   ```

2. **MAC no está en dnsmasq_hosts**
   ```bash
   # Verificar configuración
   grep "dhcp-host" /etc/dnsmasq.conf
   ```

3. **eth0 no configurada en netplan**
   ```bash
   # En el nodo
   cat /etc/netplan/*.yaml
   # Debe incluir eth0 con dhcp4: true
   ```

### Escenario 2: Nodo sin Internet

**Síntomas**: Ping a gateway OK, ping a 8.8.8.8 falla

**Diagnóstico**:
```bash
# Desde el nodo
ping 10.0.0.1      # Gateway - debe funcionar
ping 8.8.8.8       # Internet - falla
traceroute 8.8.8.8 # Ver dónde se pierde
```

**Causas y soluciones**:

1. **NAT no configurado**
   ```bash
   # En gateway
   sudo iptables -t nat -L POSTROUTING -n -v
   # Debe mostrar regla MASQUERADE

   # Si no existe:
   sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
   ```

2. **IP forwarding deshabilitado**
   ```bash
   # En gateway
   cat /proc/sys/net/ipv4/ip_forward
   # Debe ser 1

   # Si es 0:
   sudo sysctl -w net.ipv4.ip_forward=1
   ```

3. **Gateway no tiene internet**
   ```bash
   # En gateway
   ping 8.8.8.8
   # Si falla, problema con WAN (enx00e04c683da2)
   ```

### Escenario 3: DNS no resuelve

**Síntomas**: Ping a IP funciona, ping a nombre falla

**Diagnóstico**:
```bash
# Desde el nodo
nslookup google.com              # Falla
nslookup google.com 10.0.0.1     # Probar directamente al gateway
nslookup google.com 8.8.8.8      # Probar DNS externo
```

**Causas y soluciones**:

1. **Nodo no usa gateway como DNS**
   ```bash
   # En el nodo
   resolvectl status
   cat /etc/resolv.conf
   # Debe apuntar a 10.0.0.1
   ```

2. **dnsmasq no resuelve upstream**
   ```bash
   # En gateway
   nslookup google.com 127.0.0.1
   # Ver configuración
   grep "server=" /etc/dnsmasq.conf
   ```

3. **Puerto 53 bloqueado**
   ```bash
   # En gateway
   sudo ss -ulnp | grep ":53"
   ```

### Escenario 4: VPN no conecta

**Síntomas**: WireGuard no establece túnel

**Diagnóstico**:
```bash
# En Mac
sudo wg show

# En gateway
sudo wg show
sudo journalctl -u wg-quick@wg0 --since "5 min ago"
```

**Causas y soluciones**:

1. **Puerto bloqueado en modem/router**
   ```bash
   # Verificar que 51820/UDP está abierto
   # Puede requerir port forwarding en el modem
   ```

2. **Llaves incorrectas**
   ```bash
   # Verificar llave pública del servidor
   sudo cat /etc/wireguard/public.key
   # Debe coincidir con PublicKey en config del cliente
   ```

3. **Endpoint incorrecto**
   ```bash
   # El cliente debe apuntar a IP pública o IP del modem
   # No a 10.0.0.1
   ```

4. **AllowedIPs mal configurado**
   ```bash
   # En servidor, verificar allowed_ips del peer
   grep -A2 "Peer" /etc/wireguard/wg0.conf
   ```

### Escenario 5: Nodo no bootea (Netboot)

**Síntomas**: Raspberry Pi no arranca desde red

**Diagnóstico**:
```bash
# En gateway - ver logs TFTP
sudo tail -f /var/log/dnsmasq.log

# Verificar estructura TFTP
ls -la /srv/tftp/

# Verificar NFS exports
sudo exportfs -v
```

**Causas y soluciones**:

1. **TFTP no encuentra archivos**
   ```bash
   # Pi busca por serial o MAC
   # Verificar que existe directorio
   ls /srv/tftp/440dc91d/    # Por serial
   ls -la /srv/tftp/2c-cf-67-88-9e-f5  # Symlink por MAC
   ```

2. **EEPROM mal configurada**
   ```bash
   # Requiere microSD para verificar
   sudo rpi-eeprom-config
   # BOOT_ORDER debe incluir red (0x2)
   ```

3. **NFS no monta**
   ```bash
   # Verificar export existe
   sudo exportfs -v | grep rp2

   # Probar mount manual
   sudo mount -t nfs 10.0.0.1:/srv/nfs/rp2 /mnt/test
   ```

### Escenario 6: SSH "Permission denied"

**Síntomas**: No puedo hacer SSH a un nodo

**Diagnóstico**:
```bash
ssh -v admin@10.0.0.2
```

**Causas y soluciones**:

1. **Clave SSH no autorizada**
   ```bash
   # En el nodo (via NFS)
   cat /srv/nfs/rp2/home/admin/.ssh/authorized_keys
   ```

2. **Usuario no existe**
   ```bash
   grep admin /srv/nfs/rp2/etc/passwd
   ```

3. **/etc/shadow no copiado** (netboot)
   ```bash
   # Verificar que shadow existe y tiene contenido
   sudo ls -la /srv/nfs/rp2/etc/shadow
   ```

## Comandos de Diagnóstico Rápido

### Desde el Gateway

```bash
# Estado general
ip addr show
ip route show
ss -tulnp

# Servicios
systemctl status dnsmasq nfs-kernel-server wg-quick@wg0

# Conectividad
ping -c 1 10.0.0.2  # Nodo
ping -c 1 8.8.8.8   # Internet

# DHCP
cat /var/lib/misc/dnsmasq.leases

# NAT
sudo iptables -t nat -L -n -v
```

### Desde un Nodo

```bash
# Red
ip addr show eth0
ip route show

# Gateway
ping -c 1 10.0.0.1

# Internet
ping -c 1 8.8.8.8

# DNS
nslookup google.com

# NFS mount
mount | grep nfs
```

### Desde Mac (VPN)

```bash
# Estado VPN
sudo wg show

# Gateway VPN
ping 10.0.1.1

# Red interna
ping 10.0.0.1
ping 10.0.0.2

# DNS
ping rp2.homelab.local
```

## Herramientas Útiles

```bash
# Capturar tráfico
sudo tcpdump -i eth0 port 67 or port 68  # DHCP
sudo tcpdump -i eth0 port 53             # DNS
sudo tcpdump -i eth0 port 69             # TFTP

# Ver conexiones
ss -tunap

# Trazar ruta
traceroute 8.8.8.8
mtr 8.8.8.8

# Ver tabla ARP
arp -a

# Ver procesos usando red
sudo lsof -i
```
