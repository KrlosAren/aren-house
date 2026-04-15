# Configuración de Storage Local

## Problema

Los nodos bootean por NFS, lo que significa que su filesystem completo está en el gateway. Docker por defecto usa `/var/lib/docker` que estaría en NFS.
```
NFS + Docker overlay2 = ❌ No funciona
NFS + Docker vfs = ✅ Funciona pero lento
```

## Solución

Usar storage local (SSD) para Docker y k3s mientras se mantiene netboot.
```
rp2-node / rp3-node:
├── / (root)         → NFS (gateway)
├── /mnt/ssd         → SSD local (LABEL=ssd)
├── /var/lib/docker  → symlink a /mnt/ssd/docker
└── /var/lib/rancher → symlink a /mnt/ssd/rancher
```

---

## Configuración Actual

| Nodo | Dispositivo | Partición | Label | Montaje | Tamaño | Uso |
|------|-------------|-----------|-------|---------|--------|-----|
| rp2 | SSD USB | /dev/sda1 | ssd | /mnt/ssd | 500GB | Docker, k3s, Longhorn |
| rp3 | SSD USB | /dev/sda1 | ssd | /mnt/ssd | 500GB | Docker, k3s, Longhorn |

---

## Pasos de Configuración

### 1. Identificar disco
```bash
lsblk
lsblk -f  # Ver filesystems y UUIDs
```

### 2. Formatear y etiquetar (si es necesario)
```bash
# Para SSD nuevo (una sola partición)
sudo mkfs.ext4 -L ssd /dev/sda1

# Para disco existente sin label
sudo e2label /dev/sda1 ssd
```

### 3. Montar
```bash
# Crear punto de montaje
sudo mkdir -p /mnt/ssd

# Montar
sudo mount /dev/sda1 /mnt/ssd
```

### 4. Configurar fstab (persistente)
```bash
# Obtener UUID
sudo blkid /dev/sda1

# Agregar a fstab
echo 'UUID=xxxxx /mnt/ssd ext4 defaults,nofail 0 2' | sudo tee -a /etc/fstab

# Verificar
sudo mount -a
```

### 5. Configurar Docker y k3s
```bash
# Detener servicios
sudo systemctl stop docker
sudo systemctl stop k3s-agent

# Crear directorios en disco local
sudo mkdir -p /mnt/ssd/docker /mnt/ssd/rancher

# Mover datos existentes y crear symlinks
sudo mv /var/lib/docker /var/lib/docker.old
sudo ln -s /mnt/ssd/docker /var/lib/docker

sudo mv /var/lib/rancher /var/lib/rancher.old
sudo ln -s /mnt/ssd/rancher /var/lib/rancher

# Cambiar a overlay2
sudo tee /etc/docker/daemon.json << 'JSON'
{
  "storage-driver": "overlay2"
}
JSON

# Iniciar servicios
sudo systemctl start docker
sudo systemctl start k3s-agent

# Verificar
docker info | grep "Storage Driver"
```

---

## Playbook
```bash
ansible-playbook playbooks/local-storage.yml
```

El playbook:
1. Detecta disco con label `ssd`
2. Lo monta en `/mnt/ssd`
3. Crea symlinks: `/var/lib/docker` → `/mnt/ssd/docker`, `/var/lib/rancher` → `/mnt/ssd/rancher`
4. Configura Docker para usar overlay2

---

## Verificación
```bash
# Ver montaje
df -h /mnt/ssd

# Ver storage driver de Docker
docker info | grep "Storage Driver"
# Debe mostrar: overlay2

# Ver symlinks
ls -la /var/lib/docker
# Debe ser symlink a /mnt/ssd/docker

ls -la /var/lib/rancher
# Debe ser symlink a /mnt/ssd/rancher
```

---

## Storage para k3s/containerd

El mismo problema de NFS + overlay2 aplica para k3s/containerd. La solución es idéntica: symlink a disco local.

```
rp2-node / rp3-node:
├── / (root)              → NFS (gateway)
├── /mnt/ssd              → SSD local (LABEL=ssd)
├── /var/lib/docker       → symlink a /mnt/ssd/docker
└── /var/lib/rancher      → symlink a /mnt/ssd/rancher  (k3s)
```

El playbook `local-storage.yml` crea los symlinks automáticamente. Esto permite que containerd use overlay2 en vez de vfs.

Ver [ADR-010: K3s Storage en Discos Locales](decisions/010-k3s-storage-on-nfs.md) para más detalles.

### Verificación
```bash
# Verificar symlink de k3s
ls -la /var/lib/rancher
# Debe ser symlink a /mnt/ssd/rancher

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
