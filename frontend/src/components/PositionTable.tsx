import { useState } from "react";
import { api } from "../api/client";
import { useI18n } from "../i18n/I18nContext";
import type { Position } from "../types";

interface Props {
  positions: Position[];
  onChanged?: () => void;
}

export function PositionTable({ positions, onChanged }: Props) {
  const { t } = useI18n();
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
      <h3>{t("position.title")}</h3>
      {error && <p className="error">{error}</p>}
      {positions.length === 0 ? (
        <p className="muted">{t("position.empty")}</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>{t("position.colSymbol")}</th>
              <th>{t("position.colSide")}</th>
              <th>{t("position.colQty")}</th>
              <th>{t("position.colEntry")}</th>
              <th>{t("position.colMark")}</th>
              <th>{t("position.colPnl")}</th>
              <th>{t("position.colLeverage")}</th>
              <th>{t("position.colLiq")}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {positions.map((p) => (
              <tr key={p.symbol}>
                <td>{p.symbol}</td>
                <td className={p.side === "long" ? "buy" : "sell"}>
                  {p.side === "long" ? t("position.sideLong") : t("position.sideShort")}
                </td>
                <td>{p.qty}</td>
                <td>{p.entry_price}</td>
                <td>{p.mark_price}</td>
                <td className={p.unrealized_pnl >= 0 ? "buy" : "sell"}>{p.unrealized_pnl.toFixed(4)}</td>
                <td>{p.leverage}x</td>
                <td>{p.liquidation_price ? p.liquidation_price.toFixed(4) : "-"}</td>
                <td>
                  <button disabled={flattening === p.symbol} onClick={() => flatten(p.symbol)}>
                    {flattening === p.symbol ? t("position.flattening") : t("position.flatten")}
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
