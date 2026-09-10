import { useState } from "react";
import { api } from "../api/client";
import type { OrderSide, Settings } from "../types";

interface Props {
  symbol: string;
  settings: Settings | null;
  onFilled?: () => void;
}

// No quantity input: every order (manual or automatic) is sized
// server-side from the fixed margin-per-order setting (notional = margin *
// leverage) via the same validation path (internal/autotrader.
// PlaceManualOrder -> binance.FilterCache.MaxQtyForCap) - there's nothing
// for the user to size manually.
export function OrderPanel({ symbol, settings, onFilled }: Props) {
  const [status, setStatus] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState<OrderSide | null>(null);

  const submit = async (side: OrderSide) => {
    setSubmitting(side);
    setStatus(null);
    try {
      const order = await api.placeOrder(symbol, side);
      setStatus(`已送出：${order.side} ${order.qty} (約$${order.notional_usd.toFixed(2)})，狀態 ${order.status}`);
      onFilled?.();
    } catch (e) {
      setStatus(`錯誤：${String(e)}`);
    } finally {
      setSubmitting(null);
    }
  };

  return (
    <div className="panel">
      <h3>手動下單 - {symbol}</h3>
      <p className="muted small">
        市價單，固定投入{settings?.margin_usd ?? "?"}USDT保證金、{settings?.leverage ?? "?"}x槓桿
        {settings?.margin_type ?? ""}（約{settings ? (settings.margin_usd * settings.leverage).toFixed(2) : "?"}USDT名目倉位，
        伺服器端強制），不受自動交易開關影響。
      </p>
      <div className="order-form">
        <div className="side-toggle">
          <button className="buy" disabled={submitting !== null} onClick={() => submit("BUY")}>
            {submitting === "BUY" ? "送出中..." : "做多 (BUY)"}
          </button>
          <button className="sell" disabled={submitting !== null} onClick={() => submit("SELL")}>
            {submitting === "SELL" ? "送出中..." : "做空 (SELL)"}
          </button>
        </div>
        {status && <p className="order-status">{status}</p>}
      </div>
    </div>
  );
}
