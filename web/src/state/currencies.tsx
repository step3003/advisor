import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import type { ReactNode } from "react";
import { getSettings, listCurrencies } from "../api/client";
import type { CurrencyInfo } from "../api/types";
import { notifyError } from "../lib/notify";

interface CurrenciesCtx {
  currencies: CurrencyInfo[];
  codes: string[];
  defaultCurrency: string;
  refresh: () => Promise<void>;
}

const Ctx = createContext<CurrenciesCtx | null>(null);

export function CurrenciesProvider({ children }: { children: ReactNode }) {
  const [currencies, setCurrencies] = useState<CurrencyInfo[]>([]);
  const [defaultCurrency, setDefault] = useState("BYN");

  const refresh = useCallback(async () => {
    try {
      const [list, settings] = await Promise.all([listCurrencies(), getSettings()]);
      setCurrencies(list);
      setDefault(settings.defaultCurrency || "BYN");
    } catch (e) {
      notifyError(e);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const value = useMemo(
    () => ({
      currencies,
      codes: currencies.map((c) => c.code),
      defaultCurrency,
      refresh,
    }),
    [currencies, defaultCurrency, refresh],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useCurrencies(): CurrenciesCtx {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useCurrencies вне CurrenciesProvider");
  return ctx;
}
