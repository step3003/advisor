// DTO-типы API (зеркалят internal/transport/http/dto.go).

export interface Money {
  amount: string; // десятичная строка, без float
  currency: string;
}

export type EntryType = "expense" | "income";

export interface Category {
  id: string;
  name: string;
  type: EntryType;
  parentId?: string;
  color?: string;
  icon?: string;
  isBuiltin: boolean;
  archived: boolean;
}

export interface Transaction {
  id: string;
  date: string; // YYYY-MM-DD
  type: EntryType;
  categoryId: string;
  amount: Money;
  note?: string;
  recurringId?: string;
}

export interface PlanItem {
  id: string;
  period: string; // YYYY-MM
  categoryId: string;
  amount: Money;
  note?: string;
}

export interface PlanVsFactRow {
  categoryId: string;
  plan: Money;
  fact: Money;
  deviation: Money;
  remaining: Money;
  percent: number;
  overspent: boolean;
}

export interface PlanVsFact {
  period: string;
  rows: PlanVsFactRow[];
  planIncome: Money;
  planExpense: Money;
  factIncome: Money;
  factExpense: Money;
  balance: Money;
  remainingExpense: Money;
}

export interface CategoryAmount {
  categoryId: string;
  amount: Money;
}

export interface PeriodSummary {
  from: string;
  to: string;
  totalIncome: Money;
  totalExpense: Money;
  balance: Money;
  expenseByCategory: CategoryAmount[];
  incomeByCategory: CategoryAmount[];
  expenseByCurrency: Money[];
}

export interface MonthPoint {
  period: string; // YYYY-MM
  income: Money;
  expense: Money;
}

export interface Recurring {
  id: string;
  type: EntryType;
  categoryId: string;
  amount: Money;
  dayOfMonth: number;
  startDate: string;
  endDate?: string;
  autoCreateFact: boolean;
  active: boolean;
}

export interface CurrencyInfo {
  code: string;
  name: string;
}

export interface Settings {
  defaultCurrency: string;
  currencies: CurrencyInfo[];
}

export interface SmsTemplate {
  id: string;
  name: string;
  sender: string;
  pattern: string;
  amountGroup: number;
  currencyGroup: number;
  merchantGroup: number;
  fixedCurrency: string;
  type: EntryType;
  defaultCategoryId: string;
  enabled: boolean;
  priority: number;
}

export interface SmsTestResult {
  matched: boolean;
  templateName?: string;
  amount?: Money;
  type?: EntryType;
  merchant?: string;
  defaultCategoryId?: string;
}

export interface InboxDraft {
  id: string;
  rawSender: string;
  rawText: string;
  receivedAt: string;
  amount?: Money;
  type?: EntryType;
  merchant?: string;
  templateId?: string;
  resolved: boolean;
}
