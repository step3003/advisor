import { ActionIcon, Group, Text } from "@mantine/core";
import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react";
import { monthTitle, shiftMonth } from "../lib/format";

export function MonthNav({
  value,
  onChange,
}: {
  value: string;
  onChange: (ym: string) => void;
}) {
  return (
    <Group gap="xs">
      <ActionIcon variant="default" onClick={() => onChange(shiftMonth(value, -1))} aria-label="Предыдущий месяц">
        <IconChevronLeft size={18} />
      </ActionIcon>
      <Text fw={600} w={160} ta="center">
        {monthTitle(value)}
      </Text>
      <ActionIcon variant="default" onClick={() => onChange(shiftMonth(value, 1))} aria-label="Следующий месяц">
        <IconChevronRight size={18} />
      </ActionIcon>
    </Group>
  );
}
