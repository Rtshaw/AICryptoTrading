import { useEffect, useState } from "react";
import { api } from "../api/client";
import { useMarketSocket } from "../api/ws";
import { useMarketStore } from "../store/useMarketStore";
import { CandleChart } from "../components/CandleChart";
import { WatchlistPanel } from "../components/WatchlistPanel";
import { OrderPanel } from "../components/OrderPanel";
import { PositionTable } from "../components/PositionTable";
import { AISignalCard } from "../components/AISignalCard";
import { AccountPanel } from "../components/AccountPanel";
import { KillSwitchToggle } from "../components/KillSwitchToggle";
import { AutoTradeLogPanel } from "../components/AutoTradeLogPanel";
import { StrategyOptimizePanel } from "../components/StrategyOptimizePanel";
import type { AccountSummary, AutoTradeLogEntry, Position, Settings } from "../types";

export function Dashboard() {
  const selectedSymbol = useMarketStore((s) => s.selectedSymbol);
  const setSelectedSymbol = useMarketStore((s) => s.setSelectedSymbol);
  const candlesBySymbol = useMarketStore((s) => s.candles);
  const setCandles = useMarketStore((s) => s.setCandles);
  const upsertCandle = useMarketStore((s) => s.upsertCandle);
  const signalsBySymbol = useMarketStore((s) => s.signals);
  const setSignals = useMarketStore((s) => s.setSignals);
  const upsertSignal = useMarketStore((s) => s.upsertSignal);

  const [positions, setPositions] = useState<Position[]>([]);
  const [account, setAccount] = useState<AccountSummary | null>(null);
  const [settings, setSettings] = useState<Settings | null>(null);
  const [log, setLog] = useState<AutoTradeLogEntry[]>([]);
  const [watchlistRefreshToken, setWatchlistRefreshToken] = useState(0);

  useEffect(() => {
    api.getCandles(selectedSymbol).then((candles) => setCandles(selectedSymbol, candles));
    api.getSignals(selectedSymbol).then((list) => setSignals(selectedSymbol, list));
  }, [selectedSymbol, setCandles, setSignals]);

  const reloadPositions = () => {
    api.getPositions().then(setPositions).catch(() => undefined);
    api.getBalance().then(setAccount).catch(() => undefined);
  };
  const reloadLog = () => {
    api.getAutoTradeLog(undefined, 100).then(setLog).catch(() => undefined);
  };

  useEffect(() => {
    reloadPositions();
    reloadLog();
    api.getSettings().then(setSettings).catch(() => undefined);
    const interval = setInterval(reloadPositions, 20000);
    return () => clearInterval(interval);
  }, []);

  useMarketSocket((msg) => {
    switch (msg.type) {
      case "candle":
        upsertCandle(msg.data as Parameters<typeof upsertCandle>[0]);
        break;
      case "signal":
        // Broadcast for every symbol (manual button, or auto-triggered by
        // internal/autotrader.Trader.OnCandleClose on a rule transition) -
        // upsertSignal is keyed by signal.symbol so this stays correct even
        // for symbols not currently selected.
        upsertSignal(msg.data as Parameters<typeof upsertSignal>[0]);
        break;
      case "position":
        setPositions(msg.data as Position[]);
        break;
      case "balance":
        setAccount(msg.data as AccountSummary);
        break;
      case "order":
        reloadPositions();
        break;
      case "autotrade_log":
        setLog((prev) => [msg.data as AutoTradeLogEntry, ...prev].slice(0, 200));
        break;
      case "settings":
        setSettings(msg.data as Settings);
        break;
    }
  });

  const candles = candlesBySymbol[selectedSymbol] ?? [];
  const signals = signalsBySymbol[selectedSymbol] ?? [];

  return (
    <>
      <div className="dashboard-grid">
        <div className="col-left">
          <WatchlistPanel
            selected={selectedSymbol}
            onSelect={setSelectedSymbol}
            refreshToken={watchlistRefreshToken}
            settings={settings}
          />
          <AccountPanel account={account} />
        </div>
        <div className="col-center">
          <div className="panel">
            <h3>{selectedSymbol} - 5分K</h3>
            <CandleChart candles={candles} signals={signals} />
          </div>
          <PositionTable positions={positions} onChanged={reloadPositions} />
        </div>
        <div className="col-right">
          <OrderPanel symbol={selectedSymbol} settings={settings} onFilled={reloadPositions} />
          <AISignalCard symbol={selectedSymbol} />
          <KillSwitchToggle
            settings={settings}
            onChanged={(s) => {
              setSettings(s);
              setWatchlistRefreshToken((t) => t + 1);
            }}
          />
        </div>
      </div>
      <StrategyOptimizePanel />
      <AutoTradeLogPanel entries={log} />
    </>
  );
}
