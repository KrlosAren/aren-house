import { useEffect, useState } from "react";
import { api, type Estado, type EstadoCamara } from "./api";
import EnVivo from "./EnVivo";

// En vivo se corta solo: sub ≈ 150 MB/h, HD ≈ 530 MB/h si se queda abierto por olvido
const VIVO_MAX_MS = 10 * 60_000;

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

// connection_quality de Frigate → texto en español + color (valores desconocidos se muestran tal cual)
const CALIDAD: Record<string, { texto: string; clase: string }> = {
  excellent: { texto: "excelente", clase: "" },
  good: { texto: "buena", clase: "" },
  fair: { texto: "regular", clase: "ambar" },
  poor: { texto: "mala", clase: "error" },
};

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
  const [vivo, setVivo] = useState<"sub" | "hd" | null>(null);
  const [ampliada, setAmpliada] = useState(false);

  // Vista ampliada: bloquear el scroll de fondo y cerrar con Escape (teclado en Mac)
  useEffect(() => {
    if (!ampliada) return;
    document.body.style.overflow = "hidden";
    const tecla = (e: KeyboardEvent) => e.key === "Escape" && setAmpliada(false);
    window.addEventListener("keydown", tecla);
    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", tecla);
    };
  }, [ampliada]);

  // Imagen fija cada 10 s (solo si no se está viendo en vivo)
  useEffect(() => {
    if (vivo) return;
    const t = window.setInterval(() => setTick((n) => n + 1), 10_000);
    return () => window.clearInterval(t);
  }, [vivo]);

  // Corte automático del vivo
  useEffect(() => {
    if (!vivo) return;
    const t = window.setTimeout(() => {
      setVivo(null);
      onAviso("En vivo detenido tras 10 min (para no gastar datos).");
    }, VIVO_MAX_MS);
    return () => window.clearTimeout(t);
  }, [vivo, onAviso]);

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

      {/* Mismo elemento, agrandado con CSS: el video en vivo NO se desmonta ni se reconecta */}
      <div
        className={`video ${ampliada ? "ampliada" : ""}`}
        onClick={() => setAmpliada((a) => !a)}
        role="button"
        aria-label={ampliada ? "Cerrar vista ampliada" : "Ampliar cámara"}
      >
        {vivo ? (
          <>
            <EnVivo calidad={vivo} />
            <span className="en-vivo">● EN VIVO {vivo === "hd" ? "HD" : ""}</span>
          </>
        ) : (
          <img src={`/api/camara/imagen?t=${tick}${ampliada ? "&alto=720" : ""}`} alt="Vista actual de la cámara" />
        )}
        {sinVideo && <span className="sobre">Sin video</span>}
        {ampliada ? <span className="cerrar">✕</span> : <span className="ampliar">⤢</span>}
      </div>

      <div className="fila controles-vivo">
        <button className="chip" onClick={() => setVivo(vivo ? null : "sub")}>
          {vivo ? "■ Detener" : "▶ Ver en vivo"}
        </button>
        {vivo && (
          <button className={`chip ${vivo === "hd" ? "activo" : ""}`} onClick={() => setVivo(vivo === "hd" ? "sub" : "hd")}>
            HD {vivo === "hd" ? "✓" : ""}
          </button>
        )}
      </div>

      <dl className="datos">
        <div><dt>FPS</dt><dd className={sinVideo ? "error" : ""}>{camara.fps.toFixed(1)}</dd></div>
        <div>
          <dt>Conexión</dt>
          <dd className={CALIDAD[camara.calidad]?.clase ?? ""}>{CALIDAD[camara.calidad]?.texto ?? (camara.calidad || "—")}</dd>
        </div>
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
