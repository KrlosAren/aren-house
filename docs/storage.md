# Storage en Kubernetes

## Estado actual

El cluster usa **local-path provisioner** (incluido por defecto en k3s) como StorageClass principal. Los PVCs se almacenan en el disco local de cada nodo.

n8n es la excepción: usa **Longhorn** como StorageClass (`storageClassName: longhorn`).

## Discos por nodo

| Nodo | Disco | Tamaño | Montaje | Uso |
|------|-------|--------|---------|-----|
| rp1-master | SSD USB | 500GB | `/backup` | k3s data, PostgreSQL, Longhorn |
| rp2-node | SSD USB | 500GB | `/mnt/ssd` | Docker, k3s data, Longhorn |
| rp3-node | SSD USB | 500GB | `/mnt/ssd` | Docker, k3s data, Longhorn |

## Montaje de discos locales

El playbook `local-storage.yml` configura los discos en los workers:

```bash
cd homelab-ansible
ansible-playbook playbooks/local-storage.yml
```

### Qué hace

1. Detecta disco por label (`LABEL=ssd` via `blkid`)
2. Monta el disco SSD en `/mnt/ssd` via `/etc/fstab`
3. Mueve `/var/lib/docker` → `/mnt/ssd/docker` (symlink)
4. Mueve `/var/lib/rancher` → `/mnt/ssd/rancher` (symlink)
5. Configura Docker con `storage-driver: overlay2`

### Preparar discos (si no tienen label)

```bash
# En cualquier nodo worker
sudo e2label /dev/sda1 ssd
```

**Tags disponibles**: `detect`, `mount`, `docker`, `rancher`, `verify`

## local-path provisioner

StorageClass por defecto en k3s. Almacena los PVCs en `/var/lib/rancher/k3s/storage/` (o en el symlink a disco local).

```yaml
# Ejemplo de PVC con local-path
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mi-app-data
spec:
  storageClassName: local-path
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```

**Limitaciones**: No soporta replicación entre nodos. Si un nodo muere, los datos se pierden.

Ambos nodos worker tienen SSDs de 500GB, por lo que `nodeSelector` por storage ya no es necesario. Se puede usar el label `storage=ssd` si se requiere:
```yaml
nodeSelector:
  storage: ssd
```

## Longhorn

### Estado

Longhorn está **instalado** en el cluster pero no es el StorageClass por defecto. Se usa selectivamente (e.g., n8n).

### Prerequisitos (Ansible)

```bash
# Instalar dependencias iSCSI en todos los nodos
ansible-playbook playbooks/longhorn.yml

# Crear directorios de storage para Longhorn
ansible-playbook playbooks/longhorn-storage.yml
```

El playbook `longhorn.yml` instala: `open-iscsi`, `nfs-common`, `util-linux`, `cryptsetup`, y configura iSCSI InitiatorName.

El playbook `longhorn-storage.yml` crea:
- `/backup/longhorn` en rp1-master
- `/mnt/ssd/longhorn` en nodos worker (rp2, rp3)

### Instalación (Helm)

Longhorn tiene **dos pasos independientes**:

**Paso 1: Dependencias del SO (Ansible)**
```bash
ansible-playbook playbooks/longhorn.yml        # open-iscsi, nfs-common, iscsid
ansible-playbook playbooks/longhorn-storage.yml # crea /var/lib/longhorn en cada nodo
```

**Paso 2: Componentes en Kubernetes (Helm)**
```bash
helm repo add longhorn https://charts.longhorn.io
helm repo update
helm install longhorn longhorn/longhorn -n longhorn-system --create-namespace
```

Verificar que todo esté corriendo:
```bash
kubectl get pods -n longhorn-system
kubectl get storageclass   # debe aparecer "longhorn"
```

**Importante**: Sin el Paso 2, el StorageClass `longhorn` no existe y todos los PVCs que lo referencien quedan en `Pending`. Los playbooks de Ansible solo preparan el SO, no instalan Longhorn en el cluster.

### Ingress

```
k8s-apps/longhorn/
└── 01-ingres.yml    # Ingress: longhorn.k8s.homelab.local → longhorn-frontend:80
```

Acceso: `http://longhorn.k8s.homelab.local`

### Usar Longhorn en un PVC

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mi-app-data
spec:
  storageClassName: longhorn
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```

## Manifiestos relacionados

```
k8s-apps/longhorn/           # Ingress para UI de Longhorn
k8s-apps/storage-longhorn/   # Manifiesto de Longhorn (legacy, no usado)
k8s-apps/storage-learning/   # App de prueba nginx con PVC (aprendizaje)
```

## Migración local-path → Longhorn

Para migrar un PVC de local-path a Longhorn:

1. Hacer backup de los datos del PVC
2. Cambiar `storageClassName: longhorn` en el manifiesto del PVC
3. Borrar el PVC viejo y recrearlo
4. Restaurar los datos

**Nota**: No es posible cambiar el StorageClass de un PVC existente. Hay que recrearlo.
