import { useState } from "react";
import {
  Button,
  Card,
  Group,
  Select,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import { DatePickerInput } from "@mantine/dates";
import {
  IconDownload,
  IconLogout,
  IconRefresh,
} from "@tabler/icons-react";

import { apiUrl, getToken, clearToken, logout, refreshRates, setDefaultCurrency } from "../api/client";
import { useCurrencies } from "../state/currencies";
import { notifyError, notifyOk } from "../lib/notify";
import { currentMonth, firstDayOfMonth, lastDayOfMonth } from "../lib/format";

// downloadAuthed скачивает файл с эндпоинта API, передавая Bearer-токен.
async function downloadAuthed(path: string, filename: string) {
  const res = await fetch(apiUrl(path), { headers: { Authorization: `Bearer ${getToken()}` } });
  if (!res.ok) throw new Error(`Ошибка ${res.status}`);
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function SettingsPage() {
  const { codes, defaultCurrency, refresh } = useCurrencies();
  const [cur, setCur] = useState(defaultCurrency);
  const [csvRange, setCsvRange] = useState<[string, string]>([
    firstDayOfMonth(currentMonth()),
    lastDayOfMonth(currentMonth()),
  ]);

  async function saveCurrency() {
    try {
      await setDefaultCurrency(cur);
      await refresh();
      notifyOk("Валюта по умолчанию сохранена");
    } catch (e) {
      notifyError(e);
    }
  }

  async function updateRates() {
    try {
      const { updated } = await refreshRates();
      notifyOk(`Курсы обновлены (${updated})`);
    } catch {
      notifyError(new Error("Не удалось обновить курсы (нет сети). Приложение работает на кэше."));
    }
  }

  async function exportJSON() {
    try {
      await downloadAuthed("/api/export/json", "advisor-export.json");
    } catch (e) {
      notifyError(e);
    }
  }

  async function exportCSV() {
    try {
      await downloadAuthed(
        `/api/export/csv?from=${csvRange[0]}&to=${csvRange[1]}`,
        "advisor-transactions.csv",
      );
    } catch (e) {
      notifyError(e);
    }
  }

  return (
    <Stack maw={520}>
      <Title order={3}>Настройки</Title>

      <Card withBorder padding="md">
        <Title order={5} mb="xs">Валюта по умолчанию для ввода</Title>
        <Group>
          <Select data={codes.length ? codes : ["BYN"]} value={cur} onChange={(v) => v && setCur(v)} allowDeselect={false} w={140} />
          <Button onClick={saveCurrency}>Сохранить</Button>
        </Group>
      </Card>

      <Card withBorder padding="md">
        <Title order={5} mb="xs">Курсы валют</Title>
        <Text size="sm" c="dimmed" mb="sm">
          Курсы берутся из API Национального банка РБ и кэшируются на сервере.
        </Text>
        <Button variant="light" leftSection={<IconRefresh size={16} />} onClick={updateRates}>
          Обновить курсы валют
        </Button>
      </Card>

      <Card withBorder padding="md">
        <Title order={5} mb="xs">Экспорт данных</Title>
        <Stack>
          <Button variant="light" leftSection={<IconDownload size={16} />} onClick={exportJSON} w="fit-content">
            Экспорт снапшота (JSON)
          </Button>
          <Group align="flex-end" gap="xs" wrap="wrap">
            <DatePickerInput label="С" value={new Date(csvRange[0])} onChange={(d) => d && setCsvRange([d.toISOString().slice(0, 10), csvRange[1]])} valueFormat="DD.MM.YYYY" />
            <DatePickerInput label="По" value={new Date(csvRange[1])} onChange={(d) => d && setCsvRange([csvRange[0], d.toISOString().slice(0, 10)])} valueFormat="DD.MM.YYYY" />
            <Button variant="light" leftSection={<IconDownload size={16} />} onClick={exportCSV}>
              Экспорт операций (CSV)
            </Button>
          </Group>
        </Stack>
      </Card>

      <Card withBorder padding="md">
        <Button color="red" variant="light" leftSection={<IconLogout size={16} />} onClick={async () => { try { await logout(); } catch { /* всё равно выходим */ } clearToken(); location.reload(); }}>
          Выйти
        </Button>
      </Card>
    </Stack>
  );
}
