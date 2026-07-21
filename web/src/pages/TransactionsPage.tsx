import { useEffect, useState } from "react";
import {
  ActionIcon,
  Button,
  Card,
  Group,
  Modal,
  SegmentedControl,
  Stack,
  Table,
  Text,
  TextInput,
  Title,
  Loader,
  Center,
} from "@mantine/core";
import { DatePickerInput } from "@mantine/dates";
import { useDisclosure } from "@mantine/hooks";
import { IconEdit, IconPlus, IconTrash } from "@tabler/icons-react";

import {
  createTransaction,
  deleteTransaction,
  listTransactions,
  updateTransaction,
} from "../api/client";
import type { EntryType, Money, Transaction } from "../api/types";
import { MonthNav } from "../components/MonthNav";
import { CategorySelect } from "../components/CategorySelect";
import { MoneyInput } from "../components/MoneyInput";
import { useCategories } from "../state/categories";
import { useCurrencies } from "../state/currencies";
import { notifyError, notifyOk } from "../lib/notify";
import { formatMoney, currentMonth, todayISO } from "../lib/format";

function isoToDate(iso: string): Date {
  return new Date(iso + "T00:00:00");
}
function dateToISO(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export function TransactionsPage() {
  const { displayName } = useCategories();
  const { defaultCurrency } = useCurrencies();
  const [ym, setYm] = useState(currentMonth());
  const [rows, setRows] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(false);
  const [opened, { open, close }] = useDisclosure(false);

  const [editing, setEditing] = useState<Transaction | null>(null);
  const [type, setType] = useState<EntryType>("expense");
  const [date, setDate] = useState<string>(todayISO());
  const [catId, setCatId] = useState<string | null>(null);
  const [money, setMoney] = useState<Money>({ amount: "", currency: defaultCurrency });
  const [note, setNote] = useState("");

  async function load() {
    setLoading(true);
    try {
      setRows(await listTransactions(ym));
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

  function openNew() {
    setEditing(null);
    setType("expense");
    setDate(todayISO());
    setCatId(null);
    setMoney({ amount: "", currency: defaultCurrency });
    setNote("");
    open();
  }

  function openEdit(t: Transaction) {
    setEditing(t);
    setType(t.type);
    setDate(t.date);
    setCatId(t.categoryId);
    setMoney(t.amount);
    setNote(t.note ?? "");
    open();
  }

  async function save() {
    if (!catId) {
      notifyError(new Error("Выберите категорию"));
      return;
    }
    const input = { date, type, categoryId: catId, amount: money, note };
    try {
      if (editing) {
        await updateTransaction(editing.id, input);
        notifyOk("Операция обновлена");
      } else {
        await createTransaction(input);
        notifyOk("Операция добавлена");
      }
      close();
      await load();
    } catch (e) {
      notifyError(e);
    }
  }

  async function remove(t: Transaction) {
    if (!confirm("Удалить операцию?")) return;
    try {
      await deleteTransaction(t.id);
      await load();
    } catch (e) {
      notifyError(e);
    }
  }

  return (
    <Stack>
      <Group justify="space-between" wrap="wrap">
        <Title order={3}>Операции</Title>
        <MonthNav value={ym} onChange={setYm} />
      </Group>

      <Button leftSection={<IconPlus size={16} />} onClick={openNew} w="fit-content">
        Добавить
      </Button>

      {loading ? (
        <Center h={160}><Loader /></Center>
      ) : (
        <Card withBorder padding={0}>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Дата</Table.Th>
                <Table.Th>Категория</Table.Th>
                <Table.Th ta="right">Сумма</Table.Th>
                <Table.Th>Комментарий</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {rows.map((t) => (
                <Table.Tr key={t.id}>
                  <Table.Td>{t.date}</Table.Td>
                  <Table.Td>{displayName(t.categoryId)}</Table.Td>
                  <Table.Td ta="right" c={t.type === "income" ? "green" : "red"}>
                    {formatMoney(t.amount)}
                  </Table.Td>
                  <Table.Td>{t.note}</Table.Td>
                  <Table.Td>
                    <Group gap={4} justify="flex-end" wrap="nowrap">
                      <ActionIcon variant="subtle" onClick={() => openEdit(t)} aria-label="Изменить">
                        <IconEdit size={16} />
                      </ActionIcon>
                      <ActionIcon variant="subtle" color="red" onClick={() => remove(t)} aria-label="Удалить">
                        <IconTrash size={16} />
                      </ActionIcon>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
              {rows.length === 0 && (
                <Table.Tr>
                  <Table.Td colSpan={5}>
                    <Text c="dimmed" ta="center" py="md">В этом месяце операций пока нет.</Text>
                  </Table.Td>
                </Table.Tr>
              )}
            </Table.Tbody>
          </Table>
        </Card>
      )}

      <Modal opened={opened} onClose={close} title={editing ? "Изменить операцию" : "Новая операция"}>
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
          <DatePickerInput
            label="Дата"
            value={isoToDate(date)}
            onChange={(d) => d && setDate(dateToISO(d))}
            valueFormat="DD.MM.YYYY"
          />
          <CategorySelect type={type} value={catId} onChange={setCatId} required />
          <MoneyInput value={money} onChange={setMoney} />
          <TextInput label="Комментарий" value={note} onChange={(e) => setNote(e.currentTarget.value)} />
          <Group justify="flex-end">
            <Button variant="default" onClick={close}>Отмена</Button>
            <Button onClick={save}>Сохранить</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
