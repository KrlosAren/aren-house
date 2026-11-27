// Tipos mínimos de lo que usamos de VideoRTC (go2rtc). Ver video-rtc.js.
export class VideoRTC extends HTMLElement {
  /** "webrtc,mse,hls,mjpeg" por defecto; usamos solo "mse" (WebRTC necesita UDP directo al pod). */
  mode: string;
  /** "video,audio" por defecto; "video" porque el audio de la Tapo (PCMA) no va por MSE. */
  media: string;
  /** No seguir transmitiendo con la pestaña/app oculta. */
  background: boolean;
  visibilityCheck: boolean;
  /** URL del WebSocket; acepta rutas relativas ("/api/...") y las convierte a ws(s)://. */
  src: string;
  video: HTMLVideoElement;
}
