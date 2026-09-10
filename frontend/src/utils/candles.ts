import type { Candle } from "../types";

export const candleTime = (candle: Candle) => new Date(candle.ts).getTime();

// REST history and live WebSocket updates may represent the same instant
// with different string formats. Normalize by the actual instant so a
// forming (still-open) bar replaces its own prior tick, and setData
// (which requires strictly ascending, unique timestamps) never chokes.
export function normalizeCandles(candles: Candle[]): Candle[] {
  const byTime = new Map<number, Candle>();

  for (const candle of candles) {
    const time = candleTime(candle);
    if (Number.isFinite(time)) byTime.set(time, candle);
  }

  return [...byTime.entries()]
    .sort(([left], [right]) => left - right)
    .map(([, candle]) => candle);
}
