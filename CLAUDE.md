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

**Red de la casa (módem Movistar 192.168.1.0/24, WiFi "casona") — fuera del homelab:**

| Device | IP | MAC | Rol |
|--------|----|-----|-----|
| Módem Movistar (MitraStar) | 192.168.1.1 | 44:3b:14:fc:15:c0 | Router ISP, DHCP de la casa |
| rp1 WAN | 192.168.1.89 | 00:e0:4c:68:3d:a2 | `enx00e04c683da2` |
| Cámara Tapo C110 | 192.168.1.162 (reservada en el módem) | 3c:78:95:4c:6b:a4 | WiFi 2.4 "casona", RTSP :554, consumida por go2rtc |

Los pods alcanzan 192.168.1.x vía el MASQUERADE de rp1. Para ver equipos WiFi de la casa, escanear desde rp1 (cableado) — desde la Mac por WiFi se pierden dispositivos 2.4 GHz en ahorro de energía.

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

⚠️ **El DNAT solo aplica a tráfico que entra por la WAN** (`-i enx00e04c683da2`). Consecuencias:
- Todo Ingress es visible para **cualquiera en el WiFi "casona"** (invitados incluidos). No exponer por Ingress apps sin login.
- Desde fuera de casa (Tailscale) los Ingress **no** funcionan.

### Dos formas de exponer un servicio

| | Ingress (Traefik) | `tailscale serve` en rp1 (ADR-017) |
|---|---|---|
| URL | `http://miapp.k8s.homelab.local` | `https://rp1-master.tailb0ca85.ts.net:<puerto>` |
| Quién accede | Cualquiera en "casona" | Solo dispositivos del tailnet |
| Desde fuera de casa | ❌ | ✅ |
| TLS | ❌ (Cert-Manager pendiente) | ✅ Let's Encrypt automático |
| Usar para | Apps no sensibles o con login propio | Apps sensibles / sin auth (cámara, notificaciones) |

**Ingress:** `Service` (ClusterIP) + `Ingress` con host `miapp.k8s.homelab.local`. DNS ya resuelve.

**tailscale serve:**
1. Fijar `spec.clusterIP` en el Service (rp1 no resuelve el DNS del cluster; si el Service se recrea la IP cambiaría)
2. Agregar entrada a `tailscale_serve` en `homelab-ansible/playbooks/tailscale.yml`
3. `ansible-playbook playbooks/tailscale.yml --tags serve`
4. Puertos 8443+ (el 443 de rp1 lo ocupa Traefik Docker)

| Puerto | App | ClusterIP |
|---|---|---|
| 8443 | go2rtc | 10.43.92.177:1984 |
| 8444 | ntfy | 10.43.200.10:80 |
| 8445 | Frigate | 10.43.200.11:5000 |
| 8446 | Prometheus | 10.43.200.12:9090 |
| 8447 | Grafana (login propio) | 10.43.200.13:3000 |
| 8448 | Alertmanager | 10.43.200.14:9093 |

---

## Storage

| StorageClass | Provisioner | Uso |
|---|---|---|
| `local-path` (default) | rancher.io/local-path | Todas las apps stateful |

- **Longhorn removido** (ADR-016) — causaba bloqueos de kernel en rp3 (iSCSI). StorageClasses `longhorn`/`longhorn-static`, PVs huérfanos y CSIDriver eliminados (nov 2026). `local-path` es la **única** StorageClass default.
- **Siempre declarar `storageClassName: local-path`** en PVCs y en values de Helm (ej. Redis de falcosidekick, Loki) — no depender del default.
- **local-path** usa `WaitForFirstConsumer` — el PVC permanece Pending hasta que un Pod lo use, esto es normal.
- Los pods stateful tienen `nodeSelector` explícito — si se mueven de nodo pierden sus datos. Preferir workers (SSD), no rp1.
- Paths en workers: `/mnt/ssd` (SSD montado), `/mnt/ssd/rancher`, `/mnt/ssd/docker`
- Si un PV/namespace queda en `Terminating` por finalizers de un driver que ya no existe: `kubectl patch pv <pv> --type merge -p '{"metadata":{"finalizers":null}}'` (solo si el driver fue removido).

| App | Nodo pinado | Tamaño | Estado |
|-----|-------------|--------|--------|
| ntfy | rp3-node | 2Gi | Desplegado |
| Frigate | rp2-node | 100Gi (solo eventos) | Desplegado |
| Prometheus | rp2-node | 20Gi | Desplegado |
| Grafana | rp3-node | 5Gi | Desplegado |
| Alertmanager | rp3-node | 2Gi | Desplegado |
| n8n | rp3-node | 5Gi | Manifiestos, no desplegado |

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

NFS/TFTP instalados (nodos bootean desde SSD local). ⚠️ Los servicios NFS (`nfs-mountd`, `rpcbind`, …) **están corriendo** en rp1 aunque no se usan.

**iSCSI desactivado** en los 3 nodos (`playbooks/disable-iscsi.yml`): `iscsid.socket`, `iscsid.service` y `open-iscsi.service` enmascarados + `blacklist iscsi_tcp`. Reversible con `systemctl unmask`. Resto de Longhorn (ADR-016).

⚠️ **El GitHub Actions runner no está instalado** en rp1 (no hay unit `actions.runner.*` ni `~/actions-runner`).

---

## Apps k8s desplegadas

Estado real del cluster (nov 2026). Verificar con `kubectl get ns` y `kubectl get pods -A` — **no asumir** que una app corre solo porque tiene manifiestos.

| App | Namespace | Storage | Acceso |
|-----|-----------|---------|--------|
| go2rtc (cámara C110) | cameras | — (stateless) | `https://rp1-master.tailb0ca85.ts.net:8443` (tailscale serve) |
| ntfy | ntfy | local-path (2Gi, rp3) | `https://rp1-master.tailb0ca85.ts.net:8444` (tailscale serve) |
| Frigate | cameras | local-path (100Gi, rp2) | `https://rp1-master.tailb0ca85.ts.net:8445` (tailscale serve) |
| Prometheus | monitoring | local-path (20Gi, rp2) | `https://rp1-master.tailb0ca85.ts.net:8446` |
| Grafana | monitoring | local-path (5Gi, rp3) | `https://rp1-master.tailb0ca85.ts.net:8447` (login) |
| Alertmanager | monitoring | local-path (2Gi, rp3) | `https://rp1-master.tailb0ca85.ts.net:8448` |
| Blackbox Exporter, kube-state-metrics | monitoring | — | interno |
| MetalLB | metallb-system | — | — |
| kite | kube-system | — | sin Ingress |

**Con manifiestos en `k8s-apps/` pero NO desplegadas** (borradas a propósito junto con Longhorn): `n8n/`, `registry/`. Al redesplegarlas, revisar los Ingress (ver "Dos formas de exponer un servicio").

**Sin desplegar aún:** `argocd/` (solo namespace + Ingress), `falco/` (Helm values; tiene un `TODO(human)` en `sockets:` — en k3s es `/run/k3s/containerd/containerd.sock`).

### go2rtc
- Credenciales RTSP en el Secret `camera-credentials` (ns `cameras`), creado con `kubectl create secret` — **no está en git**. Comando en el header de `02-go2rtc-deployment.yml`.
- Streams: `casona` (2304x1296 + audio AAC) y `casona_sub` (baja resolución). Ver: `.../stream.html?src=casona`
- Sin auth en go2rtc a propósito: no tiene Ingress, el acceso lo controla Tailscale.
- La imagen usa ENTRYPOINT `tini`: en `args` hay que incluir el binario (`["go2rtc", "-config", ...]`).

### Frigate
- v0.18.2, detector **CPU** (Pi 5 sin acelerador): ~5 fps, inferencia ~10 ms, ~0.2 cores en reposo. Con Hailo-8L/Coral cambiar `detectors`.
- Consume los streams de **nuestro go2rtc** (`go2rtc.cameras.svc:8554`) — la Tapo mantiene una sola conexión. Detecta en `casona_sub` (1280x720), graba `casona` (copia sin recodificar; la Pi 5 no decodifica H.264 por hardware).
- Grabación **solo eventos**: continuous/motion = 0 días; alerts (person) 30 días, pre 5s/post 10s; detections (dog, cat) 7 días.
- `/config` escribible vía `initContainer` que copia el ConfigMap en cada arranque → **cambios hechos en la UI de Frigate se pierden al reiniciar; editar `01-frigate-config.yml`**.
- `/dev/shm` (256Mi) y `/tmp/cache` (1Gi) son `emptyDir` en memoria (equivalente a `--shm-size`/tmpfs).
- Puerto 5000 = UI sin auth; acceso solo por Tailscale (ADR-017).

### Notificaciones (Frigate → ntfy)
```
Frigate --MQTT frigate/reviews--> Mosquitto --> frigate-notify --HTTP + snapshot--> ntfy topic "camara" --> 📱
```
- **Mosquitto** (`05-mosquitto.yml`): solo ClusterIP, anónimo (solo pods del cluster llegan).
- **frigate-notify** v0.5.4 (`06-frigate-notify.yml`): modo `reviews` → solo **alerts** (person), `min_score: 0.7`, `notify_once`. Perros/gatos no notifican.
- Usuario ntfy `frigate`: **write-only** en `camara`. Token en Secret `frigate-notify-ntfy` (ns cameras, no en git); un `initContainer` lo inyecta reemplazando `__NTFY_TOKEN__` del ConfigMap en un `emptyDir` en memoria.
- ntfy tiene adjuntos habilitados (`attachment-cache-dir`, 5M/archivo, 72h).
- Probar sin pasar frente a la cámara (los eventos manuales tienen score 0 → usar `score`, y cerrarlos después):
  ```bash
  kubectl exec -n cameras deploy/frigate -c frigate -- curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"duration":10,"score":0.95,"draw":{}}' localhost:5000/api/events/casona/person/create
  kubectl exec -n cameras deploy/frigate -c frigate -- curl -s -X PUT -H 'Content-Type: application/json' -d '{}' localhost:5000/api/events/<id>/end
  ```

### Monitoring (ns monitoring)
- Sin Ingress: todo por tailscale serve (8446-8448). Imágenes fijadas: Prometheus v3.13.5, Grafana 13.1.10, Alertmanager v0.34.2, KSM v2.20.0.
- Targets: apiserver, kubelet, cAdvisor, pods anotados, node-exporter (systemd en los 3 nodos, :9100), blackbox ICMP/SSH, **Frigate** (`/api/metrics`), Prometheus, Alertmanager.
- Alertmanager → ntfy self-hosted, topic **`alertas`**, `?template=alertmanager` (plantilla nativa de ntfy). Usuario ntfy `alertmanager` write-only; token en Secret `alertmanager-ntfy` (montado como `credentials_file`). Antes usaba un topic público de ntfy.sh.
- Grafana secret cifrado con SOPS: `sops -d k8s-apps/monitoring-stack/grafana/secrets/05-grafana-secrets.yml | kubectl apply -f -` (llave age en la ubicación por defecto de sops).
- **Alertas** (`prometheus/09-prometheus-rules.yml`, 22 reglas en 5 grupos): nodos (caído, CPU, memoria, disco + predicción 24h, temperatura 75/82 °C, PSI I/O y memoria, NTP, reinicio), servicios (systemd caído/failed, probes blackbox), kubernetes (CrashLoop >3 reinicios/15m, pod no listo excl. Succeeded, nodo NotReady, PVC >85%), frigate (cámara sin video, frames saltados, inferencia lenta), monitoreo (target caído).
- Ruteo por severidad → prioridad ntfy: `critical` = 5 (repite c/1h), `warning` = 3 (c/4h), `info` = 2 (sin "resolved"). Inhibiciones: NodoSinRespuesta silencia el resto del nodo; TemperaturaCritica silencia TemperaturaAlta.
- Validar antes de aplicar: `promtool check rules` (en el pod de Prometheus) y `amtool check-config` (en el de Alertmanager). Tras aplicar, esperar sync del ConfigMap y `POST /-/reload` en ambos.
- Prueba de alerta: `kubectl exec -n monitoring deploy/alertmanager -- wget -qO- --header='Content-Type: application/json' --post-data='[{"labels":{"alertname":"Prueba","severity":"warning"}}]' http://localhost:9093/api/v2/alerts`
- `spec.clusterIP` es inmutable: para fijarla en un Service existente hay que borrarlo y recrearlo.
- Dashboards en carpeta "Homelab", montados juntos vía volumen `projected`: `Pods Monitor` (09), **`Frigate`** (10) y **`Nodos`** (11: hardware, PSI, servicios systemd, kernel/SO, red). Frigate y Nodos se **generan** con `grafana/dashboards/gen-*-dashboard.py` — editar el script y regenerar, no el YAML.
- node-exporter (systemd, no k8s) tiene `--collector.systemd` limitado a los units de `node_exporter_systemd_units` en `playbooks/node-exporter.yml`. Agregar un servicio a vigilar = agregarlo a esa lista. El job de Prometheus pone el label `node` (rp1-master/rp2-node/rp3-node).
- Scrape de Frigate descarta series `type="Other"` y las labels `cmdline`/`pid` (cambian en cada reinicio → alta cardinalidad).
- Prometheus recarga config con `wget -qO- --post-data='' http://localhost:9090/-/reload`, pero el ConfigMap tarda ~1 min en llegar al pod: esperar antes de recargar.

### ntfy
- `auth-default-access: deny-all`. Usuario `admin`, password en el Secret `ntfy-admin` (ns `ntfy`, no está en git): `kubectl get secret ntfy-admin -n ntfy -o jsonpath='{.data.password}' | base64 -d`
- `upstream-base-url: https://ntfy.sh` es necesario para push instantáneo en iOS (solo envía el ID del mensaje, no el contenido).
- Publicadores dentro del cluster: `http://ntfy.ntfy.svc.cluster.local/<topic>`.
- Topics: `camara` (Frigate), `alertas` (Alertmanager), `homelab` (pruebas). Usuarios: `admin` (todo), `frigate` y `alertmanager` (write-only en su topic).

---

## CI/CD

`.github/workflows/test-app.yml` — build y push al registry local en cada push a `main` que modifica `apps/**`.

- Corre en self-hosted runner (`[self-hosted, homelab]`) en rp1-master
- Security gate previo verifica que el actor sea el owner del repo (protección contra fork PRs maliciosos)
- Pushea a `registry.k8s.homelab.local` con tags: SHA completo, SHA corto, `latest`
- ⚠️ El registry **no está desplegado** actualmente → el workflow fallará hasta redesplegar `k8s-apps/registry/`
- Para agregar una nueva app: agregar detección en `detect-changes` y un nuevo job `build-<app>`

---

## Docker Stacks → k8s

| Stack | Estado |
|-------|--------|
| `stacks/observability/` | Manifiestos en `k8s-apps/monitoring-stack/` (no desplegado) |
| `stacks/n8n/` | Manifiestos en `k8s-apps/n8n/` (no desplegado) |
| `stacks/registry/` | Manifiestos en `k8s-apps/registry/` (no desplegado) |
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
| `dns-client.yml` | resolv.conf estático en workers (10.0.0.1) | nodes |
| `tailscale.yml` | VPN mesh + `tailscale serve` (`--tags serve`, lista `tailscale_serve`) | gateway |
| `node-exporter.yml` | Prometheus metrics + collector systemd (`node_exporter_systemd_units`) | all |
| `registry.yml` | Registry privado + registries.yaml en nodos | gateway |
| `github-runner.yml` | Self-hosted CI/CD runner | gateway |
| `disable-iscsi.yml` | Enmascara iSCSI (resto de Longhorn); aborta si hay sesiones activas | all |
| `update-nodes.yml` | apt upgrade | nodes |
| `reboot-nodes.yml` | Reinicio controlado | all |

**Obsoletos:** `setup-netboot-server.yml`, `prepare-node.yml`, `update-kernel.yml`, `install-basic-tools-nodes.yml`, `longhorn.yml`, `longhorn-storage.yml`

---

## Reglas de desarrollo

### Código y configuración
- Manifiestos k8s van en `k8s-apps/<nombre-app>/` con archivos numerados (`01-`, `02-`, etc.)
- Playbooks Ansible se prueban con `--check` antes de aplicar en producción (ojo: `command`/`shell` se saltan en `--check`; tareas de solo lectura llevan `check_mode: false`)
- Imágenes con versión fija (no `latest`), preferir releases con ≥2 semanas
- Secrets: o cifrados con **SOPS + age** (`.sops.yaml` cifra todo `*secret*.yml`; ej. grafana, n8n) o creados con `kubectl create secret` fuera de git. Nunca en texto plano.
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
# local-path usa WaitForFirstConsumer — el PVC queda Pending hasta que un Pod lo use, es normal
# Si el pod también está en Pending: kubectl describe pod <nombre> -n <namespace>
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

### Probe falla tras agregar auth a una app
`httpGet` a un endpoint autenticado devuelve 401 → readiness falla → rollout se cuelga. Usar `tcpSocket` o un endpoint de health sin auth.

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
├── certifications/            # Roadmap y materiales KCNA
├── homelab-ansible/
│   ├── inventory/inventory.yml
│   ├── playbooks/
│   └── roles/                 # dnsmasq, wireguard, nfs
├── k8s-apps/
│   ├── argocd/                # namespace + ingress (no desplegado)
│   ├── falco/                 # Helm values (no desplegado)
│   ├── frigate/               # NVR + detección IA (desplegado, ns cameras, rp2)
│   ├── go2rtc/                # cámara Tapo C110 (desplegado, ns cameras)
│   ├── kite/
│   ├── longhorn/              # restos, obsoleto (ADR-016)
│   ├── metallb/               # IP pool config
│   ├── monitoring-stack/      # Prometheus, Grafana, Alertmanager, Blackbox (desplegado)
│   ├── n8n/                   # (no desplegado)
│   ├── ntfy/                  # desplegado
│   ├── registry/              # (no desplegado)
│   ├── storage-learning/
│   └── test-app/
├── stacks/                    # Docker Compose (migración en curso)
└── docs/
    ├── decisions/             # ADRs 001-017
    ├── concepts/
    ├── guides/                # firewall.md, k3s-guide.md, network-troubleshooting.md
    ├── reference/             # IPs, URLs, comandos
    └── runbooks/
```

---

## Pending

- [x] k3s cluster
- [x] MetalLB
- [x] CI/CD con GitHub Actions (self-hosted runner) — roto: falta el registry y el runner no está instalado
- [x] ntfy (via tailscale serve)
- [x] go2rtc para cámara Tapo C110 (via tailscale serve)
- [x] Limpieza de restos de Longhorn (StorageClasses, PV, CSIDriver)
- [x] ADR-017: servicios sensibles via `tailscale serve` en vez de Ingress
- [x] Monitoring (Prometheus, Grafana, Alertmanager → ntfy, Blackbox, KSM, scrape Frigate)
- [ ] Redesplegar n8n, registry (revisar exposición de Ingress)
- [ ] Migrar Secrets creados a mano (camera-credentials, ntfy-admin, frigate-notify-ntfy, alertmanager-ntfy) a SOPS
- [x] Dashboard Grafana para Frigate
- [x] Panel "CPU Usado por Pod" completado
- [x] Dashboard Nodos + collector systemd en node-exporter
- [x] iSCSI desactivado y enmascarado en los 3 nodos
- [ ] Desactivar servicios NFS en rp1 si no se usan
- [ ] Reinstalar GitHub Actions runner (`playbooks/github-runner.yml`) + registry para que el CI/CD vuelva a funcionar
- [x] Alertas Prometheus (22 reglas) → ntfy `alertas` con prioridad por severidad
- [ ] Alerta "dead man's switch" (Watchdog) hacia un servicio externo: hoy si cae todo el cluster, no llega ninguna alerta
- [ ] Actualizar node_exporter (1.7.0 en el playbook)
- [ ] Falco (completar `sockets:` en values) + falcosidekick → ntfy
- [x] Frigate (detección por CPU, grabación solo eventos)
- [x] Alertas de Frigate → ntfy (Mosquitto + frigate-notify, topic `camara`)
- [ ] Evaluar acelerador (Hailo-8L AI Kit / Coral) si se agregan cámaras
- [ ] Router/AP en modo bridge (Archer C6 o Mercusys MR30G) para red IoT en 10.0.0.x → bloquear internet a la cámara con UFW
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
| 016 | local-path sobre Longhorn (iSCSI causaba bloqueos de kernel en rp3) |
| 017 | `tailscale serve` en vez de Ingress para servicios sensibles (cámara, ntfy) |
