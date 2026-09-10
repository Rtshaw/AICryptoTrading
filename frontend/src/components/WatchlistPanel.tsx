import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { Settings, WatchlistAIResult, WatchlistItem } from "../types";

interface Props {
  selected: string;
  onSelect: (symbol: string) => void;
  refreshToken?: number;
  settings: Settings | null;
}

export function WatchlistPanel({ selected, onSelect, refreshToken, settings }: Props) {
  const [items, setItems] = useState<WatchlistItem[]>([]);
  const [newSymbol, setNewSymbol] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [aiResult, setAiResult] = useState<WatchlistAIResult | null>(null);
  const [aiLoading, setAiLoading] = useState(false);

  const reload = () => {
    api.getWatchlist().then(setItems).catch((e) => setError(String(e)));
  };

  useEffect(reload, [refreshToken]);
  useEffect(() => {
    const interval = setInterval(reload, 15000);
    return () => clearInterval(interval);
  }, []);
  useEffect(() => {
    api.getLatestWatchlistAI().then(setAiResult).catch(() => {});
  }, [refreshToken]);

  const add = async () => {
    if (!newSymbol.trim()) return;
    try {
      await api.addWatchlist(newSymbol.trim().toUpperCase());
      setNewSymbol("");
      reload();
    } catch (e) {
      setError(String(e));
    }
  };

  const remove = async (symbol: string) => {
    try {
      await api.removeWatchlist(symbol);
      reload();
    } catch (e) {
      setError(String(e));
    }
  };

  const refreshAI = async () => {
    setAiLoading(true);
    setError(null);
    try {
      const result = await api.refreshWatchlistAI();
      setAiResult(result);
      reload();
    } catch (e) {
      setError(String(e));
    } finally {
      setAiLoading(false);
    }
  };

  const aiPickBySymbol = new Map((aiResult?.picks ?? []).map((p) => [p.symbol, p]));

  return (
    <div className="panel">
      <h3>關注清單</h3>
      {error && <p className="error">{error}</p>}
      <ul className="watchlist">
        {items.map((item) => {
          const aiPick = aiPickBySymbol.get(item.symbol);
          const title = aiPick
            ? `AI選股理由（近期訊號次數：${aiPick.recent_signal_count}）：${aiPick.rationale}`
            : item.tradable_at_cap
              ? `目前設定可下單，約${item.max_qty_at_cap}顆`
              : "以目前設定的保證金×槓桿換算，低於交易所最小下單量/名目金額限制，無法自動或手動下單（僅供看盤/訊號參考）";
          return (
            <li key={item.symbol}>
              <button
                className={item.symbol === selected ? "watchlist-item selected" : "watchlist-item"}
                onClick={() => onSelect(item.symbol)}
                title={title}
              >
                <span>{item.symbol}</span>
                {item.price != null && <span className="watchlist-price">{item.price}</span>}
                {aiPick && <span className="ai-pick-badge">AI×{aiPick.recent_signal_count}</span>}
                <span className={item.tradable_at_cap ? "feasible-badge yes" : "feasible-badge no"}>
                  {item.tradable_at_cap ? "可下單" : "僅看盤"}
                </span>
              </button>
              <button className="remove-btn" onClick={() => remove(item.symbol)} title="移除">
                ×
              </button>
            </li>
          );
        })}
      </ul>
      <div className="add-row">
        <input
          value={newSymbol}
          onChange={(e) => setNewSymbol(e.target.value)}
          placeholder="合約代號 例如 SOLUSDT"
          onKeyDown={(e) => e.key === "Enter" && add()}
        />
        <button onClick={add}>新增</button>
      </div>
      <div className="add-row">
        <button onClick={refreshAI} disabled={aiLoading}>
          {aiLoading ? "AI選股中…" : "AI每日選股"}
        </button>
        {aiResult?.run_date && <span className="muted small">上次選股：{aiResult.run_date}（{aiResult.picks.length}檔）</span>}
      </div>
      <p className="muted small">
        「僅看盤」代表該合約在目前設定（{settings ? `${settings.margin_usd}USDT保證金×${settings.leverage}x槓桿≈${settings.effective_notional_usd.toFixed(2)}USDT` : "保證金×槓桿"}
        ）下，交易所最小下單量/名目金額限制無法達成（例如BTCUSDT/ETHUSDT），系統只會顯示K線與AI訊號，不會自動或手動下單。
      </p>
    </div>
  );
}
