import { useEffect, useState } from "react";
import {
  Button,
  Card,
  Group,
  Modal,
  SegmentedControl,
  SimpleGrid,
  Stack,
  Table,
  Text,
  TextInput,
  Title,
  Loader,
  Center,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconCopy, IconPlus } from "@tabler/icons-react";

import { copyPreviousPlan, planVsFact, setPlan } from "../api/client";
import type { EntryType, Money, PlanVsFact } from "../api/types";
import { MonthNav } from "../components/MonthNav";
import { CategorySelect } from "../components/CategorySelect";
import { MoneyInput } from "../components/MoneyInput";
import { useCategories } from "../state/categories";
import { useCurrencies } from "../state/currencies";
import { notifyError, notifyOk } from "../lib/notify";
import { formatMoney, formatSigned, currentMonth } from "../lib/format";

export function BalancePage() {
  const { displayName } = useCategories();
  const { defaultCurrency } = useCurrencies();
  const [ym, setYm] = useState(currentMonth());
  const [pvf, setPvf] = useState<PlanVsFact | null>(null);
  const [loading, setLoading] = useState(false);
  const [opened, { open, close }] = useDisclosure(false);

  // Форма плана.
  const [type, setType] = useState<EntryType>("expense");
  const [catId, setCatId] = useState<string | null>(null);
  const [money, setMoney] = useState<Money>({ amount: "", currency: defaultCurrency });
  const [note, setNote] = useState("");

  async function load() {
    setLoading(true);
    try {
      setPvf(await planVsFact(ym));
    } catch (e) {
      notifyError(e);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ym]);

  function openPlan() {
    setType("expense");
    setCatId(null);
    setMoney({ amount: "", currency: defaultCurrency });
    setNote("");
    open();
  }

  async function savePlan() {
    if (!catId) {
      notifyError(new Error("Выберите категорию"));
      return;
    }
    try {
      await setPlan(ym, catId, money, note);
      close();
      await load();
      notifyOk("План сохранён");
    } catch (e) {
      notifyError(e);
    }
  }

  async function copyPrev() {
    try {
      const { copied } = await copyPreviousPlan(ym);
      await load();
      notifyOk(`Скопировано позиций: ${copied}`);
    } catch (e) {
      notifyError(e);
    }
  }

  return (
    <Stack>
      <Group justify="space-between" wrap="wrap">
        <Title order={3}>План / Факт</Title>
        <MonthNav value={ym} onChange={setYm} />
      </Group>

      <Group>
        <Button leftSection={<IconPlus size={16} />} onClick={openPlan}>
          Задать план
        </Button>
        <Button variant="default" leftSection={<IconCopy size={16} />} onClick={copyPrev}>
          Копировать из прошлого месяца
        </Button>
      </Group>

      {loading ? (
        <Center h={160}><Loader /></Center>
      ) : (
        <>
          <Card withBorder padding={0}>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Категория</Table.Th>
                  <Table.Th ta="right">План</Table.Th>
                  <Table.Th ta="right">Факт</Table.Th>
                  <Table.Th ta="right">Отклонение</Table.Th>
                  <Table.Th ta="right">%</Table.Th>
                  <Table.Th ta="right">Осталось</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {(pvf?.rows ?? []).map((r) => (
                  <Table.Tr key={r.categoryId}>
                    <Table.Td>
                      {displayName(r.categoryId)}
                      {r.overspent && <Text span c="red"> ⚠</Text>}
                    </Table.Td>
                    <Table.Td ta="right">{formatMoney(r.plan)}</Table.Td>
                    <Table.Td ta="right">{formatMoney(r.fact)}</Table.Td>
                    <Table.Td ta="right" c={devColor(r.deviation.amount)}>
                      {formatSigned(r.deviation)}
                    </Table.Td>
                    <Table.Td ta="right">{r.percent}%</Table.Td>
                    <Table.Td ta="right">{formatMoney(r.remaining)}</Table.Td>
                  </Table.Tr>
                ))}
                {pvf && pvf.rows.length === 0 && (
                  <Table.Tr>
                    <Table.Td colSpan={6}>
                      <Text c="dimmed" ta="center" py="md">
                        Нет данных за месяц. Задайте план или внесите факт.
                      </Text>
                    </Table.Td>
                  </Table.Tr>
                )}
              </Table.Tbody>
            </Table>
          </Card>

          {pvf && (
            <SimpleGrid cols={{ base: 2, sm: 3 }}>
              <Totals label="Плановый расход" value={formatMoney(pvf.planExpense)} />
              <Totals label="Фактический расход" value={formatMoney(pvf.factExpense)} />
              <Totals label="Осталось до конца месяца" value={formatMoney(pvf.remainingExpense)} />
              <Totals label="Плановый доход" value={formatMoney(pvf.planIncome)} />
              <Totals label="Фактический доход" value={formatMoney(pvf.factIncome)} />
              <Totals label="Остаток (доход − расход)" value={formatMoney(pvf.balance)} strong />
            </SimpleGrid>
          )}
        </>
      )}

      <Modal opened={opened} onClose={close} title="Задать план категории">
        <Stack>
          <SegmentedControl
            value={type}
            onChange={(v) => { setType(v as EntryType); setCatId(null); }}
            data={[
              { label: "Расход", value: "expense" },
              { label: "Доход", value: "income" },
            ]}
            fullWidth
          />
          <CategorySelect type={type} value={catId} onChange={setCatId} required />
          <MoneyInput value={money} onChange={setMoney} />
          <TextInput label="Комментарий" value={note} onChange={(e) => setNote(e.currentTarget.value)} />
          <Group justify="flex-end">
            <Button variant="default" onClick={close}>Отмена</Button>
            <Button onClick={savePlan}>Сохранить</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}

function devColor(amount: string): string | undefined {
  const n = parseFloat(amount);
  if (n > 0) return "red";
  if (n < 0) return "green";
  return undefined;
}

function Totals({ label, value, strong }: { label: string; value: string; strong?: boolean }) {
  return (
    <Card withBorder padding="sm">
      <Text size="xs" c="dimmed">{label}</Text>
      <Text fw={strong ? 700 : 500}>{value}</Text>
    </Card>
  );
}
