import { useEffect, useRef } from "react";
import { VideoRTC } from "./vendor/video-rtc.js";

if (!customElements.get("video-rtc")) customElements.define("video-rtc", VideoRTC);

// Video en vivo vía go2rtc (MSE sobre WebSocket, solo video). El elemento se crea a mano
// porque VideoRTC es un custom element que maneja su propio <video> y su WebSocket;
// al desmontar se quita del DOM y VideoRTC cierra la conexión.
export default function EnVivo({ calidad }: { calidad: "sub" | "hd" }) {
  const contenedor = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = document.createElement("video-rtc") as VideoRTC;
    el.mode = "mse";
    el.media = "video";
    el.background = false; // con la app oculta deja de transmitir
    contenedor.current!.appendChild(el); // connectedCallback crea el <video> interno
    // Sin controles nativos: en vivo no hay nada que adelantar, y en iOS se "comen" el toque
    // que amplía la cámara
    el.video.controls = false;
    el.src = `/api/camara/vivo?calidad=${calidad}`; // conecta al estar ya en el DOM
    return () => el.remove();
  }, [calidad]);

  return <div ref={contenedor} className="envivo" />;
}
