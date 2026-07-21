import type {
  Category,
  CurrencyInfo,
  EntryType,
  InboxDraft,
  MonthPoint,
  Money,
  PeriodSummary,
  PlanItem,
  PlanVsFact,
  Recurring,
  Settings,
  SmsTemplate,
  SmsTestResult,
  Transaction,
} from "./types";

const TOKEN_KEY = "advisor.token";

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {
    Authorization: `Bearer ${getToken()}`,
  };
  if (body !== undefined) headers["Content-Type"] = "application/json";

  const res = await fetch(path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  const data = text ? JSON.parse(text) : undefined;
  if (!res.ok) {
    const msg = data?.error ?? `Ошибка ${res.status}`;
    throw new ApiError(res.status, msg);
  }
  return data as T;
}

// --- Категории ---
export const listCategories = (includeArchived = false) =>
  req<Category[]>("GET", `/api/categories?includeArchived=${includeArchived}`);
export const createCategory = (name: string, type: EntryType, parentId?: string) =>
  req<Category>("POST", "/api/categories", { name, type, parentId: parentId ?? "" });
export const patchCategory = (id: string, patch: { name?: string; archived?: boolean }) =>
  req<Category>("PATCH", `/api/categories/${id}`, patch);
export const deleteCategory = (id: string) =>
  req<void>("DELETE", `/api/categories/${id}`);

// --- Планы ---
export const listPlans = (ym: string) =>
  req<PlanItem[]>("GET", `/api/plans?ym=${ym}`);
export const setPlan = (period: string, categoryId: string, amount: Money, note = "") =>
  req<PlanItem>("PUT", "/api/plans", { period, categoryId, amount, note });
export const copyPreviousPlan = (period: string) =>
  req<{ copied: number }>("POST", "/api/plans/copy-previous", { period });

// --- Операции ---
export const listTransactions = (ym: string) =>
  req<Transaction[]>("GET", `/api/transactions?ym=${ym}`);
export interface TxInput {
  date: string;
  type: EntryType;
  categoryId: string;
  amount: Money;
  note?: string;
}
export const createTransaction = (t: TxInput) =>
  req<Transaction>("POST", "/api/transactions", t);
export const updateTransaction = (id: string, t: TxInput) =>
  req<Transaction>("PATCH", `/api/transactions/${id}`, t);
export const deleteTransaction = (id: string) =>
  req<void>("DELETE", `/api/transactions/${id}`);

// --- Отчёты ---
export const planVsFact = (ym: string) =>
  req<PlanVsFact>("GET", `/api/reports/plan-vs-fact?ym=${ym}`);
export const periodReport = (from: string, to: string) =>
  req<PeriodSummary>("GET", `/api/reports/period?from=${from}&to=${to}`);
export const dynamics = (from: string, to: string) =>
  req<MonthPoint[]>("GET", `/api/reports/dynamics?from=${from}&to=${to}`);

// --- Повторяющиеся ---
export const listRecurring = () => req<Recurring[]>("GET", "/api/recurring");
export interface RecurringInput {
  type: EntryType;
  categoryId: string;
  amount: Money;
  dayOfMonth: number;
  startDate: string;
  endDate?: string;
  autoCreateFact: boolean;
}
export const createRecurring = (r: RecurringInput) =>
  req<Recurring>("POST", "/api/recurring", r);
export const updateRecurring = (id: string, r: RecurringInput) =>
  req<Recurring>("PATCH", `/api/recurring/${id}`, r);
export const pauseRecurring = (id: string) =>
  req<void>("POST", `/api/recurring/${id}/pause`);
export const resumeRecurring = (id: string) =>
  req<void>("POST", `/api/recurring/${id}/resume`);
export const deleteRecurring = (id: string) =>
  req<void>("DELETE", `/api/recurring/${id}`);

// --- Валюты / настройки ---
export const listCurrencies = () =>
  req<CurrencyInfo[]>("GET", "/api/currency/currencies");
export const refreshRates = (date?: string) =>
  req<{ updated: number }>("POST", "/api/currency/refresh", date ? { date } : {});
export const getSettings = () => req<Settings>("GET", "/api/settings");
export const setDefaultCurrency = (defaultCurrency: string) =>
  req<Settings>("PATCH", "/api/settings", { defaultCurrency });

// --- SMS-парсер ---
export type SmsTemplateInput = Omit<SmsTemplate, "id">;
export const listSmsTemplates = () =>
  req<SmsTemplate[]>("GET", "/api/sms/templates");
export const createSmsTemplate = (t: SmsTemplateInput) =>
  req<SmsTemplate>("POST", "/api/sms/templates", t);
export const updateSmsTemplate = (id: string, t: SmsTemplateInput) =>
  req<SmsTemplate>("PATCH", `/api/sms/templates/${id}`, t);
export const deleteSmsTemplate = (id: string) =>
  req<void>("DELETE", `/api/sms/templates/${id}`);
export const testSms = (sender: string, text: string) =>
  req<SmsTestResult>("POST", "/api/sms/test", { sender, text });

// --- Входящие (черновики) ---
export const listInbox = (unresolvedOnly = true) =>
  req<InboxDraft[]>("GET", `/api/inbox?unresolvedOnly=${unresolvedOnly}`);
export const resolveDraft = (
  id: string,
  categoryId: string,
  amount?: Money,
  type?: EntryType,
) => req<Transaction>("POST", `/api/inbox/${id}/resolve`, { categoryId, amount, type });
export const deleteDraft = (id: string) =>
  req<void>("DELETE", `/api/inbox/${id}`);

// --- Аккаунты ---
export const authStatus = () =>
  req<{ registered: boolean }>("GET", "/api/auth/status");
export const register = (username: string, password: string, deviceName = "web") =>
  req<{ token: string; username: string }>("POST", "/api/auth/register", { username, password, deviceName });
export const login = (username: string, password: string, deviceName = "web") =>
  req<{ token: string; username: string }>("POST", "/api/auth/login", { username, password, deviceName });
export const logout = () => req<void>("POST", "/api/auth/logout");

// Проверка токена: health не требует авторизации, поэтому пробуем защищённый.
export const checkAuth = () => req<Category[]>("GET", "/api/categories");
