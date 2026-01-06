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
