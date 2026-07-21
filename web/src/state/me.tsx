import { createContext, useContext, useEffect, useState } from "react";
import type { ReactNode } from "react";
import { me as fetchMe } from "../api/client";
import type { Me } from "../api/types";

const Ctx = createContext<Me>({ username: "", role: "user", isAdmin: false });

export function MeProvider({ children }: { children: ReactNode }) {
  const [me, setMe] = useState<Me>({ username: "", role: "user", isAdmin: false });
  useEffect(() => {
    fetchMe().then(setMe).catch(() => {});
  }, []);
  return <Ctx.Provider value={me}>{children}</Ctx.Provider>;
}

export function useMe(): Me {
  return useContext(Ctx);
}
