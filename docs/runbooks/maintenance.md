# Mantenimiento Rutinario

Procedimientos de mantenimiento periódico para el homelab.

## Checklist Semanal

### 1. Verificar estado de servicios

```bash
# Desde tu Mac (via VPN)
ansible gateway -m shell -a "systemctl status dnsmasq nfs-kernel-server wg-quick@wg0 --no-pager"
```

### 2. Verificar espacio en disco

```bash
# Gateway
ansible gateway -m shell -a "df -h /srv"

# Nodos
ansible nodes -m shell -a "df -h /"
```

### 3. Verificar conectividad de nodos

```bash
ansible all -m ping
```

### 4. Verificar estado de k3s

```bash
# Nodos del cluster
kubectl get nodes -o wide

# Pods del sistema (todos deben estar Running)
kubectl get pods -n kube-system

# MetalLB (speaker en cada nodo, controller activo)
kubectl get pods -n metallb-system

# Verificar que Traefik tiene IP de LoadBalancer
kubectl get svc -n kube-system traefik
```

### 5. Revisar logs de errores

```bash
# En gateway
sudo journalctl -p err --since "1 week ago" --no-pager | head -50

# Errores de k3s
sudo journalctl -u k3s -p err --since "1 week ago" --no-pager | head -20
```

## Checklist Mensual

### 1. Actualizar paquetes

```bash
# Actualizar todos los nodos (uno a la vez)
cd homelab-ansible
ansible-playbook playbooks/update-nodes.yml

# Si hay actualizaciones de kernel, reiniciar
ansible-playbook playbooks/update-nodes.yml -e "reboot=true"
```

### 2. Actualizar gateway

```bash
# En gateway (manualmente, con cuidado)
sudo apt update
sudo apt upgrade

# Si hay kernel nuevo
sudo reboot
```

### 3. Verificar integridad de NFS

```bash
# En gateway
sudo touch /srv/nfs/rp2/test-write
sudo rm /srv/nfs/rp2/test-write

sudo touch /srv/nfs/rp3/test-write
sudo rm /srv/nfs/rp3/test-write
```

### 4. Mantenimiento de k3s

```bash
# Limpiar imágenes no usadas en containerd (en cada nodo)
ansible all -m shell -a "sudo k3s crictl rmi --prune" -b

# Verificar espacio en disco local de k3s
ansible all -m shell -a "df -h /var/lib/rancher-local" -b

# Verificar que Flannel anuncia IPs correctas (deben ser 10.0.0.x)
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'

# Verificar eventos recientes (errores o warnings)
kubectl get events -A --sort-by='.lastTimestamp' | tail -20
```

### 5. Limpiar logs antiguos

```bash
# En gateway
sudo journalctl --vacuum-time=30d
```

### 6. Crear backup

```bash
# En gateway
BACKUP_DIR="/srv/backup/$(date +%Y%m)"
sudo mkdir -p $BACKUP_DIR

# Configuración
sudo tar czf $BACKUP_DIR/config.tar.gz \
  /etc/wireguard/ \
  /etc/dnsmasq.conf \
  /etc/exports \
  /etc/netplan/

echo "Backup en $BACKUP_DIR"
```

## Checklist Trimestral

### 1. Revisar seguridad

```bash
# Verificar usuarios con acceso
cat /etc/passwd | grep -v nologin | grep -v false

# Verificar llaves SSH autorizadas
cat ~/.ssh/authorized_keys

# Verificar sudoers
sudo ls -la /etc/sudoers.d/
```

### 2. Verificar claves WireGuard

```bash
# Ver fecha de creación de claves
ls -la /etc/wireguard/*.key

# Considerar rotación si tienen más de 1 año
```

### 3. Revisar espacio de backup

```bash
du -sh /srv/backup/*
# Eliminar backups antiguos si es necesario
```

### 4. Probar disaster recovery

```bash
# Verificar que puedes acceder al repo
git -C ~/aren-house pull

# Verificar que Ansible funciona
cd ~/aren-house/homelab-ansible
ansible all -m ping
ansible-playbook playbooks/gateway.yml --check
```

## Procedimientos de Mantenimiento

### Reiniciar servicio sin afectar nodos

#### dnsmasq
```bash
# Recargar configuración (sin reiniciar)
sudo systemctl reload dnsmasq

# Si necesitas reiniciar
sudo systemctl restart dnsmasq
# Los nodos mantienen su lease DHCP, no se afectan
```

#### NFS
```bash
# Recargar exports
sudo exportfs -ra

# Reiniciar NFS es más delicado
# Los nodos pueden tener errores I/O temporales
sudo systemctl restart nfs-kernel-server
```

#### WireGuard
```bash
# Agregar peer sin reiniciar
sudo wg set wg0 peer PUBLIC_KEY allowed-ips IP/32

# O sincronizar desde archivo
sudo wg syncconf wg0 <(sudo wg-quick strip wg0)
```

### Reiniciar gateway de forma segura

1. **Avisar a usuarios**
   ```bash
   wall "Gateway se reiniciará en 5 minutos"
   ```

2. **Verificar que no hay operaciones críticas**
   ```bash
   # Ver conexiones NFS activas
   ss -tn | grep 2049

   # Ver pods corriendo en k3s
   kubectl get pods -A --field-selector status.phase=Running
   ```

3. **Reiniciar**
   ```bash
   sudo reboot
   ```

4. **Verificar después del reinicio**
   ```bash
   # Servicios base
   systemctl status dnsmasq nfs-kernel-server wg-quick@wg0

   # k3s server
   sudo systemctl status k3s
   kubectl get nodes -o wide
   kubectl get pods -n kube-system

   # Nodos
   ansible all -m ping
   ```

### Mantenimiento de nodos k3s (drain/uncordon)

```bash
# 1. Marcar nodo como no programable
kubectl cordon rp2-node

# 2. Drenar pods (los mueve a otros nodos)
kubectl drain rp2-node --ignore-daemonsets --delete-emptydir-data

# 3. Realizar mantenimiento
ssh rp2-node "sudo apt update && sudo apt upgrade -y"
ssh rp2-node "sudo reboot"

# 4. Esperar a que vuelva y reactivar
kubectl uncordon rp2-node

# 5. Verificar
kubectl get nodes
kubectl get pods -o wide
```

### Actualizar kernel en nodos netboot

1. **Actualizar paquetes en el nodo**
   ```bash
   ansible-playbook playbooks/update-nodes.yml --limit rp2
   ```

2. **Copiar nuevos archivos de boot a TFTP** (desde gateway)
   ```bash
   sudo rsync -av admin@10.0.0.2:/boot/firmware/ /srv/tftp/440dc91d/
   ```

3. **Reiniciar nodo**
   ```bash
   ansible rp2-node -m reboot
   ```

4. **Verificar nuevo kernel**
   ```bash
   ansible rp2-node -m shell -a "uname -r"
   ```

### Agregar nuevo peer WireGuard

1. **Generar claves en el cliente**
   ```bash
   wg genkey | tee privatekey | wg pubkey > publickey
   ```

2. **Agregar peer en gateway.yml**
   ```yaml
   wireguard_peers:
     - name: nuevo-dispositivo
       public_key: "LLAVE_PUBLICA_DEL_CLIENTE"
       allowed_ips: "10.0.1.3/32"
   ```

3. **Aplicar cambios**
   ```bash
   ansible-playbook playbooks/gateway.yml --tags wireguard
   ```

4. **Configurar cliente**
   ```ini
   [Interface]
   PrivateKey = LLAVE_PRIVADA
   Address = 10.0.1.3/24

   [Peer]
   PublicKey = VIFt08+ZU2nQCnhXAOAMMS+ycH8d6PGLY+hcqZbXhAw=
   Endpoint = 192.168.100.x:51820
   AllowedIPs = 10.0.0.0/24, 10.0.1.0/24
   ```

### Expandir almacenamiento

1. **Conectar nuevo disco**
   ```bash
   sudo fdisk -l
   ```

2. **Particionar y formatear**
   ```bash
   sudo fdisk /dev/sdb
   sudo mkfs.ext4 /dev/sdb1
   ```

3. **Montar**
   ```bash
   sudo mkdir /srv/backup2
   sudo mount /dev/sdb1 /srv/backup2

   # Agregar a fstab
   echo "UUID=$(blkid -s UUID -o value /dev/sdb1) /srv/backup2 ext4 defaults,nofail 0 2" | sudo tee -a /etc/fstab
   ```

## Monitoreo (Futuro)

### Métricas a monitorear

| Métrica | Umbral alerta | Acción |
|---------|---------------|--------|
| Espacio disco /srv | < 20% libre | Limpiar o expandir |
| Memoria gateway | > 80% uso | Investigar procesos |
| Carga CPU | > 2.0 (5 min) | Investigar procesos |
| Servicios caídos | Cualquiera | Reiniciar servicio |
| Nodos sin ping | > 1 minuto | Verificar nodo |

### Script de monitoreo básico

```bash
#!/bin/bash
# monitor.sh - ejecutar con cron cada 5 min

# Verificar servicios
for svc in dnsmasq nfs-kernel-server wg-quick@wg0; do
  if ! systemctl is-active --quiet $svc; then
    echo "ALERTA: $svc no está activo" | wall
  fi
done

# Verificar espacio
SPACE=$(df /srv --output=pcent | tail -1 | tr -d ' %')
if [ $SPACE -gt 80 ]; then
  echo "ALERTA: /srv está al ${SPACE}%" | wall
fi

# Verificar nodos
for node in 10.0.0.2 10.0.0.3; do
  if ! ping -c 1 -W 2 $node > /dev/null 2>&1; then
    echo "ALERTA: Nodo $node no responde" | wall
  fi
done
```

### Configurar cron

```bash
# En gateway
sudo crontab -e

# Agregar:
*/5 * * * * /home/admin/monitor.sh
```
