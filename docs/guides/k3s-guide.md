# Guía Rápida de k3s

k3s es una distribución ligera de Kubernetes, ideal para entornos con recursos limitados como este homelab. Este documento provee una guía básica para interactuar con el clúster k3s.

## Instalación

La instalación del clúster está completamente automatizada a través de Ansible. El playbook se encarga de instalar el servidor en `rp1-master` y los agentes en `rp2-node` y `rp3-node`.

Para ejecutar la instalación:
```bash
# Desde el directorio homelab-ansible/
ansible-playbook playbooks/k3s.yml
```

## Acceso al Clúster

Para interactuar con el clúster desde tu máquina local, necesitas el archivo `kubeconfig`.

### 1. Copiar el archivo de configuración

Copia el archivo `k3s.yaml` desde el nodo master (`rp1-master`) a tu máquina. Es recomendable guardarlo con un nombre específico para no sobreescribir tu configuración existente.

```bash
# Asegúrate de estar conectado a la VPN (Tailscale)
scp admin@10.0.0.1:/etc/rancher/k3s/k3s.yaml ~/.kube/config-homelab
```

### 2. Modificar el kubeconfig para acceso remoto

El archivo copiado tendrá la IP local del servidor (`10.0.0.1`). Para acceder desde cualquier lugar a través de Tailscale, debes reemplazarla por la IP de Tailscale de `rp1-master`.

1.  Abre el archivo `~/.kube/config-homelab`.
2.  Busca la línea `server: https://127.0.0.1:6443` o `server: https://10.0.0.1:6443`.
3.  Reemplázala con la IP de Tailscale del master: `server: https://100.94.94.49:6443`.

### 3. Usar el nuevo kubeconfig

Puedes usar el archivo de configuración de dos maneras:

**Opción A: De forma temporal (exportando una variable de entorno)**

```bash
export KUBECONFIG=~/.kube/config-homelab
kubectl get nodes
```

**Opción B: Fusionándolo con tu config actual**

Puedes gestionar múltiples clústers fusionando los archivos.

```bash
# Fusiona tu config actual con la del homelab
export KUBECONFIG=~/.kube/config:~/.kube/config-homelab

# Puedes cambiar de contexto para apuntar a un clúster u otro
kubectl config view
kubectl config use-context default # 'default' es el nombre del contexto de k3s
```

## Comandos Básicos

Una vez configurado el acceso, puedes usar `kubectl` para inspeccionar el clúster.

```bash
# Ver el estado de los nodos
kubectl get nodes -o wide

# Ver todos los pods en todos los namespaces
kubectl get pods -A

# Ver el uso de recursos de los nodos (requiere metrics-server)
kubectl top nodes
```

## Arquitectura y Decisiones

### Storage

El backend de k3s, `containerd`, no funciona correctamente sobre NFS con el driver `overlay2`. Por esta razón, todos los datos de k3s (`/var/lib/rancher`) se almacenan en discos locales en cada nodo.

Para más detalles, consulta el ADR correspondiente:
*   [ADR-010: K3s Storage on Local Disks](../decisions/010-k3s-storage-on-nfs.md)

### Redes

Durante la instalación, el balanceador de carga por defecto de k3s (`ServiceLB`) está deshabilitado. Para exponer servicios al exterior del clúster (por ejemplo, a la red `10.0.0.0/24`), necesitarás instalar un Ingress Controller como [Traefik](https://traefik.io/traefik/) o [Nginx Ingress Controller](https://kubernetes.github.io/ingress-nginx/).

Traefik ya viene integrado con k3s, pero se puede personalizar o reemplazar.
