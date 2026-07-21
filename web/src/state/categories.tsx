import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import type { ReactNode } from "react";
import type { Category } from "../api/types";
import { listCategories } from "../api/client";
import { notifyError } from "../lib/notify";

interface CategoriesCtx {
  all: Category[]; // включая архивные
  byId: Map<string, Category>;
  displayName: (id: string) => string;
  refresh: () => Promise<void>;
}

const Ctx = createContext<CategoriesCtx | null>(null);

export function CategoriesProvider({ children }: { children: ReactNode }) {
  const [all, setAll] = useState<Category[]>([]);

  const refresh = useCallback(async () => {
    try {
      setAll(await listCategories(true));
    } catch (e) {
      notifyError(e);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const byId = useMemo(() => {
    const m = new Map<string, Category>();
    for (const c of all) m.set(c.id, c);
    return m;
  }, [all]);

  const displayName = useCallback(
    (id: string): string => {
      const c = byId.get(id);
      if (!c) return id;
      if (!c.parentId) return c.name;
      const parent = byId.get(c.parentId);
      return parent ? `${parent.name} / ${c.name}` : c.name;
    },
    [byId],
  );

  const value = useMemo(
    () => ({ all, byId, displayName, refresh }),
    [all, byId, displayName, refresh],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useCategories(): CategoriesCtx {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useCategories вне CategoriesProvider");
  return ctx;
}
