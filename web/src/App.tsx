import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import {
  AppShell,
  Burger,
  Group,
  NavLink,
  Title,
  Loader,
  Center,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import {
  IconLayoutDashboard,
  IconPlus,
  IconChartBar,
  IconCategory,
  IconRepeat,
  IconSettings,
  IconMessage,
} from "@tabler/icons-react";

import { checkAuth, getToken } from "./api/client";
import { CategoriesProvider } from "./state/categories";
import { CurrenciesProvider } from "./state/currencies";
import { TokenGate } from "./auth/TokenGate";
import { BalancePage } from "./pages/BalancePage";
import { TransactionsPage } from "./pages/TransactionsPage";
import { ReportsPage } from "./pages/ReportsPage";
import { CategoriesPage } from "./pages/CategoriesPage";
import { RecurringPage } from "./pages/RecurringPage";
import { SmsPage } from "./pages/SmsPage";
import { SettingsPage } from "./pages/SettingsPage";

type Section =
  | "balance"
  | "transactions"
  | "reports"
  | "categories"
  | "recurring"
  | "sms"
  | "settings";

const NAV: { key: Section; label: string; icon: ReactNode }[] = [
  { key: "reports", label: "Отчёты и графики", icon: <IconChartBar size={20} /> },
  { key: "balance", label: "План / Факт", icon: <IconLayoutDashboard size={20} /> },
  { key: "transactions", label: "Операции", icon: <IconPlus size={20} /> },
  { key: "recurring", label: "Повторяющиеся", icon: <IconRepeat size={20} /> },
  { key: "sms", label: "SMS-парсер", icon: <IconMessage size={20} /> },
  { key: "categories", label: "Категории", icon: <IconCategory size={20} /> },
  { key: "settings", label: "Настройки", icon: <IconSettings size={20} /> },
];

export function App() {
  const [authed, setAuthed] = useState<boolean | null>(null);
  const [section, setSection] = useState<Section>("reports");
  const [opened, { toggle, close }] = useDisclosure();

  useEffect(() => {
    if (!getToken()) {
      setAuthed(false);
      return;
    }
    checkAuth()
      .then(() => setAuthed(true))
      .catch(() => setAuthed(false));
  }, []);

  if (authed === null) {
    return (
      <Center h="100vh">
        <Loader />
      </Center>
    );
  }

  if (!authed) {
    return <TokenGate onAuthed={() => setAuthed(true)} />;
  }

  return (
    <CategoriesProvider>
      <CurrenciesProvider>
      <AppShell
        header={{ height: 56 }}
        navbar={{ width: 240, breakpoint: "sm", collapsed: { mobile: !opened } }}
        padding="md"
      >
        <AppShell.Header>
          <Group h="100%" px="md" gap="sm">
            <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
            <Title order={4}>Advisor</Title>
          </Group>
        </AppShell.Header>

        <AppShell.Navbar p="sm">
          {NAV.map((item) => (
            <NavLink
              key={item.key}
              active={section === item.key}
              label={item.label}
              leftSection={item.icon}
              onClick={() => {
                setSection(item.key);
                close();
              }}
            />
          ))}
        </AppShell.Navbar>

        <AppShell.Main>
          {section === "reports" && <ReportsPage />}
          {section === "balance" && <BalancePage />}
          {section === "transactions" && <TransactionsPage />}
          {section === "recurring" && <RecurringPage />}
          {section === "sms" && <SmsPage />}
          {section === "categories" && <CategoriesPage />}
          {section === "settings" && <SettingsPage />}
        </AppShell.Main>
      </AppShell>
      </CurrenciesProvider>
    </CategoriesProvider>
  );
}
