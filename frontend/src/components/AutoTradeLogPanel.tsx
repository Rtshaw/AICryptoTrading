import type { AutoTradeLogEntry } from "../types";
import { useI18n } from "../i18n/I18nContext";

interface Props {
  entries: AutoTradeLogEntry[];
}

// Full-width transparency log: every candle-close strategy evaluation per symbol,
// including holds and skips and why - not just the ones that placed an
// order. Placed below the 3-column grid rather than in a sidebar since a
// table like this needs horizontal room.
export function AutoTradeLogPanel({ entries }: Props) {
  const { t } = useI18n();
  const decisionLabel: Record<AutoTradeLogEntry["decision"], string> = {
    executed: t("autoTradeLog.decisionExecuted"),
    skipped: t("autoTradeLog.decisionSkipped"),
    hold: t("autoTradeLog.decisionHold"),
    error: t("autoTradeLog.decisionError"),
  };

  return (
    <div className="panel autotrade-log-panel">
      <h3>{t("autoTradeLog.title")}</h3>
      {entries.length === 0 ? (
        <p className="muted">{t("autoTradeLog.empty")}</p>
      ) : (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>{t("autoTradeLog.colTime")}</th>
                <th>{t("autoTradeLog.colSymbol")}</th>
                <th>{t("autoTradeLog.colRuleAction")}</th>
                <th>{t("autoTradeLog.colRuleReason")}</th>
                <th>{t("autoTradeLog.colResult")}</th>
                <th>{t("autoTradeLog.colReason")}</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
                <tr key={e.id}>
                  <td>{new Date(e.ts).toLocaleString()}</td>
                  <td>{e.symbol}</td>
                  <td
                    className={
                      e.rule_action === "BUY" ? "buy" : e.rule_action === "SELL" ? "sell" : undefined
                    }
                  >
                    {e.rule_action}
                  </td>
                  <td className="muted small">{e.rule_reason}</td>
                  <td className={`decision-badge ${e.decision}`}>{decisionLabel[e.decision]}</td>
                  <td className="muted small">{e.skip_reason ?? "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
