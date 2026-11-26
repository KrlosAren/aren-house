import { useEffect, useState } from "react";
import { api, type Estado, type EstadoCamara } from "./api";

interface Props {
  camara?: EstadoCamara;
  error?: string;
  homelab?: Estado["homelab"];
  onCambio: () => void;
  onAviso: (msg: string) => void;
}

// Frigate refleja el cambio en /api/stats con ~15 s de retraso: mientras tanto se muestra
// "Aplicando…" y solo se da por hecho cuando el estado REAL coincide.
const CONFIRMAR_MS = 40_000;

function haceCuanto(epoch: number) {
  const s = Math.round(Date.now() / 1000 - epoch);
  if (s < 60) return "hace un momento";
  if (s < 3600) return `hace ${Math.round(s / 60)} min`;
  if (s < 86400) return `hace ${Math.round(s / 3600)} h`;
  return new Date(epoch * 1000).toLocaleDateString("es-CL", { weekday: "short", day: "numeric", month: "short" });
}

export default function Camara({ camara, error, homelab, onCambio, onAviso }: Props) {
  const [pendiente, setPendiente] = useState<{ armada: boolean; hasta: number } | null>(null);
  const [tick, setTick] = useState(0);

  // Imagen en vivo cada 10 s
  useEffect(() => {
    const t = window.setInterval(() => setTick((n) => n + 1), 10_000);
    return () => window.clearInterval(t);
  }, []);

  // Confirmación del cambio pendiente
  useEffect(() => {
    if (!pendiente || !camara) return;
    if (camara.armada === pendiente.armada) setPendiente(null);
    else if (Date.now() > pendiente.hasta) {
      setPendiente(null);
      onAviso("Frigate no confirmó el cambio. Revisa la cámara.");
    }
  }, [camara, pendiente, onAviso]);

  if (!camara) {
    return (
      <section className="tarjeta">
        <h2>Cámara</h2>
        <p className="error">No disponible: {error}</p>
      </section>
    );
  }

  const alternar = async () => {
    const objetivo = !camara.armada;
    try {
      await api.deteccion(objetivo);
      setPendiente({ armada: objetivo, hasta: Date.now() + CONFIRMAR_MS });
      onCambio();
    } catch (e) {
      onAviso(`No se pudo ${objetivo ? "armar" : "desarmar"}: ${(e as Error).message}`);
    }
  };

  const ev = camara.ultimo_evento;
  const sinVideo = camara.fps < 1;

  return (
    <section className="tarjeta">
      <div className="fila">
        <h2>Cámara</h2>
        <span className={`insignia ${pendiente ? "gris" : camara.armada ? "verde" : "ambar"}`}>
          {pendiente ? "Aplicando…" : camara.armada ? "🔒 Armada" : "🔓 Desarmada"}
        </span>
      </div>

      <div className="video">
        <img src={`/api/camara/imagen?t=${tick}`} alt="Vista actual de la cámara" />
        {sinVideo && <span className="sobre">Sin video</span>}
      </div>

      <dl className="datos">
        <div><dt>FPS</dt><dd className={sinVideo ? "error" : ""}>{camara.fps.toFixed(1)}</dd></div>
        <div><dt>Conexión</dt><dd>{camara.calidad || "—"}</dd></div>
        <div>
          <dt>Ping</dt>
          <dd className={homelab?.ping_camara === false ? "error" : ""}>
            {homelab?.ping_camara === undefined ? "—" : homelab.ping_camara ? `${homelab.rtt_camara_ms?.toFixed(0)} ms` : "falla"}
          </dd>
        </div>
        <div><dt>Reconexiones 1h</dt><dd>{camara.reconexiones_hora}</dd></div>
      </dl>

      {ev && (
        <a className="evento" href={`/api/camara/eventos/${ev.id}/foto`} target="_blank" rel="noreferrer">
          {ev.tiene_foto && <img src={`/api/camara/eventos/${ev.id}/foto`} alt="" loading="lazy" />}
          <span>
            Último evento: <b>{ev.etiqueta}</b> {ev.score > 0 && `(${Math.round(ev.score * 100)}%)`}
            <br />
            <small>{haceCuanto(ev.inicio)}</small>
          </span>
        </a>
      )}

      <button className={`grande ${camara.armada ? "secundario" : "primario"}`} onClick={alternar} disabled={!!pendiente}>
        {camara.armada ? "Desarmar" : "Armar"}
      </button>
    </section>
  );
}
