# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Homelab with 3 Raspberry Pi 5 configured for network boot (PXE/NFS). One Pi acts as gateway/router and the other two as worker nodes that boot from the network without microSD. Includes a k3s Kubernetes cluster with MetalLB for LoadBalancer support.

## Architecture
```
Internet → Modem (192.168.100.x)
              │
         [USB-ETH] enx00e04c683da2
              │
         rp1-master (Gateway + k3s Control Plane)
         192.168.100.x WAN / 10.0.0.1 LAN
              │
         [eth0]
              │
         Switch TP-Link SG105PE (10.0.0.5)
              │
         ├── rp2-node (10.0.0.2) - Netboot, k3s worker, microSD 32GB
         └── rp3-node (10.0.0.3) - Netboot, k3s worker, SSD 240GB

Tailscale VPN: 100.x.x.x (mesh, bypasses CGNAT)
WireGuard VPN: 10.0.1.0/24 (legacy, requires port forwarding)
```

## Kubernetes (k3s)

### Cluster Architecture
```
┌─────────────────────────────────────────────────────────────┐
│                 rp1-master (Control Plane)                   │
│                                                              │
│  k3s server: API Server, Scheduler, Controller Manager       │
│  SQLite (estado), Flannel (CNI), CoreDNS, Traefik           │
│  Storage: SSD 500GB                                          │
└──────────────────────────┬───────────────────────────────────┘
                           │
            ┌──────────────┴──────────────┐
            ▼                             ▼
┌───────────────────────┐     ┌───────────────────────┐
│   rp2-node (worker)   │     │   rp3-node (worker)   │
│   kubelet, kube-proxy │     │   kubelet, kube-proxy │
│   containerd          │     │   containerd          │
│   microSD 32GB        │     │   SSD 240GB           │
│   (solo stateless)    │     │   (workloads con I/O) │
└───────────────────────┘     └───────────────────────┘
```

### Networks

| Red | Rango | Uso |
|-----|-------|-----|
| Nodos | 10.0.0.0/24 | Red física entre RPis |
| Pods | 10.42.0.0/16 | Red interna de pods |
| Services | 10.43.0.0/16 | ClusterIPs |
| MetalLB | 10.0.0.50-60 | LoadBalancer IPs |
| DHCP | 10.0.0.100-200 | Clientes DHCP |

### Node Labels
```bash
# Storage labels
kubectl label nodes rp1-master storage=ssd storage-size=500gb
kubectl label nodes rp2-node storage=sd storage-size=32gb
kubectl label nodes rp3-node storage=ssd storage-size=240gb

# Role labels
kubectl label nodes rp1-master node-role.kubernetes.io/master=""
kubectl label nodes rp2-node node-role.kubernetes.io/worker=""
kubectl label nodes rp3-node node-role.kubernetes.io/worker=""
```

### Configuración crítica

**/etc/rancher/k3s/config.yaml (solo master)**
```yaml
flannel-iface: eth0
```

**¿Por qué?** rp1-master tiene múltiples interfaces (eth0 + USB ethernet). Sin esto, Flannel puede elegir la IP incorrecta y los pods entre nodos no se comunican.

### DNS para k8s
```
*.homelab.local      → 10.0.0.1  (Traefik Docker)
*.k8s.homelab.local  → 10.0.0.50 (Traefik k3s via MetalLB)
```

### Flujo de tráfico k8s
```
Cliente → DNS (dnsmasq) → 10.0.0.50 → MetalLB → Traefik k3s → Ingress → Service → Pod
```

### Comandos útiles k8s
```bash
# Estado del cluster
kubectl get nodes -o wide

# Ver IPs de Flannel
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'

# Pods del sistema
kubectl get pods -n kube-system

# Services con LoadBalancer
kubectl get svc -A | grep LoadBalancer

# Ver FDB de Flannel (troubleshooting)
ssh rp3-node "bridge fdb show dev flannel.1"
```

## Devices

| Device | IP | MAC | Serial (TFTP) | Role |
|--------|-----|-----|---------------|------|
| rp1-master | 10.0.0.1 | 2c:cf:67:a9:b8:51 | N/A | Gateway, k3s master |
| rp2-node | 10.0.0.2 | 2c:cf:67:88:9e:f5 | 440dc91d | k3s worker (netboot) |
| rp3-node | 10.0.0.3 | 2c:cf:67:a9:b9:13 | 02671e08 | k3s worker (netboot) |
| switch | 10.0.0.5 | ec:75:0c:ff:fc:d6 | N/A | TP-Link SG105PE |

## Services on rp1-master

- **dnsmasq**: DHCP, DNS (.homelab.local, .k8s.homelab.local), TFTP
- **NFS**: Root filesystems at `/srv/nfs/{rp2,rp3}/`
- **k3s server**: Kubernetes control plane
- **Traefik (Docker)**: Reverse proxy para servicios Docker (:80/:443)
- **Tailscale**: VPN mesh (subnet router for 10.0.0.0/24)
- **NAT**: iptables MASQUERADE
- **UFW**: Firewall

## Services on Worker Nodes

- **k3s agent**: Kubernetes worker
- **containerd**: Container runtime (via k3s)

## Ansible

### Playbooks

| Playbook | Función |
|----------|---------|
| `gateway.yml` | Configuración completa de rp1-master |
| `k3s.yml` | Instalar k3s server y agents |
| `metallb.yml` | Instalar y configurar MetalLB |
| `firewall.yml` | Configurar UFW |
| `docker.yml` | Instalar Docker |
| `local-storage.yml` | Configurar storage local |
| `prepare-node.yml` | Preparar nodo para netboot |
| `update-nodes.yml` | Actualizar paquetes |
| `node-info.yml` | Info de nodos |
| `reboot-nodes.yml` | Reinicio controlado |

### Common Commands
```bash
# Test connectivity
ansible all -m ping

# Deploy k3s cluster
ansible-playbook playbooks/k3s.yml

# Deploy MetalLB
ansible-playbook playbooks/metallb.yml

# Dry-run before applying
ansible-playbook playbooks/k3s.yml --check
```

## File Structure
```
homelab-ansible/
├── playbooks/
│   ├── gateway.yml
│   ├── k3s.yml
│   ├── metallb.yml
│   └── ...
├── roles/
│   ├── wireguard/
│   ├── dnsmasq/
│   └── nfs/
└── inventory/

/srv/
├── nfs/
│   ├── rp2/          # Root filesystem for rp2
│   └── rp3/          # Root filesystem for rp3
└── tftp/
    ├── 440dc91d/     # Boot files rp2
    └── 02671e08/     # Boot files rp3

/etc/rancher/k3s/
└── config.yaml       # flannel-iface: eth0
```

## Troubleshooting

### k3s: Pods no se comunican entre nodos

**Causa:** Flannel eligió interfaz incorrecta.
```bash
# Verificar IPs anunciadas
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'

# Si rp1-master muestra 192.168.100.x en vez de 10.0.0.1:
sudo tee /etc/rancher/k3s/config.yaml << EOF
flannel-iface: eth0
EOF
sudo systemctl restart k3s
```

### k3s: LoadBalancer en pending

**Causa:** MetalLB no instalado.
```bash
ansible-playbook playbooks/metallb.yml
```

### General: Node sin internet
```bash
sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
```

## Documentation
```
docs/
├── decisions/           # ADRs
├── concepts/            # Teoría
├── guides/              # How-to
├── k3s-setup.md         # Guía de k3s y troubleshooting
├── architecture.md
└── troubleshooting.md
```

## Development Guidelines

- Document new configurations in `docs/`
- Write ADRs for architectural decisions
- Test playbooks with `--check` before applying
- Keep commits in Spanish
- All nodes use `admin` user with UID 1000

## Pending

- [x] k3s cluster - `playbooks/k3s.yml`
- [x] MetalLB - `playbooks/metallb.yml`
- [ ] Observability en k8s (Prometheus, Grafana, Loki)
- [ ] Longhorn (storage distribuido)
- [ ] Cert-Manager (certificados TLS)
- [ ] Alerting (Alertmanager)