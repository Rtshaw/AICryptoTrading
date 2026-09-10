import type { AccountSummary } from "../types";
import { useI18n } from "../i18n/I18nContext";

interface Props {
  account: AccountSummary | null;
}

export function AccountPanel({ account }: Props) {
  const { t } = useI18n();

  return (
    <div className="panel">
      <h3>{t("account.title")}</h3>
      {!account ? (
        <p className="muted">{t("account.loading")}</p>
      ) : (
        <div className="account-summary">
          <div>
            <span className="muted small">{t("account.walletBalance")}</span>
            <strong>{account.wallet_balance_usd.toFixed(2)} USDT</strong>
          </div>
          <div>
            <span className="muted small">{t("account.availableBalance")}</span>
            <strong>{account.available_balance_usd.toFixed(2)} USDT</strong>
          </div>
          <div>
            <span className="muted small">{t("account.unrealizedPnl")}</span>
            <strong className={account.total_unrealized_pnl >= 0 ? "buy" : "sell"}>
              {account.total_unrealized_pnl.toFixed(4)} USDT
            </strong>
          </div>
        </div>
      )}
    </div>
  );
}
