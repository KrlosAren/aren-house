# Guía de k3s

k3s es una distribución ligera de Kubernetes, ideal para entornos con recursos limitados como este homelab. Este documento cubre instalación, configuración, operación y troubleshooting del cluster.

## Arquitectura del Cluster

```
┌─────────────────────────────────────────────────────────────┐
│                    CONTROL PLANE (rp1-master)                │
│                                                              │
│  k3s server                                                  │
│  - API Server, Scheduler, Controller Manager                 │
│  - SQLite (estado del cluster)                              │
│  - Flannel (CNI) - flannel-iface: eth0                      │
│  - CoreDNS, Traefik (Ingress Controller)                    │
│                                                              │
│  Config: /etc/rancher/k3s/config.yaml                       │
│  Storage: /var/lib/rancher → /var/lib/rancher-local (SSD)   │
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
│   - MetalLB speaker   │     │   - MetalLB speaker   │
│                       │     │                       │
│   Storage: microSD    │     │   Storage: SSD 240GB  │
│   32GB (stateless)    │     │   (workloads con I/O) │
└───────────────────────┘     └───────────────────────┘
```

### Redes del cluster

| Red | Rango | Uso |
|-----|-------|-----|
| Nodos | 10.0.0.0/24 | Red física entre RPis |
| Pods | 10.42.0.0/16 | Red interna de pods |
| Services | 10.43.0.0/16 | ClusterIPs |
| MetalLB | 10.0.0.50-60 | LoadBalancer IPs |

### Node Labels

```bash
# Storage labels (aplicados manualmente)
kubectl label nodes rp1-master storage=ssd storage-size=500gb
kubectl label nodes rp2-node storage=sd storage-size=32gb
kubectl label nodes rp3-node storage=ssd storage-size=240gb

# Role labels
kubectl label nodes rp2-node node-role.kubernetes.io/worker=""
kubectl label nodes rp3-node node-role.kubernetes.io/worker=""
```

## Instalación

### Requisitos previos

- Tailscale instalado y configurado (`playbooks/tailscale.yml`)
- Storage local montado en nodos (`playbooks/local-storage.yml`)
- Firewall configurado (`playbooks/firewall.yml`)

### Instalar el cluster

```bash
ansible-playbook playbooks/k3s.yml
```

El playbook ejecuta estos pasos:

1. **En rp1-master (server):**
   - Habilita cgroups de memoria en el kernel (`cgroup_memory=1 cgroup_enable=memory`)
   - Crea symlink `/var/lib/rancher` → `/var/lib/rancher-local` (evita NFS, usa SSD local)
   - Crea `/etc/rancher/k3s/config.yaml` con `flannel-iface: eth0`
   - Instala k3s server con `--disable servicelb` (MetalLB lo reemplaza)
   - Configura reglas iptables FORWARD para Tailscale
   - Persiste reglas con `iptables-persistent`

2. **En rp2-node y rp3-node (agents):**
   - Obtiene el token de conexión del server
   - Habilita cgroups en el cmdline.txt de TFTP (nodos netboot)
   - Crea symlink `/var/lib/rancher` → `/var/lib/rancher-local` (usa disco local)
   - Instala k3s agent con `K3S_URL` y `K3S_TOKEN`

3. **Verificación:**
   - Muestra estado de nodos y pods del sistema

### Instalar MetalLB

```bash
ansible-playbook playbooks/metallb.yml
```

El playbook:
1. Instala MetalLB v0.14.9 desde manifests de GitHub
2. Crea IPAddressPool `homelab-pool` con rango `10.0.0.50-10.0.0.60`
3. Crea L2Advertisement para anunciar IPs via ARP
4. Espera a que Traefik obtenga su IP de LoadBalancer
5. Agrega entrada DNS `*.k8s.homelab.local` apuntando a la IP de Traefik en dnsmasq
6. Reinicia dnsmasq

### Verificar instalación

```bash
# Estado de los nodos
kubectl get nodes -o wide

# Pods del sistema (todos deberían estar Running)
kubectl get pods -n kube-system

# Verificar Flannel IPs (deben ser 10.0.0.x, NO 192.168.100.x)
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'

# Verificar MetalLB
kubectl get pods -n metallb-system
kubectl get ipaddresspool -n metallb-system

# Verificar IP de Traefik (debe tener EXTERNAL-IP)
kubectl get svc -n kube-system traefik
```

## Configuración Crítica

### flannel-iface: eth0

**Archivo:** `/etc/rancher/k3s/config.yaml` (solo en master)

```yaml
flannel-iface: eth0
```

rp1-master tiene dos interfaces de red:
- `eth0`: 10.0.0.1 (red interna LAN)
- `enx00e04c683da2`: 192.168.100.x (USB ethernet a internet)

Sin esta configuración, Flannel puede elegir la interfaz USB y anunciar la IP WAN. Los pods entre nodos no podrían comunicarse porque los workers no tienen acceso a 192.168.100.x.

### Storage local (evitar NFS)

containerd (el runtime de k3s) no funciona con overlay2 sobre NFS. Por eso, `/var/lib/rancher` se redirige a disco local en cada nodo:

```
/var/lib/rancher  →  symlink  →  /var/lib/rancher-local (disco local)
```

Ver [ADR-010: K3s Storage on Local Disks](../decisions/010-k3s-storage-on-nfs.md).

### ServiceLB deshabilitado

k3s incluye su propio ServiceLB (antes llamado Klipper), pero lo deshabilitamos con `--disable servicelb` porque usamos MetalLB en su lugar. MetalLB asigna IPs reales de la LAN en vez de exponer NodePorts.

Ver [ADR-011: MetalLB](../decisions/011-metallb.md).

## Acceso al Cluster

### Desde rp1-master (SSH)

kubectl ya está configurado para el usuario `admin`:

```bash
ssh admin@10.0.0.1
kubectl get nodes
```

### Desde tu Mac (Tailscale)

1. Copia el kubeconfig desde el master:
```bash
scp admin@10.0.0.1:/etc/rancher/k3s/k3s.yaml ~/.kube/config-homelab
```

2. Modifica la IP del server para usar Tailscale:
```bash
# Cambia 127.0.0.1 o 10.0.0.1 por la IP de Tailscale
# En el archivo ~/.kube/config-homelab, busca:
#   server: https://127.0.0.1:6443
# Y reemplaza por:
#   server: https://100.94.94.49:6443
```

3. Usa el kubeconfig:

**Opción A: Variable de entorno temporal**
```bash
export KUBECONFIG=~/.kube/config-homelab
kubectl get nodes
```

**Opción B: Fusionar con config existente**
```bash
export KUBECONFIG=~/.kube/config:~/.kube/config-homelab
kubectl config use-context default
```

## DNS y Acceso a Servicios

### Dominios

```
*.homelab.local      → 10.0.0.1   (Traefik Docker, servicios fuera de k8s)
*.k8s.homelab.local  → 10.0.0.50  (Traefik k3s via MetalLB)
```

### Flujo de tráfico

```
Cliente → DNS (dnsmasq) → 10.0.0.50 → MetalLB (ARP) → Traefik k3s → Ingress → Service → Pod
```

### Exponer una aplicación

1. Crear Deployment y Service:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mi-app
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mi-app
  template:
    metadata:
      labels:
        app: mi-app
    spec:
      containers:
        - name: mi-app
          image: nginx:alpine
          ports:
            - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: mi-app
  namespace: default
spec:
  selector:
    app: mi-app
  ports:
    - port: 80
      targetPort: 80
```

2. Crear Ingress con hostname `*.k8s.homelab.local`:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: mi-app
  namespace: default
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

3. Verificar:
```bash
# Ver que el Ingress fue creado
kubectl get ingress

# Probar acceso (desde un equipo conectado a la LAN o Tailscale)
curl http://mi-app.k8s.homelab.local
```

## Flannel (CNI)

Flannel crea una red overlay usando VXLAN (puerto UDP 8472) para comunicar pods entre nodos:

```
Pod en rp3 (10.42.3.x) → flannel.1 (VXLAN) → eth0 → eth0 → flannel.1 → Pod en rp1 (10.42.0.x)
```

Cada nodo mantiene una tabla FDB (Forwarding Database) que mapea IPs de pods a nodos físicos:

```bash
# Ver FDB de Flannel en un nodo
ssh rp3-node "bridge fdb show dev flannel.1"
```

## Operaciones

### Reiniciar k3s

```bash
# En el master
sudo systemctl restart k3s

# En un worker
sudo systemctl restart k3s-agent
```

### Ver logs

```bash
# Logs del server
sudo journalctl -u k3s -f

# Logs de un agent
sudo journalctl -u k3s-agent -f

# Logs de un pod específico
kubectl logs -n kube-system -l app=traefik
kubectl logs -n metallb-system -l app=metallb

# Logs con follow
kubectl logs -f <pod-name> -n <namespace>
```

### Drain y cordon (mantenimiento de nodos)

```bash
# Marcar nodo como no programable (no se asignan nuevos pods)
kubectl cordon rp2-node

# Drenar nodo (mueve pods a otros nodos)
kubectl drain rp2-node --ignore-daemonsets --delete-emptydir-data

# Realizar mantenimiento (reboot, actualización, etc.)
ssh rp2-node "sudo reboot"

# Reactivar nodo
kubectl uncordon rp2-node
```

### Escalar un deployment

```bash
# Escalar a 3 replicas
kubectl scale deployment mi-app --replicas=3

# Ver distribución de pods entre nodos
kubectl get pods -o wide
```

### Port-forward para debugging

```bash
# Acceder a un servicio sin Ingress
kubectl port-forward svc/mi-app 8080:80

# Acceder al dashboard de Traefik (si está habilitado)
kubectl port-forward -n kube-system svc/traefik 9000:9000
```

## Troubleshooting

### Pods no pueden comunicarse entre nodos

**Síntoma:** DNS no funciona, pods en diferentes nodos no se ven, timeouts.

**Diagnóstico:**
```bash
# Verificar IPs que anuncia cada nodo (deben ser 10.0.0.x)
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'

# Si rp1-master muestra 192.168.100.x → Flannel eligió la interfaz incorrecta
```

**Solución:**
```bash
# Verificar/crear config en el master
cat /etc/rancher/k3s/config.yaml
# Debe contener: flannel-iface: eth0

# Reiniciar k3s en todos los nodos
sudo systemctl restart k3s                              # master
ssh rp2-node "sudo systemctl restart k3s-agent"         # worker
ssh rp3-node "sudo systemctl restart k3s-agent"         # worker
```

### Service LoadBalancer en "pending"

**Síntoma:** `kubectl get svc` muestra `EXTERNAL-IP: <pending>`

**Diagnóstico:**
```bash
# Verificar pods de MetalLB
kubectl get pods -n metallb-system

# Verificar IPAddressPool
kubectl get ipaddresspool -n metallb-system

# Si no hay namespace metallb-system → MetalLB no está instalado
```

**Solución:**
```bash
ansible-playbook playbooks/metallb.yml
```

### CoreDNS no responde

**Síntoma:** `wget: bad address 'nginx'` desde un pod.

**Diagnóstico:**
```bash
# Verificar que CoreDNS está corriendo
kubectl get pods -n kube-system -l k8s-app=kube-dns

# Probar conectividad al pod de CoreDNS
kubectl run test --rm -it --image=alpine -- nslookup kubernetes.default

# Si falla → problema de red entre nodos (ver sección anterior)
```

### Nodo worker en NotReady

**Síntoma:** `kubectl get nodes` muestra un nodo como `NotReady`.

**Diagnóstico:**
```bash
# Descripción detallada del nodo
kubectl describe node rp2-node

# Verificar que k3s-agent está corriendo en el nodo
ssh rp2-node "sudo systemctl status k3s-agent"

# Verificar logs del agent
ssh rp2-node "sudo journalctl -u k3s-agent --since '5 minutes ago'"
```

**Causas comunes:**
- El nodo se reinició y k3s-agent aún no arrancó
- Disco local lleno (containerd no puede funcionar)
- Problema de red entre worker y master

### Pod en CrashLoopBackOff

```bash
# Ver eventos del pod
kubectl describe pod <pod-name> -n <namespace>

# Ver logs del contenedor (incluyendo logs del crash anterior)
kubectl logs <pod-name> -n <namespace> --previous

# Ver eventos recientes
kubectl get events -n <namespace> --sort-by='.lastTimestamp'
```

### ImagePullBackOff

```bash
# Verificar que el nodo tiene acceso a internet
ssh rp2-node "curl -s https://registry-1.docker.io/v2/ | head -1"

# Verificar si el registry está configurado
cat /etc/rancher/k3s/registries.yaml  # si existe
```

## Comandos Útiles

```bash
# Estado general
kubectl get nodes -o wide
kubectl get pods -A
kubectl get svc -A

# Recursos del sistema
kubectl top nodes                    # requiere metrics-server
kubectl top pods -A

# Investigación
kubectl describe node <node>
kubectl describe pod <pod> -n <ns>
kubectl get events -A --sort-by='.lastTimestamp'

# Logs
kubectl logs -f <pod> -n <ns>
sudo journalctl -u k3s -f           # en master
sudo journalctl -u k3s-agent -f     # en worker

# Networking
kubectl get svc -A | grep LoadBalancer
kubectl get ingress -A

# Cleanup
kubectl delete pod <pod> -n <ns>     # reiniciar pod
kubectl rollout restart deployment/<name> -n <ns>
```

## Referencias

- [k3s Documentation](https://docs.k3s.io/)
- [Flannel](https://github.com/flannel-io/flannel)
- [MetalLB](https://metallb.universe.tf/)
- [Traefik Ingress](https://doc.traefik.io/traefik/providers/kubernetes-ingress/)
- [ADR-010: k3s Storage](../decisions/010-k3s-storage-on-nfs.md)
- [ADR-011: MetalLB](../decisions/011-metallb.md)
- [k3s-setup.md](../k3s-setup.md) - Guía de instalación detallada
