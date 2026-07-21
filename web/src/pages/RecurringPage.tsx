import { useEffect, useState } from "react";
import {
  ActionIcon,
  Button,
  Card,
  Checkbox,
  Group,
  Modal,
  NumberInput,
  SegmentedControl,
  Stack,
  Table,
  Text,
  Title,
  Badge,
  Loader,
  Center,
} from "@mantine/core";
import { DatePickerInput } from "@mantine/dates";
import { useDisclosure } from "@mantine/hooks";
import {
  IconEdit,
  IconPlayerPause,
  IconPlayerPlay,
  IconPlus,
  IconTrash,
} from "@tabler/icons-react";

import {
  createRecurring,
  deleteRecurring,
  listRecurring,
  pauseRecurring,
  resumeRecurring,
  updateRecurring,
} from "../api/client";
import type { EntryType, Money, Recurring } from "../api/types";
import { CategorySelect } from "../components/CategorySelect";
import { MoneyInput } from "../components/MoneyInput";
import { useCategories } from "../state/categories";
import { useCurrencies } from "../state/currencies";
import { notifyError, notifyOk } from "../lib/notify";
import { formatMoney, todayISO } from "../lib/format";

function isoToDate(iso: string): Date {
  return new Date(iso + "T00:00:00");
}
function dateToISO(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export function RecurringPage() {
  const { displayName } = useCategories();
  const { defaultCurrency } = useCurrencies();
  const [rows, setRows] = useState<Recurring[]>([]);
  const [loading, setLoading] = useState(false);
  const [opened, { open, close }] = useDisclosure(false);

  const [editing, setEditing] = useState<Recurring | null>(null);
  const [type, setType] = useState<EntryType>("expense");
  const [catId, setCatId] = useState<string | null>(null);
  const [money, setMoney] = useState<Money>({ amount: "", currency: defaultCurrency });
  const [day, setDay] = useState(1);
  const [start, setStart] = useState(todayISO());
  const [end, setEnd] = useState<string>("");
  const [autoFact, setAutoFact] = useState(false);

  async function load() {
    setLoading(true);
    try {
      setRows(await listRecurring());
    } catch (e) {
      notifyError(e);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  function openNew() {
    setEditing(null);
    setType("expense");
    setCatId(null);
    setMoney({ amount: "", currency: defaultCurrency });
    setDay(1);
    setStart(todayISO());
    setEnd("");
    setAutoFact(false);
    open();
  }

  function openEdit(t: Recurring) {
    setEditing(t);
    setType(t.type);
    setCatId(t.categoryId);
    setMoney(t.amount);
    setDay(t.dayOfMonth);
    setStart(t.startDate);
    setEnd(t.endDate ?? "");
    setAutoFact(t.autoCreateFact);
    open();
  }

  async function save() {
    if (!catId) {
      notifyError(new Error("Выберите категорию"));
      return;
    }
    const input = {
      type, categoryId: catId, amount: money, dayOfMonth: day,
      startDate: start, endDate: end || undefined, autoCreateFact: autoFact,
    };
    try {
      if (editing) await updateRecurring(editing.id, input);
      else await createRecurring(input);
      close();
      await load();
      notifyOk("Шаблон сохранён");
    } catch (e) {
      notifyError(e);
    }
  }

  async function toggle(t: Recurring) {
    try {
      if (t.active) await pauseRecurring(t.id);
      else await resumeRecurring(t.id);
      await load();
    } catch (e) {
      notifyError(e);
    }
  }

  async function remove(t: Recurring) {
    if (!confirm("Удалить шаблон?")) return;
    try {
      await deleteRecurring(t.id);
      await load();
    } catch (e) {
      notifyError(e);
    }
  }

  return (
    <Stack>
      <Title order={3}>Повторяющиеся операции</Title>
      <Button leftSection={<IconPlus size={16} />} onClick={openNew} w="fit-content">
        Новый шаблон
      </Button>

      {loading ? (
        <Center h={160}><Loader /></Center>
      ) : (
        <Card withBorder padding={0}>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Тип</Table.Th>
                <Table.Th>Категория</Table.Th>
                <Table.Th ta="right">Сумма</Table.Th>
                <Table.Th ta="center">День</Table.Th>
                <Table.Th>Статус</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {rows.map((t) => (
                <Table.Tr key={t.id}>
                  <Table.Td>{t.type === "income" ? "Доход" : "Расход"}</Table.Td>
                  <Table.Td>{displayName(t.categoryId)}</Table.Td>
                  <Table.Td ta="right">{formatMoney(t.amount)}</Table.Td>
                  <Table.Td ta="center">{t.dayOfMonth}</Table.Td>
                  <Table.Td>
                    <Badge color={t.active ? "green" : "gray"} variant="light">
                      {t.active ? "активен" : "приостановлен"}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Group gap={2} justify="flex-end" wrap="nowrap">
                      <ActionIcon variant="subtle" onClick={() => openEdit(t)} aria-label="Изменить"><IconEdit size={16} /></ActionIcon>
                      <ActionIcon variant="subtle" onClick={() => toggle(t)} aria-label="Пауза/старт">
                        {t.active ? <IconPlayerPause size={16} /> : <IconPlayerPlay size={16} />}
                      </ActionIcon>
                      <ActionIcon variant="subtle" color="red" onClick={() => remove(t)} aria-label="Удалить"><IconTrash size={16} /></ActionIcon>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
              {rows.length === 0 && (
                <Table.Tr>
                  <Table.Td colSpan={6}>
                    <Text c="dimmed" ta="center" py="md">Шаблонов пока нет.</Text>
                  </Table.Td>
                </Table.Tr>
              )}
            </Table.Tbody>
          </Table>
        </Card>
      )}

      <Modal opened={opened} onClose={close} title="Шаблон повторяющейся операции">
        <Stack>
          <SegmentedControl
            value={type}
            onChange={(v) => { setType(v as EntryType); setCatId(null); }}
            data={[{ label: "Расход", value: "expense" }, { label: "Доход", value: "income" }]}
            fullWidth
          />
          <CategorySelect type={type} value={catId} onChange={setCatId} required />
          <MoneyInput value={money} onChange={setMoney} />
          <NumberInput label="День месяца (1–28)" value={day} onChange={(v) => setDay(Number(v) || 1)} min={1} max={28} />
          <DatePickerInput label="Дата начала" value={isoToDate(start)} onChange={(d) => d && setStart(dateToISO(d))} valueFormat="DD.MM.YYYY" />
          <DatePickerInput label="Дата окончания (опц.)" value={end ? isoToDate(end) : null} onChange={(d) => setEnd(d ? dateToISO(d) : "")} valueFormat="DD.MM.YYYY" clearable />
          <Checkbox label="Автоматически создавать факт" checked={autoFact} onChange={(e) => setAutoFact(e.currentTarget.checked)} />
          <Group justify="flex-end">
            <Button variant="default" onClick={close}>Отмена</Button>
            <Button onClick={save}>Сохранить</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
