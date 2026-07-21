import { notifications } from "@mantine/notifications";

export function notifyError(err: unknown): void {
  const msg = err instanceof Error ? err.message : String(err);
  notifications.show({ color: "red", title: "Ошибка", message: msg });
}

export function notifyOk(message: string): void {
  notifications.show({ color: "green", title: "Готово", message });
}
