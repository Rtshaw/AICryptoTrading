import { useEffect, useState } from "react";
import { api } from "../api/client";
import { useI18n } from "../i18n/I18nContext";
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
  const { t } = useI18n();
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
      <h3>{t("killSwitch.title")}</h3>
      {settings ? (
        <>
          <div className={settings.autotrade_enabled ? "kill-switch on" : "kill-switch off"}>
            <span>{settings.autotrade_enabled ? t("killSwitch.live") : t("killSwitch.paused")}</span>
            <button disabled={busy} onClick={toggle}>
              {settings.autotrade_enabled ? t("killSwitch.pause") : t("killSwitch.resume")}
            </button>
          </div>
          <p className="muted small">{t("killSwitch.description", { margin: settings.margin_usd })}</p>
          <label className="leverage-row">
            {t("killSwitch.marginLabel")}
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
            {t("killSwitch.leverageLabel")}
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
                {t("killSwitch.apply")}
              </button>
            </div>
          </label>
          <p className="muted small">
            {t("killSwitch.effectiveNotional", { notional: (marginInput * leverageInput).toFixed(2) })}
          </p>
        </>
      ) : (
        <p className="muted">{t("killSwitch.loading")}</p>
      )}
    </div>
  );
}
