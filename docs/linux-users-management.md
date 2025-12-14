# Gestión de Usuarios y Grupos en Linux

Guía sobre usuarios, grupos, permisos y su relación con Ansible y NFS.

## Conceptos fundamentales

### Tipos de usuarios

| Tipo | UID | Propósito | Puede hacer login |
|------|-----|-----------|-------------------|
| root | 0 | Administrador total | Sí (no recomendado) |
| Sistema | 1-999 | Ejecutar servicios | No |
| Normales | 1000+ | Personas reales | Sí |

### Archivos principales

| Archivo | Contenido | Permisos |
|---------|-----------|----------|
| `/etc/passwd` | Lista de usuarios | Lectura pública |
| `/etc/shadow` | Contraseñas encriptadas | Solo root |
| `/etc/group` | Lista de grupos | Lectura pública |
| `/etc/sudoers` | Permisos de sudo | Solo root |

### Anatomía de /etc/passwd
```
rp2-node:x:1000:1003:Descripcion:/home/rp2-node:/bin/bash
│        │ │    │    │           │              │
│        │ │    │    │           │              └─ Shell de login
│        │ │    │    │           └─ Directorio home
│        │ │    │    └─ GECOS (descripción/nombre completo)
│        │ │    └─ GID (grupo primario)
│        │ └─ UID (identificador único)
│        └─ "x" significa contraseña en /etc/shadow
└─ Nombre de usuario
```

### Anatomía de /etc/group
```
sudo:x:27:rp2-node,admin
│    │ │  │
│    │ │  └─ Miembros del grupo
│    │ └─ GID
│    └─ Contraseña (obsoleto)
└─ Nombre del grupo
```

## UID y GID: Por qué importan

Linux internamente usa números (UID/GID), no nombres. Los nombres son solo etiquetas legibles.
```
Archivo en disco:
┌─────────────────────────────┐
│ archivo.txt                 │
│ Owner UID: 1000             │
│ Owner GID: 1000             │
└─────────────────────────────┘

En máquina A (UID 1000 = admin):
  ls -la → -rw-r-- admin admin archivo.txt

En máquina B (UID 1000 = ubuntu):
  ls -la → -rw-r-- ubuntu ubuntu archivo.txt

¡Mismo archivo, diferente nombre mostrado!
```

### Implicaciones con NFS

NFS comparte archivos entre máquinas usando UIDs:
```
rp1-master                          rp2 (netboot via NFS)
───────────                         ──────────────────────
/srv/nfs/rp2/home/admin/            /home/admin/
  UID 1000 = rp1-master               UID 1000 = rp2-node

Problema: El archivo pertenece a UID 1000
- En rp1-master se ve como "rp1-master"
- En rp2 se ve como "rp2-node"
- Si los UIDs no coinciden, hay problemas de permisos
```

**Solución:** Usar el mismo UID para usuarios equivalentes en todas las máquinas.

## Grupos importantes

| Grupo | Propósito |
|-------|-----------|
| `sudo` | Puede ejecutar sudo |
| `docker` | Puede usar Docker sin sudo |
| `adm` | Puede leer logs en /var/log |
| `www-data` | Usuario de servidores web |

## Comandos de gestión

### Usuarios
```bash
# Ver usuario actual
whoami

# Ver UID, GID y grupos
id
id nombre_usuario

# Ver todos los usuarios (solo los que pueden login)
cat /etc/passwd | grep -E '/bin/(bash|sh|zsh)$'

# Crear usuario
sudo useradd -m -s /bin/bash nombre
# -m = crear directorio home
# -s = shell

# Crear usuario con UID específico
sudo useradd -m -s /bin/bash -u 1000 nombre

# Cambiar UID de usuario existente
sudo usermod -u NUEVO_UID nombre

# Eliminar usuario (mantiene home)
sudo userdel nombre

# Eliminar usuario y su home
sudo userdel -r nombre

# Cambiar contraseña
sudo passwd nombre
```

### Grupos
```bash
# Ver grupos de un usuario
groups nombre

# Crear grupo
sudo groupadd nombre_grupo

# Crear grupo con GID específico
sudo groupadd -g 1000 nombre_grupo

# Agregar usuario a grupo
sudo usermod -aG grupo usuario
# -a = append (no elimina de otros grupos)
# -G = grupos secundarios

# Cambiar grupo primario
sudo usermod -g grupo usuario
```

### Permisos
```bash
# Cambiar dueño
sudo chown usuario archivo
sudo chown usuario:grupo archivo
sudo chown -R usuario:grupo directorio/  # Recursivo

# Cambiar solo grupo
sudo chgrp grupo archivo

# Cambiar permisos
chmod 755 archivo  # rwxr-xr-x
chmod 644 archivo  # rw-r--r--
chmod 600 archivo  # rw-------
```

## Ansible y usuarios

### ¿Con qué usuario conecta Ansible?

El definido en `ansible_user` del inventory:
```yaml
# inventory/inventory.yml
nodes:
  hosts:
    rp2:
      ansible_host: 10.0.0.2
      ansible_user: admin        ← Conecta como "admin"
```

### Flujo de ejecución
```
Tu Mac                              Servidor
──────                              ────────
ansible-playbook playbook.yml
        │
        └──► SSH como ansible_user ────► admin@10.0.0.2
                                              │
                                    become: yes (sudo)
                                              │
                                              ▼
                                        Ejecuta como root
```

### Requisitos para Ansible

El usuario de Ansible necesita:

1. **Acceso SSH** (clave pública en authorized_keys)
2. **Permisos de sudo sin contraseña**
```bash
# /etc/sudoers.d/admin
admin ALL=(ALL) NOPASSWD: ALL
```

### Módulos de Ansible para usuarios
```yaml
# Crear usuario
- name: Crear usuario admin
  user:
    name: admin
    uid: 1000
    shell: /bin/bash
    groups: sudo,docker
    append: yes              # No eliminar de otros grupos
    create_home: yes

# Asegurar que existe el grupo
- name: Crear grupo admin
  group:
    name: admin
    gid: 1000
    state: present

# Configurar sudo sin contraseña
- name: Sudo sin contraseña para admin
  copy:
    content: "admin ALL=(ALL) NOPASSWD: ALL"
    dest: /etc/sudoers.d/admin
    mode: "0440"

# Agregar clave SSH
- name: Agregar clave SSH
  authorized_key:
    user: admin
    key: "{{ lookup('file', '~/.ssh/id_ed25519.pub') }}"
    state: present
```

## Estrategias para múltiples máquinas

### Opción A: Usuario por máquina
```
rp1-master → usuario: rp1-master (UID 1000)
rp2        → usuario: rp2-node (UID 1000)
rp3        → usuario: rp3-node (UID 1000)
```

**Pros:** Claro quién es quién
**Contras:** Confuso con NFS, diferentes nombres para mismo UID

### Opción B: Usuario unificado (recomendado)
```
rp1-master → usuario: admin (UID 1000)
rp2        → usuario: admin (UID 1000)
rp3        → usuario: admin (UID 1000)
```

**Pros:** 
- Consistente
- NFS funciona correctamente
- Una sola configuración de Ansible

**Contras:** Menos granularidad

### Opción C: Usuario de servicio + personal
```
Cada máquina:
├── admin (UID 1000)      ← Administración, Ansible, servicios
└── krlosaren (UID 1001)  ← Tu usuario personal (opcional)
```

**Pros:** Separación de responsabilidades, auditoría clara

## Troubleshooting

### "Permission denied" en archivos NFS

Los UIDs no coinciden entre máquinas.
```bash
# Ver UID real del archivo
ls -ln /ruta/archivo

# Comparar con usuario local
id nombre_usuario

# Solución: Ajustar UID o cambiar dueño
sudo chown -R 1000:1000 /ruta/
```

### Usuario no puede hacer sudo
```bash
# Verificar grupos
groups usuario

# Verificar archivo sudoers
sudo cat /etc/sudoers.d/usuario

# Verificar permisos del archivo (debe ser 440)
ls -la /etc/sudoers.d/
```

### Ansible falla al conectar
```bash
# Probar SSH manual
ssh usuario@servidor

# Probar sudo
ssh usuario@servidor "sudo whoami"
# Debe retornar "root" sin pedir contraseña
```

## Resumen de buenas prácticas

1. **Mismo UID** para usuarios equivalentes en todas las máquinas
2. **No usar root** directamente, usar sudo
3. **Sudo sin contraseña** solo para usuarios de automatización
4. **Grupos específicos** para permisos (docker, sudo, etc.)
5. **Documentar** qué usuario se usa para qué propósito
