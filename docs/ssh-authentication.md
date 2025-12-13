# Autenticación SSH en el Homelab

Guía completa sobre configuración de acceso SSH, claves públicas/privadas, y sudo.

## Conceptos fundamentales

### Métodos de autenticación SSH

| Método | Cómo funciona | Seguridad |
|--------|---------------|-----------|
| Password | Usuario y contraseña | ⚠️ Vulnerable a fuerza bruta |
| Clave pública | Par de claves (privada + pública) | ✅ Muy seguro |

Ambos métodos pueden coexistir. SSH intenta primero la clave pública, si no existe, pide contraseña.

### Par de claves SSH
```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   CLAVE PRIVADA              CLAVE PÚBLICA                  │
│   ~/.ssh/id_ed25519          ~/.ssh/id_ed25519.pub          │
│                                                             │
│   • Nunca compartir          • Se copia a servidores        │
│   • Permanece en tu máquina  • Va en authorized_keys        │
│   • Como tu contraseña       • Como tu usuario              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Flujo de autenticación
```
Tu Mac                                    Servidor
──────                                    ────────

1. ssh usuario@servidor  ─────────────────►
                                          2. Busca en authorized_keys
                                             ¿Existe clave pública?
                         ◄─────────────────  3. Envía desafío

4. Firma con clave privada ───────────────►
                                          5. Verifica firma con
                                             clave pública
                         ◄─────────────────  6. ✅ Acceso concedido
```

## Archivos importantes

### En el cliente (tu Mac)
```
~/.ssh/
├── id_ed25519        ← Clave PRIVADA (nunca compartir)
├── id_ed25519.pub    ← Clave PÚBLICA (se copia a servidores)
├── known_hosts       ← Fingerprints de servidores conocidos
└── config            ← Configuración de conexiones (opcional)
```

### En el servidor
```
/home/usuario/.ssh/
└── authorized_keys   ← Claves públicas autorizadas (una por línea)

/etc/ssh/
└── sshd_config       ← Configuración del servidor SSH

/etc/sudoers.d/
└── usuario           ← Permisos de sudo por usuario
```

## Configuración inicial de una Raspberry Pi

Cuando instalas Ubuntu Server con usuario y contraseña, SSH permite ambos métodos por defecto.

### Paso 1: Generar clave SSH (en tu Mac)
```bash
# Verificar si ya existe
ls ~/.ssh/id_ed25519.pub

# Si no existe, generar
ssh-keygen -t ed25519 -C "homelab"
```

Opciones:
- `-t ed25519` → Tipo de clave (más seguro y moderno que RSA)
- `-C "comentario"` → Identificador (aparece al final de la clave pública)

### Paso 2: Copiar clave al servidor
```bash
ssh-copy-id usuario@IP_DEL_SERVIDOR
```

Esto agrega tu clave pública a `~/.ssh/authorized_keys` en el servidor.

### Paso 3: Verificar acceso sin contraseña
```bash
ssh usuario@IP_DEL_SERVIDOR
```

Debe entrar sin pedir contraseña.

### Paso 4: Configurar sudo sin contraseña

Necesario para que Ansible funcione sin intervención.
```bash
# Conectar al servidor
ssh usuario@IP_DEL_SERVIDOR

# Crear archivo de sudoers para el usuario
echo "tu_usuario ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/tu_usuario

# Ajustar permisos (obligatorio)
sudo chmod 440 /etc/sudoers.d/tu_usuario
```

**¿Por qué chmod 440?** El archivo sudoers requiere permisos específicos: lectura solo para root y grupo. Si los permisos son incorrectos, sudo ignora el archivo.

### Paso 5 (opcional): Deshabilitar acceso con contraseña

Para mayor seguridad (recomendado en servidores expuestos a internet):
```bash
sudo nano /etc/ssh/sshd_config
```

Cambiar:
```
PasswordAuthentication no
```

Reiniciar SSH:
```bash
sudo systemctl restart sshd
```

> ⚠️ **Importante:** Antes de cerrar la sesión, abre otra terminal y verifica que puedes entrar con la clave.

## Gestión de sudo

### ¿Qué es sudo?

`sudo` permite ejecutar comandos como root (administrador). Por defecto pide contraseña.

### ¿Qué es visudo?

Editor especial para `/etc/sudoers` que valida sintaxis antes de guardar. Si cometes un error en sudoers, puedes quedar bloqueado sin poder usar sudo.
```bash
# Forma segura de editar sudoers
sudo visudo

# Forma moderna - archivos separados por usuario
sudo nano /etc/sudoers.d/nombre_usuario
```

### Sintaxis de sudoers
```
usuario  host=(usuario_destino)  opciones: comandos
│        │    │                  │         │
│        │    │                  │         └─ ALL = todos los comandos
│        │    │                  └─ NOPASSWD = sin pedir contraseña
│        │    └─ ALL = puede actuar como cualquier usuario
│        └─ ALL = desde cualquier host
└─ nombre del usuario
```

Ejemplo:
```bash
rp2-node ALL=(ALL) NOPASSWD: ALL
```

### Archivos de sudoers
```
/etc/sudoers              ← Archivo principal (no editar directamente)
/etc/sudoers.d/           ← Directorio para archivos adicionales
├── 90-cloud-init-users   ← Creado por cloud-init
├── rp2-node              ← Creado manualmente para rp2-node
└── ...
```

## Múltiples claves SSH

### En el cliente (múltiples claves privadas)
```
~/.ssh/
├── id_ed25519           ← Clave personal
├── id_rsa_trabajo       ← Clave del trabajo
├── aws_prod.pem         ← Clave para AWS
└── config               ← Define cuál usar para cada host
```

### Archivo ~/.ssh/config
```bash
# Homelab
Host rp1-master
    HostName 192.168.1.84
    User rp1-master
    IdentityFile ~/.ssh/id_ed25519

Host rp2
    HostName 10.0.0.2
    User rp2-node
    IdentityFile ~/.ssh/id_ed25519

# AWS
Host aws-prod
    HostName 52.10.20.30
    User ubuntu
    IdentityFile ~/.ssh/aws_prod.pem
```

Después solo escribes:
```bash
ssh rp2          # Usa la configuración automáticamente
ssh aws-prod     # Usa aws_prod.pem
```

### En el servidor (múltiples claves autorizadas)

El archivo `authorized_keys` puede tener múltiples claves, una por línea:
```bash
# /home/rp2-node/.ssh/authorized_keys
ssh-ed25519 AAAA...abc1 krlosaren@mac
ssh-ed25519 AAAA...xyz2 rp1-master
ssh-ed25519 AAAA...def3 laptop-trabajo
```

Cualquiera con la clave privada correspondiente puede entrar.

## Troubleshooting

### "Permission denied (publickey)"

La clave pública no está en el servidor o los permisos son incorrectos.
```bash
# Verificar que la clave está en authorized_keys
cat ~/.ssh/authorized_keys

# Verificar permisos
ls -la ~/.ssh/
# Debe ser:
# drwx------ .ssh/
# -rw------- authorized_keys

# Corregir permisos
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

### "sudo: I'm sorry, I can't do that"

El usuario no tiene permisos de sudo o el archivo sudoers tiene permisos incorrectos.
```bash
# Verificar archivo sudoers del usuario
sudo cat /etc/sudoers.d/tu_usuario

# Verificar permisos (debe ser 440)
ls -la /etc/sudoers.d/

# Corregir
sudo chmod 440 /etc/sudoers.d/tu_usuario
```

### "Host key verification failed"

El fingerprint del servidor cambió (común después de reinstalar o con netboot).
```bash
# Eliminar la entrada antigua
ssh-keygen -R IP_DEL_SERVIDOR

# Conectar de nuevo (aceptará la nueva clave)
ssh usuario@IP_DEL_SERVIDOR
```

### Ansible falla con "MODULE FAILURE"

Generalmente es problema de sudo.
```bash
# Verificar que sudo funciona sin contraseña
ssh usuario@servidor "sudo whoami"
# Debe mostrar "root" sin pedir contraseña
```

## Configuración recomendada

### Para homelab (red interna)
```bash
# /etc/ssh/sshd_config
PasswordAuthentication yes     # Conveniente para emergencias
PubkeyAuthentication yes       # Método principal
PermitRootLogin no             # Seguridad básica
```

### Para servidor en internet
```bash
# /etc/ssh/sshd_config
PasswordAuthentication no      # Solo claves (obligatorio)
PubkeyAuthentication yes
PermitRootLogin no
AllowUsers tu_usuario          # Whitelist explícita
```

## Comandos útiles
```bash
# Generar nueva clave
ssh-keygen -t ed25519 -f ~/.ssh/nombre_clave -C "descripcion"

# Copiar clave a servidor
ssh-copy-id -i ~/.ssh/clave.pub usuario@servidor

# Ver claves en tu máquina
ls -la ~/.ssh/

# Ver clave pública
cat ~/.ssh/id_ed25519.pub

# Ver claves autorizadas en servidor
cat ~/.ssh/authorized_keys

# Probar conexión con verbose (debug)
ssh -v usuario@servidor

# Verificar sudo sin contraseña
ssh usuario@servidor "sudo whoami"
```
