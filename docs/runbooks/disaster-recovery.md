# Disaster Recovery

Procedimientos para recuperar el homelab ante diferentes escenarios de fallo.

## Escenarios Cubiertos

1. [Gateway completamente muerto](#gateway-completamente-muerto)
2. [Filesystem de nodo corrupto](#filesystem-de-nodo-corrupto)
3. [Configuración rota (gateway funciona pero servicios no)](#configuración-rota)
4. [Pérdida del SSD del gateway](#pérdida-del-ssd-del-gateway)
5. [Reconstrucción total](#reconstrucción-total)

---

## Gateway Completamente Muerto

**Escenario**: El Raspberry Pi del gateway (rp1-master) murió y necesitas uno nuevo.

**Impacto**: Todos los nodos dejan de funcionar (sin DHCP, NFS, TFTP).

**Tiempo estimado**: 2-3 horas

### Prerequisitos
- Raspberry Pi 5 nuevo
- microSD con Ubuntu Server
- Acceso al SSD con datos de NFS/TFTP (si sobrevivió)

### Procedimiento

1. **Instalar Ubuntu Server en microSD**
   ```bash
   # Descargar Ubuntu Server para RPi
   # Flashear con Raspberry Pi Imager
   # Configurar usuario inicial
   ```

2. **Configurar acceso SSH desde tu Mac**
   ```bash
   ssh-copy-id admin@IP_NUEVA
   ```

3. **Clonar repositorio en tu Mac**
   ```bash
   git clone git@github.com:KrlosAren/aren-house.git
   cd aren-house/homelab-ansible
   ```

4. **Actualizar inventory con nueva IP temporal**
   ```yaml
   # inventory/inventory.yml
   gateway:
     hosts:
       rp1-master:
         ansible_host: IP_NUEVA  # IP temporal del modem
   ```

5. **Ejecutar playbook del gateway**
   ```bash
   ansible-playbook playbooks/gateway.yml
   ```

6. **Montar SSD si sobrevivió**
   ```bash
   # En el nuevo gateway
   sudo blkid  # Encontrar UUID del SSD
   sudo mkdir -p /srv
   sudo mount UUID="xxx" /srv
   ```

7. **Si el SSD no sobrevivió, reconstruir nodos**
   - Ver sección [Reconstrucción Total](#reconstrucción-total)

8. **Reinstalar Tailscale y k3s**
   ```bash
   ansible-playbook playbooks/tailscale.yml
   ansible-playbook playbooks/k3s.yml
   ansible-playbook playbooks/metallb.yml
   ```

9. **Verificar servicios**
   ```bash
   systemctl status dnsmasq nfs-kernel-server k3s
   sudo exportfs -v
   tailscale status
   kubectl get nodes -o wide
   ```

9. **Probar que los nodos bootean**
   ```bash
   # Encender nodos y verificar
   ansible nodes -m ping
   ```

---

## Filesystem de Nodo Corrupto

**Escenario**: El filesystem NFS de un nodo (ej: rp2) está corrupto.

**Impacto**: Solo ese nodo no funciona.

**Tiempo estimado**: 30-60 minutos

### Opción A: Restaurar desde backup

```bash
# En el gateway
sudo rm -rf /srv/nfs/rp2/*
sudo rsync -axv /srv/backup/rp2/ /srv/nfs/rp2/
```

### Opción B: Reconstruir desde microSD

1. **Bootear nodo con microSD**
   - Insertar microSD con Ubuntu fresco
   - Bootear el nodo

2. **Preparar el nodo**
   ```bash
   # Desde tu Mac
   ansible-playbook playbooks/prepare-node.yml --limit rp2-node
   ```

3. **Copiar filesystem a NFS**
   ```bash
   # Desde el gateway
   sudo rsync -axv admin@10.0.0.2:/ /srv/nfs/rp2/ \
     --exclude=/proc/* --exclude=/sys/* --exclude=/dev/* \
     --exclude=/tmp/* --exclude=/run/* --exclude=/boot/firmware/*
   ```

4. **Copiar archivos de boot**
   ```bash
   sudo rsync -av admin@10.0.0.2:/boot/firmware/ /srv/tftp/440dc91d/
   ```

5. **Copiar shadow**
   ```bash
   sudo scp admin@10.0.0.2:/etc/shadow /srv/nfs/rp2/etc/shadow
   ```

6. **Quitar microSD y reiniciar**

---

## Configuración Rota

**Escenario**: El gateway funciona pero los servicios no (dnsmasq, NFS, WireGuard).

**Impacto**: Variable según qué servicio falló.

**Tiempo estimado**: 15-30 minutos

### Diagnóstico

```bash
# Ver qué servicios fallan
systemctl status dnsmasq nfs-kernel-server wg-quick@wg0

# Ver logs
sudo journalctl -xe
```

### Solución: Re-aplicar Ansible

```bash
# Desde tu Mac
cd homelab-ansible
ansible-playbook playbooks/gateway.yml
```

### Si Ansible no puede conectar

1. **Verificar SSH al gateway**
   ```bash
   ssh admin@192.168.1.89.x  # IP del modem
   ```

2. **Arreglar manualmente y luego Ansible**
   ```bash
   # En el gateway
   sudo systemctl restart dnsmasq
   sudo systemctl restart nfs-kernel-server
   sudo systemctl restart wg-quick@wg0
   ```

---

## Pérdida del SSD del Gateway

**Escenario**: El SSD de 500GB con NFS/TFTP murió.

**Impacto**: Todos los nodos pierden su filesystem.

**Tiempo estimado**: 3-4 horas

### Procedimiento

1. **Instalar nuevo SSD**
   ```bash
   # Conectar SSD
   sudo fdisk -l  # Verificar que se detecta
   ```

2. **Particionar y formatear**
   ```bash
   sudo fdisk /dev/sda
   # Crear partición única
   sudo mkfs.ext4 /dev/sda1
   ```

3. **Montar en /srv**
   ```bash
   sudo mkdir -p /srv
   sudo mount /dev/sda1 /srv

   # Agregar a fstab
   UUID=$(sudo blkid -s UUID -o value /dev/sda1)
   echo "UUID=$UUID /srv ext4 defaults,nofail 0 2" | sudo tee -a /etc/fstab
   ```

4. **Crear estructura base**
   ```bash
   sudo mkdir -p /srv/nfs /srv/tftp /srv/backup
   ```

5. **Reconstruir cada nodo** (ver [Reconstrucción Total](#reconstrucción-total))

---

## Cluster k3s Caído

**Escenario**: El cluster k3s no funciona (server no responde, agents desconectados, pods no se programan).

**Impacto**: Todas las aplicaciones desplegadas en Kubernetes dejan de funcionar.

### Caso A: Solo los agents están desconectados

```bash
# Verificar estado de los agents
ssh rp2-node "sudo systemctl status k3s-agent"
ssh rp3-node "sudo systemctl status k3s-agent"

# Reiniciar agents
ssh rp2-node "sudo systemctl restart k3s-agent"
ssh rp3-node "sudo systemctl restart k3s-agent"

# Verificar que se unieron al cluster
kubectl get nodes
```

### Caso B: El server k3s no responde

```bash
# En rp1-master
sudo systemctl status k3s
sudo journalctl -u k3s --since "10 minutes ago"

# Reiniciar k3s server
sudo systemctl restart k3s

# Esperar y verificar
kubectl get nodes
kubectl get pods -n kube-system
```

### Caso C: Reinstalar k3s completo

Si k3s está corrupto y no puede recuperarse:

```bash
# 1. Desinstalar k3s en agents primero
ssh rp2-node "sudo /usr/local/bin/k3s-agent-uninstall.sh"
ssh rp3-node "sudo /usr/local/bin/k3s-agent-uninstall.sh"

# 2. Desinstalar k3s en server
sudo /usr/local/bin/k3s-uninstall.sh

# 3. Limpiar storage local
sudo rm -rf /var/lib/rancher-local/*

# 4. Reinstalar via Ansible
ansible-playbook playbooks/k3s.yml

# 5. Reinstalar MetalLB
ansible-playbook playbooks/metallb.yml

# 6. Verificar cluster
kubectl get nodes -o wide
kubectl get pods -n kube-system
kubectl get svc -n kube-system traefik
```

### Caso D: Recuperar kubeconfig

```bash
# Si perdiste acceso kubectl desde tu Mac
scp admin@10.0.0.1:/etc/rancher/k3s/k3s.yaml ~/.kube/config-homelab

# Cambiar IP al Tailscale
# En ~/.kube/config-homelab, reemplazar server: https://127.0.0.1:6443
# por server: https://100.94.94.49:6443

export KUBECONFIG=~/.kube/config-homelab
kubectl get nodes
```

**Nota:** Los manifests de aplicaciones desplegadas en k3s se pierden al reinstalar. Mantén tus manifests YAML en el repositorio git para poder re-aplicarlos.

---

## Reconstrucción Total

**Escenario**: Necesitas reconstruir todo el homelab desde cero.

**Tiempo estimado**: 4-6 horas

### Fase 1: Gateway (1-2 horas)

1. **Instalar Ubuntu en microSD del gateway**

2. **Configurar acceso inicial**
   ```bash
   ssh-copy-id admin@IP_DEL_GATEWAY
   ```

3. **Clonar repo**
   ```bash
   git clone git@github.com:KrlosAren/aren-house.git
   cd aren-house/homelab-ansible
   ```

4. **Configurar inventory**
   ```yaml
   # inventory/inventory.yml
   gateway:
     hosts:
       rp1-master:
         ansible_host: IP_DEL_GATEWAY
         ansible_user: admin
   ```

5. **Ejecutar gateway playbook**
   ```bash
   ansible-playbook playbooks/gateway.yml
   ```

6. **Configurar SSD**
   ```bash
   # En gateway
   sudo mkdir -p /srv/nfs /srv/tftp /srv/backup
   # Montar SSD en /srv
   ```

### Fase 2: Nodos (1-2 horas por nodo)

Para cada nodo (rp2, rp3):

1. **Instalar Ubuntu en microSD**

2. **Bootear y configurar SSH**
   ```bash
   ssh-copy-id admin@IP_DEL_NODO
   ```

3. **Agregar al inventory**
   ```yaml
   nodes:
     hosts:
       rp2-node:
         ansible_host: 10.0.0.2
         ansible_user: admin
   ```

4. **Preparar nodo**
   ```bash
   ansible-playbook playbooks/prepare-node.yml --limit rp2-node
   # Anotar serial y MAC
   ```

5. **Configurar servidor**
   ```bash
   ansible-playbook playbooks/setup-netboot-server.yml \
     -e "node_name=rp2 node_serial=SERIAL node_mac=MAC"
   ```

6. **Copiar filesystem** (desde gateway)
   ```bash
   sudo rsync -axv admin@10.0.0.2:/ /srv/nfs/rp2/ \
     --exclude=/proc/* --exclude=/sys/* --exclude=/dev/* \
     --exclude=/tmp/* --exclude=/run/* --exclude=/boot/firmware/*
   ```

7. **Copiar boot files**
   ```bash
   sudo rsync -av admin@10.0.0.2:/boot/firmware/ /srv/tftp/SERIAL/
   ```

8. **Copiar shadow**
   ```bash
   sudo scp admin@10.0.0.2:/etc/shadow /srv/nfs/rp2/etc/shadow
   ```

9. **Apagar, quitar microSD, encender**

10. **Verificar**
    ```bash
    ansible rp2-node -m ping
    ```

### Fase 3: Kubernetes (30-60 min)

1. **Instalar k3s**
   ```bash
   ansible-playbook playbooks/k3s.yml
   ```

2. **Instalar MetalLB**
   ```bash
   ansible-playbook playbooks/metallb.yml
   ```

3. **Verificar cluster**
   ```bash
   kubectl get nodes -o wide
   kubectl get pods -n kube-system
   kubectl get svc -n kube-system traefik  # Debe tener EXTERNAL-IP
   ```

4. **Re-aplicar manifests de aplicaciones** (si los tienes en git)
   ```bash
   kubectl apply -f mis-apps/
   ```

### Fase 4: Verificación Final

```bash
# Todos los nodos responden
ansible all -m ping

# Servicios funcionan
systemctl status dnsmasq nfs-kernel-server k3s

# k3s funciona
kubectl get nodes -o wide
kubectl get pods -A
kubectl get svc -n kube-system traefik  # Debe tener EXTERNAL-IP 10.0.0.50

# VPN conecta
tailscale status
ping 10.0.0.1
```

---

## Backups Recomendados

### Qué hacer backup

| Dato | Ubicación | Frecuencia |
|------|-----------|------------|
| Configuración Ansible | Git repo | Cada cambio |
| Claves WireGuard | /etc/wireguard/ | Única vez |
| Filesystems NFS | /srv/nfs/* | Semanal |
| dnsmasq leases | /var/lib/misc/ | Opcional |
| k3s token | /var/lib/rancher/k3s/server/node-token | Única vez |
| kubeconfig | /etc/rancher/k3s/k3s.yaml | Cada reinstalación |
| Manifests k8s | Git repo (recomendado) | Cada cambio |

### Script de backup

```bash
#!/bin/bash
# backup.sh - ejecutar en gateway

BACKUP_DIR="/srv/backup/$(date +%Y%m%d)"
mkdir -p $BACKUP_DIR

# Configuración crítica
sudo tar czf $BACKUP_DIR/etc-wireguard.tar.gz /etc/wireguard/
sudo tar czf $BACKUP_DIR/etc-dnsmasq.tar.gz /etc/dnsmasq.conf
sudo tar czf $BACKUP_DIR/etc-exports.tar.gz /etc/exports

# Filesystems de nodos (puede tardar)
for node in rp2 rp3; do
  sudo tar czf $BACKUP_DIR/nfs-$node.tar.gz /srv/nfs/$node/
done

echo "Backup completado en $BACKUP_DIR"
```

### Verificar backups

```bash
# Listar backups
ls -la /srv/backup/

# Verificar integridad
tar tzf /srv/backup/20240101/nfs-rp2.tar.gz | head
```
