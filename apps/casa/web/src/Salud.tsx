import { useEffect, useState } from "react";
import { api, type SaludCamara } from "./api";

const REFRESCO_MS = 5 * 60_000; // los tramos son de 5 min: refrescar más seguido no aporta

const TEXTO = { ok: "con video", lenta: "pérdida de ping", caida: "sin video", sin_datos: "sin datos" } as const;

// Barra de las últimas 24 h: un segmento por tramo de 5 min.
// Rojo = el video cayó (aunque sea un instante) · ámbar = se perdieron pings pero hubo video.
export default function Salud() {
  const [salud, setSalud] = useState<SaludCamara | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const cargar = () => {
      if (document.visibilityState !== "visible") return;
      api.salud(24).then((s) => { setSalud(s); setError(null); }, (e: Error) => setError(e.message));
    };
    cargar();
    const t = window.setInterval(cargar, REFRESCO_MS);
    document.addEventListener("visibilitychange", cargar);
    return () => {
      window.clearInterval(t);
      document.removeEventListener("visibilitychange", cargar);
    };
  }, []);

  if (error) return <p className="error salud-error">Salud 24 h no disponible: {error}</p>;
  if (!salud) return null;

  const claseDisp = salud.disponibilidad >= 99.5 ? "" : salud.disponibilidad >= 97 ? "ambar" : "error";

  return (
    <div className="salud">
      <div className="fila">
        <small>Últimas {salud.horas} h</small>
        <small>
          <b className={claseDisp}>{salud.disponibilidad.toFixed(1)}%</b> con video ·{" "}
          <b className={salud.cortes > 5 ? "error" : ""}>{salud.cortes}</b> cortes
        </small>
      </div>
      <div className="barra" role="img" aria-label={`Disponibilidad ${salud.disponibilidad.toFixed(1)}%, ${salud.cortes} cortes`}>
        {salud.tramos.map((t) => (
          <span key={t.inicio} className={`tramo ${t.estado}`}
            title={`${new Date(t.inicio * 1000).toLocaleTimeString("es-CL", { hour: "2-digit", minute: "2-digit" })} · ${TEXTO[t.estado]}`} />
        ))}
      </div>
      <div className="fila eje"><small>hace 24 h</small><small>12 h</small><small>ahora</small></div>
    </div>
  );
}
