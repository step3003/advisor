import { useState } from "react";
import {
  ActionIcon,
  Badge,
  Button,
  Card,
  Checkbox,
  Group,
  Modal,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import {
  IconArchive,
  IconArchiveOff,
  IconEdit,
  IconPlus,
  IconTrash,
} from "@tabler/icons-react";

import {
  createCategory,
  deleteCategory,
  patchCategory,
} from "../api/client";
import type { Category, EntryType } from "../api/types";
import { useCategories } from "../state/categories";
import { notifyError } from "../lib/notify";

type Dialog =
  | { kind: "createTop"; type: EntryType }
  | { kind: "createSub"; parent: Category }
  | { kind: "rename"; cat: Category };

export function CategoriesPage() {
  const { all, refresh } = useCategories();
  const [showArchived, setShowArchived] = useState(false);
  const [opened, { open, close }] = useDisclosure(false);
  const [dialog, setDialog] = useState<Dialog | null>(null);
  const [name, setName] = useState("");

  const visible = all.filter((c) => showArchived || !c.archived);

  function startDialog(d: Dialog, initial = "") {
    setDialog(d);
    setName(initial);
    open();
  }

  async function submit() {
    if (!dialog) return;
    if (!name.trim()) {
      notifyError(new Error("Название не может быть пустым"));
      return;
    }
    try {
      if (dialog.kind === "createTop") {
        await createCategory(name.trim(), dialog.type);
      } else if (dialog.kind === "createSub") {
        await createCategory(name.trim(), dialog.parent.type, dialog.parent.id);
      } else {
        await patchCategory(dialog.cat.id, { name: name.trim() });
      }
      close();
      await refresh();
    } catch (e) {
      notifyError(e);
    }
  }

  async function archive(c: Category, archived: boolean) {
    try {
      await patchCategory(c.id, { archived });
      await refresh();
    } catch (e) {
      notifyError(e);
    }
  }

  async function remove(c: Category) {
    if (!confirm("Удалить категорию без возможности восстановления?")) return;
    try {
      await deleteCategory(c.id);
      await refresh();
    } catch {
      notifyError(
        new Error("Нельзя удалить: по категории есть операции или планы. Используйте архивацию."),
      );
    }
  }

  function section(type: EntryType) {
    const tops = visible
      .filter((c) => c.type === type && !c.parentId)
      .sort((a, b) => a.name.localeCompare(b.name));
    return (
      <Stack gap={4}>
        {tops.map((p) => (
          <div key={p.id}>
            <Row c={p} onRename={() => startDialog({ kind: "rename", cat: p }, p.name)}
              onAddSub={() => startDialog({ kind: "createSub", parent: p })}
              onArchive={() => archive(p, !p.archived)} onDelete={() => remove(p)} />
            {visible
              .filter((c) => c.parentId === p.id)
              .sort((a, b) => a.name.localeCompare(b.name))
              .map((child) => (
                <Row key={child.id} child c={child}
                  onRename={() => startDialog({ kind: "rename", cat: child }, child.name)}
                  onArchive={() => archive(child, !child.archived)} onDelete={() => remove(child)} />
              ))}
          </div>
        ))}
      </Stack>
    );
  }

  const title =
    dialog?.kind === "rename"
      ? "Переименовать категорию"
      : dialog?.kind === "createSub"
        ? "Новая подкатегория"
        : "Новая категория";

  return (
    <Stack>
      <Title order={3}>Категории</Title>
      <Group>
        <Button leftSection={<IconPlus size={16} />} onClick={() => startDialog({ kind: "createTop", type: "expense" })}>
          Категория расхода
        </Button>
        <Button variant="light" leftSection={<IconPlus size={16} />} onClick={() => startDialog({ kind: "createTop", type: "income" })}>
          Категория дохода
        </Button>
        <Checkbox label="Показывать архивные" checked={showArchived} onChange={(e) => setShowArchived(e.currentTarget.checked)} />
      </Group>

      <Card withBorder padding="md">
        <Title order={5} mb="xs">Расходы</Title>
        {section("expense")}
      </Card>
      <Card withBorder padding="md">
        <Title order={5} mb="xs">Доходы</Title>
        {section("income")}
      </Card>

      <Modal opened={opened} onClose={close} title={title}>
        <Stack>
          <TextInput label="Название" value={name} data-autofocus
            onChange={(e) => setName(e.currentTarget.value)}
            onKeyDown={(e) => e.key === "Enter" && submit()} />
          <Group justify="flex-end">
            <Button variant="default" onClick={close}>Отмена</Button>
            <Button onClick={submit}>Сохранить</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}

function Row({
  c, child, onRename, onAddSub, onArchive, onDelete,
}: {
  c: Category;
  child?: boolean;
  onRename: () => void;
  onAddSub?: () => void;
  onArchive: () => void;
  onDelete: () => void;
}) {
  return (
    <Group justify="space-between" pl={child ? "lg" : 0} py={2}>
      <Group gap="xs">
        <Text>{c.name}</Text>
        {c.isBuiltin && <Badge size="xs" variant="light" color="gray">встроенная</Badge>}
        {c.archived && <Badge size="xs" variant="light" color="orange">в архиве</Badge>}
      </Group>
      <Group gap={2} wrap="nowrap">
        <ActionIcon variant="subtle" onClick={onRename} aria-label="Переименовать"><IconEdit size={16} /></ActionIcon>
        {onAddSub && <ActionIcon variant="subtle" onClick={onAddSub} aria-label="Подкатегория"><IconPlus size={16} /></ActionIcon>}
        <ActionIcon variant="subtle" onClick={onArchive} aria-label="Архив">
          {c.archived ? <IconArchiveOff size={16} /> : <IconArchive size={16} />}
        </ActionIcon>
        <ActionIcon variant="subtle" color="red" onClick={onDelete} aria-label="Удалить"><IconTrash size={16} /></ActionIcon>
      </Group>
    </Group>
  );
}
