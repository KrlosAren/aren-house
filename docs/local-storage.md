# Configuración de Storage Local

## Problema

Los nodos bootean por NFS, lo que significa que su filesystem completo está en el gateway. Docker por defecto usa `/var/lib/docker` que estaría en NFS.
```
NFS + Docker overlay2 = ❌ No funciona
NFS + Docker vfs = ✅ Funciona pero lento
```

## Solución

Usar storage local (microSD o SSD) para Docker mientras se mantiene netboot.
```
rp2-node:
├── / (root)         → NFS (gateway)
├── /mnt/docker      → microSD local
└── /var/lib/docker  → symlink a /mnt/docker/docker
```

---

## Configuración Actual

| Nodo | Dispositivo | Partición | Montaje | Tamaño | Uso |
|------|-------------|-----------|---------|--------|-----|
| rp2 | microSD | /dev/mmcblk0p2 | /mnt/docker | 29GB | Docker |
| rp3 | SSD | /dev/sda2 | /mnt/docker | 32GB | Docker |
| rp3 | SSD | /dev/sda3 | /mnt/storage | 433GB | Storage/Backups |

---

## Pasos de Configuración

### 1. Identificar disco
```bash
lsblk
lsblk -f  # Ver filesystems y UUIDs
```

### 2. Formatear (si es necesario)
```bash
# Para microSD existente (ya tiene ext4)
# No necesita formatear

# Para SSD nuevo
sudo mkfs.ext4 -L docker /dev/sdX2
sudo mkfs.ext4 -L storage /dev/sdX3
```

### 3. Montar
```bash
# Crear punto de montaje
sudo mkdir -p /mnt/docker

# Montar
sudo mount /dev/mmcblk0p2 /mnt/docker  # microSD
# o
sudo mount /dev/sda2 /mnt/docker       # SSD
```

### 4. Configurar fstab (persistente)
```bash
# Obtener UUID
sudo blkid /dev/mmcblk0p2

# Agregar a fstab
echo 'UUID=xxxxx /mnt/docker ext4 defaults 0 2' | sudo tee -a /etc/fstab

# Verificar
sudo mount -a
```

### 5. Configurar Docker
```bash
# Detener Docker
sudo systemctl stop docker

# Mover datos existentes
sudo mv /var/lib/docker /var/lib/docker.old

# Crear directorio en disco local
sudo mkdir -p /mnt/docker/docker

# Crear symlink
sudo ln -s /mnt/docker/docker /var/lib/docker

# Cambiar a overlay2
sudo tee /etc/docker/daemon.json << 'JSON'
{
  "storage-driver": "overlay2"
}
JSON

# Iniciar Docker
sudo systemctl start docker

# Verificar
docker info | grep "Storage Driver"
```

---

## Playbook
```bash
ansible-playbook playbooks/local-storage.yml
```

El playbook:
1. Detecta discos con label `docker` o `writable`
2. Los monta en `/mnt/docker`
3. Configura Docker para usar overlay2
4. Opcionalmente monta disco `storage`

---

## Verificación
```bash
# Ver montajes
df -h /mnt/docker /mnt/storage

# Ver storage driver de Docker
docker info | grep "Storage Driver"
# Debe mostrar: overlay2

# Ver dónde está Docker
ls -la /var/lib/docker
# Debe ser symlink a /mnt/docker/docker
```

---

## Storage para k3s/containerd

El mismo problema de NFS + overlay2 aplica para k3s/containerd. La solución es idéntica: symlink a disco local.

```
rp2-node:
├── / (root)              → NFS (gateway)
├── /mnt/docker           → microSD local
├── /var/lib/docker       → symlink a /mnt/docker/docker
└── /var/lib/rancher      → symlink a /mnt/docker/rancher  (k3s)
```

El playbook `k3s.yml` crea automáticamente:
1. Directorio `/var/lib/rancher-local` en el disco local
2. Symlink `/var/lib/rancher` → `/var/lib/rancher-local`

Esto permite que containerd use overlay2 en vez de vfs.

Ver [ADR-010: K3s Storage en Discos Locales](decisions/010-k3s-storage-on-nfs.md) para más detalles.

### Verificación
```bash
# Verificar symlink de k3s
ls -la /var/lib/rancher
# Debe ser symlink a /var/lib/rancher-local o /mnt/docker/rancher

# Verificar que containerd usa overlay
sudo crictl info | grep -i overlay
```

---

## Troubleshooting

### Docker sigue usando vfs
```bash
# Verificar symlink
ls -la /var/lib/docker

# Verificar daemon.json
cat /etc/docker/daemon.json

# Reiniciar Docker
sudo systemctl restart docker
```

### Disco no monta después de reboot
```bash
# Verificar fstab
cat /etc/fstab

# Verificar UUID
sudo blkid

# Montar manualmente
sudo mount -a

# Ver errores
sudo dmesg | tail -20
```

### Espacio insuficiente
```bash
# Ver uso
df -h

# Limpiar Docker
docker system prune -a
```
