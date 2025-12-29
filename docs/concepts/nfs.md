# NFS (Network File System)

## ¿Qué es?

NFS es un protocolo que permite montar sistemas de archivos remotos como si fueran locales. Un servidor exporta directorios y los clientes los montan.

En el homelab, usamos NFS para que los nodos (rp2, rp3) monten su root filesystem desde el gateway.

## Arquitectura

```
┌─────────────────────────────────────────────────────────────┐
│                    SERVIDOR NFS (rp1-master)                 │
│                                                              │
│   /srv/nfs/                                                  │
│   ├── rp2/    ◄─── exportado a 10.0.0.2                     │
│   └── rp3/    ◄─── exportado a 10.0.0.3                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
                          │
                          │ NFS (puerto 2049)
                          │
            ┌─────────────┴─────────────┐
            ▼                           ▼
    ┌──────────────┐           ┌──────────────┐
    │   rp2        │           │   rp3        │
    │   monta      │           │   monta      │
    │   / (root)   │           │   / (root)   │
    └──────────────┘           └──────────────┘
```

## Componentes

### Servidor

- **nfs-kernel-server**: Daemon que sirve los exports
- **rpcbind**: Servicio de portmapper para NFS
- **/etc/exports**: Configuración de qué directorios exportar

### Cliente

- **nfs-common**: Utilidades para montar NFS
- **/etc/fstab**: Configuración de qué montar

## Configuración del Servidor

### Instalar

```bash
sudo apt install nfs-kernel-server
```

### Configurar exports

El archivo `/etc/exports` define qué se exporta:

```bash
# Sintaxis:
# /ruta/local    cliente(opciones)

# Ejemplos:
/srv/nfs/rp2    10.0.0.2(rw,sync,no_subtree_check,no_root_squash)
/srv/nfs/rp3    10.0.0.3(rw,sync,no_subtree_check,no_root_squash)

# Exportar a toda una red:
/srv/nfs/shared    10.0.0.0/24(rw,sync,no_subtree_check)
```

### Opciones de export

| Opción | Significado |
|--------|-------------|
| `rw` | Lectura y escritura |
| `ro` | Solo lectura |
| `sync` | Escrituras síncronas (más seguro) |
| `async` | Escrituras asíncronas (más rápido) |
| `no_subtree_check` | No verificar que el archivo esté en el subárbol (mejor performance) |
| `no_root_squash` | Root del cliente tiene permisos de root en servidor |
| `root_squash` | Root del cliente se mapea a nobody (más seguro) |
| `all_squash` | Todos los usuarios se mapean a nobody |

### Aplicar cambios

```bash
# Recargar exports
sudo exportfs -ra

# Ver exports activos
sudo exportfs -v
```

## Configuración del Cliente

### Instalar

```bash
sudo apt install nfs-common
```

### Montar manualmente

```bash
sudo mount -t nfs 10.0.0.1:/srv/nfs/rp2 /mnt
```

### Montar automáticamente (fstab)

```bash
# /etc/fstab
10.0.0.1:/srv/nfs/rp2    /    nfs    defaults,vers=3    0    0
```

### Opciones de montaje

| Opción | Significado |
|--------|-------------|
| `vers=3` o `vers=4` | Versión de NFS |
| `hard` | Reintentar indefinidamente si falla |
| `soft` | Devolver error si falla |
| `intr` | Permitir interrumpir operaciones |
| `timeo=N` | Timeout en décimas de segundo |
| `retrans=N` | Número de reintentos |

## NFS para Netboot

Para netboot, el kernel del cliente monta NFS automáticamente. Se configura en `cmdline.txt`:

```
root=/dev/nfs nfsroot=10.0.0.1:/srv/nfs/rp2,vers=3,tcp rw ip=dhcp rootwait
```

Parámetros:
- `root=/dev/nfs`: Indica que root es NFS
- `nfsroot=IP:PATH,opciones`: Servidor y ruta
- `ip=dhcp`: Obtener IP por DHCP
- `rootwait`: Esperar a que NFS esté disponible

## UIDs y Permisos

NFS usa UIDs/GIDs numéricos, no nombres. Si el UID 1000 en el servidor es "admin" y en el cliente es "juan", ambos verán los archivos del UID 1000.

### Problema

```
Servidor: admin (UID 1000) crea archivo
Cliente:  juan (UID 1000) puede leer/escribir
          admin (UID 1001) NO puede
```

### Solución en el homelab

Todos los nodos usan el mismo usuario `admin` con UID 1000. El playbook `prepare-node.yml` se asegura de esto.

## Versiones de NFS

| Versión | Características |
|---------|-----------------|
| NFSv3 | Ampliamente compatible, sin estado |
| NFSv4 | Con estado, mejor seguridad, ACLs |
| NFSv4.1 | pNFS, mejor performance |

En el homelab usamos NFSv3 por compatibilidad con el boot del kernel.

## Comandos Útiles

### En el servidor

```bash
# Ver exports
sudo exportfs -v

# Ver clientes conectados
sudo showmount -a

# Ver estadísticas
nfsstat -s

# Ver qué procesos usan NFS
sudo lsof -N
```

### En el cliente

```bash
# Ver montajes NFS
mount | grep nfs

# Estadísticas de montaje
nfsstat -c

# Verificar conectividad
showmount -e 10.0.0.1
```

## Troubleshooting

### "mount.nfs: access denied"

1. Verificar que el export existe:
   ```bash
   sudo exportfs -v
   ```

2. Verificar que la IP del cliente está permitida:
   ```bash
   grep "rp2" /etc/exports
   ```

3. Recargar exports:
   ```bash
   sudo exportfs -ra
   ```

### "mount.nfs: Connection timed out"

1. Verificar conectividad:
   ```bash
   ping 10.0.0.1
   ```

2. Verificar que NFS está corriendo:
   ```bash
   sudo systemctl status nfs-kernel-server
   ```

3. Verificar puertos:
   ```bash
   sudo ss -tlnp | grep -E "111|2049"
   ```

### "Stale file handle"

El export cambió mientras estaba montado:
```bash
# En el cliente
sudo umount -f /mnt
sudo mount -t nfs 10.0.0.1:/path /mnt
```

### Performance lenta

1. Verificar versión:
   ```bash
   mount | grep nfs
   # Probar con vers=3 o vers=4.1
   ```

2. Aumentar tamaño de buffer:
   ```bash
   mount -o rsize=32768,wsize=32768 ...
   ```

## Seguridad

### no_root_squash

En el homelab usamos `no_root_squash` porque los nodos necesitan acceso root a su filesystem. Esto es inseguro en ambientes de producción.

### Firewalling

Si hay firewall, permitir:
- Puerto 111 (rpcbind)
- Puerto 2049 (nfs)

```bash
sudo ufw allow from 10.0.0.0/24 to any port 111
sudo ufw allow from 10.0.0.0/24 to any port 2049
```

## En el Homelab (Ansible)

El rol `nfs` configura:
1. Instala nfs-kernel-server
2. Crea directorios en /srv/nfs/
3. Genera /etc/exports
4. Recarga exports

Variables:
```yaml
nfs_base_path: /srv/nfs
nfs_network: "10.0.0.0/24"
nfs_export_options: "rw,sync,no_subtree_check,no_root_squash"
nfs_nodes:
  - name: rp2
    ip: "10.0.0.2"
```

## Referencias

- [Linux NFS-HOWTO](https://tldp.org/HOWTO/NFS-HOWTO/)
- [Ubuntu NFS Documentation](https://ubuntu.com/server/docs/service-nfs)
