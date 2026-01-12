# Guía de Uso de Playbooks

Esta guía explica cuándo y cómo usar cada playbook del proyecto.

## Diagrama de Flujo

```
┌─────────────────────────────────────────────────────────────────┐
│                    NUEVO NODO (con microSD)                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │   1. prepare-node.yml         │
              │   (Prepara nodo para netboot) │
              └───────────────────────────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │   2. setup-netboot-server.yml │
              │   (Crea estructura en gateway)│
              └───────────────────────────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │   3. Pasos manuales:          │
              │   - rsync filesystem          │
              │   - rsync boot files          │
              │   - Quitar microSD            │
              └───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    NODO FUNCIONANDO (netboot)                    │
└─────────────────────────────────────────────────────────────────┘
                              │
            ┌─────────────────┼─────────────────┐
            ▼                 ▼                 ▼
    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
    │ setup-ssh.yml│  │update-nodes.yml│ │ gateway.yml  │
    │ (SSH keys)   │  │(Actualizar)   │  │(Reconfigurar)│
    └──────────────┘  └──────────────┘  └──────────────┘
```

## Playbooks Disponibles

### gateway.yml

**Propósito**: Configurar el gateway completo (rp1-master) con todos los servicios.

**Cuándo usar**:
- Instalación inicial del gateway
- Reconfigurar después de cambios en variables
- Agregar nuevos peers de WireGuard
- Modificar hosts DHCP

**Ejecutar desde**: Tu Mac (via VPN o red local)

```bash
cd homelab-ansible

# Aplicar configuración completa
ansible-playbook playbooks/gateway.yml

# Dry-run primero
ansible-playbook playbooks/gateway.yml --check

# Solo WireGuard
ansible-playbook playbooks/gateway.yml --tags wireguard
```

**Roles incluidos**: wireguard, dnsmasq, nfs

---

### prepare-node.yml

**Propósito**: Preparar un nodo nuevo (con microSD) para netboot.

**Cuándo usar**:
- Configurar un Raspberry Pi nuevo antes de migrar a netboot
- Debe ejecutarse mientras el nodo aún tiene microSD

**Prerequisitos**:
1. Ubuntu instalado en microSD
2. `ssh-copy-id` ejecutado (acceso SSH sin contraseña)
3. Nodo agregado a `inventory/inventory.yml`

**Ejecutar desde**: Tu Mac

```bash
cd homelab-ansible

# Preparar un nodo específico
ansible-playbook playbooks/prepare-node.yml --limit rp4-node

# Ver qué hará sin aplicar
ansible-playbook playbooks/prepare-node.yml --limit rp4-node --check
```

**Qué hace**:
1. Muestra serial y MAC del nodo
2. Crea usuario admin con UID 1000
3. Configura sudo sin contraseña
4. Configura eth0 con DHCP
5. Configura EEPROM para netboot

---

### setup-netboot-server.yml

**Propósito**: Crear estructura NFS y TFTP en el gateway para un nuevo nodo.

**Cuándo usar**:
- Después de ejecutar `prepare-node.yml`
- Antes de copiar el filesystem con rsync

**Ejecutar desde**: Tu Mac

```bash
cd homelab-ansible

# Configurar servidor para nuevo nodo
ansible-playbook playbooks/setup-netboot-server.yml \
  -e "node_name=rp4" \
  -e "node_serial=abcd1234" \
  -e "node_mac=aa:bb:cc:dd:ee:ff"
```

**Variables requeridas**:
| Variable | Descripción | Ejemplo |
|----------|-------------|---------|
| node_name | Nombre del nodo | rp4 |
| node_serial | Últimos 8 chars del serial | 02671e08 |
| node_mac | MAC de eth0 | 2c:cf:67:a9:b9:13 |

**Qué crea**:
```
/srv/nfs/{node_name}/
├── proc/, sys/, dev/, tmp/, run/, mnt/

/srv/tftp/{node_serial}/
├── config.txt
└── current/
    └── cmdline.txt

Symlink: /srv/tftp/{mac-con-guiones} -> {node_serial}
```

---

### setup-ssh.yml

**Propósito**: Distribuir la clave SSH del gateway a todos los nodos.

**Cuándo usar**:
- Después de configurar un nuevo nodo
- Para permitir que rp1-master se conecte a nodos sin contraseña

**Ejecutar desde**: Tu Mac

```bash
cd homelab-ansible

# Distribuir claves a todos los nodos
ansible-playbook playbooks/setup-ssh.yml

# Solo a un nodo específico
ansible-playbook playbooks/setup-ssh.yml --limit rp2-node
```

**Qué hace**:
1. Lee clave pública de rp1-master (`/home/admin/.ssh/id_ed25519.pub`)
2. Agrega clave a `authorized_keys` de cada nodo

---

### update-nodes.yml

**Propósito**: Actualizar paquetes en los nodos de forma segura.

**Cuándo usar**:
- Mantenimiento rutinario
- Parches de seguridad
- Actualización de kernel

**Ejecutar desde**: Tu Mac

```bash
cd homelab-ansible

# Actualizar todos los nodos (uno a la vez)
ansible-playbook playbooks/update-nodes.yml

# Solo un nodo
ansible-playbook playbooks/update-nodes.yml --limit rp2

# Con reinicio automático si es necesario
ansible-playbook playbooks/update-nodes.yml -e "reboot=true"
```

**Variables opcionales**:
| Variable | Default | Descripción |
|----------|---------|-------------|
| reboot | false | Reiniciar si hay actualizaciones pendientes |
| update_kernel | false | Mostrar instrucciones para actualizar TFTP |

**Importante**: Si se actualiza el kernel, debes sincronizar manualmente los archivos de boot:
```bash
sudo rsync -av admin@10.0.0.2:/boot/firmware/ /srv/tftp/440dc91d/
```

---

### wireguard.yml

**Propósito**: Aplicar solo la configuración de WireGuard.

**Cuándo usar**:
- Agregar/remover peers VPN sin reconfigurar todo
- Debugging de VPN

**Ejecutar desde**: Tu Mac

```bash
cd homelab-ansible
ansible-playbook playbooks/wireguard.yml
```

---

### firewall.yml

**Propósito**: Configurar firewall (UFW) en gateway y nodos.

**Cuándo usar**:
- Instalación inicial del firewall
- Después de agregar nuevos servicios que requieren puertos
- Modificar reglas de acceso

**Ejecutar desde**: Tu Mac (via VPN)

```bash
cd homelab-ansible

# Configurar firewall en todos los hosts
ansible-playbook playbooks/firewall.yml

# Solo gateway
ansible-playbook playbooks/firewall.yml --limit gateway

# Solo nodos
ansible-playbook playbooks/firewall.yml --limit nodes

# Solo reglas SSH
ansible-playbook playbooks/firewall.yml --tags ssh

# Dry-run primero
ansible-playbook playbooks/firewall.yml --check
```

**Tags disponibles**:
| Tag | Descripción |
|-----|-------------|
| install | Instalar UFW |
| policy | Políticas por defecto |
| lan | Permitir todo desde LAN (netboot) |
| dhcp | Reglas DHCP broadcast |
| tftp | Reglas TFTP |
| ssh | Reglas SSH |
| wireguard | Reglas WireGuard |
| dns | Reglas DNS (VPN) |
| nfs | Reglas NFS (VPN) |
| web | Reglas HTTP/HTTPS |
| forward | Configurar forwarding |
| enable | Habilitar firewall |
| gateway | Reglas desde gateway (solo nodos) |

**Importante**:
- Ejecutar con `--check` primero para verificar cambios
- Si pierdes acceso SSH, necesitarás acceso físico al dispositivo

---

### common.yml

**Propósito**: Configuración base para todos los nodos (timezone, NTP, locales, paquetes).

**Cuándo usar**:
- Configuración inicial de un nodo nuevo
- Asegurar que todos los nodos tengan la misma configuración base
- Después de agregar un nuevo nodo al cluster

**Ejecutar desde**: Tu Mac (via VPN)

```bash
cd homelab-ansible

# Todos los nodos
ansible-playbook playbooks/common.yml

# Solo workers
ansible-playbook playbooks/common.yml --limit nodes

# Solo paquetes
ansible-playbook playbooks/common.yml --tags packages

# Solo timezone
ansible-playbook playbooks/common.yml --tags timezone
```

**Tags disponibles**:
| Tag | Descripción |
|-----|-------------|
| timezone | Configurar timezone (America/Santiago) |
| ntp | Instalar y configurar chrony |
| locales | Configurar locales (en_US.UTF-8, es_CL.UTF-8) |
| packages | Instalar paquetes básicos |
| hostname | Mostrar hostname actual |

**Paquetes instalados**:
htop, vim, curl, wget, git, tree, net-tools, dnsutils, jq

---

### node-info.yml

**Propósito**: Obtener información detallada de todos los nodos del homelab.

**Cuándo usar**:
- Verificar estado del cluster
- Diagnóstico de problemas
- Revisar recursos disponibles

**Ejecutar desde**: Tu Mac (via VPN)

```bash
cd homelab-ansible

# Todos los nodos
ansible-playbook playbooks/node-info.yml

# Solo un nodo específico
ansible-playbook playbooks/node-info.yml --limit rp2-node
```

**Información que muestra**:
- **Sistema**: hostname, OS, kernel, uptime, temperatura
- **Recursos**: memoria usada/total, disco, CPUs
- **Red**: IP eth0, MAC, conectividad a internet
- **Servicios** (solo gateway): estado de dnsmasq, NFS, WireGuard, leases DHCP

---

### reboot-nodes.yml

**Propósito**: Reiniciar nodos de forma controlada (uno a la vez).

**Cuándo usar**:
- Después de actualizar kernel
- Aplicar cambios que requieren reinicio
- Mantenimiento programado

**Ejecutar desde**: Tu Mac (via VPN)

```bash
cd homelab-ansible

# Todos los nodos (pide confirmación para cada uno)
ansible-playbook playbooks/reboot-nodes.yml

# Solo un nodo específico
ansible-playbook playbooks/reboot-nodes.yml --limit rp2-node

# Solo workers
ansible-playbook playbooks/reboot-nodes.yml --limit nodes
```

**Características**:
- **Serial: 1** - Reinicia un nodo a la vez para mantener disponibilidad
- **Confirmación** - Pide confirmación antes de reiniciar cada nodo
- **Advertencia especial** - Si es el gateway, advierte que desconectará todos los nodos
- **Verificación post-reinicio** - Muestra kernel, uptime y servicios después del reinicio

**Importante**:
- Si reinicias el gateway, perderás conectividad temporalmente
- Espera a que un nodo esté completamente disponible antes de reiniciar el siguiente

---

### update-kernel.yml

**Propósito**: Copiar kernel actualizado de NFS a TFTP y reiniciar nodos netboot.

**Cuándo usar**:
- Después de ejecutar `update-nodes.yml` que actualiza el kernel
- Cuando hay nuevo kernel disponible en `/srv/nfs/{node}/boot/`

**Ejecutar desde**: Tu Mac (via VPN)

```bash
cd homelab-ansible

# Actualizar kernel para todos los nodos
ansible-playbook playbooks/update-kernel.yml

# Solo para un nodo específico
ansible-playbook playbooks/update-kernel.yml --limit rp2-node
```

**Qué hace**:
1. En el **gateway**:
   - Busca el kernel más reciente en `/srv/nfs/{node}/boot/`
   - Copia `vmlinuz-*` e `initrd.img-*` a `/srv/tftp/{serial}/current/`
2. En los **nodos**:
   - Compara kernel en ejecución vs disponible
   - Reinicia solo si hay nuevo kernel
   - Verifica kernel después del reinicio

**Nodos configurados**:
| Nodo | NFS Path | TFTP Serial |
|------|----------|-------------|
| rp2 | /srv/nfs/rp2 | 440dc91d |
| rp3 | /srv/nfs/rp3 | 02671e08 |

---

### install-basic-tools-nodes.yml

**Propósito**: Instalar herramientas básicas adicionales en los nodos workers.

**Cuándo usar**:
- Configuración inicial de nodos
- Cuando necesitas herramientas de diagnóstico adicionales

**Ejecutar desde**: Tu Mac (via VPN)

```bash
cd homelab-ansible

# Todos los workers
ansible-playbook playbooks/install-basic-tools-nodes.yml

# Solo un nodo
ansible-playbook playbooks/install-basic-tools-nodes.yml --limit rp2-node

# Con reinicio después
ansible-playbook playbooks/install-basic-tools-nodes.yml -e "reboot=true"
```

**Herramientas instaladas**:
- speedtest-cli
- htop
- git

**Variables opcionales**:
| Variable | Default | Descripción |
|----------|---------|-------------|
| reboot | false | Reiniciar después de instalar |

**Nota**: Este playbook usa `serial: 1` para instalar en un nodo a la vez.

---

### duckdns.yml

**Propósito**: Configurar DuckDNS para actualizar IP pública automáticamente.

**Cuándo usar**:
- Configuración inicial de DNS dinámico
- Cuando tu IP pública cambia frecuentemente
- Para acceder al homelab desde internet con un dominio

**Ejecutar desde**: Tu Mac (via VPN)

```bash
cd homelab-ansible

# Configurar DuckDNS (requiere token)
ansible-playbook playbooks/duckdns.yml -e "duckdns_token=TU_TOKEN_AQUI"
```

**Variables requeridas**:
| Variable | Descripción |
|----------|-------------|
| duckdns_token | Token de autenticación de DuckDNS |

**Variables por defecto**:
| Variable | Valor | Descripción |
|----------|-------|-------------|
| duckdns_domain | aren-homelab | Subdominio en duckdns.org |
| duckdns_dir | /opt/duckdns | Directorio de instalación |

**Qué crea**:
- Script `/opt/duckdns/duck.sh` para actualizar IP
- Cron job que ejecuta cada 5 minutos
- Log en `/opt/duckdns/duck.log`

**Tags disponibles**:
| Tag | Descripción |
|-----|-------------|
| install | Crear script y ejecutar actualización inicial |
| cron | Configurar cron job |
| verify | Verificar configuración |

**Dominio resultante**: `aren-homelab.duckdns.org`

---

## Orden de Ejecución para Nuevo Nodo

1. **Instalar Ubuntu** en microSD y bootear el nuevo Raspberry Pi

2. **Configurar acceso SSH** desde tu Mac:
   ```bash
   ssh-copy-id admin@IP_DEL_NUEVO_NODO
   ```

3. **Agregar al inventory** (`inventory/inventory.yml`):
   ```yaml
   nodes:
     hosts:
       rp4-node:
         ansible_host: 10.0.0.4
         ansible_user: admin
   ```

4. **Preparar el nodo**:
   ```bash
   ansible-playbook playbooks/prepare-node.yml --limit rp4-node
   ```
   → Anota el serial y MAC que muestra

5. **Configurar servidor**:
   ```bash
   ansible-playbook playbooks/setup-netboot-server.yml \
     -e "node_name=rp4 node_serial=SERIAL node_mac=MAC"
   ```

6. **Copiar filesystem** (desde rp1-master):
   ```bash
   sudo rsync -axv admin@10.0.0.4:/ /srv/nfs/rp4/ \
     --exclude=/proc/* --exclude=/sys/* --exclude=/dev/* \
     --exclude=/tmp/* --exclude=/run/* --exclude=/boot/firmware/*
   ```

7. **Copiar archivos de boot** (desde rp1-master):
   ```bash
   sudo rsync -av admin@10.0.0.4:/boot/firmware/ /srv/tftp/SERIAL/
   ```

8. **Copiar shadow** (para mantener contraseñas):
   ```bash
   sudo scp admin@10.0.0.4:/etc/shadow /srv/nfs/rp4/etc/shadow
   ```

9. **Apagar, quitar microSD, encender**

10. **Distribuir claves SSH**:
    ```bash
    ansible-playbook playbooks/setup-ssh.yml --limit rp4-node
    ```

## Troubleshooting

### El playbook falla con "unreachable"
- Verificar que puedes hacer `ssh admin@IP_DEL_NODO`
- Verificar que el nodo está en el inventory

### El playbook falla con "permission denied"
- Verificar que `ssh-copy-id` se ejecutó
- Verificar que sudo sin contraseña está configurado

### Cambios no se aplican
- Ansible es idempotente: si el estado ya es correcto, no hace cambios
- Usa `-v` para ver más detalles
- Usa `--diff` para ver qué cambiaría
