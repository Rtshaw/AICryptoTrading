import { useState } from "react";
import { api } from "../api/client";
import { useMarketStore } from "../store/useMarketStore";
import { useI18n } from "../i18n/I18nContext";
import type { AISignal } from "../types";

interface Props {
  symbol: string;
}

export function AISignalCard({ symbol }: Props) {
  const { t } = useI18n();
  const actionLabel: Record<AISignal["action"], string> = {
    BUY: t("aiSignal.actionBuy"),
    SELL: t("aiSignal.actionSell"),
    HOLD: t("aiSignal.actionHold"),
  };
  // Sourced from the shared store (not local state) so an auto-triggered
  // signal from the backend's rule-based watcher shows up here live via the
  // WS "signal" broadcast, the same as a manually-requested one.
  const signal = useMarketStore((s) => s.latestSignal[symbol]) ?? null;
  const upsertSignal = useMarketStore((s) => s.upsertSignal);
  const [notConfigured, setNotConfigured] = useState(false);
  const [skippedReason, setSkippedReason] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const requestSignal = async () => {
    setLoading(true);
    setError(null);
    setSkippedReason(null);
    try {
      const res = await api.generateSignal(symbol);
      if (!res.configured) {
        setNotConfigured(true);
        return;
      }
      if (res.skipped_reason) {
        // Rule currently reads HOLD - same cost-saving gate the automatic
        // engine uses (internal/autotrader never calls AI on HOLD either),
        // just surfaced here so a manual click doesn't look like it did
        // nothing.
        setSkippedReason(res.skipped_reason);
        return;
      }
      // Update the store directly from the HTTP response rather than
      // waiting on the WS broadcast - keeps this working even if the socket
      // happens to be mid-reconnect. upsertSignal dedups by id so a
      // duplicate landing from both paths is harmless.
      if (res.signal) upsertSignal(res.signal);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="panel">
      <h3>{t("aiSignal.title", { symbol })}</h3>
      {notConfigured && <p className="warn">{t("aiSignal.notConfigured")}</p>}
      {skippedReason && <p className="muted small">{skippedReason}</p>}
      {error && <p className="error">{error}</p>}

      <div className="ai-section">
        <div className="ai-header">
          <span>{t("aiSignal.header")}</span>
          <button disabled={loading} onClick={requestSignal}>
            {loading ? t("aiSignal.generating") : t("aiSignal.generate")}
          </button>
        </div>
        {signal ? (
          <div className={`signal-badge ${signal.action.toLowerCase()}`}>
            <strong>{actionLabel[signal.action]}</strong>
            <span> {t("aiSignal.confidence", { pct: (signal.confidence * 100).toFixed(0) })}</span>
            {signal.entry_hint != null && <span> {t("aiSignal.entry", { price: signal.entry_hint })}</span>}
            {signal.stop_loss != null && <span> {t("aiSignal.stop", { price: signal.stop_loss })}</span>}
            {signal.take_profit != null && <span> {t("aiSignal.target", { price: signal.take_profit })}</span>}
            {signal.funding_rate != null && (
              <span> {t("aiSignal.fundingRate", { pct: (signal.funding_rate * 100).toFixed(4) })}</span>
            )}
            <p className="rationale">{signal.rationale}</p>
          </div>
        ) : (
          <p className="muted">{t("aiSignal.noSignal")}</p>
        )}
      </div>
    </div>
  );
}
