import { useState } from "react";
import { api, type CambioFoco, type EstadoFoco } from "./api";

const PRESETS: { nombre: string; cambio: CambioFoco; muestra: string }[] = [
  { nombre: "Cálido", cambio: { temp: 2700 }, muestra: "#ffb46b" },
  { nombre: "Neutro", cambio: { temp: 4000 }, muestra: "#ffe4c4" },
  { nombre: "Frío", cambio: { temp: 6500 }, muestra: "#e6f0ff" },
  { nombre: "Rojo", cambio: { color: "#ff0000" }, muestra: "#ff3b30" },
  { nombre: "Verde", cambio: { color: "#00ff40" }, muestra: "#34c759" },
  { nombre: "Azul", cambio: { color: "#0040ff" }, muestra: "#0a84ff" },
  { nombre: "Morado", cambio: { color: "#8000ff" }, muestra: "#bf5af2" },
];

// Color aproximado de lo que el foco está mostrando, para pintar el ícono
function colorActual(f: EstadoFoco) {
  if (!f.state) return "transparent";
  if (f.r !== undefined) return `rgb(${f.r},${f.g},${f.b})`;
  if (f.temp) return f.temp < 3300 ? "#ffb46b" : f.temp < 5000 ? "#ffe4c4" : "#e6f0ff";
  return "#ffe4c4";
}

function Foco({ foco, onAviso, onCambio }: { foco: EstadoFoco; onAviso: (m: string) => void; onCambio: () => void }) {
  const [local, setLocal] = useState<EstadoFoco | null>(null); // respuesta inmediata del POST
  const [ocupado, setOcupado] = useState(false);
  const [brillo, setBrillo] = useState<number | null>(null); // mientras se arrastra el slider
  const f = local && local.id === foco.id ? { ...foco, ...local } : foco;

  const cambiar = async (c: CambioFoco) => {
    setOcupado(true);
    try {
      setLocal(await api.foco(f.id, c));
      onCambio();
    } catch (e) {
      onAviso(`${f.nombre}: ${(e as Error).message}`);
    } finally {
      setOcupado(false);
      setBrillo(null);
    }
  };

  if (!f.alcanzable) {
    return (
      <div className="foco apagado">
        <span className="bombilla" />
        <div>
          <b>{f.nombre}</b>
          <small className="error">Sin red — ¿apagado en el interruptor?</small>
        </div>
      </div>
    );
  }

  return (
    <div className={`foco ${f.state ? "" : "apagado"}`}>
      <div className="fila">
        <span className="bombilla" style={{ background: colorActual(f), boxShadow: f.state ? `0 0 18px ${colorActual(f)}` : "none" }} />
        <div className="crece">
          <b>{f.nombre}</b>
          <small>{f.state ? `${f.dimming ?? 100}%` : "Apagado"}</small>
        </div>
        <label className="interruptor">
          <input type="checkbox" checked={!!f.state} disabled={ocupado} onChange={(e) => cambiar({ encendido: e.target.checked })} />
          <span />
        </label>
      </div>

      {f.state && (
        <>
          <input
            type="range" min={10} max={100} step={5}
            value={brillo ?? f.dimming ?? 100}
            disabled={ocupado}
            aria-label={`Brillo ${f.nombre}`}
            onChange={(e) => setBrillo(Number(e.target.value))}
            onPointerUp={() => brillo !== null && cambiar({ brillo })}
            onKeyUp={() => brillo !== null && cambiar({ brillo })}
          />
          <div className="presets">
            {PRESETS.map((p) => (
              <button key={p.nombre} title={p.nombre} aria-label={p.nombre} disabled={ocupado}
                style={{ background: p.muestra }} onClick={() => cambiar(p.cambio)} />
            ))}
          </div>
        </>
      )}
    </div>
  );
}

export default function Focos({ focos, onAviso, onCambio }: { focos: EstadoFoco[]; onAviso: (m: string) => void; onCambio: () => void }) {
  return (
    <section className="tarjeta">
      <h2>Focos</h2>
      {focos.length === 0 && <p className="sub">No hay focos configurados (variable FOCOS).</p>}
      {focos.map((f) => (
        <Foco key={f.id} foco={f} onAviso={onAviso} onCambio={onCambio} />
      ))}
    </section>
  );
}
