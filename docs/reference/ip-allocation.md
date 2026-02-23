# Asignación de IPs

Referencia rápida de todas las IPs y rangos del homelab.

## IPs Fijas (Dispositivos)

| IP | Dispositivo | MAC | Rol |
|----|-------------|-----|-----|
| 10.0.0.1 | rp1-master | 2c:cf:67:a9:b8:51 | Gateway, k3s master |
| 10.0.0.2 | rp2-node | 2c:cf:67:88:9e:f5 | k3s worker (netboot) |
| 10.0.0.3 | rp3-node | 2c:cf:67:a9:b9:13 | k3s worker (netboot) |
| 10.0.0.5 | switch | ec:75:0c:ff:fc:d6 | TP-Link SG105PE |

## Rangos de Red

| Rango | Propósito | Gestionado por |
|-------|-----------|----------------|
| 10.0.0.0/24 | LAN del homelab | dnsmasq (DHCP) |
| 10.0.0.1-49 | Dispositivos con IP fija | Reservado en dnsmasq |
| 10.0.0.50-60 | MetalLB LoadBalancer pool | MetalLB (L2 mode) |
| 10.0.0.100-200 | DHCP dinámico | dnsmasq |
| 10.42.0.0/16 | Pods de Kubernetes | Flannel (CNI) |
| 10.43.0.0/16 | Services de Kubernetes (ClusterIP) | k3s |
| 192.168.1.89.0/24 | WAN (modem) | Modem DHCP |

## VPN

| Rango | VPN | Estado |
|-------|-----|--------|
| 100.x.x.x | Tailscale mesh | Activo (primary) |
| 10.0.1.0/24 | WireGuard | Legacy/backup |

## MetalLB - IPs asignadas

| IP | Servicio | Namespace |
|----|----------|-----------|
| 10.0.0.50 | Traefik (Ingress Controller) | kube-system |

> Las IPs 10.0.0.51-60 están disponibles para futuros servicios LoadBalancer.

## DNS

| Dominio | Resuelve a | Propósito |
|---------|------------|-----------|
| *.homelab.local | 10.0.0.1 | Servicios Docker (Traefik Docker) |
| *.k8s.homelab.local | 192.168.1.89 | Servicios k8s (DNAT → MetalLB 10.0.0.50 → Traefik k3s) |
| rp1-master.homelab.local | 10.0.0.1 | Gateway |
| rp2-node.homelab.local | 10.0.0.2 | Worker 2 |
| rp3-node.homelab.local | 10.0.0.3 | Worker 3 |
