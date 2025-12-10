# TFTP (Trivial File Transfer Protocol)

## Qué es

Protocolo simple de transferencia de archivos. Es una versión minimalista de FTP, diseñado para ser lo más ligero posible.

## Para qué sirve

Principalmente para **boot por red (PXE)**:

- Dispositivos sin disco/microSD necesitan descargar su sistema operativo
- TFTP es tan simple que cabe en la ROM de una tarjeta de red
- No requiere autenticación (confianza en la red local)

## Comparación con otros protocolos

| Característica | TFTP | FTP | HTTP |
|---------------|------|-----|------|
| Autenticación | No | Sí | Opcional |
| Listado de archivos | No | Sí | Sí |
| Protocolo | UDP | TCP | TCP |
| Complejidad | Mínima | Alta | Media |
| Uso principal | Boot | Transferencias | Web |

## Cómo funciona

```
[Cliente]                      [Servidor TFTP]
    |                                |
    |-- RRQ (Read Request) --------->|  "Dame archivo X"
    |                                |
    |<-------- DATA (bloque 1) ------|  [512 bytes]
    |                                |
    |-- ACK (bloque 1) ------------->|  "Recibido"
    |                                |
    |<-------- DATA (bloque 2) ------|  [512 bytes]
    |                                |
    |-- ACK (bloque 2) ------------->|  "Recibido"
    |           ...                  |
    |<-------- DATA (último) --------|  [< 512 bytes = fin]
    |                                |
    |-- ACK (último) --------------->|  "Completo"
```

### Características

- **Bloques de 512 bytes**: Transferencia en chunks pequeños
- **Sin conexión**: Usa UDP, cada paquete es independiente
- **Último bloque < 512 bytes**: Indica fin de transferencia
- **Sin navegación**: Debes conocer el nombre exacto del archivo

## Puerto

| Puerto | Protocolo | Uso |
|--------|-----------|-----|
| 69/udp | TFTP | Peticiones iniciales |
| >1024/udp | TFTP | Transferencia de datos (puerto dinámico) |

## En el homelab

Usamos TFTP para **netboot de Raspberry Pi** (arrancar sin microSD):

```bash
# /etc/dnsmasq.conf
enable-tftp
tftp-root=/srv/tftp

# Estructura de archivos
/srv/tftp/
├── bootcode.bin      # Firmware inicial RPi
├── start.elf         # GPU firmware
├── kernel8.img       # Kernel Linux ARM64
├── config.txt        # Configuración boot
└── cmdline.txt       # Parámetros kernel
```

El flujo de boot:
1. RPi enciende, busca DHCP
2. DHCP responde con IP + dirección servidor TFTP
3. RPi descarga `bootcode.bin` via TFTP
4. Continúa descargando kernel y arranca

Ver: [ADR-003 dnsmasq](../decisions/003-dnsmasq-dhcp-dns-tftp.md)

## Comandos útiles

```bash
# Cliente TFTP (para pruebas)
tftp 10.0.0.1
> get bootcode.bin
> quit

# Ver logs de dnsmasq TFTP
sudo journalctl -u dnsmasq | grep -i tftp

# Listar archivos servidos
ls -la /srv/tftp/

# Verificar que puerto 69 está escuchando
sudo ss -ulnp | grep :69
```

## Seguridad

TFTP no tiene autenticación. Cualquiera en la red puede descargar archivos.

Mitigaciones:
- Solo servir en interfaz interna (`interface=eth0`)
- No poner archivos sensibles en `/srv/tftp`
- Firewall: solo permitir desde red interna
