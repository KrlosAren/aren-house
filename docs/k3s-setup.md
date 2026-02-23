# K3s Setup - Guía de Instalación y Troubleshooting

## Arquitectura del Cluster
```
┌─────────────────────────────────────────────────────────────┐
│                    CONTROL PLANE (rp1-master)                │
│                                                              │
│  k3s server                                                  │
│  - API Server                                                │
│  - Scheduler                                                 │
│  - Controller Manager                                        │
│  - SQLite (estado del cluster)                              │
│  - Flannel (CNI)                                            │
│  - CoreDNS                                                   │
│  - Traefik (Ingress Controller)                             │
└──────────────────────────┬───────────────────────────────────┘
                           │
            ┌──────────────┴──────────────┐
            ▼                             ▼
┌───────────────────────┐     ┌───────────────────────┐
│   rp2-node (worker)   │     │   rp3-node (worker)   │
│                       │     │                       │
│   k3s agent           │     │   k3s agent           │
│   - kubelet           │     │   - kubelet           │
│   - kube-proxy        │     │   - kube-proxy        │
│   - containerd        │     │   - containerd        │
│                       │     │                       │
│   Storage: microSD    │     │   Storage: SSD 240GB  │
│   32GB (solo compute) │     │   (workloads con I/O) │
└───────────────────────┘     └───────────────────────┘
```

## Instalación

### Requisitos previos

- Tailscale instalado y configurado
- Storage local montado (SSD/microSD)
- Firewall configurado

### Playbook
```bash
ansible-playbook playbooks/k3s.yml
```

El playbook:
1. Configura cgroups para memoria
2. Crea symlink de `/var/lib/rancher` a storage local (evita NFS)
3. Configura `flannel-iface: eth0` (crítico si hay múltiples interfaces)
4. Instala k3s server en gateway
5. Instala k3s agents en workers

## Configuración importante

### /etc/rancher/k3s/config.yaml (solo master)
```yaml
flannel-iface: eth0
```

**¿Por qué?** El gateway (rp1) tiene múltiples interfaces:
- `eth0`: 10.0.0.1 (red interna del cluster)
- `enx00e04c683da2`: 192.168.1.89.x (USB ethernet a internet)

Sin esta configuración, Flannel puede elegir la interfaz incorrecta y los pods en diferentes nodos no pueden comunicarse.

## Networking

### ¿Automático o manual?

Al instalar k3s, la mayoría de las redes se configuran **automáticamente** con valores por defecto. En nuestro playbook `k3s.yml` la instalación es:

```bash
curl -sfL https://get.k3s.io | sh -s - server \
  --write-kubeconfig-mode 644 \
  --disable servicelb
```

No se pasan flags de red (`--cluster-cidr`, `--service-cidr`), por lo que k3s usa sus defaults.

### Rangos de red

| Red | Rango | Quién lo define | Configurable con |
|-----|-------|-----------------|------------------|
| Nodos (LAN) | 10.0.0.0/24 | Nosotros (dnsmasq DHCP) | dnsmasq config |
| Pods | 10.42.0.0/16 | k3s default (Flannel) | `--cluster-cidr` |
| Services | 10.43.0.0/16 | k3s default | `--service-cidr` |
| MetalLB | 10.0.0.50-60 | Nosotros | `k8s-apps/metallb/metallb-config.yml` |
| DHCP | 10.0.0.100-200 | Nosotros (dnsmasq) | dnsmasq config |

### Red de Pods (10.42.0.0/16) - automática

Flannel gestiona esta red. Cada nodo recibe un `/24` del rango `10.42.0.0/16`:

```
rp1-master  →  10.42.0.0/24  (pods del master)
rp2-node    →  10.42.1.0/24  (pods del worker 2)
rp3-node    →  10.42.2.0/24  (pods del worker 3)
```

Cada pod recibe una IP de este rango. La asignación es automática y no requiere configuración. Flannel se encarga del tunneling VXLAN (puerto 8472/UDP) para que pods de distintos nodos se comuniquen.

```
Pod en rp3 (10.42.2.x) → flannel.1 → eth0 → eth0 → flannel.1 → Pod en rp1 (10.42.0.x)
                          VXLAN       UDP 8472       VXLAN
```

### Red de Services (10.43.0.0/16) - automática

Son IPs virtuales, no hay tráfico real en la red física. Cuando creas un Service tipo ClusterIP, k3s le asigna una IP de este rango automáticamente. `kube-proxy` maneja el ruteo internamente con reglas iptables dentro de cada nodo.

```yaml
# Ejemplo: al crear un Service, k3s asigna una ClusterIP automáticamente
apiVersion: v1
kind: Service
metadata:
  name: mi-app
spec:
  type: ClusterIP       # ← k3s asigna ej: 10.43.127.45 automáticamente
  selector:
    app: mi-app
  ports:
    - port: 80
```

Estas IPs solo son alcanzables desde dentro del cluster (otros pods, nodos). Para exponer un servicio fuera del cluster se usa LoadBalancer (MetalLB) o Ingress.

### MetalLB (10.0.0.50-60) - configurado por nosotros

A diferencia de las redes anteriores, el pool de MetalLB lo elegimos nosotros. Son IPs reales de nuestra LAN que MetalLB anuncia via ARP (L2 mode), lo que permite que cualquier máquina en la red `10.0.0.0/24` las alcance.

```yaml
# k8s-apps/metallb/metallb-config.yml
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: homelab-pool
  namespace: metallb-system
spec:
  addresses:
    - 10.0.0.50-10.0.0.60    # ← Nosotros elegimos este rango
```

Actualmente solo Traefik usa una IP de este pool (10.0.0.50). Las IPs 10.0.0.51-60 están disponibles para futuros servicios LoadBalancer.

### Lo que configuramos nosotros vs lo automático

| Configuración | Tipo | Dónde |
|---------------|------|-------|
| `flannel-iface: eth0` | Manual | `/etc/rancher/k3s/config.yaml` |
| `--disable servicelb` | Manual | Playbook `k3s.yml` (para usar MetalLB) |
| Pool MetalLB `10.0.0.50-60` | Manual | `k8s-apps/metallb/metallb-config.yml` |
| DNS `*.k8s.homelab.local → 192.168.1.89` | Manual | dnsmasq config (DNAT → 10.0.0.50) |
| Pod CIDR `10.42.0.0/16` | Automático | Default de k3s |
| Service CIDR `10.43.0.0/16` | Automático | Default de k3s |
| Flannel VXLAN | Automático | Incluido en k3s |
| CoreDNS | Automático | Incluido en k3s |
| Traefik Ingress Controller | Automático | Incluido en k3s |

### Flannel (CNI)

Flannel es el CNI (Container Network Interface) incluido en k3s. Crea la red overlay entre nodos.

Cada nodo mantiene una tabla FDB (Forwarding Database):
```bash
# Ver FDB de un nodo
bridge fdb show dev flannel.1
```

### Verificar la red

```bash
# Ver qué IP de Flannel anuncia cada nodo (debe ser 10.0.0.x)
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'

# Ver el CIDR de pods asignado a cada nodo
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.podCIDR}{"\n"}{end}'

# Ver los Services y sus ClusterIPs
kubectl get svc -A

# Ver el LoadBalancer IP de Traefik
kubectl get svc -n kube-system traefik
```

## MetalLB

Provee LoadBalancer para bare-metal. Asigna IPs reales a Services tipo LoadBalancer.

### Instalación
```bash
ansible-playbook playbooks/metallb.yml
```

### Verificar
```bash
# Ver IP asignada a Traefik
kubectl get svc -n kube-system traefik

# Debería mostrar EXTERNAL-IP: 10.0.0.50
```

## Acceso a servicios

### Flujo de tráfico
```
Cliente (Mac/LAN)
       │
       │ nginx.k8s.homelab.local
       ▼
┌─────────────────────────┐
│  dnsmasq                │
│  *.k8s.homelab.local    │
│  → 192.168.1.89         │
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│  UFW (DNAT)             │
│  :80/:443 → 10.0.0.50  │
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│  MetalLB                │
│  Anuncia 10.0.0.50      │
│  via ARP                │
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│  Traefik (k3s)          │
│  LoadBalancer           │
│  Rutea por Host header  │
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│  Service → Pod          │
└─────────────────────────┘
```

### Exponer una aplicación

1. Crear Deployment y Service
2. Crear Ingress con hostname `*.k8s.homelab.local`
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: mi-app
  namespace: mi-namespace
spec:
  rules:
    - host: mi-app.k8s.homelab.local
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: mi-app
                port:
                  number: 80
```

## Troubleshooting

### Pods no pueden comunicarse entre nodos

**Síntoma:** DNS no funciona, pods en diferentes nodos no se ven.

**Diagnóstico:**
```bash
# Verificar IPs que anuncia cada nodo
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'

# Verificar FDB de Flannel
ssh rp3-node "bridge fdb show dev flannel.1"
```

**Causa común:** Flannel eligió la interfaz de red incorrecta.

**Solución:**
```bash
# En el master, crear/editar /etc/rancher/k3s/config.yaml
flannel-iface: eth0

# Reiniciar k3s
sudo systemctl restart k3s

# Reiniciar agents
ssh rp2-node "sudo systemctl restart k3s-agent"
ssh rp3-node "sudo systemctl restart k3s-agent"
```

### Service LoadBalancer en "pending"

**Síntoma:** `kubectl get svc` muestra `EXTERNAL-IP: <pending>`

**Causa:** MetalLB no está instalado o configurado.

**Solución:**
```bash
# Verificar pods de MetalLB
kubectl get pods -n metallb-system

# Verificar IPAddressPool
kubectl get ipaddresspool -n metallb-system
```

### Mover data de k3s a otro disco (certificados corruptos)

**Síntoma:** Después de mover `/var/lib/rancher` a otro disco (ej: de microSD a SSD), k3s server crashea con:
```
level=fatal msg="failed to start controllers: ... tls: failed to verify certificate: x509: certificate signed by unknown authority"
```

Los agents también fallan con `token CA hash does not match the Cluster CA certificate hash`.

**Causa:** Al mover los archivos de k3s a otro disco, los certificados TLS pueden quedar inconsistentes. k3s tiene una cadena de confianza interna:

```
CA raíz (server/tls/server-ca.crt)
  ├── firma → certificados del API server
  ├── firma → certificados de clientes (kubelet, controller, etc.)
  └── genera → token con hash de la CA embebido (K10<hash>::server:<secret>)
```

Si los archivos TLS se copian parcialmente, se regeneran a medias al reiniciar, o quedan symlinks rotos, la cadena se rompe: los componentes internos del server no confían en su propia CA, y los agents no reconocen el token.

**Cómo mover la data correctamente:**
```bash
# 1. Parar k3s
sudo systemctl stop k3s

# 2. Copiar TODO de una vez, preservando permisos y symlinks
sudo cp -a /var/lib/rancher/* /nuevo/disco/k3s-data/

# 3. Reemplazar con symlink
sudo rm -rf /var/lib/rancher
sudo ln -s /nuevo/disco/k3s-data /var/lib/rancher

# 4. Reiniciar
sudo systemctl start k3s
```

**Si ya se corrompió — regenerar certificados:**
```bash
# En el master:
sudo systemctl stop k3s
sudo rm -rf /backup/k3s-data/server/tls
sudo rm -f /backup/k3s-data/agent/client-ca.crt /backup/k3s-data/agent/server-ca.crt
sudo systemctl start k3s

# Obtener el nuevo token (¡cambia al regenerar!)
sudo cat /backup/k3s-data/server/token
# Ejemplo: K103c96d3a9...::server:c490bd4601c0666e949887eab670b20c
#                                   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
#                                   Este es el secret para los agents
```

```bash
# En cada agent: actualizar token y limpiar certs cacheados
sudo systemctl stop k3s-agent

# Actualizar el token en el env file
echo 'K3S_TOKEN=<nuevo-secret>
K3S_URL=https://10.0.0.1:6443' | sudo tee /etc/systemd/system/k3s-agent.service.env

# Limpiar certificados cacheados del agent
sudo rm -rf /var/lib/rancher/k3s/agent/tls
sudo rm -f /var/lib/rancher/k3s/agent/client-ca.crt
sudo rm -f /var/lib/rancher/k3s/agent/server-ca.crt
sudo rm -f /var/lib/rancher/k3s/agent/kubelet.kubeconfig
sudo rm -f /var/lib/rancher/k3s/agent/kube-proxy.kubeconfig

sudo systemctl start k3s-agent
```

**Nota para nodos netboot:** El storage de k3s en los agents DEBE estar en disco local (microSD/SSD), no en NFS. overlayfs no funciona sobre NFS. Verificar que el symlink apunte al disco correcto:
```bash
# Correcto (disco local):
/var/lib/rancher → /mnt/docker/k3s-data    # rp2 (microSD)
/var/lib/rancher → /mnt/docker/rancher      # rp3 (SSD)

# INCORRECTO (NFS):
/var/lib/rancher → /var/lib/rancher-local   # ← si esto está en NFS, falla
```

**Incidente real (2026-02):** Se movió la data del master de microSD a SSD (`/backup/k3s-data`). Los certificados quedaron inconsistentes, con 3 hashes de CA distintos. Se resolvió regenerando TLS completo en el server, actualizando tokens en agents, y corrigiendo los symlinks de storage en los workers.

### CoreDNS no responde

**Síntoma:** `wget: bad address 'nginx'` desde un pod.

**Diagnóstico:**
```bash
# Verificar que CoreDNS está corriendo
kubectl get pods -n kube-system -l k8s-app=kube-dns

# Probar conectividad al pod de CoreDNS
kubectl run test --rm -it --image=alpine -- ping -c 2 <IP-de-coredns>
```

**Causa:** Problema de red entre nodos (ver sección anterior).

## Comandos útiles
```bash
# Estado del cluster
kubectl get nodes -o wide

# Pods del sistema
kubectl get pods -n kube-system

# Ver logs de un pod
kubectl logs -n kube-system -l app=traefik

# Describir un recurso
kubectl describe node rp1-master

# Entrar a un pod
kubectl exec -it <pod> -- sh

# Port-forward para debugging
kubectl port-forward svc/nginx -n storage-lab 8080:80
```

## Referencias

- [k3s Documentation](https://docs.k3s.io/)
- [Flannel](https://github.com/flannel-io/flannel)
- [MetalLB](https://metallb.universe.tf/)
