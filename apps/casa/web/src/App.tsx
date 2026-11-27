import { useCallback, useEffect, useRef, useState } from "react";
import { api, type Estado } from "./api";
import Camara from "./Camara";
import Focos from "./Focos";
import Homelab from "./Homelab";

const INTERVALO_MS = 5000;

// SHA con el que se compiló ESTE frontend (VITE_VERSION en el Dockerfile). "dev" en local.
const VERSION_WEB: string = import.meta.env.VITE_VERSION || "dev";
const REPO = "https://github.com/KrlosAren/aren-house";
const corto = (sha: string) => (sha === "dev" ? "dev" : sha.slice(0, 7));

export default function App() {
  const [estado, setEstado] = useState<Estado | null>(null);
  const [sinConexion, setSinConexion] = useState(false);
  const [aviso, setAviso] = useState<string | null>(null);
  const timer = useRef<number | undefined>(undefined);

  const refrescar = useCallback(async () => {
    try {
      setEstado(await api.estado());
      setSinConexion(false);
    } catch {
      setSinConexion(true); // típicamente: Tailscale desconectado en el teléfono
    }
  }, []);

  // Polling solo con la app visible: en segundo plano no gasta batería ni golpea a Frigate
  useEffect(() => {
    const arrancar = () => {
      window.clearInterval(timer.current);
      if (document.visibilityState === "visible") {
        refrescar();
        timer.current = window.setInterval(refrescar, INTERVALO_MS);
      }
    };
    arrancar();
    document.addEventListener("visibilitychange", arrancar);
    return () => {
      window.clearInterval(timer.current);
      document.removeEventListener("visibilitychange", arrancar);
    };
  }, [refrescar]);

  const mostrarAviso = useCallback((msg: string) => {
    setAviso(msg);
    window.setTimeout(() => setAviso(null), 5000);
  }, []);

  return (
    <main>
      <header>
        <h1>Casa</h1>
        <span className={`punto ${sinConexion ? "rojo" : estado ? "verde" : ""}`} />
        <span className="sub">
          {sinConexion ? "Sin conexión — ¿Tailscale activo?" : estado ? new Date(estado.hora).toLocaleTimeString("es-CL") : "Cargando…"}
        </span>
      </header>

      {estado && (
        <>
          <Camara camara={estado.camara} error={estado.camara_error} homelab={estado.homelab} onCambio={refrescar} onAviso={mostrarAviso} />
          <Focos focos={estado.focos} onAviso={mostrarAviso} onCambio={refrescar} />
          <Homelab homelab={estado.homelab} error={estado.homelab_error} />
        </>
      )}

      {/* El service worker deja la PWA con el frontend viejo hasta recargar: si el servidor
          ya tiene otra versión, se ofrece recargar en vez de mezclar versiones en silencio. */}
      {estado && estado.version !== VERSION_WEB && VERSION_WEB !== "dev" && (
        <button className="grande primario" onClick={() => location.reload()}>
          Hay una versión nueva ({corto(estado.version)}) — tocar para actualizar
        </button>
      )}

      <footer className="sub">
        {VERSION_WEB === "dev" ? (
          "versión dev"
        ) : (
          <a href={`${REPO}/commit/${VERSION_WEB}`} target="_blank" rel="noreferrer">
            versión {corto(VERSION_WEB)}
          </a>
        )}
        {estado && estado.version !== VERSION_WEB && ` · servidor ${corto(estado.version)}`}
      </footer>

      {aviso && <div className="aviso" role="alert">{aviso}</div>}
    </main>
  );
}
