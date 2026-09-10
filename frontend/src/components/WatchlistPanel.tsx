import { useEffect, useState } from "react";
import { api } from "../api/client";
import { useI18n } from "../i18n/I18nContext";
import type { Settings, WatchlistAIResult, WatchlistItem } from "../types";

interface Props {
  selected: string;
  onSelect: (symbol: string) => void;
  refreshToken?: number;
  settings: Settings | null;
}

export function WatchlistPanel({ selected, onSelect, refreshToken, settings }: Props) {
  const { t } = useI18n();
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

  const sizing = settings
    ? t("watchlist.footnoteSizing", {
        margin: settings.margin_usd,
        leverage: settings.leverage,
        notional: settings.effective_notional_usd.toFixed(2),
      })
    : t("watchlist.footnoteSizingFallback");

  return (
    <div className="panel">
      <h3>{t("watchlist.title")}</h3>
      {error && <p className="error">{error}</p>}
      <ul className="watchlist">
        {items.map((item) => {
          const aiPick = aiPickBySymbol.get(item.symbol);
          const title = aiPick
            ? t("watchlist.aiRationalePrefix", { count: aiPick.recent_signal_count, rationale: aiPick.rationale })
            : item.tradable_at_cap
              ? t("watchlist.tradableTooltip", { qty: item.max_qty_at_cap ?? "-" })
              : t("watchlist.chartOnlyTooltip");
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
                  {item.tradable_at_cap ? t("watchlist.tradable") : t("watchlist.chartOnly")}
                </span>
              </button>
              <button className="remove-btn" onClick={() => remove(item.symbol)} title={t("watchlist.remove")}>
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
          placeholder={t("watchlist.addPlaceholder")}
          onKeyDown={(e) => e.key === "Enter" && add()}
        />
        <button onClick={add}>{t("watchlist.add")}</button>
      </div>
      <div className="add-row">
        <button onClick={refreshAI} disabled={aiLoading}>
          {aiLoading ? t("watchlist.aiRefreshing") : t("watchlist.aiRefresh")}
        </button>
        {aiResult?.run_date && (
          <span className="muted small">
            {t("watchlist.lastRefresh", { date: aiResult.run_date, count: aiResult.picks.length })}
          </span>
        )}
      </div>
      <p className="muted small">{t("watchlist.footnote", { sizing })}</p>
    </div>
  );
}
