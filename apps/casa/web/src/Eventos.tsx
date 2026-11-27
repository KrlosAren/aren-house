import { useEffect, useState } from "react";
import { api, type EventoLista } from "./api";

const ETIQUETA: Record<string, string> = { person: "Persona", dog: "Perro", cat: "Gato" };

function hora(epoch: number) {
  const d = new Date(epoch * 1000);
  const hoy = new Date().toDateString() === d.toDateString();
  const hm = d.toLocaleTimeString("es-CL", { hour: "2-digit", minute: "2-digit" });
  return hoy ? hm : `${d.toLocaleDateString("es-CL", { weekday: "short", day: "numeric" })} ${hm}`;
}

const duracion = (e: EventoLista) => (e.fin ? `${Math.max(1, Math.round(e.fin - e.inicio))} s` : "en curso");

// Historial con miniaturas; al tocar uno se reproduce el clip grabado por Frigate.
// `ultimoId` viene del polling de /api/estado: si cambia, hay un evento nuevo y se recarga.
export default function Eventos({ ultimoId }: { ultimoId?: string }) {
  const [eventos, setEventos] = useState<EventoLista[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [verTodos, setVerTodos] = useState(false);
  const [abierto, setAbierto] = useState<EventoLista | null>(null);

  useEffect(() => {
    api.eventos(20).then(
      (lista) => { setEventos(lista); setError(null); },
      (e: Error) => setError(e.message),
    );
  }, [ultimoId]);

  useEffect(() => {
    if (!abierto) return;
    document.body.style.overflow = "hidden";
    const tecla = (e: KeyboardEvent) => e.key === "Escape" && setAbierto(null);
    window.addEventListener("keydown", tecla);
    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", tecla);
    };
  }, [abierto]);

  const visibles = eventos ? (verTodos ? eventos : eventos.slice(0, 6)) : [];

  return (
    <section className="tarjeta">
      <h2>Eventos</h2>
      {error && <p className="error">No disponible: {error}</p>}
      {eventos?.length === 0 && <p className="sub">Sin eventos grabados.</p>}

      <div className="eventos">
        {visibles.map((e) => (
          <button key={e.id} className="evento-celda" disabled={!e.tiene_clip || !e.fin} onClick={() => setAbierto(e)}>
            <img src={`/api/camara/eventos/${e.id}/miniatura`} alt="" loading="lazy" />
            <span className="evento-info">
              <b>{ETIQUETA[e.etiqueta] ?? e.etiqueta}</b> {Math.round(e.score * 100)}%
              <small>{hora(e.inicio)} · {duracion(e)}</small>
            </span>
            {e.tiene_clip && e.fin && <span className="play">▶</span>}
          </button>
        ))}
      </div>

      {eventos && eventos.length > 6 && (
        <button className="chip" onClick={() => setVerTodos(!verTodos)}>
          {verTodos ? "Ver menos" : `Ver ${eventos.length - 6} más`}
        </button>
      )}

      {abierto && (
        <div className="visor" onClick={() => setAbierto(null)}>
          <span className="cerrar">✕</span>
          {/* El clip lo sirve casa desde su caché con Range (206): Safari lo necesita */}
          <video src={`/api/camara/eventos/${abierto.id}/clip`} controls autoPlay playsInline
            onClick={(ev) => ev.stopPropagation()} />
          <p className="visor-pie">
            {ETIQUETA[abierto.etiqueta] ?? abierto.etiqueta} · {hora(abierto.inicio)} · {duracion(abierto)}
          </p>
        </div>
      )}
    </section>
  );
}
