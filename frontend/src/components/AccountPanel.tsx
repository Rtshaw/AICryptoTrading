import type { AccountSummary } from "../types";

interface Props {
  account: AccountSummary | null;
}

export function AccountPanel({ account }: Props) {
  return (
    <div className="panel">
      <h3>帳戶</h3>
      {!account ? (
        <p className="muted">載入中…</p>
      ) : (
        <div className="account-summary">
          <div>
            <span className="muted small">錢包餘額</span>
            <strong>{account.wallet_balance_usd.toFixed(2)} USDT</strong>
          </div>
          <div>
            <span className="muted small">可用餘額</span>
            <strong>{account.available_balance_usd.toFixed(2)} USDT</strong>
          </div>
          <div>
            <span className="muted small">未實現損益</span>
            <strong className={account.total_unrealized_pnl >= 0 ? "buy" : "sell"}>
              {account.total_unrealized_pnl.toFixed(4)} USDT
            </strong>
          </div>
        </div>
      )}
    </div>
  );
}
