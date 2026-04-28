# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Homelab con 3 Raspberry Pi 5. `rp1-master` actúa como gateway/router/k3s control plane. `rp2-node` y `rp3-node` son workers con SSD local.

**Usuario:** Experiencia sólida en Docker, aprendiendo Kubernetes paso a paso. Migra servicios Docker → k3s de forma gradual.

---

## Comandos comunes

```bash
# Ansible — correr desde homelab-ansible/
cd homelab-ansible
ansible-playbook playbooks/<playbook>.yml --check   # dry-run
ansible-playbook playbooks/<playbook>.yml --limit gateway
ansible-playbook playbooks/<playbook>.yml --tags nat,forward

# kubectl
kubectl get pods -A
kubectl logs -n <ns> <pod> --tail=50
kubectl rollout restart deployment/<name> -n <ns>
kubectl apply -f k8s-apps/<app>/

# SSH a nodos
ssh -J admin@100.107.98.121 admin@10.0.0.2   # via Tailscale
ssh admin@10.0.0.1                            # gateway directo (si en LAN)

# Verificar estado del cluster
kubectl get nodes -o wide
kubectl get svc -A | grep LoadBalancer
```

---

## Arquitectura de red

```
Internet (CGNAT)
      │
 Modem ISP (192.168.1.x/24)
      │
 enx00e04c683da2  ← WAN, IP dinámica DHCP del ISP (actualmente 192.168.1.89)
      │
 rp1-master (10.0.0.1) ← Gateway + k3s control plane
      │
 eth0 LAN 10.0.0.0/24
      │
 Switch TP-Link (10.0.0.5)
      ├── rp2-node  10.0.0.2  k3s worker, SSD 500GB
      └── rp3-node  10.0.0.3  k3s worker, SSD 500GB

Tailscale VPN: 100.x.x.x (mesh, bypasses CGNAT)
  rp1-master Tailscale IP: 100.107.98.121
  Mac Tailscale IP:        100.70.50.39
```

| Device | IP LAN | MAC | Rol |
|--------|--------|-----|-----|
| rp1-master | 10.0.0.1 | 2c:cf:67:a9:b8:51 | Gateway, k3s master |
| rp2-node | 10.0.0.2 | 2c:cf:67:88:9e:f5 | k3s worker |
| rp3-node | 10.0.0.3 | 2c:cf:67:a9:b9:13 | k3s worker |
| switch | 10.0.0.5 | ec:75:0c:ff:fc:d6 | TP-Link SG105PE |

| Red | Rango | Uso |
|-----|-------|-----|
| LAN | 10.0.0.0/24 | Red física |
| DHCP | 10.0.0.100-200 | Clientes dinámicos |
| MetalLB | 10.0.0.50-60 | LoadBalancer IPs |
| Pods | 10.42.0.0/16 | Flannel |
| Services | 10.43.0.0/16 | ClusterIPs |

**IP WAN de rp1 es dinámica.** Verificar antes de conectar:
```bash
tailscale status | grep rp1-master   # via Tailscale (preferido)
ssh admin@10.0.0.1 "ip addr show enx00e04c683da2 | grep inet"
```

---

## Configuraciones críticas

### k3s: flannel-iface (OBLIGATORIO en master)

`/etc/rancher/k3s/config.yaml` en rp1-master:
```yaml
flannel-iface: eth0
```
rp1 tiene múltiples interfaces. Sin esto, Flannel elige la IP WAN y los pods entre nodos no se comunican.

### dnsmasq: interfaces

dnsmasq escucha en **tres interfaces** — si falta alguna, el DNS falla para ese cliente:

```ini
interface=eth0           # LAN — DHCP + DNS
interface=enx00e04c683da2  # WAN — solo DNS
interface=tailscale0     # Tailscale — solo DNS (Mac y otros clientes VPN)
no-dhcp-interface=enx00e04c683da2
no-dhcp-interface=tailscale0
```

Template: `homelab-ansible/roles/dnsmasq/templates/dnsmasq.conf.j2`

### DNS desde Mac (Tailscale)

Los archivos `/etc/resolver/` deben apuntar a la IP Tailscale de rp1, no a 10.0.0.1:
```
/etc/resolver/homelab.local      → nameserver 100.107.98.121
/etc/resolver/k8s.homelab.local  → nameserver 100.107.98.121
```
Si la IP Tailscale cambia (reinstalación): `tailscale status | grep rp1-master` y actualizar ambos archivos.

### Firewall: UFW + k3s no usar netfilter-persistent

Tres componentes modifican iptables al arrancar: **UFW**, **k3s** (kube-router + Flannel), **Tailscale**. Pueden pisarse entre sí.

**Regla**: nunca usar `netfilter-persistent` junto con UFW. Son mecanismos de persistencia incompatibles — `netfilter-persistent save` guarda un snapshot que sobrescribe las reglas de UFW en el próximo boot.

Las reglas NAT/FORWARD van en `/etc/ufw/before.rules` (gestionado por `playbooks/firewall.yml`). Las reglas FORWARD deben ser **explícitas** — no confiar solo en `DEFAULT_FORWARD_POLICY="ACCEPT"` porque kube-router se inserta primero en la cadena.

Si se ejecutó `netfilter-persistent save` por error:
```bash
sudo systemctl disable netfilter-persistent
sudo rm -f /etc/iptables/rules.v4 /etc/iptables/rules.v6
ansible-playbook playbooks/firewall.yml --limit gateway --tags nat
```

### IPs estáticas en nodos (netplan)

rp2 y rp3 usan IP estática via netplan — **no dependen de DHCP**. Un bug en rp1 enruta los DHCP broadcasts a la interfaz WAN (ADR-015).

`/etc/netplan/01-network.yaml` en cada nodo:
```yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: false
      addresses: [10.0.0.X/24]
      routes: [{to: default, via: 10.0.0.1}]
      nameservers:
        addresses: [10.0.0.1]
        search: [homelab.local]
```

---

## Flujo de tráfico k8s

```
Cliente → DNS → *.k8s.homelab.local → 192.168.1.89 (WAN rp1)
       → DNAT :80/:443 → 10.0.0.50 (MetalLB)
       → Traefik k3s → Ingress → Service → Pod
```

- `*.homelab.local` → 10.0.0.1 (Traefik Docker, servicios legacy)
- `*.k8s.homelab.local` → IP WAN rp1 → DNAT → MetalLB 10.0.0.50

Para exponer un servicio nuevo:
1. `Service` (ClusterIP) apuntando a los pods
2. `Ingress` con host `miapp.k8s.homelab.local`
3. DNS ya resuelve automáticamente

---

## Storage

| StorageClass | Provisioner | Uso |
|---|---|---|
| `local-path` (default) | rancher.io/local-path | Todas las apps stateful |

- **Longhorn removido** (ADR-016) — causaba bloqueos de kernel en rp3 (iSCSI). Todos los PVCs migrados a `local-path`.
- **local-path** usa `WaitForFirstConsumer` — el PVC permanece Pending hasta que un Pod lo use, esto es normal.
- Los pods stateful tienen `nodeAffinity` explícita — si se mueven de nodo pierden sus datos.
- Paths en workers: `/mnt/ssd` (SSD montado), `/mnt/ssd/rancher`, `/mnt/ssd/docker`

| App | Nodo pinado | Tamaño |
|-----|-------------|--------|
| Prometheus | rp2-node | 20Gi |
| Grafana | rp3-node | 5Gi |
| n8n | rp3-node | 5Gi |
| Registry | — | — |

---

## Servicios en rp1-master

| Servicio | Tipo | Función |
|----------|------|---------|
| dnsmasq | systemd | DHCP (LAN) + DNS (.homelab.local, .k8s.homelab.local) |
| k3s server | systemd | Kubernetes control plane |
| Traefik | Docker | Reverse proxy servicios legacy (:80/:443) |
| Tailscale | systemd | VPN mesh + subnet router (10.0.0.0/24) |
| UFW | systemd | Firewall — reglas en /etc/ufw/before.rules |
| PostgreSQL | Docker + systemd | Base de datos para apps k8s (5432) |
| GitHub Actions Runner | systemd | CI/CD self-hosted |

NFS/TFTP instalados pero inactivos (nodos bootean desde SSD local).

---

## Apps k8s desplegadas

| App | Namespace | StorageClass | Ingress |
|-----|-----------|-------------|---------|
| Prometheus | monitoring | local-path (20Gi, rp2) | prometheus.k8s.homelab.local |
| Grafana | monitoring | local-path (5Gi, rp3) | grafana.k8s.homelab.local |
| Alertmanager | monitoring | local-path (2Gi) | alertmanager.k8s.homelab.local |
| Blackbox Exporter | monitoring | — | — |
| kube-state-metrics | monitoring | — | — |
| n8n | n8n-system | local-path (5Gi, rp3) | n8n.k8s.homelab.local |
| Registry | registry | local-path | registry.k8s.homelab.local |
| kite | kube-system | — | kite.k8s.homelab.local |

---

## CI/CD

`.github/workflows/test-app.yml` — build y push al registry local en cada push a `main` que modifica `apps/**`.

- Corre en self-hosted runner (`[self-hosted, homelab]`) en rp1-master
- Security gate previo verifica que el actor sea el owner del repo (protección contra fork PRs maliciosos)
- Pushea a `registry.k8s.homelab.local` con tags: SHA completo, SHA corto, `latest`
- Para agregar una nueva app: agregar detección en `detect-changes` y un nuevo job `build-<app>`

---

## Docker Stacks → k8s

| Stack | Estado |
|-------|--------|
| `stacks/observability/` | Migrado → `k8s-apps/monitoring-stack/` |
| `stacks/n8n/` | Migrado → `k8s-apps/n8n/` |
| `stacks/registry/` | Migrado → `k8s-apps/registry/` |
| `stacks/pihole/` | En Docker, pendiente migrar |
| `stacks/router/` | Traefik Docker, coexiste con Traefik k3s |
| `stacks/storage-apps/` | PostgreSQL en Docker+systemd (ADR-013, fuera del cluster) |

---

## Ansible

Ansible corre desde Mac. Nodos accesibles via ProxyJump por rp1.

**Antes de correr playbooks:** verificar IP WAN de rp1 en `inventory/inventory.yml`.

| Playbook | Función | Target |
|----------|---------|--------|
| `gateway.yml` | Config completa rp1 (dnsmasq, WireGuard, NFS) | gateway |
| `common.yml` | Timezone, NTP, paquetes base | all |
| `firewall.yml` | UFW + NAT rules en before.rules | all |
| `k3s.yml` | Instalar k3s server y agents | all |
| `docker.yml` | Instalar Docker | nodes |
| `local-storage.yml` | Montar SSD en /mnt/ssd | nodes |
| `longhorn.yml` | Dependencias OS de Longhorn (iSCSI) | all |
| `longhorn-storage.yml` | Crear /var/lib/longhorn y /mnt/ssd/longhorn | all |
| `dns-client.yml` | resolv.conf estático en workers (10.0.0.1) | nodes |
| `tailscale.yml` | VPN mesh | all |
| `node-exporter.yml` | Prometheus metrics | all |
| `registry.yml` | Registry privado + registries.yaml en nodos | gateway |
| `github-runner.yml` | Self-hosted CI/CD runner | gateway |
| `update-nodes.yml` | apt upgrade | nodes |
| `reboot-nodes.yml` | Reinicio controlado | all |

**Obsoletos (netboot):** `setup-netboot-server.yml`, `prepare-node.yml`, `update-kernel.yml`, `install-basic-tools-nodes.yml`

---

## Reglas de desarrollo

### Código y configuración
- Manifiestos k8s van en `k8s-apps/<nombre-app>/` con archivos numerados (`01-`, `02-`, etc.)
- Playbooks Ansible se prueban con `--check` antes de aplicar en producción
- Todos los nodos usan usuario `admin` con UID 1000
- Commits en español

### Documentación
- Nuevas configuraciones se documentan en `docs/`
- Decisiones arquitectónicas → ADR en `docs/decisions/` (nunca borrar, solo marcar superseded)
- Guías de operación → `docs/guides/`
- Runbooks para procedimientos → `docs/runbooks/`
- Referencia rápida (IPs, URLs, comandos) → `docs/reference/`

### Cuando agregar un ADR
Siempre que se tome una decisión que afecte la arquitectura: cambio de tecnología, decisión de dónde correr un servicio, cambio en networking, storage, boot, etc.

---

## Troubleshooting rápido

### Pods en Pending — PVC no se bindea
```bash
kubectl describe pvc <nombre> -n <namespace>
# Si dice "could not find StorageClass longhorn":
helm install longhorn longhorn/longhorn -n longhorn-system --create-namespace
```

### k3s: pods no se comunican entre nodos
```bash
# Verificar que Flannel usa eth0 (no WAN)
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'
# Fix: flannel-iface: eth0 en /etc/rancher/k3s/config.yaml + restart k3s
```

### Nodos sin internet después de reboot de rp1
Causa probable: `netfilter-persistent` sobrescribió reglas UFW.
```bash
# Diagnóstico (debe mostrar src 10.0.0.0/24)
ssh admin@10.0.0.1 "sudo iptables -t nat -L POSTROUTING -n -v | grep enx"
# Fix temporal
ssh admin@10.0.0.1 "sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE"
# Fix permanente
ansible-playbook playbooks/firewall.yml --limit gateway --tags nat
ssh admin@10.0.0.1 "sudo systemctl disable netfilter-persistent; sudo rm -f /etc/iptables/rules.v4"
```

### DNS no resuelve desde Mac via Tailscale
```bash
tailscale status | grep rp1-master   # IP actual de rp1
dig @100.107.98.121 grafana.k8s.homelab.local  # probar directo
# Si timeout: dnsmasq no escucha en tailscale0 → ansible-playbook playbooks/gateway.yml
# Si IP cambió: actualizar /etc/resolver/homelab.local y /etc/resolver/k8s.homelab.local
```

### Prometheus no responde HTTP (acepta TCP pero cuelga)
Causa: goroutines acumuladas esperando timeouts al k8s API (`10.43.0.1:443`). Ocurre si el cluster arrancó con conectividad inestable.
```bash
kubectl rollout restart deployment/prometheus -n monitoring
# Si se repite frecuentemente: aumentar scrape_timeout en 02-prometheus-config.yml
```
El config tiene `scrape_timeout: 10s` para limitar el bloqueo. Sin este valor, el default es igual al `scrape_interval` (30s) y los goroutines se acumulan hasta saturar el runtime.

### LoadBalancer en pending
```bash
kubectl get pods -n metallb-system
ansible-playbook playbooks/metallb.yml
```

### Nodo sin IP
Los nodos usan netplan (no DHCP). Verificar `/etc/netplan/01-network.yaml` y `sudo netplan apply`.

### SSH a nodos desde Mac
```bash
ssh -J admin@100.107.98.121 admin@10.0.0.2   # via Tailscale (preferido)
ssh -J admin@192.168.1.89 admin@10.0.0.2     # via WAN (si IP conocida)
```

---

## Estructura del repositorio

```
aren-house/
├── CLAUDE.md
├── .github/workflows/         # CI/CD (build & push al registry local)
├── apps/
│   └── test-app/              # Express app de prueba
├── homelab-ansible/
│   ├── inventory/inventory.yml
│   ├── playbooks/
│   └── roles/                 # dnsmasq, wireguard, nfs
├── k8s-apps/
│   ├── longhorn/              # Ingress Longhorn UI
│   ├── metallb/               # IP pool config
│   ├── monitoring-stack/      # Prometheus, Grafana, Alertmanager, Blackbox
│   ├── n8n/
│   ├── ntfy/                  # ntfy + ntfy-alertmanager bridge
│   ├── registry/
│   └── test-app/
├── stacks/                    # Docker Compose (migración en curso)
└── docs/
    ├── decisions/             # ADRs 001-015
    ├── concepts/
    ├── guides/                # firewall.md, k3s-guide.md, network-troubleshooting.md
    ├── reference/             # IPs, URLs, comandos
    └── runbooks/
```

---

## Pending

- [x] k3s cluster
- [x] MetalLB
- [x] Prometheus + Grafana en k8s
- [x] Alertmanager en k8s
- [x] Registry privado en k8s
- [x] n8n en k8s
- [x] CI/CD con GitHub Actions (self-hosted runner)
- [x] Longhorn instalado (Helm)
- [ ] ntfy + ntfy-alertmanager bridge (manifiestos en `k8s-apps/ntfy/`, falta desplegar)
- [ ] Loki (logs centralizados)
- [ ] Cert-Manager (TLS)
- [ ] Migrar Pi-hole al cluster
- [ ] Split DNS en Tailscale Admin (eliminar /etc/resolver/ manual)
- [ ] IP estática WAN para rp1

---

## Historial de decisiones clave

| ADR | Decisión |
|-----|----------|
| 001 | Tailscale sobre WireGuard (CGNAT del ISP) |
| 007 | Docker storage local (overlay2) sobre NFS (vfs) |
| 011 | MetalLB para LoadBalancer real sobre NodePort |
| 012 | k3s sobre k8s completo (ARM, menor consumo) |
| 013 | PostgreSQL fuera del cluster (Docker+systemd) |
| 014 | Boot desde SSD local sobre netboot PXE/NFS |
| 015 | IPs estáticas via netplan sobre DHCP en nodos |
