# KCNA Roadmap — Kubernetes and Cloud Native Associate

## Objetivo

Obtener la certificación KCNA dominando los conceptos de Kubernetes y el ecosistema Cloud Native, apoyado en el homelab (k3s en 3 Raspberry Pi 5) como entorno de práctica real.

**Examen:** 60 preguntas multiple choice · 90 minutos · Passing score: 75%

---

## Dominios del examen

| Dominio | Peso | Semanas |
|---------|------|---------|
| Kubernetes Fundamentals | 46% | 1, 2, 3, 4, 5 |
| Container Orchestration | 22% | 2 |
| Cloud Native Architecture | 16% | 7 |
| Cloud Native Observability | 8% | 6 |
| Cloud Native Application Delivery | 8% | 7 |

---

## Estado actual

- [x] Docker — experiencia sólida
- [x] k3s cluster (3 nodos Raspberry Pi 5)
- [x] Deployments, Services, Ingress, PVCs en producción
- [x] Prometheus + Grafana + Alertmanager
- [x] CI/CD con GitHub Actions (self-hosted runner)
- [x] MetalLB, Traefik, local-path storage
- [ ] RBAC y seguridad profunda
- [ ] StatefulSets, DaemonSets, Jobs
- [ ] NetworkPolicy
- [ ] Ecosistema CNCF completo
- [ ] Helm avanzado

---

## Semanas

| Semana | Tema | Archivo | Estado |
|--------|------|---------|--------|
| 1 | Arquitectura interna de k8s | [semana-01.md](semana-01.md) | Pendiente |
| 2 | Workloads: ReplicaSet, DaemonSet, Job, CronJob | [semana-02.md](semana-02.md) | Pendiente |
| 3 | Networking profundo | [semana-03.md](semana-03.md) | Pendiente |
| 4 | Storage y Configuration | [semana-04.md](semana-04.md) | Pendiente |
| 5 | Seguridad: RBAC y ServiceAccounts | [semana-05.md](semana-05.md) | Pendiente |
| 6 | Observabilidad | [semana-06.md](semana-06.md) | Pendiente |
| 7 | Cloud Native Architecture y Application Delivery | [semana-07.md](semana-07.md) | Pendiente |
| 8 | Repaso + simulacros | [semana-08.md](semana-08.md) | Pendiente |

---

## Cómo usar este roadmap

Cada archivo de semana tiene:
- **Conceptos** — qué estudiar y por qué importa para el examen
- **Práctica en homelab** — comandos concretos a correr en el cluster
- **Checklist diario** — progreso día a día
- **Preguntas de repaso** — simulación del estilo del examen

Avanza a tu ritmo. Marca cada ítem cuando lo completes.

---

## Recursos

- [Documentación oficial Kubernetes](https://kubernetes.io/docs/)
- [CNCF Landscape](https://landscape.cncf.io/)
- Libro: *Kubernetes Up & Running* — O'Reilly
- Canal: TechWorld with Nana (YouTube)
- Simulador: killer.sh (incluido con el voucher del examen)
