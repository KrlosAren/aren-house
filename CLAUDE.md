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
| `common.yml` | Config base del sistema (timezone, NTP, locales, paquetes) |
| `k3s.yml` | Instalar k3s server y agents con Tailscale forwarding |
| `metallb.yml` | Instalar y configurar MetalLB (pool 10.0.0.50-60) |
| `firewall.yml` | Configurar UFW en gateway y nodos |
| `docker.yml` | Instalar Docker con storage driver vfs para NFS boot |
| `local-storage.yml` | Configurar y montar discos locales en nodos |
| `setup-netboot-server.yml` | Preparar estructura NFS y TFTP para netboot |
| `prepare-node.yml` | Preparar nodo para netboot |
| `setup-ssh.yml` | Distribuir claves SSH del gateway a los nodos |
| `wireguard.yml` | Instalar y configurar WireGuard VPN |
| `tailscale.yml` | Instalar y configurar Tailscale VPN mesh |
| `duckdns.yml` | Configurar DuckDNS para actualización de IP pública |
| `node-exporter.yml` | Instalar Prometheus node_exporter en todos los hosts |
| `registry.yml` | Configurar registry privado local para Docker/containerd |
| `install-basic-tools-nodes.yml` | Instalar herramientas básicas en nodos worker |
| `update-nodes.yml` | Actualizar paquetes |
| `update-kernel.yml` | Actualizar kernel en TFTP de nodos netboot |
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
├── ansible.cfg
├── inventory/
│   └── inventory.yml
├── playbooks/
│   ├── gateway.yml
│   ├── common.yml
│   ├── k3s.yml
│   ├── metallb.yml
│   ├── firewall.yml
│   ├── docker.yml
│   ├── local-storage.yml
│   ├── setup-netboot-server.yml
│   ├── prepare-node.yml
│   ├── setup-ssh.yml
│   ├── wireguard.yml
│   ├── tailscale.yml
│   ├── duckdns.yml
│   ├── node-exporter.yml
│   ├── registry.yml
│   ├── install-basic-tools-nodes.yml
│   ├── update-nodes.yml
│   ├── update-kernel.yml
│   ├── node-info.yml
│   └── reboot-nodes.yml
└── roles/
    ├── wireguard/
    ├── dnsmasq/
    └── nfs/

# Documentación (en ../docs/ relativo a homelab-ansible)
../docs/
├── architecture.md
├── k3s-setup.md
├── troubleshooting.md
├── ansible-guide.md
├── dns-setup.md
├── docker-setup.md
├── firewall-guide.md
├── linux-users-management.md
├── local-storage.md
├── netboot-concepts.md
├── netboot-node-setup.md
├── observability.md
├── ssh-authentication.md
├── tailscale-setup.md
├── decisions/        # ADRs (001-011)
├── concepts/         # Teoría (15 archivos)
├── guides/           # How-to (5 archivos)
└── runbooks/         # Operaciones (disaster-recovery, maintenance)

# En los nodos remotos
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

La documentación está en `../docs/` (relativo a homelab-ansible, es decir en la raíz del monorepo `aren-house/docs/`).

```
docs/
├── architecture.md            # Arquitectura general del homelab
├── k3s-setup.md               # Guía de k3s y troubleshooting
├── troubleshooting.md         # Troubleshooting general
├── ansible-guide.md           # Guía de Ansible
├── dns-setup.md               # Configuración DNS
├── docker-setup.md            # Configuración Docker
├── firewall-guide.md          # Guía de firewall/UFW
├── linux-users-management.md  # Gestión de usuarios Linux
├── local-storage.md           # Storage local en nodos
├── netboot-concepts.md        # Conceptos de netboot
├── netboot-node-setup.md      # Setup de nodos netboot
├── observability.md           # Observabilidad (Prometheus, Grafana)
├── ssh-authentication.md      # Autenticación SSH
├── tailscale-setup.md         # Configuración Tailscale
├── decisions/                 # ADRs (Architectural Decision Records)
│   ├── 001-wireguard-over-openvpn.md
│   ├── 002-network-segmentation.md
│   ├── 003-dnsmasq-dhcp-dns-tftp.md
│   ├── 004-ip-forwarding-nat.md
│   ├── 005-ufw-firewall.md
│   ├── 006-netboot-vs-local.md
│   ├── 007-docker-storage-overlay.md
│   ├── 008-tailscale-cgnat.md
│   ├── 009-cgnat-workaround.md
│   ├── 010-k3s-storage-on-nfs.md
│   └── 011-metallb.md
├── concepts/                  # Teoría y conceptos
│   ├── dhcp.md, dns.md, nat.md, nfs.md, pxe.md, tftp.md
│   ├── ip-forwarding.md, iptables-basics.md, ufw.md
│   ├── netplan.md, systemd.md, vpn.md, wireguard.md
│   └── raspberry-pi-eeprom.md
├── guides/                    # How-to guides
│   ├── firewall.md
│   ├── k3s-guide.md
│   ├── network-troubleshooting.md
│   ├── playbook-usage.md
│   └── service-management.md
└── runbooks/                  # Runbooks operacionales
    ├── disaster-recovery.md
    └── maintenance.md
```

## Project History

El proyecto ha evolucionado a través del tiempo. Cada decisión está documentada con ADRs (Architectural Decision Records) en `docs/decisions/`. Es importante preservar el historial y nunca borrar decisiones anteriores, solo marcarlas como superseded.

### Evolución de VPN
1. **OpenVPN** (evaluado) → Descartado por complejidad (ADR-001)
2. **WireGuard** (2025-12) → Implementado como VPN primaria. Role: `roles/wireguard/`, playbook: `wireguard.yml` (ADR-001)
3. **Tailscale** (2025-12) → Reemplazó a WireGuard como VPN primaria porque el ISP (Entel Chile) usa CGNAT, bloqueando conexiones entrantes. WireGuard queda como backup/legacy (ADR-008, ADR-009)

### Evolución de Storage
1. **NFS puro** → Docker usaba driver `vfs` sobre NFS (lento)
2. **Storage local** (2025-12) → microSD/SSD local para Docker con overlay2, symlink strategy (ADR-007)
3. **k3s storage local** (2026-01) → Mismo principio para containerd/k3s: `/var/lib/rancher` → `/var/lib/rancher-local` (ADR-010)

### Evolución de Networking k8s
1. **NodePort** → Servicios k8s expuestos en puertos altos (30000+)
2. **Traefik Docker como proxy** → Doble salto, configuración duplicada
3. **MetalLB** (2026-01) → LoadBalancer nativo con IPs reales (10.0.0.50-60), un solo entry point (ADR-011)

### Evolución de acceso a internet
1. **IP pública + port forwarding** → Asumido inicialmente
2. **Descubrimiento de CGNAT** → ISP comparte IP pública, port forwarding no funciona (ADR-009)
3. **DuckDNS + Tailscale** → DuckDNS trackea IP pública, Tailscale resuelve acceso remoto

## Development Guidelines

- Document new configurations in `docs/`
- Write ADRs for architectural decisions. Never delete old ADRs, mark them as superseded
- When a technology is replaced, document the evolution (why it was chosen, why it was replaced)
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