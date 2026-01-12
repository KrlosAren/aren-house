# Troubleshooting

Problemas comunes y soluciones encontradas durante la configuración del homelab.

---

## Netboot

### Nodo no bootea después de configurar firewall

**Síntoma:** El nodo muestra "netboot init failed"

**Causa:** El firewall bloquea DHCP o TFTP broadcast

**Solución:**
```bash
# En el gateway, agregar reglas por interfaz
sudo ufw allow in on eth0 to any port 67:68 proto udp
sudo ufw allow in on eth0 to any port 69 proto udp
sudo ufw allow in on eth0 from 10.0.0.0/24
```

### Error "Couldn't stat '/etc/ufw/user.rules'" en nodos

**Síntoma:** ufw falla al configurar en nodos NFS boot

**Causa:** ufw no se inicializó correctamente sobre NFS

**Solución:**
```bash
# Reinstalar ufw
sudo apt purge ufw -y
sudo apt install ufw -y
```

### Perdí acceso SSH a los nodos

**Síntoma:** No puedo conectar por SSH después de configurar firewall

**Causa:** Las reglas de firewall bloquearon SSH

**Solución (desde gateway via NFS):**
```bash
# Deshabilitar ufw en el nodo editando su filesystem
sudo sed -i 's/ENABLED=yes/ENABLED=no/' /srv/nfs/rp2/etc/ufw/ufw.conf

# Reiniciar el nodo (desconectar/conectar alimentación)
```

---

## Docker

### Error "overlay... invalid argument"

**Síntoma:** `docker run` falla con error de overlay

**Causa:** overlay2 no funciona sobre NFS

**Solución:**
```bash
# Usar storage local o cambiar a vfs
sudo tee /etc/docker/daemon.json << 'EOF'
{
  "storage-driver": "vfs"
}
EOF
sudo systemctl restart docker
```

**Mejor solución:** Usar microSD/SSD local para Docker

### Docker pull muy lento

**Síntoma:** Descargar imágenes toma mucho tiempo

**Causa:** Usando driver vfs sobre NFS

**Solución:** Configurar storage local con overlay2 (ver [local-storage.md](local-storage.md))

---

## VPN / Acceso Remoto

### WireGuard no conecta desde internet

**Síntoma:** `transfer: 0 B received` en `wg show`

**Causa:** ISP usa CGNAT, puerto 51820 no llega

**Verificar:**
```bash
# Ver IP de la WAN del modem
# Si es 100.64.x.x - 100.127.x.x = CGNAT
```

**Solución:** Usar Tailscale en lugar de WireGuard directo

### Tailscale no puede acceder a la subred

**Síntoma:** Puedo conectar al gateway pero no a los nodos

**Causa:** Subnet router no aprobado

**Solución:**
1. Ir a https://login.tailscale.com/admin/machines
2. Encontrar rp1-master
3. Edit route settings → Aprobar 10.0.0.0/24

### DNS .homelab.local no resuelve desde VPN

**Síntoma:** `ping rp1.homelab.local` falla

**Causa:** Mac no usa el DNS del homelab

**Solución:**
```bash
sudo mkdir -p /etc/resolver
echo "nameserver 10.0.0.1" | sudo tee /etc/resolver/homelab.local
```

---

## Firewall

### Orden de reglas bloquea acceso

**Síntoma:** Pierdo acceso SSH al aplicar firewall

**Causa:** Policy `deny` se aplica antes que las reglas `allow`

**Solución:** En el playbook, crear reglas `allow` ANTES de configurar policy `deny`:
```yaml
# CORRECTO
1. Crear reglas allow
2. Configurar policy deny
3. Habilitar ufw

# INCORRECTO
1. Configurar policy deny  ← Bloquea todo
2. Crear reglas allow      ← Nunca llega aquí
```

### NFS no funciona a través del firewall

**Síntoma:** Nodos no pueden montar NFS

**Causa:** NFS usa puertos dinámicos además de 2049 y 111

**Solución:**
```bash
# Permitir todo desde LAN
sudo ufw allow in on eth0 from 10.0.0.0/24
```

---

## Paquetes / Actualizaciones

### apt 404 errors en nodos

**Síntoma:** `apt update` falla con 404

**Causa:** Cache stale en Ubuntu development version

**Solución:**
```bash
sudo rm -rf /var/lib/apt/lists/*
sudo apt update
```

### flash-kernel falla en NFS boot

**Síntoma:** Error buscando `/boot/firmware/current/cmdline.txt`

**Causa:** flash-kernel espera boot local, no TFTP

**Solución:**
```bash
# Deshabilitar hooks
sudo chmod -x /etc/initramfs/post-update.d/flash-kernel
sudo chmod -x /etc/kernel/postinst.d/zz-flash-kernel
sudo chmod -x /etc/kernel/postrm.d/zz-flash-kernel

# Marcar como hold
sudo apt-mark hold flash-kernel flash-kernel-piboot
```

### snapd falla en NFS boot

**Síntoma:** `setcap: Operation not supported`

**Causa:** NFS no soporta Linux capabilities

**Solución:**
```bash
sudo apt remove --purge snapd -y
```

---

## Storage

### Disco no monta después de reboot

**Síntoma:** /mnt/docker vacío después de reiniciar

**Causa:** Falta entrada en fstab

**Solución:**
```bash
# Obtener UUID
sudo blkid /dev/mmcblk0p2

# Agregar a fstab
echo 'UUID=xxxxx /mnt/docker ext4 defaults 0 2' | sudo tee -a /etc/fstab

# Verificar
sudo mount -a
```

### Docker no usa el disco local

**Síntoma:** Docker sigue usando NFS

**Causa:** Symlink no creado o daemon.json incorrecto

**Solución:**
```bash
# Verificar symlink
ls -la /var/lib/docker
# Debe apuntar a /mnt/docker/docker

# Verificar config
cat /etc/docker/daemon.json
# Debe tener "storage-driver": "overlay2"

# Reiniciar
sudo systemctl restart docker
```

---

## Comandos de Diagnóstico

### Red
```bash
# Ver interfaces
ip addr

# Ver rutas
ip route

# Ver conexiones
ss -tulnp

# Test DNS
nslookup rp2.homelab.local 10.0.0.1

# Test conectividad
ping -c 2 10.0.0.1
```

### Firewall
```bash
# Ver reglas
sudo ufw status verbose

# Ver logs de bloqueos
sudo tail -f /var/log/ufw.log
```

### Docker
```bash
# Ver info
docker info

# Ver contenedores
docker ps -a

# Ver logs
docker logs <container>

# Ver uso de disco
docker system df
```

### Systemd
```bash
# Ver estado de servicio
systemctl status <service>

# Ver logs
journalctl -u <service> -f

# Reiniciar servicio
sudo systemctl restart <service>
```
