import { Dashboard } from "./pages/Dashboard";
import { useI18n } from "./i18n/I18nContext";

export default function App() {
  const { lang, setLang, t } = useI18n();

  return (
    <div className="app">
      <header className="app-header">
        <div>
          <h1>{t("app.title")}</h1>
          <span className="muted small">{t("app.subtitle")}</span>
        </div>
        <button className="lang-toggle" onClick={() => setLang(lang === "zh-TW" ? "en" : "zh-TW")}>
          {t("app.langToggle")}
        </button>
      </header>
      <main>
        <Dashboard />
      </main>
    </div>
  );
}
