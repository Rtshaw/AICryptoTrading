import { Dashboard } from "./pages/Dashboard";

export default function App() {
  return (
    <div className="app">
      <header className="app-header">
        <h1>永續合約 AI 交易 Dashboard</h1>
        <span className="muted small">Binance USDS-M Perpetual Futures · 5分K · 每筆固定保證金</span>
      </header>
      <main>
        <Dashboard />
      </main>
    </div>
  );
}
