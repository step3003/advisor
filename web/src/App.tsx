import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import {
  AppShell,
  Box,
  Center,
  Drawer,
  Group,
  Loader,
  NavLink,
  Text,
  Title,
  UnstyledButton,
} from "@mantine/core";
import { useDisclosure, useMediaQuery } from "@mantine/hooks";
import {
  IconLayoutDashboard,
  IconPlus,
  IconChartBar,
  IconCategory,
  IconRepeat,
  IconSettings,
  IconMessage,
  IconDotsCircleHorizontal,
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

interface NavItem {
  key: Section;
  label: string;
  short: string;
  icon: ReactNode;
}

const NAV: NavItem[] = [
  { key: "reports", label: "Отчёты и графики", short: "Отчёты", icon: <IconChartBar size={22} /> },
  { key: "balance", label: "План / Факт", short: "План/Факт", icon: <IconLayoutDashboard size={22} /> },
  { key: "transactions", label: "Операции", short: "Операции", icon: <IconPlus size={22} /> },
  { key: "recurring", label: "Повторяющиеся", short: "Повтор.", icon: <IconRepeat size={22} /> },
  { key: "sms", label: "SMS-парсер", short: "SMS", icon: <IconMessage size={22} /> },
  { key: "categories", label: "Категории", short: "Категории", icon: <IconCategory size={22} /> },
  { key: "settings", label: "Настройки", short: "Настройки", icon: <IconSettings size={22} /> },
];

// На телефоне в нижней панели — 3 основных раздела + «Ещё».
const PRIMARY: Section[] = ["reports", "balance", "transactions"];

function pageFor(section: Section): ReactNode {
  switch (section) {
    case "reports": return <ReportsPage />;
    case "balance": return <BalancePage />;
    case "transactions": return <TransactionsPage />;
    case "recurring": return <RecurringPage />;
    case "sms": return <SmsPage />;
    case "categories": return <CategoriesPage />;
    case "settings": return <SettingsPage />;
  }
}

export function App() {
  const [authed, setAuthed] = useState<boolean | null>(null);
  const [section, setSection] = useState<Section>("reports");
  const isMobile = useMediaQuery("(max-width: 48em)");

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
    return <Center h="100dvh"><Loader /></Center>;
  }
  if (!authed) {
    return <TokenGate onAuthed={() => setAuthed(true)} />;
  }

  const content = (
    <CategoriesProvider>
      <CurrenciesProvider>
        {isMobile
          ? <MobileShell section={section} setSection={setSection} />
          : <DesktopShell section={section} setSection={setSection} />}
      </CurrenciesProvider>
    </CategoriesProvider>
  );
  return content;
}

// --- Десктоп: боковая панель ---
function DesktopShell({ section, setSection }: { section: Section; setSection: (s: Section) => void }) {
  return (
    <AppShell header={{ height: 56 }} navbar={{ width: 240, breakpoint: "sm" }} padding="md">
      <AppShell.Header>
        <Group h="100%" px="md">
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
            onClick={() => setSection(item.key)}
          />
        ))}
      </AppShell.Navbar>
      <AppShell.Main>{pageFor(section)}</AppShell.Main>
    </AppShell>
  );
}

// --- Телефон: контент + нижняя панель вкладок ---
function MobileShell({ section, setSection }: { section: Section; setSection: (s: Section) => void }) {
  const [moreOpen, more] = useDisclosure(false);
  const active = NAV.find((n) => n.key === section)!;

  function go(s: Section) {
    setSection(s);
    more.close();
  }

  return (
    <Box style={{ display: "flex", flexDirection: "column", height: "100dvh" }}>
      <Box
        component="header"
        px="md"
        style={{
          display: "flex",
          alignItems: "center",
          borderBottom: "1px solid var(--mantine-color-default-border)",
          paddingTop: "env(safe-area-inset-top)",
          height: "calc(52px + env(safe-area-inset-top))",
        }}
      >
        <Title order={4}>{active.short}</Title>
      </Box>

      <Box style={{ flex: 1, overflowY: "auto", WebkitOverflowScrolling: "touch" }} p="sm" pb={80}>
        {pageFor(section)}
      </Box>

      <Box
        component="nav"
        style={{
          display: "flex",
          borderTop: "1px solid var(--mantine-color-default-border)",
          background: "var(--mantine-color-body)",
          paddingBottom: "env(safe-area-inset-bottom)",
        }}
      >
        {PRIMARY.map((key) => {
          const item = NAV.find((n) => n.key === key)!;
          return <TabButton key={key} icon={item.icon} label={item.short} active={section === key} onClick={() => go(key)} />;
        })}
        <TabButton
          icon={<IconDotsCircleHorizontal size={22} />}
          label="Ещё"
          active={!PRIMARY.includes(section)}
          onClick={more.open}
        />
      </Box>

      <Drawer opened={moreOpen} onClose={more.close} position="bottom" title="Разделы" size="auto">
        {NAV.filter((n) => !PRIMARY.includes(n.key)).map((item) => (
          <NavLink
            key={item.key}
            active={section === item.key}
            label={item.label}
            leftSection={item.icon}
            onClick={() => go(item.key)}
          />
        ))}
      </Drawer>
    </Box>
  );
}

function TabButton({ icon, label, active, onClick }: { icon: ReactNode; label: string; active: boolean; onClick: () => void }) {
  return (
    <UnstyledButton
      onClick={onClick}
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        gap: 2,
        padding: "8px 0 10px",
        color: active ? "var(--mantine-color-blue-6)" : "var(--mantine-color-dimmed)",
      }}
    >
      {icon}
      <Text fw={active ? 600 : 400} style={{ fontSize: 10 }}>{label}</Text>
    </UnstyledButton>
  );
}
