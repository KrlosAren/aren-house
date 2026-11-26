import type { Estado } from "./api";

// Umbrales iguales a las alertas de Prometheus: Pi 75 °C, x86 85 °C
function claseTemp(nodo: string, t?: number) {
  if (t === undefined) return "";
  const limite = nodo.startsWith("pc") ? 85 : 75;
  return t >= limite ? "error" : t >= limite - 10 ? "ambar" : "";
}

export default function Homelab({ homelab, error }: { homelab?: Estado["homelab"]; error?: string }) {
  return (
    <section className="tarjeta">
      <h2>Homelab</h2>
      {!homelab ? (
        <p className="error">Prometheus no disponible: {error}</p>
      ) : (
        <table className="nodos">
          <thead>
            <tr><th>Nodo</th><th>CPU</th><th>Temp</th></tr>
          </thead>
          <tbody>
            {homelab.nodos.map((n) => (
              <tr key={n.nombre}>
                <td><span className={`punto ${n.arriba ? "verde" : "rojo"}`} />{n.nombre}</td>
                <td>{n.cpu !== undefined ? `${n.cpu.toFixed(0)}%` : "—"}</td>
                <td className={claseTemp(n.nombre, n.temperatura)}>
                  {n.temperatura !== undefined ? `${n.temperatura.toFixed(0)} °C` : "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
