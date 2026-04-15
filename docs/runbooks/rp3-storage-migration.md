# Migración Storage rp3-node: 3 particiones → 1 partición

## Contexto

rp3-node tiene un SSD de 500GB con layout legacy de 3 particiones:
- `sda1`: 512MB (vacía)
- `sda2`: 32GB con label `docker` → montada en `/mnt/docker`
- `sda3`: 433GB con label `storage` → montada en `/mnt/storage`

**Objetivo**: Una sola partición `sda1` con label `ssd` montada en `/mnt/ssd` (igual que rp2-node).

## Pre-requisitos

- Acceso SSH a rp1-master y rp3-node
- kubectl configurado
- Espacio suficiente en rp1-master para backup temporal

## Pasos

### 1. Drain del nodo

```bash
kubectl drain rp3-node --ignore-daemonsets --delete-emptydir-data
```

Verificar que los pods se movieron:
```bash
kubectl get pods -A -o wide | grep rp3
```

### 2. Detener servicios en rp3

Desde rp3-node (o via NFS desde rp1-master editando `/srv/nfs/rp3/`):
```bash
ssh admin@10.0.0.3
sudo systemctl stop k3s-agent
sudo systemctl stop docker
```

### 3. Backup desde rp1-master via SSH

Los datos del SSD no son visibles via NFS (son mount points locales en rp3). El backup se hace via SSH:

```bash
# Desde rp1-master
sudo mkdir -p /tmp/rp3-backup

# Backup k3s data
sudo rsync -av admin@10.0.0.3:/mnt/docker/rancher/ /tmp/rp3-backup/rancher/

# Backup Docker data
sudo rsync -av admin@10.0.0.3:/mnt/docker/docker/ /tmp/rp3-backup/docker/

# Backup Longhorn data (si existe)
sudo rsync -av admin@10.0.0.3:/mnt/storage/longhorn/ /tmp/rp3-backup/longhorn/ 2>/dev/null || echo "No longhorn data"
```

### 4. Desmontar particiones en rp3

```bash
ssh admin@10.0.0.3

# Verificar montajes actuales
df -h | grep sda

# Desmontar
sudo umount /mnt/storage
sudo umount /mnt/docker
```

### 5. Reparticionar SSD

```bash
# En rp3-node
sudo fdisk /dev/sda

# Dentro de fdisk:
# p        (ver particiones actuales)
# d → 3    (borrar partición 3)
# d → 2    (borrar partición 2)
# d → 1    (borrar partición 1)
# n → p → 1 → Enter → Enter  (nueva partición única, todo el disco)
# w        (escribir y salir)

# Formatear con label
sudo mkfs.ext4 -L ssd /dev/sda1
```

### 6. Montar nueva partición

```bash
# En rp3-node
sudo mkdir -p /mnt/ssd
sudo mount /dev/sda1 /mnt/ssd

# Actualizar fstab
UUID=$(sudo blkid -s UUID -o value /dev/sda1)
echo "UUID=$UUID /mnt/ssd ext4 defaults,nofail 0 2" | sudo tee -a /etc/fstab

# Verificar (eliminar entradas viejas de /mnt/docker y /mnt/storage del fstab)
sudo vim /etc/fstab
```

### 7. Restaurar datos

Desde rp1-master via SSH (el SSD ya está montado en rp3 en `/mnt/ssd`):
```bash
# Crear estructura en rp3
ssh admin@10.0.0.3 "sudo mkdir -p /mnt/ssd/{rancher,docker,longhorn}"

# Restaurar k3s data
sudo rsync -av /tmp/rp3-backup/rancher/ admin@10.0.0.3:/mnt/ssd/rancher/

# Restaurar Docker data
sudo rsync -av /tmp/rp3-backup/docker/ admin@10.0.0.3:/mnt/ssd/docker/

# Restaurar Longhorn data (si existe)
sudo rsync -av /tmp/rp3-backup/longhorn/ admin@10.0.0.3:/mnt/ssd/longhorn/ 2>/dev/null || echo "No longhorn data"
```

### 8. Actualizar symlinks en rp3

```bash
ssh admin@10.0.0.3

# Docker symlink
sudo rm /var/lib/docker
sudo ln -s /mnt/ssd/docker /var/lib/docker

# k3s/rancher symlink
sudo rm /var/lib/rancher
sudo ln -s /mnt/ssd/rancher /var/lib/rancher
```

### 9. Limpiar mount points viejos

```bash
ssh admin@10.0.0.3
sudo rmdir /mnt/docker /mnt/storage 2>/dev/null || true
```

### 10. Iniciar servicios

```bash
ssh admin@10.0.0.3
sudo systemctl start docker
sudo systemctl start k3s-agent
```

### 11. Uncordon y verificar

```bash
kubectl uncordon rp3-node

# Verificar nodo Ready
kubectl get nodes

# Verificar pods se levantan
kubectl get pods -A -o wide | grep rp3
```

### 12. Actualizar node labels

```bash
kubectl label nodes rp3-node storage-size=500gb --overwrite
```

### 13. Limpiar backup temporal

```bash
# Desde rp1-master, cuando todo funcione bien
sudo rm -rf /tmp/rp3-backup
```

---

## Estandarizar label en rp2-node

rp2-node ya tiene una sola partición en `/mnt/ssd`, pero el label del disco puede no ser `ssd`. Para estandarizar:

```bash
ssh admin@10.0.0.2

# Verificar label actual
sudo e2label /dev/sda1

# Si no dice "ssd", cambiar (no requiere desmontar)
sudo e2label /dev/sda1 ssd

# Actualizar node label en k8s
kubectl label nodes rp2-node storage=ssd storage-size=500gb --overwrite
```

---

## Verificación final

```bash
# Ambos nodos deben mostrar /mnt/ssd
ssh admin@10.0.0.2 "df -h /mnt/ssd && ls -la /var/lib/docker /var/lib/rancher"
ssh admin@10.0.0.3 "df -h /mnt/ssd && ls -la /var/lib/docker /var/lib/rancher"

# Labels correctos
kubectl get nodes --show-labels | grep storage

# Todos los pods corriendo
kubectl get pods -A
```

---

## Rollback

Si algo falla durante la migración de rp3:

1. Los datos originales están en `/tmp/rp3-backup` en rp1-master
2. Si el SSD ya fue reparticionado, restaurar desde backup al nuevo layout
3. Si no se reparticionó, simplemente montar las particiones originales de vuelta

**Nota**: Este runbook es una operación de una sola vez. Después de completar, ejecutar `ansible-playbook playbooks/local-storage.yml` debería funcionar con el nuevo layout estandarizado.
