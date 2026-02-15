# Comandos de Referencia Rápida

## Ansible

```bash
# Verificar conectividad
ansible all -m ping

# Ejecutar playbook
ansible-playbook playbooks/<playbook>.yml

# Dry-run (verificar sin aplicar)
ansible-playbook playbooks/<playbook>.yml --check

# Limitar a un host
ansible-playbook playbooks/<playbook>.yml --limit rp2-node

# Ver más detalle
ansible-playbook playbooks/<playbook>.yml -v

# Ver inventario
ansible-inventory --list
```

## kubectl (k3s)

### Cluster
```bash
# Estado de nodos
kubectl get nodes -o wide

# Pods del sistema
kubectl get pods -n kube-system

# Todos los recursos en un namespace
kubectl get all -n <namespace>

# Ver eventos recientes
kubectl get events --sort-by=.metadata.creationTimestamp
```

### Pods
```bash
# Listar pods
kubectl get pods -A

# Ver logs de un pod
kubectl logs <pod-name> -n <namespace>
kubectl logs <pod-name> -n <namespace> -f    # seguir logs

# Describir pod (ver eventos, estado)
kubectl describe pod <pod-name> -n <namespace>

# Shell en un pod
kubectl exec -it <pod-name> -n <namespace> -- /bin/sh
```

### Services e Ingress
```bash
# Ver services con LoadBalancer
kubectl get svc -A | grep LoadBalancer

# Ver todos los ingress
kubectl get ingress -A

# Ver detalle de ingress
kubectl describe ingress <name> -n <namespace>
```

### Storage
```bash
# Ver PersistentVolumeClaims
kubectl get pvc -A

# Ver PersistentVolumes
kubectl get pv

# Ver StorageClasses
kubectl get sc
```

### Deployments
```bash
# Ver deployments
kubectl get deployments -A

# Escalar
kubectl scale deployment <name> -n <namespace> --replicas=2

# Reiniciar deployment
kubectl rollout restart deployment <name> -n <namespace>

# Ver estado del rollout
kubectl rollout status deployment <name> -n <namespace>
```

### Networking (Flannel)
```bash
# Ver IPs de Flannel por nodo
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.flannel\.alpha\.coreos\.com/public-ip}{"\n"}{end}'

# Ver FDB de Flannel (desde un nodo)
bridge fdb show dev flannel.1
```

### Troubleshooting
```bash
# Pod atascado en Pending
kubectl describe pod <pod> -n <ns>    # ver Events

# DNS dentro del cluster
kubectl run tmp-shell --rm -i --tty --image nicolaka/netshoot -- /bin/bash
nslookup kubernetes.default
nslookup <service>.<namespace>.svc.cluster.local

# Ver logs de Traefik (ingress)
kubectl logs -l app.kubernetes.io/name=traefik -n kube-system

# Ver config de MetalLB
kubectl get ipaddresspool -n metallb-system
kubectl get l2advertisement -n metallb-system
```

## k3s

```bash
# Estado del servicio (master)
sudo systemctl status k3s

# Estado del agente (workers)
sudo systemctl status k3s-agent

# Reiniciar k3s
sudo systemctl restart k3s          # master
sudo systemctl restart k3s-agent    # workers

# Ver logs
sudo journalctl -u k3s -f
sudo journalctl -u k3s-agent -f

# Token del cluster (para agregar nodos)
sudo cat /var/lib/rancher/k3s/server/node-token
```

## Servicios del Sistema

```bash
# dnsmasq
sudo systemctl status dnsmasq
sudo tail -f /var/log/dnsmasq.log
sudo cat /var/lib/misc/dnsmasq.leases    # ver leases DHCP

# NFS
sudo exportfs -v                          # ver exports
sudo showmount -e localhost               # listar shares

# Tailscale
sudo tailscale status
sudo tailscale ip                         # ver IP de Tailscale

# WireGuard
sudo wg show

# UFW (firewall)
sudo ufw status verbose
sudo ufw status numbered
```

## Mantenimiento de Nodos

```bash
# Drenar nodo antes de mantenimiento
kubectl drain <node> --ignore-daemonsets --delete-emptydir-data

# Restaurar nodo
kubectl uncordon <node>

# Ver labels de un nodo
kubectl get node <node> --show-labels

# Agregar label
kubectl label nodes <node> <key>=<value>
```

## Aplicar manifiestos k8s

```bash
# Aplicar un archivo
kubectl apply -f <file>.yml

# Aplicar un directorio completo (en orden alfabético)
kubectl apply -f k8s-apps/<app>/

# Eliminar recursos de un archivo
kubectl delete -f <file>.yml
```
