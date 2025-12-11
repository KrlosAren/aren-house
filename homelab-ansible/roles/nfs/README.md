# Role: nfs

Instala y configura NFS server para servir root filesystems a nodos que bootean por red (PXE/netboot).

## Función

Exporta directorios via NFS para que los nodos worker monten su root filesystem desde el gateway.

## Variables

| Variable | Default | Descripción |
|----------|---------|-------------|
| `nfs_base_path` | `/srv/nfs` | Directorio base para filesystems |
| `nfs_network` | `10.0.0.0/24` | Red permitida para montar NFS |
| `nfs_export_options` | `rw,sync,no_subtree_check,no_root_squash` | Opciones de exportación |
| `nfs_nodes` | `[]` | Lista de nodos con netboot |

## Ejemplo de uso
```yaml
- hosts: gateway
  become: yes
  vars:
    nfs_nodes:
      - name: rp2
        ip: "10.0.0.2"
      - name: rp3
        ip: "10.0.0.3"
  roles:
    - nfs
```

## Estructura creada
```
/srv/nfs/
├── rp2/    ← Root filesystem de rp2
└── rp3/    ← Root filesystem de rp3
```

## Verificación
```bash
# Ver exports activos
sudo exportfs -v

# Ver estado del servicio
sudo systemctl status nfs-kernel-server

# Probar montaje desde un cliente
sudo mount -t nfs 10.0.0.1:/srv/nfs/rp2 /mnt
```

## Notas

- Los directorios se crean vacíos. El root filesystem debe copiarse manualmente con rsync.
- La opción `no_root_squash` es necesaria para que el nodo pueda bootear correctamente.
- Este role no gestiona el contenido de los filesystems, solo la configuración del servidor NFS.
