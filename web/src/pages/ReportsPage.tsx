import { useEffect, useMemo, useState } from "react";
import {
  Card,
  Group,
  SegmentedControl,
  SimpleGrid,
  Stack,
  Text,
  Title,
  Loader,
  Center,
} from "@mantine/core";
import { DatePickerInput } from "@mantine/dates";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";

import { dynamics, periodReport } from "../api/client";
import type { MonthPoint, PeriodSummary } from "../api/types";
import { useCategories } from "../state/categories";
import { notifyError } from "../lib/notify";
import {
  formatMoney,
  moneyToNumber,
  monthShort,
  lastDayOfMonth,
} from "../lib/format";

const EXPENSE = "#e03131";
const INCOME = "#2f9e44";

type Preset = "day" | "week" | "month" | "quarter" | "year" | "custom";

function ymOf(iso: string): string {
  return iso.slice(0, 7);
}
function isoToday(): string {
  const d = new Date();
  return d.toISOString().slice(0, 10);
}
function addDays(iso: string, days: number): string {
  const d = new Date(iso + "T00:00:00");
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}
function addMonths(iso: string, months: number): string {
  const d = new Date(iso + "T00:00:00");
  d.setMonth(d.getMonth() + months);
  return d.toISOString().slice(0, 10);
}

function presetRange(p: Preset): [string, string] {
  const today = isoToday();
  const ym = ymOf(today);
  switch (p) {
    case "day":
      return [today, today];
    case "week":
      return [addDays(today, -6), today];
    case "month":
      return [`${ym}-01`, lastDayOfMonth(ym)];
    case "quarter":
      return [`${ymOf(addMonths(today, -2))}-01`, lastDayOfMonth(ym)];
    case "year":
      return [`${ymOf(addMonths(today, -11))}-01`, lastDayOfMonth(ym)];
    default:
      return [`${ym}-01`, lastDayOfMonth(ym)];
  }
}

export function ReportsPage() {
  const { displayName } = useCategories();
  const [preset, setPreset] = useState<Preset>("month");
  const [range, setRange] = useState<[string, string]>(presetRange("month"));
  const [summary, setSummary] = useState<PeriodSummary | null>(null);
  const [points, setPoints] = useState<MonthPoint[]>([]);
  const [loading, setLoading] = useState(false);

  const [from, to] = range;

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    Promise.all([periodReport(from, to), dynamics(ymOf(from), ymOf(to))])
      .then(([s, p]) => {
        if (cancelled) return;
        setSummary(s);
        setPoints(p);
      })
      .catch(notifyError)
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [from, to]);

  function onPreset(p: Preset) {
    setPreset(p);
    if (p !== "custom") setRange(presetRange(p));
  }

  const catData = useMemo(
    () =>
      (summary?.expenseByCategory ?? []).map((c) => ({
        name: displayName(c.categoryId),
        value: moneyToNumber(c.amount),
        label: formatMoney(c.amount),
      })),
    [summary, displayName],
  );

  const dynData = useMemo(
    () =>
      points.map((p) => ({
        name: monthShort(p.period),
        Доход: moneyToNumber(p.income),
        Расход: moneyToNumber(p.expense),
      })),
    [points],
  );

  return (
    <Stack>
      <Title order={3}>Отчёты и графики</Title>

      <Group justify="space-between" wrap="wrap">
        <SegmentedControl
          value={preset}
          onChange={(v) => onPreset(v as Preset)}
          data={[
            { label: "День", value: "day" },
            { label: "Неделя", value: "week" },
            { label: "Месяц", value: "month" },
            { label: "Квартал", value: "quarter" },
            { label: "Год", value: "year" },
            { label: "Период", value: "custom" },
          ]}
        />
        {preset === "custom" && (
          <Group gap="xs">
            <DatePickerInput
              value={new Date(from)}
              onChange={(d) => d && setRange([d.toISOString().slice(0, 10), to])}
              valueFormat="DD.MM.YYYY"
              label="С"
            />
            <DatePickerInput
              value={new Date(to)}
              onChange={(d) => d && setRange([from, d.toISOString().slice(0, 10)])}
              valueFormat="DD.MM.YYYY"
              label="По"
            />
          </Group>
        )}
      </Group>

      {loading ? (
        <Center h={200}>
          <Loader />
        </Center>
      ) : (
        <>
          <SimpleGrid cols={{ base: 1, sm: 3 }}>
            <StatCard title="Доход за период" value={summary ? formatMoney(summary.totalIncome) : "—"} color={INCOME} />
            <StatCard title="Расход за период" value={summary ? formatMoney(summary.totalExpense) : "—"} color={EXPENSE} />
            <StatCard title="Баланс" value={summary ? formatMoney(summary.balance) : "—"} />
          </SimpleGrid>

          <Card withBorder padding="md">
            <Title order={5} mb="sm">Расходы по категориям (BYN)</Title>
            {catData.length === 0 ? (
              <Text c="dimmed">Нет данных за период.</Text>
            ) : (
              <ResponsiveContainer width="100%" height={Math.max(160, catData.length * 36)}>
                <BarChart data={catData} layout="vertical" margin={{ left: 24, right: 16 }}>
                  <XAxis type="number" hide />
                  <YAxis type="category" dataKey="name" width={140} tick={{ fontSize: 12 }} />
                  <Tooltip formatter={(_v, _n, p) => (p?.payload?.label ?? "")} />
                  <Bar dataKey="value" fill={EXPENSE} radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </Card>

          <Card withBorder padding="md">
            <Title order={5} mb="sm">Динамика доход/расход по месяцам (BYN)</Title>
            {dynData.length === 0 ? (
              <Text c="dimmed">Нет данных за период.</Text>
            ) : (
              <ResponsiveContainer width="100%" height={260}>
                <BarChart data={dynData} margin={{ left: 8, right: 8 }}>
                  <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                  <YAxis tick={{ fontSize: 12 }} />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="Доход" fill={INCOME} radius={[4, 4, 0, 0]} />
                  <Bar dataKey="Расход" fill={EXPENSE} radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </Card>

          {catData.length > 0 && (
            <Card withBorder padding="md">
              <Title order={5} mb="sm">Топ категорий по расходу</Title>
              <Stack gap={4}>
                {catData.slice(0, 8).map((c) => (
                  <Group key={c.name} justify="space-between">
                    <Text size="sm">{c.name}</Text>
                    <Text size="sm" fw={600}>{c.label}</Text>
                  </Group>
                ))}
              </Stack>
            </Card>
          )}
        </>
      )}
    </Stack>
  );
}

function StatCard({ title, value, color }: { title: string; value: string; color?: string }) {
  return (
    <Card withBorder padding="md">
      <Text size="xs" c="dimmed">{title}</Text>
      <Text size="xl" fw={700} c={color}>{value}</Text>
    </Card>
  );
}
