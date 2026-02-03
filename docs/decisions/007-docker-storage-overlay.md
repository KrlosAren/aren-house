# 007. Storage Driver para Docker en NFS Boot

**Fecha:** 2025-12-15
**Estado:** Aceptado

## Contexto

Los nodos bootean por NFS (ver [ADR-006](006-netboot-vs-local.md)), lo que significa que `/var/lib/docker` está en un filesystem NFS. Docker por defecto usa `overlay2` como storage driver.

El problema: **overlay2 no funciona sobre NFS**.

```
$ docker run hello-world
Error: mount source: "overlay"... invalid argument
```

## Opciones

### Opción A: Usar driver vfs
- Funciona sobre NFS
- Muy lento (copia completa de archivos en cada capa)
- No requiere hardware adicional

### Opción B: Storage local (elegida)
- Usar microSD/SSD para `/var/lib/docker`
- Permite usar overlay2 (rápido)
- Requiere disco en cada nodo

## Decisión

Usar **storage local** para Docker con driver **overlay2**.

Configuración:
```
/mnt/docker          ← microSD/SSD montado
/mnt/docker/docker   ← datos de Docker
/var/lib/docker      ← symlink a /mnt/docker/docker
```

## Consecuencias

### Positivas
- Performance mucho mejor que vfs
- Los nodos ya tienen microSD (no usada para boot)
- SSD disponible para rp3 (500GB)

### Negativas
- Gestión de dos storages por nodo (NFS para OS, local para Docker)
- Configurar fstab para montaje persistente
- Datos de Docker no centralizados

## Implementación

```bash
# Montar disco local
sudo mount /dev/mmcblk0p2 /mnt/docker

# Crear symlink
sudo ln -s /mnt/docker/docker /var/lib/docker

# Configurar daemon.json
{
  "storage-driver": "overlay2"
}
```

## Configuración Actual

| Nodo | Dispositivo | Montaje | Tamaño |
|------|-------------|---------|--------|
| rp2 | microSD | /mnt/docker | 29GB |
| rp3 | SSD | /mnt/docker | 240GB |

## Referencias

- [local-storage.md](../local-storage.md)
- Playbook: `playbooks/local-storage.yml`
