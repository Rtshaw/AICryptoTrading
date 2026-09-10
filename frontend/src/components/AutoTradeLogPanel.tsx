import type { AutoTradeLogEntry } from "../types";

interface Props {
  entries: AutoTradeLogEntry[];
}

const decisionLabel: Record<AutoTradeLogEntry["decision"], string> = {
  executed: "已下單",
  skipped: "略過",
  hold: "觀望",
  error: "錯誤",
};

// Full-width transparency log: every candle-close strategy evaluation per symbol,
// including holds and skips and why - not just the ones that placed an
// order. Placed below the 3-column grid rather than in a sidebar since a
// table like this needs horizontal room.
export function AutoTradeLogPanel({ entries }: Props) {
  return (
    <div className="panel autotrade-log-panel">
      <h3>自動交易紀錄</h3>
      {entries.length === 0 ? (
        <p className="muted">尚無紀錄（每根5分K收盤都會評估一次，最新的會出現在這裡）</p>
      ) : (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>時間</th>
                <th>合約</th>
                <th>規則判斷</th>
                <th>規則理由</th>
                <th>結果</th>
                <th>原因</th>
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
