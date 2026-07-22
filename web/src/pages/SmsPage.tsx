import { useEffect, useMemo, useState } from "react";
import {
  ActionIcon,
  Alert,
  Badge,
  Box,
  Button,
  Card,
  Checkbox,
  Code,
  Group,
  Modal,
  NumberInput,
  SegmentedControl,
  Select,
  Stack,
  Switch,
  Table,
  Text,
  TextInput,
  Textarea,
  Title,
  UnstyledButton,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconEdit, IconPlus, IconTrash, IconTestPipe } from "@tabler/icons-react";

import {
  assignMerchant,
  createRule,
  createSmsTemplate,
  createTemplateFromSample,
  deleteDraft,
  deleteRule,
  deleteSmsTemplate,
  listInbox,
  listMerchants,
  listRules,
  listSmsTemplates,
  resolveDraft,
  testSms,
  updateSmsTemplate,
} from "../api/client";
import type { CategoryRule, EntryType, InboxDraft, Merchant, Money, SignalKind, SmsTemplate, SmsTestResult } from "../api/types";
import { CategorySelect } from "../components/CategorySelect";
import { MoneyInput } from "../components/MoneyInput";
import { useCategories } from "../state/categories";
import { useCurrencies } from "../state/currencies";
import { useMe } from "../state/me";
import { notifyError, notifyOk } from "../lib/notify";
import { formatMoney } from "../lib/format";

const EMPTY: Omit<SmsTemplate, "id"> = {
  name: "",
  sender: "",
  pattern: "",
  amountGroup: 1,
  currencyGroup: 0,
  merchantGroup: 0,
  captureKind: "merchant",
  fixedCurrency: "BYN",
  type: "expense",
  defaultCategoryId: "",
  enabled: true,
  priority: 0,
};

export function SmsPage() {
  const { displayName } = useCategories();
  const { isAdmin } = useMe();
  const [templates, setTemplates] = useState<SmsTemplate[]>([]);
  const [drafts, setDrafts] = useState<InboxDraft[]>([]);
  const [rules, setRules] = useState<CategoryRule[]>([]);
  const [merchants, setMerchants] = useState<Merchant[]>([]);

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

  // Сборка шаблона «по образцу».
  const [teachOpened, teach] = useDisclosure(false);
  const [teachDraft, setTeachDraft] = useState<InboxDraft | null>(null);

  async function load() {
    try {
      const [t, d, rs, ms] = await Promise.all([listSmsTemplates(), listInbox(true), listRules(), listMerchants()]);
      setTemplates(t);
      setDrafts(d);
      setRules(rs);
      setMerchants(ms);
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
          <Title order={5}>Шаблоны разбора {!isAdmin && <Text span size="xs" c="dimmed">(общие, меняет админ)</Text>}</Title>
          {isAdmin && (
            <Button size="xs" leftSection={<IconPlus size={14} />} onClick={openNew}>
              Новый шаблон
            </Button>
          )}
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
                  {isAdmin && (
                    <Group gap={2} justify="flex-end" wrap="nowrap">
                      <ActionIcon variant="subtle" onClick={() => openEdit(t)}><IconEdit size={16} /></ActionIcon>
                      <ActionIcon variant="subtle" color="red" onClick={() => removeTemplate(t.id)}><IconTrash size={16} /></ActionIcon>
                    </Group>
                  )}
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
                {testResult.merchant ? <>, контрагент: <b>{testResult.merchant}</b></> : null}
                {testResult.defaultCategoryId ? `, категория: ${displayName(testResult.defaultCategoryId)}` : " (без категории → во входящие)"}
              </Alert>
            ) : (
              <Alert color="orange" title="Ни один шаблон не совпал">Проверьте отправителя и regex.</Alert>
            )
          )}
        </Stack>
      </Card>

      {/* Справочник контрагентов */}
      <MerchantsCard merchants={merchants} onChange={load} />

      {/* Правила «контрагент → категория» */}
      <RulesCard rules={rules} onChange={load} />

      {/* Входящие */}
      <Card withBorder padding="md">
        <Title order={5} mb="sm">Входящие ({drafts.length})</Title>
        <Table striped>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Дата</Table.Th>
              <Table.Th>Отправитель</Table.Th>
              <Table.Th>Текст</Table.Th>
              <Table.Th>Контрагент</Table.Th>
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
                    {isAdmin && (
                      <Button size="xs" variant="light" color="grape" onClick={() => { setTeachDraft(d); teach.open(); }}>По образцу</Button>
                    )}
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
      {teachDraft && (
        <TeachModal opened={teachOpened} onClose={teach.close} draft={teachDraft} onDone={async () => { teach.close(); await load(); }} />
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
          <NumberInput label="Группа признака (0 = нет)" min={0} value={form.merchantGroup} onChange={(v) => set("merchantGroup", Number(v) || 0)} />
        </Group>
        {form.merchantGroup > 0 && (
          <Select
            label="Тип признака"
            description="Что ловит эта группа: контрагента (покупка) или счёт (ЕРИП/перевод)"
            data={[{ label: "Контрагент", value: "merchant" }, { label: "Счёт", value: "account" }]}
            value={form.captureKind}
            onChange={(v) => v && set("captureKind", v as SignalKind)}
            allowDeselect={false}
            w={220}
          />
        )}
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
  const [remember, setRemember] = useState(true);

  async function submit() {
    if (!catId) {
      notifyError(new Error("Выберите категорию"));
      return;
    }
    try {
      await resolveDraft(draft.id, catId, draft.amount ? undefined : money, type, remember && !!draft.merchant);
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
        {draft.merchant && (
          <Checkbox
            checked={remember}
            onChange={(e) => setRemember(e.currentTarget.checked)}
            label={`Запомнить: «${draft.merchant}» → эта категория (будущие SMS разнесутся сами)`}
          />
        )}
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={submit}>Создать операцию</Button>
        </Group>
      </Stack>
    </Modal>
  );
}

type AssignFn = (name: string, categoryId: string, label: string) => void;

// MerchantsCard — строгий список распознавания из SMS, разделён на «Контрагентов»
// (покупки) и «Счета» (ЕРИП/переводы). Категория (и для счетов — название)
// закрепляются прямо за признаком по точному совпадению имени.
function MerchantsCard({ merchants, onChange }: { merchants: Merchant[]; onChange: () => void }) {
  const assign: AssignFn = async (name, categoryId, label) => {
    try {
      await assignMerchant(name, categoryId, label);
      onChange();
      notifyOk(`«${label || name}» сохранён`);
    } catch (e) {
      notifyError(e);
    }
  };

  const contractors = merchants.filter((m) => m.kind !== "account");
  const accounts = merchants.filter((m) => m.kind === "account");

  return (
    <Card withBorder padding="md">
      <Title order={5} mb="xs">Контрагенты и счета</Title>
      <Text size="sm" c="dimmed" mb="sm">
        Признаки из SMS копятся автоматически (каждый отдельно, точно по имени).
        Закрепите категорию — и будущие платежи разнесутся сами. Счёт — это номер
        договора (ЕРИП/перевод), ему можно задать понятное название.
      </Text>

      <Text fw={600} size="sm" mb={4}>Контрагенты <Text span c="dimmed" size="xs">(покупки)</Text></Text>
      <Table striped mb="lg">
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Контрагент</Table.Th>
            <Table.Th ta="right">Встреч</Table.Th>
            <Table.Th ta="right">Оборот</Table.Th>
            <Table.Th>Категория</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {contractors.map((m) => <MerchantRow key={m.name} m={m} onAssign={assign} />)}
          {contractors.length === 0 && (
            <Table.Tr><Table.Td colSpan={4}><Text c="dimmed" ta="center" py="sm">Контрагентов пока нет.</Text></Table.Td></Table.Tr>
          )}
        </Table.Tbody>
      </Table>

      <Text fw={600} size="sm" mb={4}>Счета <Text span c="dimmed" size="xs">(ЕРИП / переводы)</Text></Text>
      <Table striped>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Счёт</Table.Th>
            <Table.Th>Название</Table.Th>
            <Table.Th ta="right">Встреч</Table.Th>
            <Table.Th ta="right">Оборот</Table.Th>
            <Table.Th>Категория</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {accounts.map((m) => <AccountRow key={m.name} m={m} onAssign={assign} />)}
          {accounts.length === 0 && (
            <Table.Tr><Table.Td colSpan={5}><Text c="dimmed" ta="center" py="sm">Счетов пока нет.</Text></Table.Td></Table.Tr>
          )}
        </Table.Tbody>
      </Table>
    </Card>
  );
}

function MerchantRow({ m, onAssign }: { m: Merchant; onAssign: AssignFn }) {
  return (
    <Table.Tr>
      <Table.Td>{m.name}</Table.Td>
      <Table.Td ta="right">{m.seenCount}</Table.Td>
      <Table.Td ta="right">{formatMoney(m.total)}</Table.Td>
      <Table.Td>
        <div style={{ minWidth: 190 }}>
          <CategorySelect
            type="expense"
            value={m.categoryId ?? null}
            onChange={(id) => id && onAssign(m.name, id, m.label ?? "")}
            hideLabel
            placeholder="Назначить…"
          />
        </div>
      </Table.Td>
    </Table.Tr>
  );
}

// AccountRow — строка счёта: номер + редактируемое название + категория.
function AccountRow({ m, onAssign }: { m: Merchant; onAssign: AssignFn }) {
  const [label, setLabel] = useState(m.label ?? "");
  return (
    <Table.Tr>
      <Table.Td><Text size="sm" style={{ fontFamily: "monospace" }}>{m.name}</Text></Table.Td>
      <Table.Td>
        <TextInput
          size="xs"
          placeholder="напр. Электроэнергия"
          value={label}
          onChange={(e) => setLabel(e.currentTarget.value)}
          onBlur={() => { if (label !== (m.label ?? "")) onAssign(m.name, m.categoryId ?? "", label); }}
        />
      </Table.Td>
      <Table.Td ta="right">{m.seenCount}</Table.Td>
      <Table.Td ta="right">{formatMoney(m.total)}</Table.Td>
      <Table.Td>
        <div style={{ minWidth: 170 }}>
          <CategorySelect
            type="expense"
            value={m.categoryId ?? null}
            onChange={(id) => id && onAssign(m.name, id, label)}
            hideLabel
            placeholder="Назначить…"
          />
        </div>
      </Table.Td>
    </Table.Tr>
  );
}

// --- Сборка шаблона «по образцу» ---

type TeachRole = "amount" | "currency" | "merchant";
interface Tok { text: string; start: number; end: number; }

function tokenize(s: string): Tok[] {
  const out: Tok[] = [];
  const re = /\S+/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(s)) !== null) out.push({ text: m[0], start: m.index, end: m.index + m[0].length });
  return out;
}

const ROLE_COLOR: Record<TeachRole, string> = { amount: "blue", currency: "teal", merchant: "grape" };

// TeachModal — учим разбор на реальном SMS: тыкаем в слова, отмечая сумму/валюту/
// признак; система собирает regex-шаблон и сразу создаёт операцию.
function TeachModal({ opened, onClose, draft, onDone }: {
  opened: boolean; onClose: () => void; draft: InboxDraft; onDone: () => void;
}) {
  const tokens = useMemo(() => tokenize(draft.rawText), [draft.rawText]);
  const [role, setRole] = useState<TeachRole>("amount");
  const [amountI, setAmountI] = useState<number | null>(null);
  const [currI, setCurrI] = useState<number | null>(null);
  const [merchIs, setMerchIs] = useState<number[]>([]);
  const [kind, setKind] = useState<SignalKind>("merchant");
  const [type, setType] = useState<EntryType>(draft.type ?? "expense");
  const [catId, setCatId] = useState<string | null>(null);
  const [name, setName] = useState(draft.rawSender || "Новый шаблон");

  function click(i: number) {
    if (role === "amount") setAmountI((p) => (p === i ? null : i));
    else if (role === "currency") setCurrI((p) => (p === i ? null : i));
    else setMerchIs((p) => (p.includes(i) ? p.filter((x) => x !== i) : [...p, i].sort((a, b) => a - b)));
  }
  function roleOf(i: number): TeachRole | null {
    if (i === amountI) return "amount";
    if (i === currI) return "currency";
    if (merchIs.includes(i)) return "merchant";
    return null;
  }

  const amountText = amountI != null ? tokens[amountI].text : "";
  const currencyText = currI != null ? tokens[currI].text : "";
  const merchantText = merchIs.length
    ? draft.rawText.slice(tokens[merchIs[0]].start, tokens[merchIs[merchIs.length - 1]].end)
    : "";

  async function save() {
    if (!amountText) { notifyError(new Error("Отметьте сумму")); return; }
    if (!catId) { notifyError(new Error("Выберите категорию")); return; }
    try {
      const res = await createTemplateFromSample({
        draftId: draft.id, name, sender: draft.rawSender, text: draft.rawText,
        amountText, currencyText, fixedCurrency: currencyText ? "" : "BYN",
        merchantText, captureKind: kind, type, categoryId: catId,
      });
      onDone();
      notifyOk(res.transactionId ? "Шаблон создан, операция добавлена" : "Шаблон создан");
    } catch (e) {
      notifyError(e);
    }
  }

  return (
    <Modal opened={opened} onClose={onClose} title="Шаблон по образцу" size="lg">
      <Stack gap="sm">
        <Text size="sm" c="dimmed">
          Выбери роль ниже, затем ткни в нужные слова сообщения. Признак (контрагент или счёт)
          можно собрать из нескольких соседних слов.
        </Text>
        <Group gap="xs">
          <Button size="xs" variant={role === "amount" ? "filled" : "light"} color="blue" onClick={() => setRole("amount")}>Сумма</Button>
          <Button size="xs" variant={role === "currency" ? "filled" : "light"} color="teal" onClick={() => setRole("currency")}>Валюта</Button>
          <Button size="xs" variant={role === "merchant" ? "filled" : "light"} color="grape" onClick={() => setRole("merchant")}>Контрагент / счёт</Button>
        </Group>

        <Box style={{ lineHeight: 2.2, padding: 8, border: "1px solid var(--mantine-color-default-border)", borderRadius: 6 }}>
          {tokens.map((tok, i) => {
            const r = roleOf(i);
            return (
              <UnstyledButton
                key={i}
                onClick={() => click(i)}
                style={{
                  padding: "2px 5px", margin: 1, borderRadius: 4,
                  fontFamily: "monospace", fontSize: 13,
                  background: r ? `var(--mantine-color-${ROLE_COLOR[r]}-light)` : "transparent",
                  color: r ? `var(--mantine-color-${ROLE_COLOR[r]}-light-color)` : undefined,
                }}
              >
                {tok.text}
              </UnstyledButton>
            );
          })}
        </Box>

        <Group gap="lg" wrap="wrap">
          <Text size="sm"><b>Сумма:</b> {amountText || "—"}</Text>
          <Text size="sm"><b>Валюта:</b> {currencyText || "BYN (по умолч.)"}</Text>
          <Text size="sm"><b>{kind === "account" ? "Счёт" : "Контрагент"}:</b> {merchantText || "—"}</Text>
        </Group>

        {merchantText && (
          <SegmentedControl
            size="xs"
            value={kind}
            onChange={(v) => setKind(v as SignalKind)}
            data={[{ label: "Это контрагент", value: "merchant" }, { label: "Это счёт", value: "account" }]}
          />
        )}

        <TextInput label="Название шаблона" value={name} onChange={(e) => setName(e.currentTarget.value)} />
        <SegmentedControl
          value={type}
          onChange={(v) => { setType(v as EntryType); setCatId(null); }}
          data={[{ label: "Расход", value: "expense" }, { label: "Доход", value: "income" }]}
          fullWidth
        />
        <CategorySelect
          type={type}
          value={catId}
          onChange={setCatId}
          required
          label={merchantText ? "Категория для этого контрагента/счёта" : "Категория для всех таких сообщений"}
        />

        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={save}>Создать шаблон</Button>
        </Group>
      </Stack>
    </Modal>
  );
}

// RulesCard — правила «контрагент → категория»: список + добавление.
function RulesCard({ rules, onChange }: { rules: CategoryRule[]; onChange: () => void }) {
  const { displayName } = useCategories();
  const [pattern, setPattern] = useState("");
  const [catId, setCatId] = useState<string | null>(null);

  async function add() {
    if (!pattern.trim() || !catId) {
      notifyError(new Error("Укажите контрагента и категорию"));
      return;
    }
    try {
      await createRule(pattern.trim(), catId);
      setPattern("");
      setCatId(null);
      onChange();
      notifyOk("Правило добавлено");
    } catch (e) {
      notifyError(e);
    }
  }

  async function remove(id: string) {
    try {
      await deleteRule(id);
      onChange();
    } catch (e) {
      notifyError(e);
    }
  }

  return (
    <Card withBorder padding="md">
      <Title order={5} mb="xs">Правила по подстроке <Text span size="xs" c="dimmed">(дополнительно)</Text></Title>
      <Text size="sm" c="dimmed" mb="sm">
        Необязательный слой для массовых паттернов: если имя контрагента содержит подстроку —
        операция относится к категории. Точная привязка контрагента (выше) приоритетнее.
      </Text>
      <Table striped mb="sm">
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Контрагент содержит</Table.Th>
            <Table.Th>Категория</Table.Th>
            <Table.Th />
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {rules.map((r) => (
            <Table.Tr key={r.id}>
              <Table.Td>{r.pattern}</Table.Td>
              <Table.Td>{displayName(r.categoryId)}</Table.Td>
              <Table.Td>
                <ActionIcon variant="subtle" color="red" onClick={() => remove(r.id)} style={{ float: "right" }}>
                  <IconTrash size={16} />
                </ActionIcon>
              </Table.Td>
            </Table.Tr>
          ))}
          {rules.length === 0 && (
            <Table.Tr><Table.Td colSpan={3}><Text c="dimmed" ta="center" py="sm">Правил пока нет.</Text></Table.Td></Table.Tr>
          )}
        </Table.Tbody>
      </Table>
      <Group align="flex-end" gap="xs" wrap="wrap">
        <TextInput label="Контрагент (подстрока)" placeholder="YANDEX" value={pattern} onChange={(e) => setPattern(e.currentTarget.value)} />
        <div style={{ minWidth: 200 }}>
          <CategorySelect type="expense" value={catId} onChange={setCatId} label="Категория" />
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={add}>Добавить</Button>
      </Group>
    </Card>
  );
}
