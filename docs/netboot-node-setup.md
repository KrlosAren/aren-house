# Configuración de Netboot para Nuevos Nodos

Guía paso a paso para configurar una Raspberry Pi nueva para bootear desde la red (PXE/NFS).

## Requisitos previos

- Raspberry Pi con microSD y Ubuntu Server instalado
- Gateway (rp1-master) funcionando con dnsmasq y NFS
- Acceso SSH al gateway
- El nodo conectado a la red (WiFi o cable temporal)

## Información a recopilar

Antes de empezar, necesitas obtener del nuevo nodo:
```bash
# Serial de la Raspberry Pi (últimos 8 caracteres)
cat /proc/cpuinfo | grep Serial

# MAC address de eth0
cat /sys/class/net/eth0/address
```

## Paso 1: Acceso SSH inicial

Desde tu Mac, copia tu clave SSH al nuevo nodo:
```bash
ssh-copy-id usuario_inicial@IP_DEL_NODO
```

Verifica que puedes entrar sin contraseña:
```bash
ssh usuario_inicial@IP_DEL_NODO
```

## Paso 2: Crear usuario admin (UID 1000)

### 2.1 Crear usuario temporal
```bash
# En el nuevo nodo
sudo useradd -m -s /bin/bash -G sudo tempadmin
sudo passwd tempadmin
```

### 2.2 Configurar SSH y sudo para tempadmin
```bash
# Copiar clave SSH
sudo mkdir -p /home/tempadmin/.ssh
sudo cp ~/.ssh/authorized_keys /home/tempadmin/.ssh/
sudo chown -R tempadmin:tempadmin /home/tempadmin/.ssh
sudo chmod 700 /home/tempadmin/.ssh
sudo chmod 600 /home/tempadmin/.ssh/authorized_keys

# Configurar sudo sin contraseña
echo "tempadmin ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/tempadmin
sudo chmod 440 /etc/sudoers.d/tempadmin
```

### 2.3 Conectar como tempadmin y renombrar usuario
```bash
# Salir y reconectar
exit
ssh tempadmin@IP_DEL_NODO

# Renombrar usuario (cambiar 'usuario_inicial' por el nombre real)
sudo usermod -l admin usuario_inicial
sudo groupmod -n admin usuario_inicial
sudo mv /home/usuario_inicial /home/admin
sudo usermod -d /home/admin admin

# Configurar sudo para admin
echo "admin ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/admin
sudo chmod 440 /etc/sudoers.d/admin
```

### 2.4 Verificar y limpiar
```bash
# Salir y reconectar como admin
exit
ssh admin@IP_DEL_NODO

# Verificar
id admin
sudo whoami

# Eliminar usuario temporal
sudo userdel -r tempadmin
sudo rm /etc/sudoers.d/tempadmin
```

## Paso 3: Configurar red eth0

Si el nodo está conectado por WiFi y eth0 no tiene IP, necesitas configurar netplan.

### 3.1 Verificar estado de eth0
```bash
ip addr show eth0
```

Si dice `state DOWN` o no tiene IP, continúa:

### 3.2 Editar netplan
```bash
sudo nano /etc/netplan/50-cloud-init.yaml
```

Agregar la sección de eth0:
```yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: true
  wifis:
    # ... mantener configuración WiFi existente si la hay
```

### 3.3 Aplicar configuración
```bash
sudo netplan apply
ip addr show eth0
```

Debería mostrar la IP asignada por dnsmasq (ej: 10.0.0.3).

## Paso 4: Agregar nodo a dnsmasq (si es nuevo)

En el gateway (rp1-master), agregar el nodo a la configuración de dnsmasq si no está:

### 4.1 Actualizar playbook

Editar `playbooks/gateway.yml` y agregar en `dnsmasq_hosts`:
```yaml
dnsmasq_hosts:
  - name: rp2
    mac: "2c:cf:67:88:9e:f5"
    ip: "10.0.0.2"
  - name: rp3
    mac: "2c:cf:67:a9:b9:13"    # ← Nueva MAC
    ip: "10.0.0.3"               # ← Nueva IP
```

### 4.2 Aplicar cambios
```bash
ansible-playbook playbooks/gateway.yml
```

## Paso 5: Distribuir clave SSH del gateway

Ejecutar el playbook de SSH para que rp1-master pueda conectarse al nuevo nodo:
```bash
ansible-playbook playbooks/setup-ssh.yml
```

## Paso 6: Copiar root filesystem a NFS

Desde rp1-master:

### 6.1 Asegurar que root tiene clave SSH
```bash
sudo mkdir -p /root/.ssh
sudo cp /home/admin/.ssh/id_ed25519* /root/.ssh/
sudo chmod 600 /root/.ssh/id_ed25519
sudo chmod 644 /root/.ssh/id_ed25519.pub
```

### 6.2 Copiar filesystem
```bash
sudo rsync -axv --progress admin@10.0.0.X:/ /srv/nfs/rpX/ \
  --exclude=/proc/* \
  --exclude=/sys/* \
  --exclude=/dev/* \
  --exclude=/tmp/* \
  --exclude=/run/* \
  --exclude=/mnt/* \
  --exclude=/boot/firmware/*
```

Reemplazar `X` con el número del nodo.

### 6.3 Crear directorios virtuales
```bash
sudo mkdir -p /srv/nfs/rpX/{proc,sys,dev,tmp,run,mnt}
sudo chmod 1777 /srv/nfs/rpX/tmp
```

### 6.4 Crear dispositivos esenciales
```bash
sudo mknod -m 666 /srv/nfs/rpX/dev/null c 1 3
sudo mknod -m 666 /srv/nfs/rpX/dev/zero c 1 5
sudo mknod -m 666 /srv/nfs/rpX/dev/random c 1 8
sudo mknod -m 666 /srv/nfs/rpX/dev/urandom c 1 9
sudo mknod -m 600 /srv/nfs/rpX/dev/console c 5 1
sudo mknod -m 666 /srv/nfs/rpX/dev/tty c 5 0
```

### 6.5 Configurar fstab para NFS boot
```bash
sudo nano /srv/nfs/rpX/etc/fstab
```

Reemplazar contenido con:
```
proc            /proc   proc    defaults          0 0
sysfs           /sys    sysfs   defaults          0 0
tmpfs           /tmp    tmpfs   defaults          0 0
tmpfs           /run    tmpfs   defaults          0 0
```

## Paso 7: Configurar TFTP

### 7.1 Crear directorio con serial
```bash
# Usar los últimos 8 caracteres del serial
sudo mkdir -p /srv/tftp/SERIAL_DEL_NODO

# Crear symlink por MAC (opcional pero útil)
sudo ln -s /srv/tftp/SERIAL_DEL_NODO /srv/tftp/MAC-DEL-NODO
```

Formato de MAC para symlink: `2c-cf-67-a9-b9-13` (guiones en lugar de dos puntos).

### 7.2 Copiar archivos de boot
```bash
sudo rsync -av admin@10.0.0.X:/boot/firmware/ /srv/tftp/SERIAL_DEL_NODO/
```

### 7.3 Configurar cmdline.txt
```bash
sudo nano /srv/tftp/SERIAL_DEL_NODO/cmdline.txt
```

Contenido:
```
console=serial0,115200 console=tty1 root=/dev/nfs nfsroot=10.0.0.1:/srv/nfs/rpX,vers=3,tcp rw ip=dhcp rootwait
```

## Paso 8: Configurar EEPROM para netboot

En el nodo (con microSD aún conectada):

### 8.1 Crear archivo de configuración
```bash
cat > /tmp/boot.conf << 'EOF'
BOOT_ORDER=0xf2461
TFTP_PREFIX=2
NET_INSTALL_AT_POWER_ON=0
EOF
```

**BOOT_ORDER explicado:**
- `1` = SD card
- `4` = USB
- `6` = NVMe
- `2` = Network
- `f` = Reiniciar si todo falla

`0xf2461` = Intenta SD → NVMe → USB → Network → Reiniciar

### 8.2 Aplicar configuración
```bash
sudo rpi-eeprom-config --apply /tmp/boot.conf
```

### 8.3 Verificar
```bash
sudo rpi-eeprom-config
```

## Paso 9: Probar netboot

1. Apagar el nodo: `sudo poweroff`
2. Retirar la microSD
3. Encender el nodo
4. Verificar desde rp1-master:
```bash
# Ver logs de DHCP/TFTP
sudo tail -f /var/log/dnsmasq.log

# Probar conexión cuando bootee
ping 10.0.0.X
ssh admin@10.0.0.X
```

## Troubleshooting

### El nodo no obtiene IP
```bash
# En rp1-master, verificar leases
cat /var/lib/misc/dnsmasq.leases

# Verificar que la MAC está en dnsmasq.conf
grep "MAC_DEL_NODO" /etc/dnsmasq.conf
```

### TFTP no envía archivos
```bash
# Verificar que TFTP está escuchando
sudo ss -ulnp | grep 69

# Verificar permisos de archivos
ls -la /srv/tftp/SERIAL_DEL_NODO/
```

### NFS no monta
```bash
# Verificar exports
sudo exportfs -v

# Verificar que el nodo está en la lista
grep rpX /etc/exports
```

### SSH no funciona después de netboot

El archivo `/etc/shadow` puede no tener las credenciales correctas:
```bash
# Desde rp1-master, copiar shadow del nodo (bootear con SD temporalmente)
# O recrear el archivo sudoers
echo "admin ALL=(ALL) NOPASSWD: ALL" | sudo tee /srv/nfs/rpX/etc/sudoers.d/admin
sudo chmod 440 /srv/nfs/rpX/etc/sudoers.d/admin
```

### El nodo no tiene internet

Verificar que NAT está configurado en rp1-master:
```bash
sudo iptables -t nat -L POSTROUTING -v
```

Si está vacío, agregar la regla:
```bash
sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
```

## Resumen de archivos importantes

| Ubicación | Descripción |
|-----------|-------------|
| `/srv/nfs/rpX/` | Root filesystem del nodo |
| `/srv/tftp/SERIAL/` | Archivos de boot (kernel, initrd, config) |
| `/srv/tftp/SERIAL/cmdline.txt` | Parámetros de boot |
| `/etc/dnsmasq.conf` | Configuración DHCP/DNS/TFTP |
| `/etc/exports` | Exports NFS |

## Checklist para nuevo nodo

- [ ] Obtener serial y MAC
- [ ] Crear usuario admin (UID 1000)
- [ ] Configurar eth0 en netplan
- [ ] Agregar a dnsmasq_hosts en playbook
- [ ] Distribuir clave SSH con playbook
- [ ] Copiar filesystem a /srv/nfs/rpX
- [ ] Crear directorios virtuales y dispositivos
- [ ] Configurar fstab para NFS
- [ ] Copiar archivos de boot a TFTP
- [ ] Configurar cmdline.txt
- [ ] Configurar EEPROM
- [ ] Retirar microSD y probar netboot
