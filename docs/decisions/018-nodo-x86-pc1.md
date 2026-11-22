# ADR-018: Primer nodo x86_64 (pc1-node) en el cluster arm64

**Estado:** Aceptado
**Fecha:** 2026-11-22

## Contexto

El cluster tenía tres Raspberry Pi 5 (arm64): rp1-master (control plane + gateway) y dos workers con SSD. Había una PC de escritorio disponible:

| | pc1-node |
|---|---|
| Placa | MSI B85-G43 GAMING (MS-7816), BIOS V12.2 (2013) |
| CPU | Intel i7-4770 (Haswell), 4C/8T, AVX2 |
| iGPU | HD 4600 — VAAPI H.264 (Quick Sync) |
| RAM | 12 GB (4 GB CL9 + 8 GB CL10, Kingston HyperX) → 11 GiB usables |
| Disco | WD Blue 1 TB **HDD** (no SSD) |
| Red | Killer E220x 1 GbE (`enp2s0`) + WiFi (`wlp3s0`, desactivado) |
| SO | Ubuntu Server 26.04.1 LTS, kernel 7.0 |

Más CPU y RAM que cualquier Pi, y una iGPU que decodifica H.264 por hardware (la Pi 5 no puede).

## Decisión

Agregar la PC como **worker k3s** (`pc1-node`, `10.0.0.4`), igual que rp2/rp3:

- **Red:** IP estática por netplan en `enp2s0` (ADR-015), gateway y DNS en rp1. WiFi eliminado de netplan (la config original quedó en `/root/00-installer-config.yaml.bak`): un nodo con salida propia a "casona" saltaría el NAT/UFW de rp1.
- **Usuario:** `b85-server` (instalador) renombrado a `admin` con `usermod -l` → conserva **UID 1000** (convención de los playbooks).
- **Storage:** LV propio `ubuntu-vg/rancher` de 600 G montado **directamente** en `/var/lib/rancher` (sin symlink). Imágenes y PVs `local-path` no pueden llenar `/` (100 G). Quedan ~228 G libres en el VG para crecer con `lvextend -r`.
- **k3s agent:** misma versión que el server (`INSTALL_K3S_VERSION` leído de rp1) y `flannel-iface: enp2s0` vía `k3s_flannel_iface` en el inventario.
- **Playbooks multi-arquitectura:** la arquitectura del binario sale de `ansible_facts['architecture']` (`node-exporter.yml`, `github-runner.yml`); el paso de cgroups en `cmdline.txt` corre solo en `aarch64`.
- **Monitoreo:** node-exporter + probes ICMP/SSH en Prometheus. Las alertas de temperatura usan `type="x86_pkg_temp"` (85/95 °C) en x86 y `type="cpu-thermal"` (75/82 °C) en Pi; las zonas `acpitz` se ignoran.

### BIOS

| Ajuste | Valor | Motivo |
|---|---|---|
| Restore after AC Power Loss | Power On | Vuelve sola tras un corte de luz, como las Pi |
| Windows 8.1/10 WHQL Support | Enabled | UEFI puro (CSM off) |
| Fast Boot | Enabled | |
| LAN Option ROM | Disabled | Sin PXE |
| **DRAM Frequency** | **DDR3-1333 (fijo)** | Con módulos distintos, en *Auto* re-entrenaba la RAM en cada arranque: **POST 65 s → 9 s** (boot total 1 min 23 s → 27 s) |

⚠️ Si se resetea la CMOS o se actualiza el BIOS, reaplicar estos ajustes (sobre todo DRAM Frequency).

## Consecuencias

**Positivo:**
- +4C/8T y +11 GiB de RAM para el cluster; el nodo más potente.
- iGPU con VAAPI: candidato para Frigate (decodificar el stream principal por hardware).
- 590 G en `/var/lib/rancher` para cargas con mucho volumen (grabaciones).

**Negativo:**
- **Cluster multi-arquitectura:** el scheduler no mira la arquitectura de la imagen. Una imagen solo-arm64 que caiga en pc1-node falla con `exec format error`. Revisar `docker manifest inspect <imagen>` antes de desplegar, o fijar `nodeSelector: kubernetes.io/arch`.
- **HDD:** bien para escritura secuencial (grabaciones), mal para I/O aleatorio. Prometheus sigue en el SSD de rp2.
- Mayor consumo eléctrico que una Pi (~30-60 W en reposo vs ~5 W).
- OpenVINO **en GPU** no soporta Haswell (requiere Gen8+); en CPU con AVX2 sí.

## Notas operativas

- Acceso: `ssh admin@10.0.0.4` (vía subnet router de Tailscale).
- El MASQUERADE de rp1 y el DNAT dependen del cable WAN de rp1 (`enx00e04c683da2`). Al instalar pc1-node se soltó y todo el homelab quedó inaccesible desde fuera; pc1-node (en la LAN) sirvió para diagnosticar.
