import { useState } from "react";
import { api } from "../api/client";
import type { Position } from "../types";

interface Props {
  positions: Position[];
  onChanged?: () => void;
}

export function PositionTable({ positions, onChanged }: Props) {
  const [flattening, setFlattening] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const flatten = async (symbol: string) => {
    setFlattening(symbol);
    setError(null);
    try {
      await api.flatten(symbol);
      onChanged?.();
    } catch (e) {
      setError(String(e));
    } finally {
      setFlattening(null);
    }
  };

  return (
    <div className="panel">
      <h3>持倉</h3>
      {error && <p className="error">{error}</p>}
      {positions.length === 0 ? (
        <p className="muted">目前無持倉</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>合約</th>
              <th>方向</th>
              <th>數量</th>
              <th>進場價</th>
              <th>標記價</th>
              <th>未實現損益</th>
              <th>槓桿</th>
              <th>強平價</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {positions.map((p) => (
              <tr key={p.symbol}>
                <td>{p.symbol}</td>
                <td className={p.side === "long" ? "buy" : "sell"}>{p.side === "long" ? "多" : "空"}</td>
                <td>{p.qty}</td>
                <td>{p.entry_price}</td>
                <td>{p.mark_price}</td>
                <td className={p.unrealized_pnl >= 0 ? "buy" : "sell"}>{p.unrealized_pnl.toFixed(4)}</td>
                <td>{p.leverage}x</td>
                <td>{p.liquidation_price ? p.liquidation_price.toFixed(4) : "-"}</td>
                <td>
                  <button disabled={flattening === p.symbol} onClick={() => flatten(p.symbol)}>
                    {flattening === p.symbol ? "平倉中…" : "平倉"}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
