# Role: dnsmasq

Instala y configura dnsmasq como servidor DHCP, DNS y TFTP para el homelab.

## Funciones

| Servicio | Descripción |
|----------|-------------|
| DHCP | Asigna IPs automáticamente, IPs fijas por MAC address |
| DNS | Resuelve nombres locales (.homelab.local) y reenvía consultas externas |
| TFTP | Sirve archivos de boot para PXE (netboot Raspberry Pi) |

## Variables

| Variable | Default | Descripción |
|----------|---------|-------------|
| `dnsmasq_interface` | `eth0` | Interfaz de red interna |
| `dnsmasq_wan_interface` | `enx00e04c683da2` | Interfaz WAN (hacia modem) |
| `dnsmasq_dns_servers` | `[1.1.1.1, 8.8.8.8]` | DNS upstream |
| `dnsmasq_domain` | `homelab.local` | Dominio local |
| `dnsmasq_network.gateway` | `10.0.0.1` | IP del gateway |
| `dnsmasq_network.range_start` | `10.0.0.100` | Inicio rango DHCP dinámico |
| `dnsmasq_network.range_end` | `10.0.0.200` | Fin rango DHCP dinámico |
| `dnsmasq_tftp_root` | `/srv/tftp` | Directorio raíz TFTP |
| `dnsmasq_hosts` | `[]` | Lista de hosts con IP fija |

## Ejemplo de uso
```yaml
- hosts: gateway
  become: yes
  vars:
    dnsmasq_hosts:
      - name: rp2
        mac: "2c:cf:67:88:9e:f5"
        ip: "10.0.0.2"
      - name: rp3
        mac: "2c:cf:67:a9:b9:13"
        ip: "10.0.0.3"
      - name: switch
        mac: "ec:75:0c:ff:fc:d6"
        ip: "10.0.0.5"
  roles:
    - dnsmasq
```

## Verificación
```bash
# Estado del servicio
sudo systemctl status dnsmasq

# Ver leases DHCP
cat /var/lib/misc/dnsmasq.leases

# Probar resolución DNS
ping rp2.homelab.local

# Ver logs
sudo tail -f /var/log/dnsmasq.log
```

## Dependencias

Configura también netplan para que el gateway use dnsmasq como su propio DNS.
