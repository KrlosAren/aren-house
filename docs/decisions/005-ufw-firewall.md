# ADR-005: UFW como Firewall para Gateway y Nodos

## Estado

Aceptado

## Contexto

El homelab necesita un firewall para:

1. **Proteger el gateway** de accesos no autorizados desde WAN
2. **Segmentar tráfico** entre redes (LAN, VPN, WAN)
3. **Limitar servicios** a las redes que los necesitan (DNS/DHCP solo LAN, SSH desde LAN/VPN)
4. **Proteger nodos** permitiendo solo tráfico necesario para netboot y administración

Actualmente solo existen reglas iptables para NAT (MASQUERADE) y FORWARD de WireGuard, sin restricciones de entrada.

## Decisión

### Usar UFW (Uncomplicated Firewall)

Elegimos UFW como interfaz para gestionar iptables por:

1. **Sintaxis simple**: `ufw allow from 10.0.0.0/24 to any port 22`
2. **Integración con Ansible**: módulo `ufw` nativo
3. **Comentarios en reglas**: facilita auditoría
4. **Rate limiting**: protección contra fuerza bruta (`ufw limit`)

### Estrategia de Reglas

#### Políticas por defecto
- **Entrada**: DENY (denegar todo por defecto)
- **Salida**: ALLOW (permitir todo)
- **Forward**: ACCEPT (necesario para NAT)

#### Gateway (rp1-master)

**Desde LAN (eth0)**: Permitir todo el tráfico.

Justificación: Los nodos netboot requieren DHCP, TFTP y NFS para arrancar. Es más simple y confiable permitir todo desde la interfaz LAN que gestionar reglas individuales. La seguridad perimetral se aplica en WAN.

**Desde VPN y WAN**:

| Puerto | Protocolo | Desde | Servicio |
|--------|-----------|-------|----------|
| 22 | TCP | VPN, WAN (limit) | SSH |
| 53 | TCP/UDP | VPN | DNS |
| 111 | TCP | VPN | RPC (NFS) |
| 2049 | TCP | VPN | NFS |
| 51820 | UDP | ANY | WireGuard |
| 80, 443 | TCP | ANY | HTTP/S (futuro) |

#### Nodos (rp2, rp3)

| Puerto | Protocolo | Desde | Servicio |
|--------|-----------|-------|----------|
| 22 | TCP | LAN, VPN | SSH |
| * | * | Gateway | Todo (netboot) |

### Implementación via Ansible

Playbook `firewall.yml` que:
1. Instala UFW
2. Configura políticas por defecto
3. Aplica reglas específicas por grupo (gateway vs nodes)
4. Habilita forwarding para NAT
5. Activa el firewall

## Alternativas Consideradas

### iptables directo
- **Pro**: Control total, sin capas de abstracción
- **Contra**: Sintaxis compleja, difícil de mantener
- **Decisión**: UFW genera iptables internamente, mejor UX

### nftables
- **Pro**: Sucesor moderno de iptables, mejor rendimiento
- **Contra**: Menos soporte en Ansible, curva de aprendizaje
- **Decisión**: No adoptado, UFW es suficiente para el caso de uso

### firewalld
- **Pro**: Zonas de red, recarga dinámica
- **Contra**: Más complejo, pensado para servidores empresariales
- **Decisión**: Overhead innecesario para homelab

## Consecuencias

### Positivas
- Gateway y nodos protegidos por firewall
- Servicios limitados a redes autorizadas
- Rate limiting en SSH previene fuerza bruta
- Configuración reproducible via Ansible
- Fácil auditoría con `ufw status verbose`

### Negativas
- UFW y WireGuard PostUp/PostDown pueden generar reglas duplicadas
- Requiere cuidado al modificar reglas manualmente (usar Ansible)

### Consideraciones de Netboot
- Los nodos dependen del gateway para TFTP/NFS
- La regla "permitir todo desde gateway" es necesaria para boot
- Si el firewall bloquea NFS, los nodos no arrancan

## Comandos de Verificación

```bash
# Ver estado del firewall
sudo ufw status verbose

# Ver reglas numeradas
sudo ufw status numbered

# Ver reglas iptables generadas
sudo iptables -L -n -v
```

## Referencias

- [UFW Manual](https://help.ubuntu.com/community/UFW)
- [Ansible UFW Module](https://docs.ansible.com/ansible/latest/collections/community/general/ufw_module.html)
- ADR-004: IP Forwarding y NAT
