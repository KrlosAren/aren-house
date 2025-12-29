# Gestión de Servicios

Guía para administrar los servicios del homelab.

## Servicios en el Gateway (rp1-master)

| Servicio | Unit systemd | Puerto | Función |
|----------|--------------|--------|---------|
| dnsmasq | dnsmasq.service | 53, 67, 69 | DHCP, DNS, TFTP |
| NFS | nfs-kernel-server.service | 2049, 111 | Root filesystem para netboot |
| WireGuard | wg-quick@wg0.service | 51820 | VPN |

## Comandos Básicos

### Ver estado de todos los servicios

```bash
# Estado resumido
systemctl status dnsmasq nfs-kernel-server wg-quick@wg0

# Estado detallado de uno
sudo systemctl status dnsmasq
```

### Iniciar/Detener/Reiniciar

```bash
# Reiniciar servicio
sudo systemctl restart dnsmasq

# Detener servicio
sudo systemctl stop dnsmasq

# Iniciar servicio
sudo systemctl start dnsmasq

# Recargar configuración sin reiniciar (si el servicio lo soporta)
sudo systemctl reload dnsmasq
```

### Habilitar/Deshabilitar al boot

```bash
# Habilitar inicio automático
sudo systemctl enable dnsmasq

# Deshabilitar inicio automático
sudo systemctl disable dnsmasq

# Ver si está habilitado
systemctl is-enabled dnsmasq
```

## Dependencias entre Servicios

```
                    ┌─────────────────┐
                    │   networking    │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│    dnsmasq      │ │   nfs-server    │ │   wg-quick@wg0  │
│  (DHCP/DNS/TFTP)│ │   (NFS)         │ │   (VPN)         │
└─────────────────┘ └─────────────────┘ └─────────────────┘
         │                   │
         │                   │
         ▼                   ▼
┌─────────────────────────────────────┐
│         Nodos (rp2, rp3)            │
│   Dependen de DHCP, TFTP, NFS       │
└─────────────────────────────────────┘
```

### Orden de inicio correcto

1. **networking** - Red debe estar lista
2. **dnsmasq** - DHCP/DNS/TFTP para que nodos puedan bootear
3. **nfs-kernel-server** - NFS para root filesystem
4. **wg-quick@wg0** - VPN (opcional, no crítico para nodos)

### Orden de parada seguro

1. Avisar a usuarios que se va a reiniciar
2. Si es posible, apagar nodos primero (evita errores NFS)
3. Detener servicios en orden inverso

## Health Checks

### Script de verificación rápida

```bash
#!/bin/bash
# health-check.sh

echo "=== Estado de Servicios ==="
for svc in dnsmasq nfs-kernel-server wg-quick@wg0; do
    status=$(systemctl is-active $svc)
    echo "$svc: $status"
done

echo ""
echo "=== Puertos Escuchando ==="
sudo ss -tulnp | grep -E ":(53|67|69|111|2049|51820) "

echo ""
echo "=== DHCP Leases ==="
cat /var/lib/misc/dnsmasq.leases

echo ""
echo "=== NFS Exports ==="
sudo exportfs -v

echo ""
echo "=== WireGuard ==="
sudo wg show
```

### Verificaciones individuales

#### dnsmasq
```bash
# Servicio activo
systemctl is-active dnsmasq

# Puertos escuchando
sudo ss -ulnp | grep dnsmasq

# Resolución DNS funciona
nslookup rp2.homelab.local 127.0.0.1

# DHCP leases
cat /var/lib/misc/dnsmasq.leases
```

#### NFS
```bash
# Servicio activo
systemctl is-active nfs-kernel-server

# Exports configurados
sudo exportfs -v

# Puerto escuchando
sudo ss -tlnp | grep 2049

# Montar desde un cliente para probar
# (desde otro host)
sudo mount -t nfs 10.0.0.1:/srv/nfs/rp2 /mnt/test
```

#### WireGuard
```bash
# Servicio activo
systemctl is-active wg-quick@wg0

# Interfaz existe
ip addr show wg0

# Estado de peers
sudo wg show

# Puerto escuchando
sudo ss -ulnp | grep 51820
```

## Logs

### Ubicación de logs

| Servicio | Log |
|----------|-----|
| dnsmasq | /var/log/dnsmasq.log |
| NFS | journalctl -u nfs-kernel-server |
| WireGuard | journalctl -u wg-quick@wg0 |
| Sistema | /var/log/syslog, journalctl |

### Ver logs en tiempo real

```bash
# dnsmasq (DHCP, DNS, TFTP)
sudo tail -f /var/log/dnsmasq.log

# NFS
sudo journalctl -u nfs-kernel-server -f

# WireGuard
sudo journalctl -u wg-quick@wg0 -f

# Todo el sistema
sudo journalctl -f
```

### Filtrar logs por tiempo

```bash
# Últimas 2 horas
journalctl -u dnsmasq --since "2 hours ago"

# Desde el último boot
journalctl -u nfs-kernel-server -b

# Hoy
journalctl -u wg-quick@wg0 --since today
```

## Problemas Comunes

### dnsmasq no inicia

**Síntoma**: `systemctl status dnsmasq` muestra error

**Causas comunes**:
1. Puerto 53 ocupado por systemd-resolved
   ```bash
   sudo systemctl disable systemd-resolved
   sudo systemctl stop systemd-resolved
   ```

2. Error de sintaxis en configuración
   ```bash
   dnsmasq --test
   ```

3. Interfaz de red no existe
   ```bash
   ip link show eth0
   ```

### NFS no exporta

**Síntoma**: `exportfs -v` no muestra exports

**Causas comunes**:
1. Error de sintaxis en /etc/exports
   ```bash
   sudo exportfs -ra  # Muestra errores
   ```

2. Directorio no existe
   ```bash
   ls -la /srv/nfs/
   ```

3. rpcbind no está corriendo
   ```bash
   systemctl status rpcbind
   ```

### WireGuard no conecta

**Síntoma**: Clientes no pueden establecer túnel

**Causas comunes**:
1. Llaves incorrectas
   ```bash
   sudo cat /etc/wireguard/public.key
   # Verificar que coincide con la del cliente
   ```

2. Puerto bloqueado
   ```bash
   sudo ss -ulnp | grep 51820
   ```

3. IP forwarding deshabilitado
   ```bash
   cat /proc/sys/net/ipv4/ip_forward  # Debe ser 1
   ```

## Reinicio Seguro del Gateway

### Antes de reiniciar

1. Verificar que no hay operaciones críticas en nodos:
   ```bash
   # Ver procesos en nodos via NFS
   ps aux | grep -E "rsync|apt|dpkg"
   ```

2. Avisar si hay usuarios conectados:
   ```bash
   who
   ```

### Durante el reinicio

Los nodos seguirán funcionando con:
- Su kernel en memoria
- Cache de NFS (si está configurado)
- Pero no podrán escribir a disco

### Después del reinicio

1. Verificar todos los servicios:
   ```bash
   ./health-check.sh
   ```

2. Verificar que los nodos están accesibles:
   ```bash
   ansible nodes -m ping
   ```

## Mantenimiento Programado

### Actualizar configuración de dnsmasq

```bash
# 1. Editar configuración
sudo nano /etc/dnsmasq.conf

# 2. Verificar sintaxis
dnsmasq --test

# 3. Recargar (sin interrumpir servicio)
sudo systemctl reload dnsmasq
```

### Actualizar exports NFS

```bash
# 1. Editar exports
sudo nano /etc/exports

# 2. Aplicar cambios sin reiniciar
sudo exportfs -ra

# 3. Verificar
sudo exportfs -v
```

### Agregar peer WireGuard

```bash
# 1. Editar wg0.conf
sudo nano /etc/wireguard/wg0.conf

# 2. Aplicar sin reiniciar túnel
sudo wg syncconf wg0 <(sudo wg-quick strip wg0)

# 3. Verificar
sudo wg show
```
