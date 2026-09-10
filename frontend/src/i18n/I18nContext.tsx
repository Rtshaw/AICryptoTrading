import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { translations, type Lang } from "./translations";

const STORAGE_KEY = "lang";

function loadInitialLang(): Lang {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "en" || stored === "zh-TW") return stored;
  } catch {
    // localStorage unavailable (private browsing etc.) - fall through to default
  }
  return "zh-TW";
}

type TranslateFn = (key: keyof (typeof translations)["zh-TW"], vars?: Record<string, string | number>) => string;

interface I18nContextValue {
  lang: Lang;
  setLang: (lang: Lang) => void;
  t: TranslateFn;
}

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(loadInitialLang);

  const setLang = useCallback((next: Lang) => {
    setLangState(next);
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // best-effort persistence only
    }
  }, []);

  const t = useCallback<TranslateFn>(
    (key, vars) => {
      const template = translations[lang][key] ?? translations["zh-TW"][key] ?? String(key);
      if (!vars) return template;
      return Object.entries(vars).reduce(
        (acc, [k, v]) => acc.split(`{{${k}}}`).join(String(v)),
        template,
      );
    },
    [lang],
  );

  const value = useMemo(() => ({ lang, setLang, t }), [lang, setLang, t]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}
