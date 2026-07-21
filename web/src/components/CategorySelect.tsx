import { Select } from "@mantine/core";
import type { EntryType } from "../api/types";
import { useCategories } from "../state/categories";

// CategorySelect — выбор категории/подкатегории заданного типа (только активные).
// Значение — id категории. Подкатегории показываются как «Родитель / Ребёнок».
export function CategorySelect({
  type,
  value,
  onChange,
  label = "Категория",
  required,
  hideLabel,
  placeholder = "Выберите категорию",
}: {
  type: EntryType;
  value: string | null;
  onChange: (id: string | null) => void;
  label?: string;
  required?: boolean;
  hideLabel?: boolean;
  placeholder?: string;
}) {
  const { all } = useCategories();

  const active = all.filter((c) => !c.archived && c.type === type);
  const parents = active.filter((c) => !c.parentId);

  const data = parents.flatMap((p) => {
    const children = active.filter((c) => c.parentId === p.id);
    return [
      { value: p.id, label: p.name },
      ...children.map((c) => ({ value: c.id, label: `   ${p.name} / ${c.name}` })),
    ];
  });

  return (
    <Select
      label={hideLabel ? undefined : label}
      placeholder={placeholder}
      data={data}
      value={value}
      onChange={onChange}
      searchable
      required={required}
      nothingFoundMessage="Не найдено"
    />
  );
}
