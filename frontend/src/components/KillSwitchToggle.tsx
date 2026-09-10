import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { Settings } from "../types";

interface Props {
  settings: Settings | null;
  onChanged: (s: Settings) => void;
}

// The runtime kill switch: pauses automatic order EXECUTION only (the rule
// engine and AI signal generation keep running so the dashboard stays
// informative even while paused - internal/autotrader.execute checks this
// first, before any sizing/order-placement work). Manual orders and the
// position-flatten button are NOT gated by this switch.
export function KillSwitchToggle({ settings, onChanged }: Props) {
  const [busy, setBusy] = useState(false);
  const [leverageInput, setLeverageInput] = useState(settings?.leverage ?? 3);
  const [marginInput, setMarginInput] = useState(settings?.margin_usd ?? 5);

  useEffect(() => {
    if (settings) {
      setLeverageInput(settings.leverage);
      setMarginInput(settings.margin_usd);
    }
  }, [settings]);

  const toggle = async () => {
    if (!settings) return;
    setBusy(true);
    try {
      const updated = await api.updateSettings({ autotrade_enabled: !settings.autotrade_enabled });
      onChanged(updated);
    } finally {
      setBusy(false);
    }
  };

  const applySizing = async () => {
    setBusy(true);
    try {
      const updated = await api.updateSettings({ leverage: leverageInput, margin_usd: marginInput });
      onChanged(updated);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="panel">
      <h3>自動交易</h3>
      {settings ? (
        <>
          <div className={settings.autotrade_enabled ? "kill-switch on" : "kill-switch off"}>
            <span>{settings.autotrade_enabled ? "運作中（LIVE）" : "已暫停"}</span>
            <button disabled={busy} onClick={toggle}>
              {settings.autotrade_enabled ? "暫停自動下單" : "恢復自動下單"}
            </button>
          </div>
          <p className="muted small">
            暫停只影響自動下單，AI訊號與規則判斷仍持續運作並顯示。手動下單與平倉不受此開關影響。
            每筆下單固定投入{settings.margin_usd}USDT保證金，名目倉位=保證金×槓桿，隨槓桿倍數變動，
            沒有另外的名目金額上限。
          </p>
          <label className="leverage-row">
            保證金（USDT，每筆固定）
            <div className="leverage-input">
              <input
                type="number"
                min={1}
                max={50}
                step={0.5}
                value={marginInput}
                onChange={(e) => setMarginInput(Number(e.target.value))}
              />
            </div>
          </label>
          <label className="leverage-row">
            槓桿（1-25x）
            <div className="leverage-input">
              <input
                type="number"
                min={1}
                max={25}
                value={leverageInput}
                onChange={(e) => setLeverageInput(Number(e.target.value))}
              />
              <button
                disabled={busy || (leverageInput === settings.leverage && marginInput === settings.margin_usd)}
                onClick={applySizing}
              >
                套用
              </button>
            </div>
          </label>
          <p className="muted small">
            套用後名目倉位約為 {(marginInput * leverageInput).toFixed(2)} USDT。
          </p>
        </>
      ) : (
        <p className="muted">載入中…</p>
      )}
    </div>
  );
}
