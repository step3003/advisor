import { useEffect, useMemo, useState } from "react";
import {
  ActionIcon,
  Badge,
  Box,
  Button,
  Card,
  Checkbox,
  Code,
  Collapse,
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
import { IconChevronDown, IconEdit, IconTrash } from "@tabler/icons-react";

import {
  assignMerchant,
  createSmsTemplate,
  createTemplateFromSample,
  deleteDraft,
  deleteSmsTemplate,
  listInbox,
  listMerchants,
  listSmsTemplates,
  resolveDraft,
  restoreDraft,
  updateSmsTemplate,
} from "../api/client";
import type { EntryType, InboxDraft, Merchant, Money, SignalKind, SmsAction, SmsTemplate } from "../api/types";
import { CategorySelect } from "../components/CategorySelect";
import { MoneyInput } from "../components/MoneyInput";
import { useCurrencies } from "../state/currencies";
import { useMe } from "../state/me";
import { notifyError, notifyOk } from "../lib/notify";
import { formatMoney } from "../lib/format";

const EMPTY: Omit<SmsTemplate, "id"> = {
  name: "",
  sender: "",
  pattern: "",
  action: "operation",
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
  const { isAdmin } = useMe();
  const [templates, setTemplates] = useState<SmsTemplate[]>([]);
  const [drafts, setDrafts] = useState<InboxDraft[]>([]);
  const [filtered, setFiltered] = useState<InboxDraft[]>([]);
  const [merchants, setMerchants] = useState<Merchant[]>([]);
  const [advanced, setAdvanced] = useState(false); // раскрыть шаблоны (типы сообщений)
  const [showFiltered, setShowFiltered] = useState(false);

  const [tplOpened, tpl] = useDisclosure(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<Omit<SmsTemplate, "id">>(EMPTY);

  // Разбор черновика вручную (формат известен, нужна только категория).
  const [resolveOpened, resolveDlg] = useDisclosure(false);
  const [draft, setDraft] = useState<InboxDraft | null>(null);

  // Сборка шаблона «по образцу» (формат неизвестен).
  const [teachOpened, teach] = useDisclosure(false);
  const [teachDraft, setTeachDraft] = useState<InboxDraft | null>(null);

  async function load() {
    try {
      const [t, d, f, ms] = await Promise.all([
        listSmsTemplates(), listInbox("pending"), listInbox("filtered"), listMerchants(),
      ]);
      setTemplates(t);
      setDrafts(d);
      setFiltered(f);
      setMerchants(ms);
    } catch (e) {
      notifyError(e);
    }
  }

  const tplName = (id?: string) => templates.find((t) => t.id === id)?.name ?? "—";

  useEffect(() => {
    void load();
  }, []);

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

  // Одна точка входа для черновика: если формат уже распознан (есть сумма) —
  // просто категоризируем; если формат новый и мы админ — учим «по образцу».
  function openDraft(d: InboxDraft) {
    if (!d.amount && isAdmin) {
      setTeachDraft(d);
      teach.open();
    } else {
      setDraft(d);
      resolveDlg.open();
    }
  }

  return (
    <Stack>
      <Title order={3}>SMS-парсер</Title>
      <Text size="sm" c="dimmed">
        Android-приложение шлёт банковские SMS на сервер. Знакомые — разносятся сами;
        новые попадают во «Входящие»: нажми «Разобрать» и один раз укажи, где сумма и
        контрагент/счёт — дальше такие сообщения парсятся автоматически.
      </Text>

      {/* Входящие — главное */}
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
                    <Button size="xs" variant="light" onClick={() => openDraft(d)}>Разобрать</Button>
                    <ActionIcon variant="subtle" color="red" onClick={async () => { await deleteDraft(d.id); void load(); }}><IconTrash size={16} /></ActionIcon>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
            {drafts.length === 0 && (
              <Table.Tr><Table.Td colSpan={6}><Text c="dimmed" ta="center" py="sm">Входящих нет — всё разобрано.</Text></Table.Td></Table.Tr>
            )}
          </Table.Tbody>
        </Table>
      </Card>

      {/* Отфильтрованные (архив мусора) — свёрнуто */}
      <Card withBorder padding="md">
        <Group justify="space-between" style={{ cursor: "pointer" }} onClick={() => setShowFiltered((v) => !v)}>
          <Title order={5}>Отфильтрованные <Text span size="xs" c="dimmed">({filtered.length})</Text></Title>
          <IconChevronDown size={18} style={{ transform: showFiltered ? "rotate(180deg)" : undefined, transition: "transform .15s" }} />
        </Group>
        <Collapse in={showFiltered}>
          <Text size="sm" c="dimmed" mt="sm" mb="sm">
            Сообщения, отброшенные шаблонами-мусором (баланс, коды и т.п.). Не удалены — на случай,
            если фильтр поймал лишнее: тогда «Вернуть» отправит сообщение обратно во «Входящие».
          </Text>
          <Table striped>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Дата</Table.Th>
                <Table.Th>Текст</Table.Th>
                <Table.Th>Отфильтровано типом</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filtered.map((d) => (
                <Table.Tr key={d.id}>
                  <Table.Td>{d.receivedAt}</Table.Td>
                  <Table.Td><Text size="sm" lineClamp={2} maw={280}>{d.rawText}</Text></Table.Td>
                  <Table.Td><Badge variant="light" color="gray">{tplName(d.templateId)}</Badge></Table.Td>
                  <Table.Td>
                    <Group gap={4} justify="flex-end" wrap="nowrap">
                      <Button size="xs" variant="subtle" onClick={async () => { await restoreDraft(d.id); void load(); }}>Вернуть</Button>
                      <ActionIcon variant="subtle" color="red" onClick={async () => { await deleteDraft(d.id); void load(); }}><IconTrash size={16} /></ActionIcon>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
              {filtered.length === 0 && (
                <Table.Tr><Table.Td colSpan={4}><Text c="dimmed" ta="center" py="sm">Пока ничего не отфильтровано.</Text></Table.Td></Table.Tr>
              )}
            </Table.Tbody>
          </Table>
        </Collapse>
      </Card>

      {/* Справочник контрагентов и счетов */}
      <MerchantsCard merchants={merchants} onChange={load} />

      {/* Шаблоны — продвинутое, свёрнуто */}
      <Card withBorder padding="md">
        <Group justify="space-between" style={{ cursor: "pointer" }} onClick={() => setAdvanced((v) => !v)}>
          <Title order={5}>Типы сообщений <Text span size="xs" c="dimmed">({templates.length})</Text></Title>
          <IconChevronDown size={18} style={{ transform: advanced ? "rotate(180deg)" : undefined, transition: "transform .15s" }} />
        </Group>
        <Collapse in={advanced}>
          <Text size="sm" c="dimmed" mt="sm">
            Типы (шаблоны) создаются сами при разборе «по образцу». Операция — извлекает сумму/контрагента;
            мусор — отбрасывает сообщение. Здесь можно посмотреть, поправить regex вручную или удалить.
          </Text>
          <Table striped mt="sm">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Название</Table.Th>
                <Table.Th>Действие</Table.Th>
                <Table.Th>Отправитель</Table.Th>
                <Table.Th>Признак</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {templates.map((t) => (
                <Table.Tr key={t.id}>
                  <Table.Td>{t.name}</Table.Td>
                  <Table.Td>
                    {t.action === "discard"
                      ? <Badge color="gray" variant="light">выкидывать</Badge>
                      : <Badge color="blue" variant="light">операция</Badge>}
                  </Table.Td>
                  <Table.Td>{t.sender || "любой"}</Table.Td>
                  <Table.Td>
                    {t.action === "discard" ? <Text c="dimmed" size="sm">—</Text>
                      : t.merchantGroup > 0 ? (t.captureKind === "account" ? "счёт" : "контрагент")
                      : <Text c="dimmed" size="sm">нет</Text>}
                  </Table.Td>
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
                <Table.Tr><Table.Td colSpan={5}><Text c="dimmed" ta="center" py="sm">Типов пока нет — создай первый через «Разобрать» во Входящих.</Text></Table.Td></Table.Tr>
              )}
            </Table.Tbody>
          </Table>
        </Collapse>
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

// TeachModal — учим разбор на реальном SMS: выбираем действие (операция/мусор),
// тыкаем в слова, отмечая поля/признак-мусора; система собирает шаблон.
function TeachModal({ opened, onClose, draft, onDone }: {
  opened: boolean; onClose: () => void; draft: InboxDraft; onDone: () => void;
}) {
  const tokens = useMemo(() => tokenize(draft.rawText), [draft.rawText]);
  const [action, setAction] = useState<SmsAction>("operation");
  const [role, setRole] = useState<TeachRole>("amount");
  const [amountI, setAmountI] = useState<number | null>(null);
  const [currI, setCurrI] = useState<number | null>(null);
  const [merchIs, setMerchIs] = useState<number[]>([]);
  const [kind, setKind] = useState<SignalKind>("merchant");
  const [type, setType] = useState<EntryType>(draft.type ?? "expense");
  const [catId, setCatId] = useState<string | null>(null);
  const [name, setName] = useState(draft.rawSender || "Новый тип");

  const discard = action === "discard";

  function click(i: number) {
    if (discard || role === "merchant") {
      setMerchIs((p) => (p.includes(i) ? p.filter((x) => x !== i) : [...p, i].sort((a, b) => a - b)));
    } else if (role === "amount") setAmountI((p) => (p === i ? null : i));
    else setCurrI((p) => (p === i ? null : i));
  }
  function roleOf(i: number): TeachRole | null {
    if (!discard && i === amountI) return "amount";
    if (!discard && i === currI) return "currency";
    if (merchIs.includes(i)) return "merchant";
    return null;
  }

  const amountText = amountI != null ? tokens[amountI].text : "";
  const currencyText = currI != null ? tokens[currI].text : "";
  const marked = merchIs.length
    ? draft.rawText.slice(tokens[merchIs[0]].start, tokens[merchIs[merchIs.length - 1]].end)
    : "";

  async function save() {
    if (!name.trim()) { notifyError(new Error("Задайте название типа")); return; }
    try {
      if (discard) {
        if (!marked) { notifyError(new Error("Отметьте слово-признак мусора")); return; }
        await createTemplateFromSample({
          draftId: draft.id, name, sender: draft.rawSender, text: draft.rawText,
          action: "discard", signatureText: marked,
          amountText: "", currencyText: "", fixedCurrency: "", merchantText: "", captureKind: kind,
          type, categoryId: "",
        });
        onDone();
        notifyOk("Тип-фильтр создан, сообщение в архиве");
        return;
      }
      if (!amountText) { notifyError(new Error("Отметьте сумму")); return; }
      if (!catId) { notifyError(new Error("Выберите категорию")); return; }
      const res = await createTemplateFromSample({
        draftId: draft.id, name, sender: draft.rawSender, text: draft.rawText,
        action: "operation",
        amountText, currencyText, fixedCurrency: currencyText ? "" : "BYN",
        merchantText: marked, captureKind: kind, type, categoryId: catId,
      });
      onDone();
      notifyOk(res.transactionId ? "Тип создан, операция добавлена" : "Тип создан");
    } catch (e) {
      notifyError(e);
    }
  }

  return (
    <Modal opened={opened} onClose={onClose} title="Новый тип по образцу" size="lg">
      <Stack gap="sm">
        <SegmentedControl
          value={action}
          onChange={(v) => setAction(v as SmsAction)}
          data={[{ label: "Создать операцию", value: "operation" }, { label: "Это мусор (выкидывать)", value: "discard" }]}
          fullWidth
        />
        <Text size="sm" c="dimmed">
          {discard
            ? "Отметь слово-признак, по которому узнавать такой мусор (напр. «Dostupno», «Balance»)."
            : "Выбери роль ниже, затем ткни в нужные слова. Признак (контрагент/счёт) можно из нескольких слов."}
        </Text>

        {!discard && (
          <Group gap="xs">
            <Button size="xs" variant={role === "amount" ? "filled" : "light"} color="blue" onClick={() => setRole("amount")}>Сумма</Button>
            <Button size="xs" variant={role === "currency" ? "filled" : "light"} color="teal" onClick={() => setRole("currency")}>Валюта</Button>
            <Button size="xs" variant={role === "merchant" ? "filled" : "light"} color="grape" onClick={() => setRole("merchant")}>Контрагент / счёт</Button>
          </Group>
        )}

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

        {discard ? (
          <Text size="sm"><b>Признак мусора:</b> {marked || "—"}</Text>
        ) : (
          <>
            <Group gap="lg" wrap="wrap">
              <Text size="sm"><b>Сумма:</b> {amountText || "—"}</Text>
              <Text size="sm"><b>Валюта:</b> {currencyText || "BYN (по умолч.)"}</Text>
              <Text size="sm"><b>{kind === "account" ? "Счёт" : "Контрагент"}:</b> {marked || "—"}</Text>
            </Group>
            {marked && (
              <SegmentedControl
                size="xs"
                value={kind}
                onChange={(v) => setKind(v as SignalKind)}
                data={[{ label: "Это контрагент", value: "merchant" }, { label: "Это счёт", value: "account" }]}
              />
            )}
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
              label={marked ? "Категория для этого контрагента/счёта" : "Категория для всех таких сообщений"}
            />
          </>
        )}

        <TextInput label="Название типа" value={name} onChange={(e) => setName(e.currentTarget.value)} />

        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={save}>{discard ? "Создать фильтр" : "Создать тип"}</Button>
        </Group>
      </Stack>
    </Modal>
  );
}
