import { useEffect, useState } from "react";
import {
  ActionIcon,
  Alert,
  Badge,
  Button,
  Card,
  Checkbox,
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
  createRule,
  createSmsTemplate,
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
import type { CategoryRule, EntryType, InboxDraft, Merchant, Money, SmsTemplate, SmsTestResult } from "../api/types";
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
                {testResult.merchant ? <>, продавец: <b>{testResult.merchant}</b></> : null}
                {testResult.defaultCategoryId ? `, категория: ${displayName(testResult.defaultCategoryId)}` : " (без категории → во входящие)"}
              </Alert>
            ) : (
              <Alert color="orange" title="Ни один шаблон не совпал">Проверьте отправителя и regex.</Alert>
            )
          )}
        </Stack>
      </Card>

      {/* Справочник продавцов */}
      <MerchantsCard merchants={merchants} onChange={load} />

      {/* Правила «продавец → категория» */}
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

// MerchantsCard — авто-накапливаемый справочник продавцов из SMS. Назначение
// категории прямо из строки создаёт правило «продавец → категория».
function MerchantsCard({ merchants, onChange }: { merchants: Merchant[]; onChange: () => void }) {
  async function assign(name: string, categoryId: string) {
    try {
      await createRule(name, categoryId);
      onChange();
      notifyOk(`«${name}» → категория назначена`);
    } catch (e) {
      notifyError(e);
    }
  }

  return (
    <Card withBorder padding="md">
      <Title order={5} mb="xs">Справочник продавцов</Title>
      <Text size="sm" c="dimmed" mb="sm">
        Продавцы копятся автоматически из SMS. Назначьте категорию — создастся правило,
        и будущие платежи этого продавца будут разноситься сами. Оборот считается по
        основной валюте продавца.
      </Text>
      <Table striped>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Продавец</Table.Th>
            <Table.Th ta="right">Встреч</Table.Th>
            <Table.Th ta="right">Оборот</Table.Th>
            <Table.Th>Категория</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {merchants.map((m) => (
            <MerchantRow key={m.name} m={m} onAssign={assign} />
          ))}
          {merchants.length === 0 && (
            <Table.Tr><Table.Td colSpan={4}><Text c="dimmed" ta="center" py="sm">Продавцов пока нет — появятся после разбора SMS.</Text></Table.Td></Table.Tr>
          )}
        </Table.Tbody>
      </Table>
    </Card>
  );
}

function MerchantRow({ m, onAssign }: { m: Merchant; onAssign: (name: string, categoryId: string) => void }) {
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
            onChange={(id) => id && onAssign(m.name, id)}
            hideLabel
            placeholder="Назначить…"
          />
        </div>
      </Table.Td>
    </Table.Tr>
  );
}

// RulesCard — правила «продавец → категория»: список + добавление.
function RulesCard({ rules, onChange }: { rules: CategoryRule[]; onChange: () => void }) {
  const { displayName } = useCategories();
  const [pattern, setPattern] = useState("");
  const [catId, setCatId] = useState<string | null>(null);

  async function add() {
    if (!pattern.trim() || !catId) {
      notifyError(new Error("Укажите продавца и категорию"));
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
      <Title order={5} mb="xs">Правила «продавец → категория»</Title>
      <Text size="sm" c="dimmed" mb="sm">
        Если название продавца из SMS содержит эту подстроку — операция автоматически
        относится к указанной категории (без «Входящих»).
      </Text>
      <Table striped mb="sm">
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Продавец содержит</Table.Th>
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
        <TextInput label="Продавец (подстрока)" placeholder="YANDEX" value={pattern} onChange={(e) => setPattern(e.currentTarget.value)} />
        <div style={{ minWidth: 200 }}>
          <CategorySelect type="expense" value={catId} onChange={setCatId} label="Категория" />
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={add}>Добавить</Button>
      </Group>
    </Card>
  );
}
