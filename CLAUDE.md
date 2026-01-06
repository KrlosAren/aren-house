# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Homelab with 3 Raspberry Pi 5 configured for network boot (PXE/NFS). One Pi acts as gateway/router and the other two as worker nodes that boot from the network without microSD.

## Architecture

```
Internet → Modem (192.168.100.x)
              │
         [USB-ETH] enx00e04c683da2
              │
         rp1-master (Gateway)
         192.168.100.x WAN / 10.0.0.1 LAN
              │
         [eth0]
              │
         Switch TP-Link SG105PE (10.0.0.5)
              │
         ├── rp2 (10.0.0.2) - Netboot
         └── rp3 (10.0.0.3) - Netboot

WireGuard VPN: 10.0.1.0/24
```

## Devices

| Device | IP | MAC | Serial (TFTP) | Role |
|--------|-----|-----|---------------|------|
| rp1-master | 10.0.0.1 | 2c:cf:67:a9:b8:51 | N/A | Gateway, DHCP, DNS, TFTP, NFS |
| rp2 | 10.0.0.2 | 2c:cf:67:88:9e:f5 | 440dc91d | Worker (netboot) |
| rp3 | 10.0.0.3 | 2c:cf:67:a9:b9:13 | 02671e08 | Worker (netboot) |
| switch | 10.0.0.5 | ec:75:0c:ff:fc:d6 | N/A | TP-Link SG105PE |

## Networks

| Network | Range | Interface | Purpose |
|---------|-------|-----------|---------|
| WAN | 192.168.100.x | enx00e04c683da2 | DHCP from modem |
| LAN Homelab | 10.0.0.0/24 | eth0 | Internal segmented network |
| VPN | 10.0.1.0/24 | wg0 | Remote access via WireGuard |

## Services on rp1-master

- **dnsmasq**: DHCP (fixed IPs by MAC), DNS (.homelab.local), TFTP (boot files)
- **NFS**: Root filesystems at `/srv/nfs/{rp2,rp3}/`
- **WireGuard**: VPN for remote access
- **NAT**: iptables MASQUERADE for internet access

## File Structure on Gateway

```
/srv/
├── nfs/
│   ├── rp2/          # Root filesystem for rp2
│   └── rp3/          # Root filesystem for rp3
├── tftp/
│   ├── 440dc91d/     # Boot files rp2 (by serial)
│   ├── 02671e08/     # Boot files rp3 (by serial)
│   ├── 2c-cf-67-88-9e-f5 -> 440dc91d  # Symlink by MAC
│   └── 2c-cf-67-a9-b9-13 -> 02671e08  # Symlink by MAC
└── backup/           # 163GB available
```

## Ansible

All commands run from `homelab-ansible/` directory.

### Inventory Hosts

| Ansible Host | IP | Group |
|--------------|-----|-------|
| rp1-master | 10.0.0.1 | gateway |
| rp2-node | 10.0.0.2 | nodes |
| rp3-node | 10.0.0.3 | nodes |

Connection requires VPN (WireGuard) active to reach 10.0.0.x network.

### Structure

```
homelab-ansible/
├── ansible.cfg              # inventory path, roles_path
├── inventory/inventory.yml  # Host definitions (gateway, nodes groups)
├── playbooks/
│   ├── gateway.yml          # Configure complete rp1-master
│   ├── wireguard.yml        # Configure WireGuard VPN
│   ├── firewall.yml         # Configure UFW firewall on gateway and nodes
│   ├── setup-netboot-server.yml  # Configure netboot server (TFTP/NFS)
│   ├── setup-ssh.yml        # Distribute SSH keys to nodes
│   ├── prepare-node.yml     # Prepare new node for netboot
│   ├── common.yml           # Base config: timezone, NTP, locales, packages
│   ├── node-info.yml        # Get system info from all nodes
│   ├── reboot-nodes.yml     # Controlled reboot (one at a time)
│   ├── update-nodes.yml     # Update packages on nodes
│   └── update-kernel.yml    # Update kernel on nodes
└── roles/
    ├── wireguard/           # VPN with IP forwarding
    ├── dnsmasq/             # DHCP, DNS, TFTP (with host-record for rp1)
    └── nfs/                 # NFS server and exports
```

### Role Variables

Variables in `defaults/main.yml`, overridden in playbooks:

- **wireguard**: `wireguard_peers` (VPN clients list)
- **dnsmasq**:
  - `dnsmasq_hosts` (MAC→IP mappings for DHCP)
  - `dnsmasq_network.dns` (hostname for host-record, e.g., "rp1")
- **nfs**: `nfs_nodes` (netboot nodes)

### Common Commands

```bash
# Test connectivity
ansible all -m ping

# Deploy gateway configuration
ansible-playbook playbooks/gateway.yml

# Dry-run before applying
ansible-playbook playbooks/gateway.yml --check

# Verbose output
ansible-playbook playbooks/gateway.yml -v

# Base configuration (timezone, NTP, locales, packages)
ansible-playbook playbooks/common.yml
ansible-playbook playbooks/common.yml --limit nodes    # Solo workers
ansible-playbook playbooks/common.yml --tags packages  # Solo paquetes

# Get node information (uptime, temp, memory, disk, services)
ansible-playbook playbooks/node-info.yml

# Controlled reboot (one at a time, with confirmation)
ansible-playbook playbooks/reboot-nodes.yml
ansible-playbook playbooks/reboot-nodes.yml --limit rp2-node
```

## Gateway Commands

```bash
# View DHCP leases
cat /var/lib/misc/dnsmasq.leases

# View boot logs
sudo tail -f /var/log/dnsmasq.log

# View NFS exports
sudo exportfs -v

# VPN control (from Mac)
sudo wg-quick up ~/homelab.conf
sudo wg-quick down ~/homelab.conf
```

## Key Configuration Files

| File | Purpose |
|------|---------|
| `/etc/dnsmasq.conf` | DHCP, DNS, TFTP |
| `/etc/exports` | NFS exports |
| `/etc/netplan/01-network.yaml` | Network configuration |
| `/etc/wireguard/wg0.conf` | VPN |

## Common Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| Node without internet | Missing NAT | `sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE` |
| SSH "Permission denied" | Missing /etc/shadow | Copy shadow from microSD to NFS |
| Node doesn't boot | TFTP searches by MAC | Create symlink with MAC in /srv/tftp/ |
| eth0 without IP | Not configured in netplan | Add eth0 with dhcp4: true |

## Documentation

```
docs/
├── decisions/           # ADRs (architectural decisions)
│   ├── 001-wireguard-over-openvpn.md
│   ├── 002-network-segmentation.md
│   ├── 003-dnsmasq-dhcp-dns-tftp.md
│   ├── 004-ip-forwarding-nat.md
│   └── 005-ufw-firewall.md
├── concepts/            # Theory: DHCP, DNS, TFTP, PXE, NAT, VPN, iptables, systemd, NFS, EEPROM, UFW
├── guides/              # How-to: playbook-usage, firewall, service-management, network-troubleshooting
├── runbooks/            # Procedures: disaster-recovery, maintenance
├── ansible-guide.md     # Complete Ansible guide
├── dns-setup.md         # DNS configuration guide
├── netboot-node-setup.md
├── netboot-concepts.md
├── ssh-authentication.md
└── linux-users-management.md
```

## Development Guidelines

- Document new configurations in `docs/`
- Write ADRs for architectural decisions in `docs/decisions/`
- Update role READMEs when adding new variables
- Test playbooks with `--check` before applying
- Keep commits in Spanish
- All nodes use `admin` user with UID 1000 for NFS consistency

## Pending

- [x] Firewall (ufw) with rules between networks - `playbooks/firewall.yml`
- [ ] Docker on nodes
- [ ] k3s cluster
- [ ] Monitoring with Prometheus/Grafana

## DNS

El gateway (rp1-master) actúa como servidor DNS local via dnsmasq.

### Resolución de nombres

| Hostname | IP |
|----------|-----|
| rp1.homelab.local | 10.0.0.1 |
| rp2.homelab.local | 10.0.0.2 |
| rp3.homelab.local | 10.0.0.3 |

### Configuración en macOS (cliente VPN)

Para resolver nombres `.homelab.local` desde tu Mac:
```bash
sudo mkdir -p /etc/resolver
sudo bash -c 'echo "nameserver 10.0.0.1" > /etc/resolver/homelab.local'
```

### Tipos de registros en dnsmasq

- `dhcp-host=MAC,nombre,IP` - Para clientes DHCP (rp2, rp3)
- `host-record=nombre,nombre.dominio,IP` - Para hosts estáticos (rp1)
