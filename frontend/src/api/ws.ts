import { useEffect, useRef } from "react";
import type { WSMessage } from "../types";

const WS_URL = import.meta.env.VITE_WS_URL ?? "ws://localhost:8280/ws/market";

/**
 * Subscribes to the backend's /ws/market broadcast and calls onMessage for
 * every candle/signal/order/position/balance/autotrade_log/funding/settings
 * event. Reconnects with a fixed backoff on drop.
 */
export function useMarketSocket(onMessage: (msg: WSMessage) => void) {
  const handlerRef = useRef(onMessage);
  handlerRef.current = onMessage;

  useEffect(() => {
    let socket: WebSocket | null = null;
    let closedByEffect = false;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    const connect = () => {
      socket = new WebSocket(WS_URL);
      socket.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data) as WSMessage;
          handlerRef.current(msg);
        } catch {
          // ignore malformed frames
        }
      };
      socket.onclose = () => {
        if (!closedByEffect) {
          retryTimer = setTimeout(connect, 3000);
        }
      };
    };

    connect();

    return () => {
      closedByEffect = true;
      if (retryTimer) clearTimeout(retryTimer);
      socket?.close();
    };
  }, []);
}
