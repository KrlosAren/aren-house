# CLAUDE.md

Guía para Claude Code cuando trabaja en este repositorio.

## Project Overview

Homelab con 3 Raspberry Pi 5 configuradas para network boot (PXE/NFS). Una Pi actúa como gateway/router y las otras dos como workers que bootean desde la red sin microSD. Incluye un cluster k3s con MetalLB para LoadBalancer.

**Contexto del usuario:** Experiencia sólida en Docker, aprendiendo Kubernetes (k3s) paso a paso. Actualmente migrando servicios Docker al cluster k3s de forma gradual.

## Architecture
```
Internet (CGNAT) → Modem (192.168.1.89.x)
                        │
                   [USB-ETH] enx00e04c683da2
                        │
                   rp1-master (Gateway + k3s Control Plane)
                   192.168.1.89.x WAN / 10.0.0.1 LAN
                        │
                   [eth0] LAN 10.0.0.0/24
                        │
                   Switch TP-Link SG105PE (10.0.0.5)
                        │
                   ├── rp2-node (10.0.0.2) - k3s worker, microSD 32GB (solo stateless)
                   └── rp3-node (10.0.0.3) - k3s worker, SSD 240GB (workloads con I/O)

Tailscale VPN: 100.x.x.x (mesh, bypasses CGNAT) - primary
WireGuard VPN: 10.0.1.0/24 (legacy/backup)
```

## Devices

| Device | IP | MAC | Serial (TFTP) | Role |
|--------|-----|-----|---------------|------|
| rp1-master | 10.0.0.1 | 2c:cf:67:a9:b8:51 | N/A | Gateway, k3s master, SSD 500GB |
| rp2-node | 10.0.0.2 | 2c:cf:67:88:9e:f5 | 440dc91d | k3s worker (netboot), microSD 32GB |
| rp3-node | 10.0.0.3 | 2c:cf:67:a9:b9:13 | 02671e08 | k3s worker (netboot), SSD 240GB |
| switch | 10.0.0.5 | ec:75:0c:ff:fc:d6 | N/A | TP-Link SG105PE |

## Networks

| Red | Rango | Uso |
|-----|-------|-----|
| Nodos (LAN) | 10.0.0.0/24 | Red física entre RPis |
| Pods | 10.42.0.0/16 | Red interna de pods (Flannel) |
| Services | 10.43.0.0/16 | ClusterIPs |
| MetalLB | 10.0.0.50-60 | LoadBalancer IPs |
| DHCP | 10.0.0.100-200 | Clientes DHCP dinámico |

## Kubernetes (k3s)

### Ingress y flujo de tráfico
```
Cliente → DNS (dnsmasq) → 192.168.1.89 → DNAT → 10.0.0.50 → MetalLB → Traefik k3s → Ingress → Service → Pod
```

- `*.homelab.local` → 10.0.0.1 (Traefik Docker, servicios legacy)
- `*.k8s.homelab.local` → 192.168.1.89 (DNAT → MetalLB 10.0.0.50, Traefik k3s)
- dnsmasq escucha en LAN (eth0) + WAN (enx00e04c683da2), DHCP solo en LAN
- DNAT en firewall redirige WAN :80/:443 → 10.0.0.50 (MetalLB)

Para exponer un servicio nuevo en k8s:
1. Crear Service (ClusterIP) apuntando a los pods
2. Crear Ingress con host `miapp.k8s.homelab.local`
3. DNS ya resuelve `*.k8s.homelab.local` → 192.168.1.89 → DNAT → MetalLB

### Storage
- **local-path** (provisioner incluido en k3s) para PVCs
- Workloads con I/O → `nodeSelector: kubernetes.io/hostname: rp3-node` (SSD)
- Longhorn evaluado pero **no implementado** - se activará cuando se necesite replicación

### Configuración crítica

**/etc/rancher/k3s/config.yaml (solo master)**
```yaml
flannel-iface: eth0
```
rp1-master tiene múltiples interfaces (eth0 + USB ethernet). Sin esto, Flannel elige la IP incorrecta y los pods entre nodos no se comunican.

### Node Labels
```bash
kubectl label nodes rp1-master storage=ssd storage-size=500gb
kubectl label nodes rp2-node storage=sd storage-size=32gb
kubectl label nodes rp3-node storage=ssd storage-size=240gb
```

## Services on rp1-master

- **dnsmasq**: DHCP, DNS (.homelab.local, .k8s.homelab.local), TFTP
- **NFS**: Root filesystems en `/srv/nfs/{rp2,rp3}/`
- **k3s server**: Kubernetes control plane
- **Traefik (Docker)**: Reverse proxy para servicios Docker (:80/:443)
- **Tailscale**: VPN mesh (subnet router for 10.0.0.0/24)
- **NAT**: iptables MASQUERADE
- **UFW**: Firewall

## Docker Stacks → k8s Migration

Los Docker stacks en `stacks/` se migran gradualmente al cluster k3s:

| Stack | Estado | k8s |
|-------|--------|-----|
| `stacks/observability/` | Prometheus migrado, Grafana pendiente | `k8s-apps/monitoring-stack/` |
| `stacks/n8n/` | En Docker | Pendiente |
| `stacks/pihole/` | En Docker | Pendiente |
| `stacks/registry/` | Migrado a k8s (Docker stack legacy) | `k8s-apps/registry/` |
| `stacks/router/` | Traefik Docker | Coexiste con Traefik k3s |

## File Structure
```
aren-house/
├── CLAUDE.md                  # Este archivo
├── README.md                  # Overview del proyecto
├── homelab-ansible/           # Automatización con Ansible
│   ├── ansible.cfg
│   ├── inventory/inventory.yml
│   ├── playbooks/             # 20 playbooks (ver tabla abajo)
│   └── roles/                 # wireguard, dnsmasq, nfs
├── k8s-apps/                  # Manifiestos de Kubernetes
│   ├── README.md
│   ├── metallb/               # Configuración MetalLB (IP pool)
│   ├── monitoring-stack/      # Prometheus en k8s (7 manifiestos, numerados 00-06)
│   ├── storage-learning/      # App de prueba nginx con PVC
│   └── storage-longhorn/      # Manifiesto Longhorn (no aplicado)
├── stacks/                    # Docker Compose stacks (en migración a k8s)
│   ├── n8n/
│   ├── observability/         # Prometheus + Grafana (legacy)
│   ├── pihole/
│   ├── registry/
│   └── router/                # Traefik
└── docs/                      # Documentación
    ├── architecture.md
    ├── k3s-setup.md
    ├── troubleshooting.md
    ├── observability.md
    ├── dns-setup.md
    ├── docker-setup.md
    ├── local-storage.md
    ├── ansible-guide.md
    ├── linux-users-management.md
    ├── netboot-concepts.md
    ├── netboot-node-setup.md
    ├── ssh-authentication.md
    ├── tailscale-setup.md
    ├── reference/             # Referencia rápida
    │   ├── ip-allocation.md
    │   ├── ports.md
    │   └── commands.md
    ├── decisions/             # ADRs (001-012)
    ├── concepts/              # Teoría (15 archivos)
    ├── guides/                # How-to
    │   ├── firewall.md        # Guía unificada de firewall/UFW
    │   ├── k3s-guide.md
    │   ├── network-troubleshooting.md
    │   ├── playbook-usage.md
    │   └── service-management.md
    └── runbooks/              # Operaciones
        ├── disaster-recovery.md
        └── maintenance.md
```

### Remote paths (en los nodos)
```
/srv/nfs/{rp2,rp3}/           # Root filesystem NFS
/srv/tftp/{440dc91d,02671e08}/ # Boot files TFTP
/etc/rancher/k3s/config.yaml   # flannel-iface: eth0
```

## Ansible Playbooks

| Playbook | Función | Target |
|----------|---------|--------|
| `gateway.yml` | Configuración completa de rp1-master | gateway |
| `common.yml` | Config base (timezone, NTP, locales, paquetes) | all |
| `k3s.yml` | Instalar k3s server y agents | all |
| `metallb.yml` | Instalar MetalLB (pool 10.0.0.50-60) | master |
| `firewall.yml` | Configurar UFW | all |
| `docker.yml` | Instalar Docker (vfs para NFS boot) | nodes |
| `local-storage.yml` | Montar discos locales en nodos | nodes |
| `setup-netboot-server.yml` | Preparar NFS/TFTP para netboot | gateway |
| `prepare-node.yml` | Preparar nodo para netboot | nodes |
| `setup-ssh.yml` | Distribuir claves SSH | nodes |
| `wireguard.yml` | Configurar WireGuard VPN | gateway |
| `tailscale.yml` | Configurar Tailscale VPN mesh | all |
| `duckdns.yml` | Configurar DuckDNS (IP pública) | gateway |
| `node-exporter.yml` | Instalar Prometheus node_exporter | all |
| `registry.yml` | Registry privado local | gateway |
| `install-basic-tools-nodes.yml` | Herramientas básicas en workers | nodes |
| `update-nodes.yml` | Actualizar paquetes | nodes |
| `update-kernel.yml` | Actualizar kernel en TFTP | gateway+nodes |
| `node-info.yml` | Info de nodos | all |
| `reboot-nodes.yml` | Reinicio controlado | all |

## k8s-apps Conventions

- Cada app vive en su propio directorio dentro de `k8s-apps/`
- Archivos numerados con prefijo (`00-`, `01-`, ...) para orden de aplicación
- Cada app usa su propio namespace
- Ingress usa dominio `*.k8s.homelab.local`
- Aplicar con: `kubectl apply -f k8s-apps/<app>/`

## Troubleshooting rápido

### k3s: Pods no se comunican entre nodos
```bash
# Verificar IPs de Flannel (debe ser 10.0.0.x, NO 192.168.1.89.x)
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'
# Fix: flannel-iface: eth0 en /etc/rancher/k3s/config.yaml + restart k3s
```

### k3s: LoadBalancer en pending
```bash
ansible-playbook playbooks/metallb.yml
```

### Node sin internet
```bash
sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
```

## Project History

Cada decisión está documentada con ADRs en `docs/decisions/`. Nunca borrar ADRs, solo marcar como superseded.

### Evolución de VPN
1. **OpenVPN** → Descartado por complejidad (ADR-001)
2. **WireGuard** (2025-12) → VPN primaria. Role: `roles/wireguard/` (ADR-001)
3. **Tailscale** (2025-12) → Reemplazó WireGuard por CGNAT del ISP (ADR-008, ADR-009)

### Evolución de Storage
1. **NFS puro** → Docker usaba driver `vfs` sobre NFS (lento)
2. **Storage local** (2025-12) → microSD/SSD local con overlay2 (ADR-007)
3. **k3s storage local** (2026-01) → `/var/lib/rancher` → `/var/lib/rancher-local` (ADR-010)

### Evolución de Orquestación
1. **Docker standalone** (2025-12) → Sin orquestación
2. **k3s** (2026-01) → Cluster k8s, menor consumo en ARM (ADR-012)

### Evolución de Networking k8s
1. **NodePort** → Puertos altos (30000+)
2. **Traefik Docker como proxy** → Doble salto
3. **MetalLB** (2026-01) → IPs reales 10.0.0.50-60 (ADR-011)

### Evolución de Observabilidad
1. **Docker standalone** (2025-12) → Prometheus + Grafana en Docker en rp2-node
2. **k8s migration** (2026-02) → Prometheus migrado a k8s con service discovery nativo

## Development Guidelines

- Documentar nuevas configuraciones en `docs/`
- Escribir ADRs para decisiones arquitectónicas en `docs/decisions/`
- Cuando se reemplaza una tecnología, documentar la evolución
- Probar playbooks con `--check` antes de aplicar
- Commits en español
- Todos los nodos usan usuario `admin` con UID 1000
- Manifiestos k8s nuevos van en `k8s-apps/<nombre-app>/` con archivos numerados

## Pending

- [x] k3s cluster - `playbooks/k3s.yml`
- [x] MetalLB - `playbooks/metallb.yml`
- [x] Prometheus en k8s - `k8s-apps/monitoring-stack/`
- [ ] Grafana en k8s
- [ ] Loki (logs centralizados)
- [ ] Cert-Manager (certificados TLS)
- [ ] Alerting (Alertmanager)
- [ ] Migrar stacks Docker restantes al cluster
