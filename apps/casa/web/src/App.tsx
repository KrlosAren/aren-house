import { useCallback, useEffect, useRef, useState } from "react";
import { api, type Estado } from "./api";
import Camara from "./Camara";
import Focos from "./Focos";
import Homelab from "./Homelab";

const INTERVALO_MS = 5000;

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

      {aviso && <div className="aviso" role="alert">{aviso}</div>}
    </main>
  );
}
