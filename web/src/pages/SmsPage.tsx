import { useEffect, useState } from "react";
import {
  ActionIcon,
  Alert,
  Badge,
  Button,
  Card,
  Code,
  Group,
  Modal,
  NumberInput,
  SegmentedControl,
  Stack,
  Switch,
  Table,
  Text,
  TextInput,
  Textarea,
  Title,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconEdit, IconPlus, IconTrash, IconTestPipe } from "@tabler/icons-react";

import {
  createSmsTemplate,
  deleteDraft,
  deleteSmsTemplate,
  listInbox,
  listSmsTemplates,
  resolveDraft,
  testSms,
  updateSmsTemplate,
} from "../api/client";
import type { EntryType, InboxDraft, Money, SmsTemplate, SmsTestResult } from "../api/types";
import { CategorySelect } from "../components/CategorySelect";
import { MoneyInput } from "../components/MoneyInput";
import { useCategories } from "../state/categories";
import { useCurrencies } from "../state/currencies";
import { notifyError, notifyOk } from "../lib/notify";
import { formatMoney } from "../lib/format";

const EMPTY: Omit<SmsTemplate, "id"> = {
  name: "",
  sender: "",
  pattern: "",
  amountGroup: 1,
  currencyGroup: 0,
  merchantGroup: 0,
  fixedCurrency: "BYN",
  type: "expense",
  defaultCategoryId: "",
  enabled: true,
  priority: 0,
};

export function SmsPage() {
  const { displayName } = useCategories();
  const [templates, setTemplates] = useState<SmsTemplate[]>([]);
  const [drafts, setDrafts] = useState<InboxDraft[]>([]);

  const [tplOpened, tpl] = useDisclosure(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<Omit<SmsTemplate, "id">>(EMPTY);

  // Тест.
  const [testSender, setTestSender] = useState("");
  const [testText, setTestText] = useState("");
  const [testResult, setTestResult] = useState<SmsTestResult | null>(null);

  // Разбор черновика.
  const [resolveOpened, resolveDlg] = useDisclosure(false);
  const [draft, setDraft] = useState<InboxDraft | null>(null);

  async function load() {
    try {
      const [t, d] = await Promise.all([listSmsTemplates(), listInbox(true)]);
      setTemplates(t);
      setDrafts(d);
    } catch (e) {
      notifyError(e);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  function openNew() {
    setEditingId(null);
    setForm(EMPTY);
    tpl.open();
  }
  function openEdit(t: SmsTemplate) {
    setEditingId(t.id);
    const { id: _id, ...rest } = t;
    void _id;
    setForm(rest);
    tpl.open();
  }

  async function saveTemplate() {
    try {
      if (editingId) await updateSmsTemplate(editingId, form);
      else await createSmsTemplate(form);
      tpl.close();
      await load();
      notifyOk("Шаблон сохранён");
    } catch (e) {
      notifyError(e);
    }
  }

  async function removeTemplate(id: string) {
    if (!confirm("Удалить шаблон?")) return;
    try {
      await deleteSmsTemplate(id);
      await load();
    } catch (e) {
      notifyError(e);
    }
  }

  async function runTest() {
    try {
      setTestResult(await testSms(testSender, testText));
    } catch (e) {
      notifyError(e);
    }
  }

  function openResolve(d: InboxDraft) {
    setDraft(d);
    resolveDlg.open();
  }

  return (
    <Stack>
      <Title order={3}>SMS-парсер</Title>
      <Text size="sm" c="dimmed">
        Настройте шаблоны разбора банковских SMS. Android-приложение шлёт сырой текст SMS
        на сервер, а сервер по шаблонам извлекает сумму и создаёт операцию. Нераспознанные
        SMS попадают во «Входящие» для ручного разбора.
      </Text>

      {/* Шаблоны */}
      <Card withBorder padding="md">
        <Group justify="space-between" mb="sm">
          <Title order={5}>Шаблоны разбора</Title>
          <Button size="xs" leftSection={<IconPlus size={14} />} onClick={openNew}>
            Новый шаблон
          </Button>
        </Group>
        <Table striped>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Название</Table.Th>
              <Table.Th>Отправитель</Table.Th>
              <Table.Th>Тип</Table.Th>
              <Table.Th>Категория</Table.Th>
              <Table.Th>Вкл.</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {templates.map((t) => (
              <Table.Tr key={t.id}>
                <Table.Td>{t.name}</Table.Td>
                <Table.Td>{t.sender || "любой"}</Table.Td>
                <Table.Td>{t.type === "income" ? "Доход" : "Расход"}</Table.Td>
                <Table.Td>{t.defaultCategoryId ? displayName(t.defaultCategoryId) : <Text c="dimmed" size="sm">во входящие</Text>}</Table.Td>
                <Table.Td>{t.enabled ? <Badge color="green" variant="light">да</Badge> : <Badge color="gray" variant="light">нет</Badge>}</Table.Td>
                <Table.Td>
                  <Group gap={2} justify="flex-end" wrap="nowrap">
                    <ActionIcon variant="subtle" onClick={() => openEdit(t)}><IconEdit size={16} /></ActionIcon>
                    <ActionIcon variant="subtle" color="red" onClick={() => removeTemplate(t.id)}><IconTrash size={16} /></ActionIcon>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
            {templates.length === 0 && (
              <Table.Tr><Table.Td colSpan={6}><Text c="dimmed" ta="center" py="sm">Шаблонов пока нет.</Text></Table.Td></Table.Tr>
            )}
          </Table.Tbody>
        </Table>
      </Card>

      {/* Тест */}
      <Card withBorder padding="md">
        <Title order={5} mb="sm">Проверка на примере</Title>
        <Stack gap="xs">
          <TextInput label="Отправитель" placeholder="Priorbank" value={testSender} onChange={(e) => setTestSender(e.currentTarget.value)} />
          <Textarea label="Текст SMS" autosize minRows={2} placeholder="Oplata 45.20 BYN v EUROOPT" value={testText} onChange={(e) => setTestText(e.currentTarget.value)} />
          <Button variant="light" leftSection={<IconTestPipe size={16} />} onClick={runTest} w="fit-content">Проверить</Button>
          {testResult && (
            testResult.matched ? (
              <Alert color="green" title={`Совпал шаблон: ${testResult.templateName}`}>
                Сумма: <b>{testResult.amount ? formatMoney(testResult.amount) : "—"}</b>, тип:{" "}
                {testResult.type === "income" ? "доход" : "расход"}
                {testResult.merchant ? <>, продавец: <b>{testResult.merchant}</b></> : null}
                {testResult.defaultCategoryId ? `, категория: ${displayName(testResult.defaultCategoryId)}` : " (без категории → во входящие)"}
              </Alert>
            ) : (
              <Alert color="orange" title="Ни один шаблон не совпал">Проверьте отправителя и regex.</Alert>
            )
          )}
        </Stack>
      </Card>

      {/* Входящие */}
      <Card withBorder padding="md">
        <Title order={5} mb="sm">Входящие ({drafts.length})</Title>
        <Table striped>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Дата</Table.Th>
              <Table.Th>Отправитель</Table.Th>
              <Table.Th>Текст</Table.Th>
              <Table.Th>Продавец</Table.Th>
              <Table.Th>Сумма</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {drafts.map((d) => (
              <Table.Tr key={d.id}>
                <Table.Td>{d.receivedAt}</Table.Td>
                <Table.Td>{d.rawSender}</Table.Td>
                <Table.Td><Text size="sm" lineClamp={2} maw={260}>{d.rawText}</Text></Table.Td>
                <Table.Td>{d.merchant || <Text c="dimmed" size="sm">—</Text>}</Table.Td>
                <Table.Td>{d.amount ? formatMoney(d.amount) : <Text c="dimmed" size="sm">?</Text>}</Table.Td>
                <Table.Td>
                  <Group gap={4} justify="flex-end" wrap="nowrap">
                    <Button size="xs" variant="light" onClick={() => openResolve(d)}>Разобрать</Button>
                    <ActionIcon variant="subtle" color="red" onClick={async () => { await deleteDraft(d.id); void load(); }}><IconTrash size={16} /></ActionIcon>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
            {drafts.length === 0 && (
              <Table.Tr><Table.Td colSpan={6}><Text c="dimmed" ta="center" py="sm">Входящих нет.</Text></Table.Td></Table.Tr>
            )}
          </Table.Tbody>
        </Table>
      </Card>

      <TemplateModal opened={tplOpened} onClose={tpl.close} form={form} setForm={setForm} onSave={saveTemplate} />
      {draft && (
        <ResolveModal opened={resolveOpened} onClose={resolveDlg.close} draft={draft} onDone={async () => { resolveDlg.close(); await load(); }} />
      )}
    </Stack>
  );
}

function TemplateModal({
  opened, onClose, form, setForm, onSave,
}: {
  opened: boolean;
  onClose: () => void;
  form: Omit<SmsTemplate, "id">;
  setForm: (f: Omit<SmsTemplate, "id">) => void;
  onSave: () => void;
}) {
  const { codes } = useCurrencies();
  const set = <K extends keyof Omit<SmsTemplate, "id">>(k: K, v: Omit<SmsTemplate, "id">[K]) =>
    setForm({ ...form, [k]: v });

  return (
    <Modal opened={opened} onClose={onClose} title="Шаблон разбора SMS" size="lg">
      <Stack gap="xs">
        <TextInput label="Название" value={form.name} onChange={(e) => set("name", e.currentTarget.value)} />
        <TextInput label="Отправитель (подстрока, пусто = любой)" value={form.sender} onChange={(e) => set("sender", e.currentTarget.value)} />
        <Textarea label="Regex по тексту SMS" autosize minRows={2} value={form.pattern} onChange={(e) => set("pattern", e.currentTarget.value)}
          description="Сумму заключите в группу (скобки). Пример: Oplata (\d+[.,]\d{2}) BYN" />
        <Group grow>
          <NumberInput label="Группа суммы" min={1} value={form.amountGroup} onChange={(v) => set("amountGroup", Number(v) || 1)} />
          <NumberInput label="Группа валюты (0 = фикс.)" min={0} value={form.currencyGroup} onChange={(v) => set("currencyGroup", Number(v) || 0)} />
          <NumberInput label="Группа продавца (0 = нет)" min={0} value={form.merchantGroup} onChange={(v) => set("merchantGroup", Number(v) || 0)} />
        </Group>
        {form.currencyGroup === 0 && (
          <TextInput label="Фиксированная валюта" value={form.fixedCurrency} onChange={(e) => set("fixedCurrency", e.currentTarget.value.toUpperCase())}
            list="cur-list" />
        )}
        <datalist id="cur-list">{codes.map((c) => <option key={c} value={c} />)}</datalist>
        <SegmentedControl value={form.type} onChange={(v) => set("type", v as EntryType)}
          data={[{ label: "Расход", value: "expense" }, { label: "Доход", value: "income" }]} />
        <CategorySelect type={form.type} value={form.defaultCategoryId || null}
          onChange={(id) => set("defaultCategoryId", id ?? "")}
          label="Категория (пусто = во входящие на ручной разбор)" />
        <Group>
          <Switch label="Включён" checked={form.enabled} onChange={(e) => set("enabled", e.currentTarget.checked)} />
          <NumberInput label="Приоритет" value={form.priority} onChange={(v) => set("priority", Number(v) || 0)} w={120} />
        </Group>
        <Text size="xs" c="dimmed">Совет: используйте раздел «Проверка на примере», чтобы подобрать regex.</Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={onSave}>Сохранить</Button>
        </Group>
      </Stack>
    </Modal>
  );
}

function ResolveModal({
  opened, onClose, draft, onDone,
}: {
  opened: boolean;
  onClose: () => void;
  draft: InboxDraft;
  onDone: () => void;
}) {
  const { defaultCurrency } = useCurrencies();
  const [type, setType] = useState<EntryType>(draft.type ?? "expense");
  const [catId, setCatId] = useState<string | null>(null);
  const [money, setMoney] = useState<Money>(draft.amount ?? { amount: "", currency: defaultCurrency });

  async function submit() {
    if (!catId) {
      notifyError(new Error("Выберите категорию"));
      return;
    }
    try {
      await resolveDraft(draft.id, catId, draft.amount ? undefined : money, type);
      onDone();
      notifyOk("Операция создана");
    } catch (e) {
      notifyError(e);
    }
  }

  return (
    <Modal opened={opened} onClose={onClose} title="Разобрать входящее SMS">
      <Stack gap="xs">
        <Code block>{draft.rawText}</Code>
        <SegmentedControl value={type} onChange={(v) => { setType(v as EntryType); setCatId(null); }}
          data={[{ label: "Расход", value: "expense" }, { label: "Доход", value: "income" }]} fullWidth />
        {!draft.amount && <MoneyInput value={money} onChange={setMoney} />}
        <CategorySelect type={type} value={catId} onChange={setCatId} required />
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={submit}>Создать операцию</Button>
        </Group>
      </Stack>
    </Modal>
  );
}
