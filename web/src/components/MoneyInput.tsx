import { Group, NumberInput, Select } from "@mantine/core";
import type { Money } from "../api/types";
import { useCurrencies } from "../state/currencies";

// MoneyInput — ввод суммы + выбор валюты. Возвращает Money (десятичная строка).
export function MoneyInput({
  value,
  onChange,
  label = "Сумма",
}: {
  value: Money;
  onChange: (m: Money) => void;
  label?: string;
}) {
  const { codes } = useCurrencies();
  const options = codes.length ? codes : ["BYN"];

  const num = value.amount === "" ? "" : Number(value.amount);

  return (
    <Group grow align="flex-end" gap="xs">
      <NumberInput
        label={label}
        value={num}
        onChange={(v) =>
          onChange({ ...value, amount: v === "" ? "" : Number(v).toFixed(2) })
        }
        decimalScale={2}
        min={0}
        step={1}
        thousandSeparator=" "
      />
      <Select
        label="Валюта"
        data={options}
        value={value.currency}
        onChange={(c) => c && onChange({ ...value, currency: c })}
        maw={110}
        allowDeselect={false}
      />
    </Group>
  );
}
