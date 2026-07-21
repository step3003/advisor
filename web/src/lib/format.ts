import type { Money } from "../api/types";

// formatMoney форматирует денежную строку с разделением тысяч и кодом валюты.
export function formatMoney(m: Money): string {
  return `${groupThousands(m.amount)} ${m.currency}`;
}

// formatAmount — только число (без валюты).
export function formatAmount(m: Money): string {
  return groupThousands(m.amount);
}

// formatSigned добавляет явный знак «+» для положительных сумм.
export function formatSigned(m: Money): string {
  const isNeg = m.amount.trim().startsWith("-");
  const s = groupThousands(m.amount);
  return isNeg ? s : `+${s}`;
}

// toNumber парсит десятичную строку в число (для графиков; отображение — строкой).
export function moneyToNumber(m: Money): number {
  return parseFloat(m.amount) || 0;
}

function groupThousands(dec: string): string {
  const neg = dec.startsWith("-");
  const body = neg ? dec.slice(1) : dec;
  const [intPart, fracPart] = body.split(".");
  const grouped = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, " ");
  const out = fracPart !== undefined ? `${grouped},${fracPart}` : grouped;
  return neg ? `−${out}` : out;
}

const MONTHS = [
  "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
  "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
];

// monthTitle: "2026-07" → "Июль 2026".
export function monthTitle(ym: string): string {
  const [y, m] = ym.split("-").map(Number);
  return `${MONTHS[(m ?? 1) - 1]} ${y}`;
}

// monthShort: "2026-07" → "07.26".
export function monthShort(ym: string): string {
  const [y, m] = ym.split("-").map(Number);
  return `${String(m).padStart(2, "0")}.${String(y).slice(2)}`;
}

// shiftMonth сдвигает "YYYY-MM" на delta месяцев.
export function shiftMonth(ym: string, delta: number): string {
  const [y, m] = ym.split("-").map(Number);
  const idx = (y * 12 + (m - 1)) + delta;
  const ny = Math.floor(idx / 12);
  const nm = (idx % 12) + 1;
  return `${ny}-${String(nm).padStart(2, "0")}`;
}

// currentMonth возвращает текущий "YYYY-MM".
export function currentMonth(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
}

// todayISO возвращает сегодня в "YYYY-MM-DD".
export function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

// firstDayOfMonth / lastDayOfMonth для "YYYY-MM".
export function firstDayOfMonth(ym: string): string {
  return `${ym}-01`;
}
export function lastDayOfMonth(ym: string): string {
  const [y, m] = ym.split("-").map(Number);
  const day = new Date(y, m, 0).getDate();
  return `${ym}-${String(day).padStart(2, "0")}`;
}
