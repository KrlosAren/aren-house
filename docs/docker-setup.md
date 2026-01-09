# Configuración de Docker en el Homelab

## Arquitectura
```
rp1-master (Gateway)
    │
    ├── /srv/nfs/rp2/var/lib/docker  ← Storage de Docker para rp2
    └── /srv/nfs/rp3/var/lib/docker  ← Storage de Docker para rp3
            │
            │ NFS
            │
    ┌───────┴───────┐
    │               │
    ▼               ▼
┌──────────┐  ┌──────────┐
│   rp2    │  │   rp3    │
│  Docker  │  │  Docker  │
│  (vfs)   │  │  (vfs)   │
└──────────┘  └──────────┘
```

---

## ¿Por qué vfs en lugar de overlay?

Docker usa **storage drivers** para gestionar las capas de imágenes y contenedores.

### Storage drivers disponibles

| Driver | Descripción | Rendimiento | Soporta NFS |
|--------|-------------|-------------|-------------|
| overlay2 | Por defecto, usa overlayfs | Rápido | ❌ No |
| vfs | Copia completa de archivos | Lento | ✅ Sí |
| btrfs | Usa filesystem btrfs | Rápido | ❌ No |
| zfs | Usa filesystem zfs | Rápido | ❌ No |

### El problema con NFS boot

Los nodos (rp2, rp3) bootean por NFS. Su filesystem completo, incluyendo `/var/lib/docker`, está en el servidor NFS (rp1-master).
```
overlay2 sobre NFS:
    │
    ▼
Error: "mount source overlay... invalid argument"
```

**overlayfs no puede montarse sobre NFS** porque NFS no soporta las operaciones de filesystem que overlay2 necesita.

### La solución: vfs

El driver `vfs` no usa overlayfs. En su lugar, copia los archivos completos para cada capa.
```
vfs sobre NFS:
    │
    ▼
Funciona ✅ (pero más lento y usa más espacio)
```

---

## Instalación

### Playbook
```bash
ansible-playbook playbooks/docker.yml
```

### Qué hace el playbook

1. Instala dependencias (ca-certificates, curl, gnupg)
2. Agrega repositorio oficial de Docker
3. Instala Docker CE, CLI, containerd, buildx, compose
4. Configura storage driver `vfs` en `/etc/docker/daemon.json`
5. Agrega usuario `admin` al grupo `docker`
6. Habilita y arranca el servicio

### Configuración del storage driver

El archivo `/etc/docker/daemon.json` contiene:
```json
{
  "storage-driver": "vfs"
}
```

---

## Uso de Docker

### Comandos básicos
```bash
# Ver versión
docker --version
docker compose version

# Ver información del sistema (incluye storage driver)
docker info

# Listar contenedores
docker ps        # En ejecución
docker ps -a     # Todos

# Listar imágenes
docker images

# Ejecutar contenedor
docker run -d --name mi-app -p 8080:80 nginx

# Ver logs
docker logs mi-app

# Detener y eliminar
docker stop mi-app
docker rm mi-app
```

### Docker Compose
```bash
# Crear archivo docker-compose.yml
cat > docker-compose.yml << 'COMPOSE'
services:
  web:
    image: nginx
    ports:
      - "8080:80"
COMPOSE

# Iniciar
docker compose up -d

# Ver estado
docker compose ps

# Ver logs
docker compose logs -f

# Detener
docker compose down
```

---

## Consideraciones de rendimiento

### vfs es más lento

| Operación | overlay2 | vfs |
|-----------|----------|-----|
| Pull imagen | Rápido (capas compartidas) | Lento (copia todo) |
| Crear contenedor | Instantáneo | Lento (copia imagen) |
| Uso de disco | Eficiente | Mayor uso |

### Cuándo importa

- **Desarrollo/testing**: vfs es suficiente
- **Producción con alta carga**: Considerar SSD local

### Alternativa: SSD local en cada nodo

Para mejor rendimiento, puedes conectar un SSD a cada nodo:
```
rp2-node
    │
    └── USB 3.0 → SSD
        └── /var/lib/docker (ext4, local)
```

Esto permite usar overlay2 en lugar de vfs.

---

## Troubleshooting

### Error: "overlay... invalid argument"

**Causa**: Docker intenta usar overlay2 sobre NFS.

**Solución**: Configurar vfs en `/etc/docker/daemon.json`:
```bash
sudo mkdir -p /etc/docker
echo '{"storage-driver": "vfs"}' | sudo tee /etc/docker/daemon.json
sudo systemctl restart docker
```

### Error: "permission denied"

**Causa**: Usuario no está en grupo docker.

**Solución**:
```bash
sudo usermod -aG docker $USER
# Cerrar sesión y volver a entrar
```

### Contenedor no inicia
```bash
# Ver logs del contenedor
docker logs <nombre-contenedor>

# Ver logs de Docker daemon
sudo journalctl -u docker -f
```

### Verificar storage driver actual
```bash
docker info | grep "Storage Driver"
# Debe mostrar: Storage Driver: vfs
```

---

## Limpieza

Docker puede acumular imágenes, contenedores y volúmenes no usados:
```bash
# Eliminar contenedores detenidos
docker container prune

# Eliminar imágenes sin usar
docker image prune

# Eliminar volúmenes sin usar
docker volume prune

# Limpiar todo lo no usado
docker system prune -a
```

**Importante en NFS**: Como vfs usa más espacio, la limpieza regular es importante para no llenar el disco del gateway.

---

## Ejemplo: Desplegar Nginx
```bash
# En rp2-node
ssh admin@10.0.0.2

# Ejecutar Nginx
docker run -d \
  --name nginx \
  --restart unless-stopped \
  -p 80:80 \
  nginx

# Verificar
curl http://localhost

# Ver desde tu Mac (via Tailscale)
curl http://10.0.0.2
```

---

## Próximos pasos

- [ ] Agregar SSD a nodos para mejor rendimiento
- [ ] Configurar Docker registry privado
- [ ] Instalar Portainer para gestión web
- [ ] Migrar a k3s para orquestación
