// Tipos y llamadas a la API Go (apps/casa/*.go). Mismos nombres de campo que el JSON.

export interface Evento {
  id: string;
  etiqueta: string;
  inicio: number; // epoch s
  score: number;
  tiene_foto: boolean;
}

export interface EstadoCamara {
  armada: boolean;
  fps: number;
  calidad: string;
  reconexiones_hora: number;
  ultimo_evento: Evento | null;
}

export interface EstadoFoco {
  id: string;
  nombre: string;
  ip: string;
  alcanzable: boolean;
  error?: string;
  state?: boolean;
  dimming?: number;
  r?: number;
  g?: number;
  b?: number;
  temp?: number;
  sceneId?: number;
  rssi?: number;
}

export interface Nodo {
  nombre: string;
  arriba: boolean;
  temperatura?: number;
  cpu?: number;
}

export interface Estado {
  version: string; // SHA del servidor
  hora: string;
  camara?: EstadoCamara;
  camara_error?: string;
  focos: EstadoFoco[];
  homelab?: { nodos: Nodo[]; ping_camara?: boolean; rtt_camara_ms?: number };
  homelab_error?: string;
}

export interface EventoLista extends Evento {
  fin: number | null; // null = en curso
  tiene_clip: boolean;
}

export interface SaludCamara {
  horas: number;
  paso_min: number;
  tramos: { inicio: number; estado: "ok" | "lenta" | "caida" | "sin_datos" }[];
  disponibilidad: number;
  cortes: number;
}

export interface CambioFoco {
  encendido?: boolean;
  brillo?: number;
  color?: string; // #rrggbb
  temp?: number;
}

async function pedir<T>(ruta: string, init?: RequestInit): Promise<T> {
  const r = await fetch(ruta, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  const body = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(body.error ?? `HTTP ${r.status}`);
  return body as T;
}

export const api = {
  estado: () => pedir<Estado>("/api/estado"),
  deteccion: (activa: boolean) =>
    pedir<{ activa: boolean }>("/api/camara/deteccion", { method: "POST", body: JSON.stringify({ activa }) }),
  eventos: (limite = 20) => pedir<EventoLista[]>(`/api/camara/eventos?limite=${limite}`),
  salud: (horas = 24) => pedir<SaludCamara>(`/api/camara/salud?horas=${horas}`),
  foco: (id: string, cambio: CambioFoco) =>
    pedir<EstadoFoco>(`/api/focos/${id}`, { method: "POST", body: JSON.stringify(cambio) }),
};
