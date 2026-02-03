# Guía de Ansible para Homelab

Guía práctica de Ansible basada en lo aprendido configurando el homelab.

## Conceptos Básicos

### ¿Qué es Ansible?

Ansible es una herramienta de automatización que ejecuta tareas en servidores remotos via SSH. No requiere instalar agentes en los nodos.
```
Tu Mac (control node)
    │
    │ SSH
    ▼
┌─────────────────┐
│ rp1-master      │
│ rp2-node        │  ← managed nodes
│ rp3-node        │
└─────────────────┘
```

### Estructura de un Proyecto Ansible
```
homelab-ansible/
├── ansible.cfg           # Configuración de Ansible
├── inventory/
│   └── inventory.yml     # Lista de hosts y grupos
├── playbooks/
│   ├── gateway.yml       # Playbook para configurar gateway
│   ├── update-nodes.yml  # Playbook para actualizar nodos
│   └── node-info.yml     # Playbook para ver información
├── roles/
│   ├── dnsmasq/          # Role para dnsmasq
│   ├── nfs/              # Role para NFS
│   └── wireguard/        # Role para WireGuard
└── docs/
    └── ansible-guide.md  # Esta guía
```

## Inventory

El inventory define qué hosts gestiona Ansible y cómo agruparlos.

### Ejemplo: inventory/inventory.yml
```yaml
all:
  children:
    gateway:
      hosts:
        rp1-master:
          ansible_host: 10.0.0.1
          ansible_user: admin
    nodes:
      hosts:
        rp2-node:
          ansible_host: 10.0.0.2
          ansible_user: admin
        rp3-node:
          ansible_host: 10.0.0.3
          ansible_user: admin
```

### Grupos

- `all` - Todos los hosts
- `gateway` - Solo el gateway
- `nodes` - Solo los nodos workers

### Comandos útiles
```bash
# Listar todos los hosts
ansible all --list-hosts

# Listar hosts de un grupo
ansible nodes --list-hosts

# Ping a todos los hosts
ansible all -m ping

# Ping solo a nodos
ansible nodes -m ping
```

## Playbooks

Un playbook es un archivo YAML que define tareas a ejecutar.

### Estructura básica
```yaml
---
# Comentario describiendo el playbook
- name: Nombre del play
  hosts: all              # En qué hosts ejecutar
  become: yes             # Usar sudo
  serial: 1               # Ejecutar en N hosts a la vez

  vars:
    mi_variable: "valor"

  tasks:
    - name: Nombre de la tarea
      modulo:
        parametro: valor
```

### Parámetros comunes del play

| Parámetro | Descripción | Ejemplo |
|-----------|-------------|---------|
| `hosts` | Hosts o grupos donde ejecutar | `all`, `nodes`, `rp2-node` |
| `become` | Ejecutar como root (sudo) | `yes`, `no` |
| `serial` | Hosts a procesar en paralelo | `1`, `2`, `50%` |
| `gather_facts` | Recopilar info del sistema | `yes`, `no` |

### Ejecutar playbooks
```bash
# Ejecutar playbook
ansible-playbook playbooks/mi-playbook.yml

# Limitar a un host
ansible-playbook playbooks/mi-playbook.yml --limit rp2-node

# Limitar a un grupo
ansible-playbook playbooks/mi-playbook.yml --limit nodes

# Modo check (dry-run, no hace cambios)
ansible-playbook playbooks/mi-playbook.yml --check

# Paso a paso (confirmar cada tarea)
ansible-playbook playbooks/mi-playbook.yml --step

# Pasar variables
ansible-playbook playbooks/mi-playbook.yml -e "variable=valor"
```

## Módulos Comunes

### command - Ejecutar comando simple
```yaml
- name: Obtener kernel
  command: uname -r
  register: kernel_result
  changed_when: false
```

**Nota:** No soporta pipes (`|`) ni redirecciones (`>`).

### shell - Ejecutar comando con shell
```yaml
- name: Obtener uso de memoria
  shell: free -h | grep Mem | awk '{print $3}'
  register: memory_result
  changed_when: false
```

**Nota:** Soporta pipes, redirecciones y variables de entorno.

### apt - Gestionar paquetes
```yaml
# Instalar paquete
- name: Instalar nginx
  apt:
    name: nginx
    state: present

# Actualizar cache e instalar
- name: Instalar con cache actualizado
  apt:
    name: htop
    state: present
    update_cache: yes

# Actualizar todos los paquetes
- name: Actualizar sistema
  apt:
    upgrade: safe
    autoremove: yes

# Eliminar paquete
- name: Eliminar paquete
  apt:
    name: snapd
    state: absent
    purge: yes
```

### copy - Copiar archivos
```yaml
# Copiar contenido directamente
- name: Crear archivo de configuración
  copy:
    content: |
      linea 1
      linea 2
    dest: /etc/mi-config.conf
    mode: "0644"

# Copiar archivo local al remoto
- name: Copiar script
  copy:
    src: files/mi-script.sh
    dest: /usr/local/bin/mi-script.sh
    mode: "0755"
```

### template - Copiar con variables
```yaml
- name: Crear configuración desde template
  template:
    src: templates/config.j2
    dest: /etc/mi-app/config.conf
    mode: "0644"
```

### file - Gestionar archivos/directorios
```yaml
# Crear directorio
- name: Crear directorio
  file:
    path: /srv/data
    state: directory
    mode: "0755"

# Crear symlink
- name: Crear enlace simbólico
  file:
    src: /srv/data
    dest: /data
    state: link

# Cambiar permisos
- name: Cambiar permisos
  file:
    path: /etc/mi-config.conf
    mode: "0600"
    owner: root
    group: root
```

### service / systemd - Gestionar servicios
```yaml
# Iniciar y habilitar servicio
- name: Iniciar nginx
  service:
    name: nginx
    state: started
    enabled: yes

# Reiniciar servicio
- name: Reiniciar dnsmasq
  systemd:
    name: dnsmasq
    state: restarted
    daemon_reload: yes
```

### reboot - Reiniciar sistema
```yaml
- name: Reiniciar nodo
  reboot:
    msg: "Reinicio por mantenimiento"
    reboot_timeout: 300
    pre_reboot_delay: 5
    post_reboot_delay: 30
```

### wait_for_connection - Esperar conexión
```yaml
- name: Esperar a que el nodo esté disponible
  wait_for_connection:
    delay: 10
    timeout: 120
```

### debug - Mostrar información
```yaml
# Mostrar mensaje simple
- name: Mostrar variable
  debug:
    msg: "El kernel es {{ kernel_result.stdout }}"

# Mostrar lista formateada
- name: Mostrar información
  vars:
    info_lines:
      - "Hostname: {{ ansible_hostname }}"
      - "IP: {{ ansible_host }}"
  debug:
    msg: "{{ info_lines }}"
```

### pause - Pausar ejecución
```yaml
# Pausa con confirmación
- name: Confirmar acción
  pause:
    prompt: "¿Continuar? (yes/no)"
  register: confirmacion

# Pausa por tiempo
- name: Esperar 30 segundos
  pause:
    seconds: 30
```

### stat - Verificar si archivo existe
```yaml
- name: Verificar si existe archivo
  stat:
    path: /var/run/reboot-required
  register: archivo_existe

- name: Hacer algo si existe
  debug:
    msg: "El archivo existe"
  when: archivo_existe.stat.exists
```

## Variables y Facts

### gather_facts

Cuando `gather_facts: yes`, Ansible recopila información del sistema:
```yaml
ansible_hostname          # rp2-node
ansible_distribution      # Ubuntu
ansible_distribution_version  # 25.10
ansible_processor_vcpus   # 4
ansible_memtotal_mb       # 8192
ansible_default_ipv4.address  # 10.0.0.2
```

Ver todos los facts disponibles:
```bash
ansible rp2-node -m setup
ansible rp2-node -m setup -a "filter=ansible_distribution*"
```

### register

Guarda el resultado de una tarea:
```yaml
- name: Obtener kernel
  command: uname -r
  register: resultado

# resultado contiene:
# {
#   "stdout": "6.17.0-1006-raspi",
#   "stderr": "",
#   "rc": 0,
#   "changed": true,
#   "stdout_lines": ["6.17.0-1006-raspi"]
# }

- name: Usar resultado
  debug:
    msg: "Kernel: {{ resultado.stdout }}"
```

### vars - Variables locales
```yaml
# En el play
vars:
  mi_variable: "valor"
  lista:
    - item1
    - item2

# En una tarea
- name: Tarea con variables locales
  vars:
    variable_local: "solo para esta tarea"
  debug:
    msg: "{{ variable_local }}"
```

### Pasar variables en línea de comandos
```bash
ansible-playbook playbook.yml -e "variable=valor"
ansible-playbook playbook.yml -e "var1=uno var2=dos"
```

## Condicionales (when)

### Sintaxis básica
```yaml
# Comparación simple
when: ansible_distribution == "Ubuntu"

# Negación
when: not archivo.stat.exists

# AND (lista = AND implícito)
when:
  - condicion1
  - condicion2

# AND explícito
when: condicion1 and condicion2

# OR
when: condicion1 or condicion2

# Verificar si string contiene algo
when: "'gateway' in group_names"
when: "'error' in resultado.stderr"

# Variable definida
when: mi_var is defined
when: mi_var is not defined

# Resultado de registro
when: resultado.rc == 0
when: resultado.stdout == "active"
```

### Filtro default
```yaml
# Si user_input no existe, usa 'yes'
when: confirmacion.user_input | default('yes') != 'yes'
```

## Control de Flujo

### changed_when

Controla cuándo una tarea se marca como "changed":
```yaml
# Nunca marcar como changed (solo lectura)
- name: Obtener información
  command: uname -r
  register: kernel
  changed_when: false

# Marcar changed según condición
- name: Ejecutar script
  command: /opt/script.sh
  register: script_result
  changed_when: "'updated' in script_result.stdout"
```

### failed_when

Controla cuándo una tarea se considera fallida:
```yaml
- name: Verificar servicio
  command: systemctl status mi-servicio
  register: status
  failed_when: status.rc not in [0, 3]
```

### ignore_errors

Continuar aunque falle:
```yaml
- name: Tarea que puede fallar
  command: /comando/que/puede/fallar
  ignore_errors: yes
```

### meta: end_host

Terminar ejecución para el host actual (sin fallar):
```yaml
- name: Saltar si no se confirma
  meta: end_host
  when: confirmacion.user_input != 'yes'
```

## Tags

Permite ejecutar solo partes del playbook:
```yaml
tasks:
  - name: Obtener información
    command: uname -r
    tags: info

  - name: Reiniciar
    reboot:
    tags: reboot

  - name: Verificar
    command: systemctl status sshd
    tags:
      - verify
      - info
```
```bash
# Solo tasks con tag "info"
ansible-playbook playbook.yml --tags info

# Todo excepto "reboot"
ansible-playbook playbook.yml --skip-tags reboot

# Múltiples tags
ansible-playbook playbook.yml --tags "info,verify"

# Listar tags disponibles
ansible-playbook playbook.yml --list-tags
```

## Roles

Un role es una forma de organizar playbooks reutilizables.

### Estructura de un role
```
roles/
└── mi_role/
    ├── tasks/
    │   └── main.yml      # Tareas principales
    ├── handlers/
    │   └── main.yml      # Handlers
    ├── templates/
    │   └── config.j2     # Templates Jinja2
    ├── files/
    │   └── script.sh     # Archivos estáticos
    ├── vars/
    │   └── main.yml      # Variables del role
    └── defaults/
        └── main.yml      # Variables por defecto
```

### Usar un role
```yaml
- name: Configurar servidor
  hosts: all
  become: yes

  roles:
    - mi_role
    - otro_role
```

### Handlers

Los handlers se ejecutan al final del play si fueron notificados:
```yaml
# tasks/main.yml
- name: Copiar configuración
  template:
    src: config.j2
    dest: /etc/mi-app/config.conf
  notify: Reiniciar mi-app

# handlers/main.yml
- name: Reiniciar mi-app
  service:
    name: mi-app
    state: restarted
```

## Jinja2 Filters

Ansible usa Jinja2 para templates y expresiones:
```yaml
# default - valor por defecto si no existe
{{ variable | default('valor_defecto') }}

# upper/lower - mayúsculas/minúsculas
{{ hostname | upper }}

# replace - reemplazar texto
{{ mac_address | replace(':', '-') }}

# indent - indentar texto
{{ texto_multilinea | indent(4) }}

# to_yaml/to_json - convertir formato
{{ mi_dict | to_yaml }}

# first/last - primer/último elemento
{{ mi_lista | first }}

# length - longitud
{{ mi_lista | length }}

# ternario (if/else en una línea)
{{ 'Sí' if condicion else 'No' }}
```

## Comandos Útiles
```bash
# Ver documentación de un módulo
ansible-doc apt
ansible-doc copy
ansible-doc -l  # Listar todos los módulos

# Ejecutar comando ad-hoc
ansible all -m command -a "uptime"
ansible nodes -m shell -a "df -h /"

# Verificar sintaxis
ansible-playbook playbook.yml --syntax-check

# Ver qué hosts serían afectados
ansible-playbook playbook.yml --list-hosts

# Ver qué tareas se ejecutarían
ansible-playbook playbook.yml --list-tasks

# Modo verbose
ansible-playbook playbook.yml -v    # básico
ansible-playbook playbook.yml -vv   # más detalle
ansible-playbook playbook.yml -vvv  # debug
```

## Playbooks del Proyecto

### Orden de ejecución recomendado (setup inicial)

Para configurar el homelab desde cero, ejecutar los playbooks en este orden:

```
1. common.yml           → Config base en todos los nodos
2. setup-ssh.yml        → Distribuir claves SSH del gateway a nodos
3. gateway.yml          → Roles: wireguard, dnsmasq, nfs
4. firewall.yml         → UFW en gateway y nodos
5. tailscale.yml        → VPN mesh (requiere auth manual)
6. docker.yml           → Docker en todos los nodos
7. local-storage.yml    → Montar discos locales
8. k3s.yml              → Cluster Kubernetes
9. metallb.yml          → LoadBalancer para k3s
10. node-exporter.yml   → Métricas Prometheus
```

### Playbooks de operación (uso recurrente)

| Playbook | Uso | Ejemplo |
|----------|-----|---------|
| `update-nodes.yml` | Actualizar paquetes | `ansible-playbook playbooks/update-nodes.yml` |
| `update-kernel.yml` | Actualizar kernel en TFTP | `ansible-playbook playbooks/update-kernel.yml` |
| `node-info.yml` | Ver info de todos los nodos | `ansible-playbook playbooks/node-info.yml` |
| `reboot-nodes.yml` | Reinicio controlado | `ansible-playbook playbooks/reboot-nodes.yml` |

### Playbooks de setup adicional

| Playbook | Uso | Notas |
|----------|-----|-------|
| `setup-netboot-server.yml` | Preparar NFS/TFTP para nuevo nodo | Requiere `-e "node_name=rp4 node_serial=SERIAL node_mac=MAC"` |
| `prepare-node.yml` | Preparar nodo para netboot | Configura EEPROM, elimina snapd |
| `duckdns.yml` | DNS dinámico | Requiere `-e "duckdns_token=TOKEN"` |
| `registry.yml` | Registry privado Docker/k3s | Configura insecure-registries |
| `install-basic-tools-nodes.yml` | Herramientas extra en nodos | speedtest-cli, htop, git |

### Inventory actual

```yaml
all:
  children:
    gateway:
      hosts:
        rp1-master:
          ansible_host: 192.168.100.18  # WAN IP
          ansible_user: admin
          ansible_python_interpreter: /usr/bin/python3
    nodes:
      hosts:
        rp2-node:
          ansible_host: 10.0.0.2
          ansible_user: admin
        rp3-node:
          ansible_host: 10.0.0.3
          ansible_user: admin
```

**Grupos disponibles:**
- `all` - Los 3 nodos
- `gateway` - Solo rp1-master
- `nodes` - Solo rp2-node y rp3-node

### Roles del proyecto

| Role | Playbook que lo usa | Función |
|------|-------------------|---------|
| `dnsmasq` | `gateway.yml` | DHCP, DNS (.homelab.local), TFTP |
| `nfs` | `gateway.yml` | NFS server para netboot |
| `wireguard` | `gateway.yml` | VPN server (legacy, reemplazado por Tailscale) |

## Buenas Prácticas

1. **Usa `changed_when: false`** para tareas de solo lectura
2. **Usa `serial: 1`** cuando reinicies o actualices para no perder conectividad
3. **Agrupa hosts lógicamente** en el inventory
4. **Usa roles** para código reutilizable
5. **Documenta** tus playbooks con comentarios
6. **Usa `--check`** antes de ejecutar en producción
7. **Versiona** tu código con git
