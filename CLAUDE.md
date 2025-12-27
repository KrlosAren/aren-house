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

### Structure

```
homelab-ansible/
├── ansible.cfg              # inventory path, roles_path
├── inventory/inventory.yml  # Host definitions (gateway, nodes groups)
├── playbooks/
│   ├── gateway.yml          # Configure complete rp1-master
│   ├── setup-ssh.yml        # Distribute SSH keys to nodes
│   └── prepare-node.yml     # Prepare new node for netboot
└── roles/
    ├── wireguard/           # VPN with IP forwarding
    ├── dnsmasq/             # DHCP, DNS, TFTP
    └── nfs/                 # NFS server and exports
```

### Role Variables

Variables in `defaults/main.yml`, overridden in playbooks:

- **wireguard**: `wireguard_peers` (VPN clients list)
- **dnsmasq**: `dnsmasq_hosts` (MAC→IP mappings)
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
├── decisions/           # ADRs (WireGuard, segmentation, dnsmasq)
├── concepts/            # DHCP, DNS, TFTP, PXE, NAT, VPN theory
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

- [ ] Firewall (ufw) with rules between networks
- [ ] Docker on nodes
- [ ] k3s cluster
- [ ] Monitoring with Prometheus/Grafana
