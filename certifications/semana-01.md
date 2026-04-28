# Semana 1 — Arquitectura interna de Kubernetes

**Dominio KCNA:** Kubernetes Fundamentals (46%)
**Estado:** Pendiente

---

## Objetivo de la semana

Entender qué hace cada componente de Kubernetes por dentro — no solo "cómo usarlo" sino qué pasa cuando corres un `kubectl apply`. Esta semana cubre el 20% del examen por sí sola.

---

## Conceptos

### Control Plane (corre en rp1-master)

| Componente | Función |
|-----------|---------|
| **kube-apiserver** | Punto de entrada de todas las operaciones. Todo pasa por aquí — kubectl, kubelet, otros componentes |
| **etcd** | Base de datos distribuida key-value. Guarda el estado deseado del cluster. Si se pierde etcd, se pierde el cluster |
| **kube-scheduler** | Decide en qué nodo corre cada Pod nuevo según recursos disponibles, afinidades y restricciones |
| **kube-controller-manager** | Ejecuta los controladores: ReplicaSet controller, Node controller, Job controller, etc. |
| **cloud-controller-manager** | Integra con APIs de cloud providers (AWS, GCP). En k3s/homelab: MetalLB lo reemplaza para LoadBalancer |

### Node components (corren en rp2, rp3)

| Componente | Función |
|-----------|---------|
| **kubelet** | Agente en cada nodo. Recibe instrucciones del API server y asegura que los pods corran |
| **kube-proxy** | Gestiona reglas de red (iptables/ipvs) para que los Services funcionen |
| **Container runtime** | Corre los contenedores. En k3s: containerd (no Docker) |

### Flujo de un `kubectl apply`

```
kubectl apply -f deployment.yml
      │
      ▼
  API Server  ←── autentica + valida el YAML
      │
      ▼
    etcd       ←── persiste el estado deseado
      │
      ▼
  Controller   ←── detecta que faltan pods, crea ReplicaSet
      │
      ▼
  Scheduler    ←── elige el nodo (rp2 o rp3) según recursos
      │
      ▼
  kubelet      ←── recibe la spec del pod, le dice a containerd que lo corra
      │
      ▼
  containerd   ←── descarga imagen, crea contenedor
```

### k3s vs k8s completo

k3s empaqueta todos los componentes del control plane en un solo binario. En lugar de etcd usa SQLite por defecto (o PostgreSQL externo). Los componentes son los mismos conceptualmente.

---

## Práctica en homelab

### Día 1 — Ver los componentes corriendo

```bash
# En k3s los componentes del control plane NO corren como pods — corren embebidos en el binario
# Pero puedes ver los pods del sistema
kubectl get pods -n kube-system -o wide

# Ver el proceso k3s en rp1
ssh admin@10.0.0.1 "ps aux | grep k3s"

# Ver los agentes en los workers
ssh -J admin@100.107.98.121 admin@10.0.0.2 "ps aux | grep k3s"
```

### Día 2 — Inspeccionar el API server

```bash
# La versión del API server
kubectl version

# Todos los recursos que el API server conoce
kubectl api-resources

# Ver los API groups disponibles
kubectl api-versions | sort
```

### Día 3 — Entender etcd a través del estado del cluster

```bash
# etcd guarda todo — cada objeto que ves es una entrada en etcd
kubectl get all -A | wc -l   # cuántos objetos hay en el cluster

# Ver el estado de un nodo como lo ve etcd (a través del API server)
kubectl get node rp2-node -o json
```

### Día 4 — El scheduler en acción

```bash
# Ver en qué nodo scheduló cada pod y por qué
kubectl get pods -A -o wide

# Forzar que un pod vaya a un nodo específico y ver cómo el scheduler respeta la restricción
kubectl run test-scheduler --image=busybox --overrides='{"spec":{"nodeName":"rp2-node"}}' -- sleep 60
kubectl get pod test-scheduler -o wide
kubectl delete pod test-scheduler
```

### Día 5 — El kubelet y containerd

```bash
# Ver los logs del kubelet (k3s-agent en workers)
ssh -J admin@100.107.98.121 admin@10.0.0.2 "journalctl -u k3s-agent --tail=30"

# Ver los contenedores que containerd está corriendo (no Docker)
ssh -J admin@100.107.98.121 admin@10.0.0.2 "sudo ctr containers list"
ssh -J admin@100.107.98.121 admin@10.0.0.2 "sudo crictl ps"
```

---

## Checklist diario

- [ ] Día 1 — Componentes del control plane: qué hace cada uno
- [ ] Día 2 — API server: inspeccionar recursos y API groups
- [ ] Día 3 — etcd: entender que todo el estado del cluster vive ahí
- [ ] Día 4 — Scheduler: observar cómo elige nodos
- [ ] Día 5 — kubelet + containerd: ver el agente en los workers
- [ ] Día 6 — Leer el capítulo 1-2 de *Kubernetes Up & Running*
- [ ] Día 7 — Responder preguntas de repaso (abajo)

---

## Preguntas de repaso

1. ¿Qué componente persiste el estado deseado del cluster?
2. Si el kube-scheduler cae, ¿los pods existentes siguen corriendo?
3. ¿Cuál es la diferencia entre kubelet y kube-proxy?
4. ¿Qué pasa si etcd pierde datos?
5. ¿Por qué k3s usa un solo binario en lugar de múltiples procesos?
6. ¿Qué componente es responsable de que un Deployment mantenga el número de réplicas correcto?
7. En tu cluster, ¿dónde corre el kube-scheduler?

---

## Respuestas

<details>
<summary>Ver respuestas</summary>

1. **etcd** — es la base de datos del cluster
2. **Sí** — el scheduler solo actúa al crear pods nuevos. Los pods existentes los gestiona el kubelet en cada nodo
3. **kubelet** corre pods (habla con containerd). **kube-proxy** gestiona reglas de red para Services
4. El cluster pierde todo su estado — los pods en los nodos siguen corriendo pero Kubernetes ya no puede gestionarlos
5. Para simplificar instalación y reducir consumo de recursos — ideal para edge y homelab
6. El **ReplicaSet controller** dentro del kube-controller-manager
7. En **rp1-master**, embebido en el proceso k3s server

</details>

---

[← Volver al roadmap](roadmap.md) | [Semana 2 →](semana-02.md)
