# ADR 011: MetalLB para LoadBalancer en Bare-Metal

## Estado

Aceptado

## Fecha

2026-01-31

## Contexto

En Kubernetes, los Services tipo `LoadBalancer` están diseñados para integrarse con cloud providers (AWS, GCP, Azure) que proveen balanceadores de carga externos automáticamente. En un cluster bare-metal/homelab como el nuestro, estos services quedan en estado `pending` indefinidamente porque no hay nada que les asigne una IP externa.

Teníamos dos opciones para exponer servicios:

1. **NodePort**: Expone un puerto alto (30000-32767) en todos los nodos
2. **LoadBalancer con MetalLB**: Asigna IPs reales de nuestra red

### Arquitectura previa
```
Cliente → Traefik (Docker) :80 → Traefik (k3s) :31718 → Service → Pod
              10.0.0.1              NodePort
```

Requería:
- Configurar Traefik Docker como proxy hacia k3s
- Mantener archivos de configuración por cada app
- Doble salto de red

## Decisión

Instalar MetalLB en modo Layer 2 (L2) para proveer LoadBalancer nativo en el cluster k3s.

### Configuración

**Pool de IPs:** `10.0.0.50-60`
- Fuera del rango DHCP (10.0.0.100-200)
- Fuera de las IPs fijas de nodos (10.0.0.1-3)

**Modo:** L2 (ARP)
- MetalLB responde peticiones ARP para las IPs del pool
- Simple, funciona en cualquier red
- No requiere configuración en router/switch

### Arquitectura nueva
```
Cliente → DNS (dnsmasq) → MetalLB (10.0.0.50) → Traefik (k3s) → Service → Pod
```

## Consecuencias

### Positivas

- **Simplificación**: Un solo punto de entrada para apps de k8s
- **IPs reales**: Services obtienen IPs accesibles desde la LAN
- **Estándar**: Comportamiento similar a cloud providers
- **Separación clara**: `*.homelab.local` (Docker) vs `*.k8s.homelab.local` (k8s)

### Negativas

- **Componente adicional**: Más pods corriendo en el cluster
- **Limitación L2**: Todo el tráfico pasa por un solo nodo (el que "posee" la IP)
- **Configuración DNS**: Requiere actualizar dnsmasq para los nuevos subdominios

### Neutrales

- Traefik Docker sigue siendo útil para servicios que corren fuera de k8s

## Alternativas consideradas

### 1. Solo NodePort
```
Cliente → 10.0.0.x:31718 → Traefik (k3s) → Service → Pod
```

**Rechazado porque:**
- Puertos no estándar (31718 en vez de 80)
- Difícil de recordar
- No profesional para exponer servicios

### 2. Traefik Docker como único entry point
```
Cliente → Traefik (Docker) → k3s NodePort → Service → Pod
```

**Rechazado porque:**
- Doble salto de red
- Configuración duplicada (Ingress en k8s + file en Traefik Docker)
- Más complejo de mantener

### 3. HostPort en Traefik k3s

Hacer que Traefik k3s escuche directamente en puerto 80/443 del nodo.

**Rechazado porque:**
- Conflicto con Traefik Docker en rp1
- Menos flexible que MetalLB

## Implementación
```bash
# Playbook de instalación
ansible-playbook playbooks/metallb.yml
```

### Verificación
```bash
# Ver IP asignada a Traefik
kubectl get svc -n kube-system traefik
# EXTERNAL-IP: 10.0.0.50

# Probar acceso
curl http://nginx.k8s.homelab.local
```

## Referencias

- [MetalLB Documentation](https://metallb.universe.tf/)
- [MetalLB Layer 2 Mode](https://metallb.universe.tf/concepts/layer2/)
- [Kubernetes LoadBalancer Services](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer)
