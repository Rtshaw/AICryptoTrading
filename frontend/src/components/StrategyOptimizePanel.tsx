import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { BacktestMetrics, OptimizeReport, SBParams } from "../types";

function MetricsRow({ label, m }: { label: string; m: BacktestMetrics | null | undefined }) {
  if (!m) return null;
  return (
    <tr>
      <td>{label}</td>
      <td>{m.total_trades}</td>
      <td>{m.win_rate.toFixed(1)}%</td>
      <td className={m.total_return_pct >= 0 ? "buy" : "sell"}>{m.total_return_pct.toFixed(2)}%</td>
      <td>{m.sharpe.toFixed(2)}</td>
      <td>{m.max_drawdown_pct.toFixed(2)}%</td>
    </tr>
  );
}

function tunedParamsSummary(p: SBParams): string {
  return `擺幅回看${p.swing_lookback}根 / FVG≥${p.min_fvg_size_pct}% / sweep於${p.max_bars_for_sweep}根內 / 停損緩衝${p.stop_buffer_pct}% / 風報比${p.risk_reward_ratio}`;
}

function fixedRulesSummary(p: SBParams): string {
  return `位移實體≥${(p.min_displacement_body_pct * 100).toFixed(0)}% / OTE${(p.ote_min_retrace * 100).toFixed(0)}-${(p.ote_max_retrace * 100).toFixed(0)}% / Breaker重疊${p.require_breaker_confluence ? "必要" : "非必要"} / SMT背離${p.require_smt_divergence ? "必要" : "非必要"}`;
}

// Manually-triggered backtest + grid-search optimization (internal/backtest.Optimize)
// for the Silver Bullet strategy's parameters - see internal/autotrader for
// how the live pipeline picks up whatever this saves. One-time tuning per
// the user's explicit choice, not a recurring scheduled job.
export function StrategyOptimizePanel() {
  const [current, setCurrent] = useState<{ params: SBParams; train: BacktestMetrics | null; validation: BacktestMetrics | null; optimizedAt: string | null } | null>(null);
  const [report, setReport] = useState<OptimizeReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = () => {
    api
      .getStrategyParams()
      .then((res) =>
        setCurrent({ params: res.params, train: res.train_metrics, validation: res.validation_metrics, optimizedAt: res.optimized_at }),
      )
      .catch(() => undefined);
  };
  useEffect(reload, []);

  const runOptimize = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.optimizeStrategy();
      setReport(res);
      reload();
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="panel">
      <h3>Silver Bullet 策略調參</h3>
      <p className="muted small">
        用過去60天的歷史K線回測，網格搜尋參數組合，依「驗證期」(最近30%資料，模型從未拿來調參)的Sharpe排序——只有驗證期也表現夠好的參數組合才會被採用，避免overfitting。這是手動觸發的一次性動作，不會自動排程重跑。
      </p>
      <button disabled={loading} onClick={runOptimize}>
        {loading ? "回測+調參中（可能需要一段時間）..." : "執行回測並自動調參"}
      </button>
      {error && <p className="error">{error}</p>}

      {current && (
        <div style={{ marginTop: 12 }}>
          <p className="muted small">
            目前套用中的參數{current.optimizedAt ? `（${new Date(current.optimizedAt).toLocaleString()}調參）` : "（預設值，尚未調參）"}：
            <br />
            {tunedParamsSummary(current.params)}
            <br />
            （ICT2026固定規則：{fixedRulesSummary(current.params)}）
          </p>
          {(current.train || current.validation) && (
            <div className="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>期間</th>
                    <th>交易次數</th>
                    <th>勝率</th>
                    <th>總報酬</th>
                    <th>Sharpe</th>
                    <th>最大回撤</th>
                  </tr>
                </thead>
                <tbody>
                  <MetricsRow label="訓練期" m={current.train} />
                  <MetricsRow label="驗證期" m={current.validation} />
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {report && (
        <div style={{ marginTop: 12 }}>
          <p className="muted small">
            本次評估了{report.candidates_evaluated}組參數，{report.candidates_passed}組通過驗證期門檻（交易次數≥5且Sharpe&gt;0）。
            切分時間點：{new Date(report.split_time).toLocaleString()}
          </p>
        </div>
      )}
    </div>
  );
}
