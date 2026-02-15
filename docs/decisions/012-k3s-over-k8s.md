# ADR 012: K3s sobre Kubernetes Vanilla

## Estado

Aceptado

## Fecha

2026-01-23

## Contexto

Necesitamos un orquestador de contenedores para el cluster de 3 Raspberry Pi 5 con 8GB de RAM. Los nodos worker (rp2, rp3) bootean por NFS desde el gateway (rp1-master), lo que impone restricciones de storage (overlayfs no funciona sobre NFS).

Requisitos:
- Bajo consumo de recursos (ARM, 8GB RAM por nodo)
- Compatibilidad con arquitectura ARM64
- Instalación automatizable con Ansible
- API compatible con Kubernetes estándar (útil para aprendizaje CKA)
- Funcionar con nodos que bootean por NFS

## Decisión

Usar **k3s** como distribución de Kubernetes.

### Razones principales

- **Menor consumo de recursos**: ~512MB RAM vs ~2GB de k8s vanilla
- **Binario único**: ~50MB, sin dependencias externas
- **Componentes incluidos**: Traefik (Ingress), CoreDNS (DNS), Flannel (CNI), local-path-provisioner (storage)
- **Instalación simple**: Un solo script/comando por nodo

### Lo que k3s cambia respecto a k8s vanilla

| Componente | k8s vanilla | k3s |
|-----------|-------------|-----|
| Base de datos | etcd (cluster) | SQLite (embedded) |
| Container runtime | Docker o containerd | Solo containerd |
| Cloud providers | Incluidos | Removidos |
| Storage drivers | Todos | Solo los esenciales |
| Binarios | Múltiples (~1GB) | Uno solo (~50MB) |

### Lo que se mantiene igual

- **API 100% compatible**: Misma API de Kubernetes, certificación CNCF
- **kubectl**: Idéntico, mismos comandos y flags
- **Manifiestos YAML**: Exactamente iguales, portables
- **Helm**: Funciona sin cambios
- **CRDs y Operators**: Compatibles
- **Networking**: Mismos conceptos (Services, Ingress, NetworkPolicy)

### Limitaciones

- HA menos probado que etcd (SQLite no es distribuido)
- Escala hasta ~500 nodos (suficiente para homelab)
- Menos opciones enterprise out-of-the-box

## Consecuencias

### Positivas

- **Recursos**: El cluster funciona con solo 3 RPi de 8GB sin problemas de memoria
- **Operaciones**: Instalación y actualización son un solo comando por nodo
- **Automatización**: Playbook `k3s.yml` instala todo el cluster
- **Aprendizaje**: API idéntica a k8s, conocimientos transferibles a CKA

### Negativas

- **HA limitado**: SQLite no soporta múltiples masters de forma nativa (se puede cambiar a etcd si se necesita)
- **Menos granularidad**: Algunos componentes vienen bundled y no se pueden reemplazar fácilmente

## Alternativas consideradas

### 1. k8s vanilla (kubeadm)

**Rechazado porque:**
- Consumo de recursos (~2GB RAM solo para el control plane)
- Instalación más compleja (múltiples binarios, certificados manuales)
- Requiere etcd separado
- Overkill para 3 nodos ARM

### 2. MicroK8s (Canonical)

**Rechazado porque:**
- Basado en snap, que no funciona sobre NFS (`setcap: Operation not supported`)
- Problemas documentados con netboot en este homelab

### 3. kind / minikube

**Rechazado porque:**
- Diseñados para desarrollo local, no para clusters de producción/homelab
- kind requiere Docker como host
- minikube es single-node

## Implementación

```bash
# Instalar cluster completo
ansible-playbook playbooks/k3s.yml
```

### Configuración crítica

```yaml
# /etc/rancher/k3s/config.yaml (solo master)
flannel-iface: eth0
```

**¿Por qué?** rp1-master tiene múltiples interfaces de red:
- `eth0`: 10.0.0.1 (red interna del cluster)
- `enx00e04c683da2`: 192.168.1.89.x (USB ethernet a internet)

Sin `flannel-iface: eth0`, Flannel puede elegir la interfaz incorrecta y los pods entre nodos no se comunican.

## Referencias

- [K3s Documentation](https://docs.k3s.io/)
- [K3s Architecture](https://docs.k3s.io/architecture)
- [CNCF Certified Kubernetes](https://www.cncf.io/training/certification/software-conformance/)
- [ADR-010: Storage de k3s en discos locales](010-k3s-storage-on-nfs.md)
- [ADR-011: MetalLB para LoadBalancer](011-metallb.md)
