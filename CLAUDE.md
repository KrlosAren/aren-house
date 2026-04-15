# CLAUDE.md

Guía para Claude Code cuando trabaja en este repositorio.

## Project Overview

Homelab con 3 Raspberry Pi 5. Una Pi actúa como gateway/router y las otras dos como workers con SSD local. Incluye un cluster k3s con MetalLB para LoadBalancer.

**Contexto del usuario:** Experiencia sólida en Docker, aprendiendo Kubernetes (k3s) paso a paso. Actualmente migrando servicios Docker al cluster k3s de forma gradual.

## Architecture
```
Internet (CGNAT) → Modem ISP (192.168.1.x/24)
                        │
                   [USB-ETH] enx00e04c683da2 (IP dinámica DHCP del ISP)
                        │
                   rp1-master (Gateway + k3s Control Plane)
                   WAN: dinámica (192.168.1.x) / LAN: 10.0.0.1 (estática)
                        │
                   [eth0] LAN 10.0.0.0/24
                        │
                   Switch TP-Link SG105PE (10.0.0.5)
                        │
                   ├── rp2-node (10.0.0.2) - k3s worker, SSD 500GB, IP estática via netplan
                   └── rp3-node (10.0.0.3) - k3s worker, SSD 500GB, IP estática via netplan

Tailscale VPN: 100.x.x.x (mesh, bypasses CGNAT) - primary
WireGuard VPN: 10.0.1.0/24 (legacy/backup)
```

**Nota:** La IP WAN de rp1 es dinámica (DHCP del ISP). Usar Tailscale o verificar IP actual con `ip addr show enx00e04c683da2` antes de conectar.

## Devices

| Device | IP | MAC | Role |
|--------|-----|-----|------|
| rp1-master | 10.0.0.1 (LAN estática) | 2c:cf:67:a9:b8:51 | Gateway, k3s master, SSD 500GB |
| rp2-node | 10.0.0.2 (netplan estática) | 2c:cf:67:88:9e:f5 | k3s worker, SSD 500GB |
| rp3-node | 10.0.0.3 (netplan estática) | 2c:cf:67:a9:b9:13 | k3s worker, SSD 500GB |
| switch | 10.0.0.5 | ec:75:0c:ff:fc:d6 | TP-Link SG105PE |

## Networks

| Red | Rango | Uso |
|-----|-------|-----|
| Nodos (LAN) | 10.0.0.0/24 | Red física entre RPis |
| Pods | 10.42.0.0/16 | Red interna de pods (Flannel) |
| Services | 10.43.0.0/16 | ClusterIPs |
| MetalLB | 10.0.0.50-60 | LoadBalancer IPs |
| DHCP | 10.0.0.100-200 | Clientes DHCP dinámico (switch, dispositivos futuros) |

## Kubernetes (k3s)

### Ingress y flujo de tráfico
```
Cliente → DNS (dnsmasq) → IP WAN rp1 → DNAT → 10.0.0.50 → MetalLB → Traefik k3s → Ingress → Service → Pod
```

- `*.homelab.local` → 10.0.0.1 (Traefik Docker, servicios legacy)
- `*.k8s.homelab.local` → IP WAN rp1 (DNAT → MetalLB 10.0.0.50, Traefik k3s)
- dnsmasq escucha en LAN (eth0) + WAN (enx00e04c683da2), DHCP solo en LAN
- DNAT en firewall redirige WAN :80/:443 → 10.0.0.50 (MetalLB)

Para exponer un servicio nuevo en k8s:
1. Crear Service (ClusterIP) apuntando a los pods
2. Crear Ingress con host `miapp.k8s.homelab.local`
3. DNS ya resuelve `*.k8s.homelab.local` → IP WAN rp1 → DNAT → MetalLB

### Storage
- **local-path** (provisioner incluido en k3s) como StorageClass por defecto
- **Longhorn** instalado, usado selectivamente (e.g., n8n PVC)
- Workloads con I/O → `nodeSelector: kubernetes.io/hostname: rp3-node` (SSD)
- PostgreSQL fuera del cluster en Docker+systemd (ADR-013)

### Configuración crítica

**/etc/rancher/k3s/config.yaml (solo master)**
```yaml
flannel-iface: eth0
```
rp1-master tiene múltiples interfaces (eth0 + USB ethernet). Sin esto, Flannel elige la IP incorrecta y los pods entre nodos no se comunican.

### Node Labels
```bash
kubectl label nodes rp1-master storage=ssd storage-size=500gb
kubectl label nodes rp2-node storage=ssd storage-size=500gb
kubectl label nodes rp3-node storage=ssd storage-size=500gb
```

## Services on rp1-master

- **dnsmasq**: DHCP (dispositivos LAN), DNS (.homelab.local, .k8s.homelab.local)
- **k3s server**: Kubernetes control plane
- **Traefik (Docker)**: Reverse proxy para servicios Docker (:80/:443)
- **Tailscale**: VPN mesh (subnet router for 10.0.0.0/24)
- **NAT**: iptables MASQUERADE
- **UFW**: Firewall
- **PostgreSQL (Docker)**: Base de datos para apps k8s (puerto 5432)
- **GitHub Actions Runner**: Self-hosted CI/CD runner

**Nota:** NFS/TFTP siguen instalados en rp1 pero ya no están en uso activo (los nodos bootean desde SSD local).

## Networking: IPs estáticas en nodos

Los nodos rp2 y rp3 usan IP estática via netplan — **no dependen de DHCP**.

**Motivo:** Un bug en rp1 causa que los DHCP OFFERs a `255.255.255.255` sean enrutados por la interfaz WAN en vez de eth0, debido a la interacción entre k3s/kube-router y la tabla de routing `local` del kernel. Ver ADR-015.

Netplan de nodos (`/etc/netplan/01-network.yaml`):
```yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: false
      addresses:
        - 10.0.0.X/24  # .2 para rp2, .3 para rp3
      routes:
        - to: default
          via: 10.0.0.1
      nameservers:
        addresses:
          - 10.0.0.1
        search:
          - homelab.local
```

## Ansible

### Conectividad
Ansible se ejecuta desde Mac. Los nodos (10.0.0.x) son accesibles vía ProxyJump por rp1:

```yaml
# inventory/inventory.yml
ansible_ssh_common_args: '-o ProxyJump=admin@<IP_WAN_rp1>'
```

**Importante:** La IP WAN de rp1 cambia en cada reboot del ISP. Verificar antes de correr playbooks:
```bash
# Ver IP actual de rp1
ssh admin@<ultima_ip_conocida> "ip addr show enx00e04c683da2 | grep inet"
```

### Playbooks

| Playbook | Función | Target |
|----------|---------|--------|
| `gateway.yml` | Configuración completa de rp1-master | gateway |
| `common.yml` | Config base (timezone, NTP, locales, paquetes) | all |
| `k3s.yml` | Instalar k3s server y agents | all |
| `metallb.yml` | Instalar MetalLB (pool 10.0.0.50-60) | master |
| `firewall.yml` | Configurar UFW | all |
| `docker.yml` | Instalar Docker | nodes |
| `local-storage.yml` | Montar discos locales en nodos | nodes |
| `setup-ssh.yml` | Distribuir claves SSH de rp1 a nodos | nodes |
| `wireguard.yml` | Configurar WireGuard VPN | gateway |
| `tailscale.yml` | Configurar Tailscale VPN mesh | all |
| `duckdns.yml` | Configurar DuckDNS (IP pública) | gateway |
| `node-exporter.yml` | Instalar Prometheus node_exporter | all |
| `registry.yml` | Registry privado local | gateway |
| `update-nodes.yml` | Actualizar paquetes | nodes |
| `node-info.yml` | Info de nodos | all |
| `reboot-nodes.yml` | Reinicio controlado | all |
| `github-runner.yml` | Instalar GitHub Actions runner | gateway |
| `longhorn.yml` | Instalar dependencias Longhorn (iSCSI) | all |
| `longhorn-storage.yml` | Crear directorios storage Longhorn | all |
| `dns-client.yml` | Configurar resolv.conf estático en workers | nodes |

**Playbooks obsoletos (netboot):** `setup-netboot-server.yml`, `prepare-node.yml`, `update-kernel.yml`, `install-basic-tools-nodes.yml`

## Docker Stacks → k8s Migration

| Stack | Estado | k8s |
|-------|--------|-----|
| `stacks/observability/` | Migrado a k8s (Prometheus + Grafana) | `k8s-apps/monitoring-stack/` |
| `stacks/n8n/` | Migrado a k8s | `k8s-apps/n8n/` |
| `stacks/pihole/` | En Docker | Pendiente |
| `stacks/registry/` | Migrado a k8s | `k8s-apps/registry/` |
| `stacks/router/` | Traefik Docker | Coexiste con Traefik k3s |
| `stacks/storage-apps/` | PostgreSQL en Docker+systemd | Fuera del cluster (ADR-013) |

## File Structure
```
aren-house/
├── CLAUDE.md                  # Este archivo
├── README.md                  # Overview del proyecto
├── .github/workflows/         # GitHub Actions CI/CD
│   └── test-app.yml           # Build & push al registry local
├── apps/                      # Código fuente de aplicaciones
│   └── test-app/              # Express app de prueba
├── homelab-ansible/           # Automatización con Ansible
│   ├── ansible.cfg
│   ├── inventory/inventory.yml
│   ├── playbooks/
│   └── roles/                 # wireguard, dnsmasq, nfs
├── k8s-apps/                  # Manifiestos de Kubernetes
│   ├── longhorn/              # Ingress para Longhorn UI
│   ├── metallb/               # Configuración MetalLB (IP pool)
│   ├── monitoring-stack/      # Prometheus + Grafana en k8s
│   ├── n8n/                   # n8n (Longhorn PVC)
│   ├── registry/              # Docker Registry + UI
│   ├── storage-learning/      # App de prueba nginx con PVC
│   └── test-app/              # Test app (imagen del registry local)
├── stacks/                    # Docker Compose stacks (en migración a k8s)
└── docs/                      # Documentación
    ├── decisions/             # ADRs (001-015)
    ├── concepts/              # Teoría
    ├── guides/                # How-to
    ├── reference/             # Referencia rápida (IPs, ports, URLs, commands)
    └── runbooks/              # Operaciones
```

## Troubleshooting rápido

### k3s: Pods no se comunican entre nodos
```bash
# Verificar IPs de Flannel (debe ser 10.0.0.x, NO IP WAN)
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'
# Fix: flannel-iface: eth0 en /etc/rancher/k3s/config.yaml + restart k3s
```

### k3s: LoadBalancer en pending
```bash
ansible-playbook playbooks/metallb.yml
```

### Nodo sin internet
```bash
sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
```

### Nodo sin IP (DHCP no funciona)
Los nodos usan IP estática via netplan. Si un nodo no tiene IP, verificar `/etc/netplan/01-network.yaml` y correr `sudo netplan apply`.

### SSH a nodos desde Mac
```bash
# Requiere ProxyJump por rp1
ssh -J admin@<IP_WAN_rp1> admin@10.0.0.2  # rp2
ssh -J admin@<IP_WAN_rp1> admin@10.0.0.3  # rp3
```

## Project History

Cada decisión está documentada con ADRs en `docs/decisions/`. Nunca borrar ADRs, solo marcar como superseded.

### Evolución de Boot de nodos
1. **Netboot PXE/NFS** (2025-12) → Workers bootean desde rp1 via red (ADR-006)
2. **SSD local** (2026-04) → Cada nodo tiene Ubuntu en SSD propio, independiente de rp1 (ADR-014)
3. **IPs estáticas netplan** (2026-04) → Sin dependencia de DHCP en nodos (ADR-015)

### Evolución de VPN
1. **OpenVPN** → Descartado por complejidad (ADR-001)
2. **WireGuard** (2025-12) → VPN primaria (ADR-001)
3. **Tailscale** (2025-12) → Reemplazó WireGuard por CGNAT del ISP (ADR-008, ADR-009)

### Evolución de Storage
1. **NFS puro** → Docker usaba driver `vfs` sobre NFS (lento)
2. **Storage local** (2025-12) → SSD local con overlay2 (ADR-007)
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
2. **k8s migration** (2026-02) → Prometheus + Grafana migrados a k8s con service discovery nativo

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
- [x] Grafana en k8s - `k8s-apps/monitoring-stack/grafana/`
- [x] Registry privado en k8s - `k8s-apps/registry/`
- [x] n8n en k8s - `k8s-apps/n8n/`
- [x] CI/CD con GitHub Actions - `.github/workflows/test-app.yml`
- [x] Longhorn (dependencias instaladas) - `playbooks/longhorn.yml`
- [x] Migración nodos a SSD local - ADR-014
- [x] IPs estáticas en nodos - ADR-015
- [ ] Loki (logs centralizados)
- [ ] Cert-Manager (certificados TLS)
- [ ] Alerting (Alertmanager)
- [ ] Migrar Pi-hole al cluster
- [ ] IP estática WAN para rp1 (actualmente dinámica del ISP)
