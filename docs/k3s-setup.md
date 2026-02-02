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
- `enx00e04c683da2`: 192.168.100.x (USB ethernet a internet)

Sin esta configuración, Flannel puede elegir la interfaz incorrecta y los pods en diferentes nodos no pueden comunicarse.

## Networking

### Rangos de red

| Red | Rango | Uso |
|-----|-------|-----|
| Nodos | 10.0.0.0/24 | Red física entre RPis |
| Pods | 10.42.0.0/16 | Red interna de pods |
| Services | 10.43.0.0/16 | ClusterIPs |
| MetalLB | 10.0.0.50-60 | LoadBalancer IPs |

### Flannel (CNI)

Flannel crea una red overlay usando VXLAN para comunicar pods entre nodos:
```
Pod en rp3 (10.42.3.x) → flannel.1 → eth0 → eth0 → flannel.1 → Pod en rp1 (10.42.0.x)
                         VXLAN        UDP 8472      VXLAN
```

Cada nodo mantiene una tabla FDB (Forwarding Database):
```bash
# Ver FDB de un nodo
bridge fdb show dev flannel.1
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
│  → 10.0.0.50            │
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
