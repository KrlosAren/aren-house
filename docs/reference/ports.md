# Puertos y Servicios

Referencia rápida de todos los puertos usados en el homelab.

## Gateway (rp1-master)

| Puerto | Protocolo | Servicio | Acceso |
|--------|-----------|----------|--------|
| 22 | TCP | SSH | LAN, VPN, WAN (limit) |
| 53 | TCP/UDP | DNS (dnsmasq) | LAN, VPN, WAN |
| 67-68 | UDP | DHCP (dnsmasq) | LAN |
| 69 | UDP | TFTP (dnsmasq) | LAN |
| 80 | TCP | HTTP (DNAT → MetalLB/Traefik k3s) | Anywhere |
| 443 | TCP | HTTPS (DNAT → MetalLB/Traefik k3s) | Anywhere |
| 111 | TCP/UDP | RPC/portmapper (NFS) | LAN |
| 2049 | TCP/UDP | NFS | LAN |
| 5000 | TCP | Docker Registry | LAN |
| 6443 | TCP | k3s API Server | LAN, Tailscale, WAN |
| 8472 | UDP | Flannel VXLAN | LAN |
| 9100 | TCP | node_exporter (Prometheus) | LAN |
| 10250 | TCP | kubelet | LAN |
| 51820 | UDP | WireGuard | Anywhere |

## Workers (rp2-node, rp3-node)

| Puerto | Protocolo | Servicio | Acceso |
|--------|-----------|----------|--------|
| 22 | TCP | SSH | LAN, VPN |
| 8472 | UDP | Flannel VXLAN | LAN |
| 9100 | TCP | node_exporter | LAN |
| 10250 | TCP | kubelet | LAN |

## Kubernetes Services

| Puerto | Servicio | Tipo | IP Externa |
|--------|----------|------|------------|
| 80/443 | Traefik (k3s) | LoadBalancer | 10.0.0.50 |
| 9090 | Prometheus | ClusterIP (via Ingress) | prometheus.k8s.homelab.local |

## Docker Stacks (en migración a k8s)

| Puerto | Servicio | Host | Estado |
|--------|----------|------|--------|
| 9090 | Prometheus | rp2-node | Migrado a k8s |
| 3000 | Grafana | rp2-node | Pendiente migrar |
| 5678 | n8n | - | En Docker |
