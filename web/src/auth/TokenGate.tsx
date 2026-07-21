import { useState } from "react";
import {
  Button,
  Card,
  Center,
  PasswordInput,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import { checkAuth, setToken } from "../api/client";

export function TokenGate({ onAuthed }: { onAuthed: () => void }) {
  const [value, setValue] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    setError("");
    setLoading(true);
    setToken(value.trim());
    try {
      await checkAuth();
      onAuthed();
    } catch {
      setError("Неверный токен или сервер недоступен");
    } finally {
      setLoading(false);
    }
  }

  return (
    <Center h="100vh" p="md">
      <Card withBorder shadow="sm" padding="lg" w={360} maw="100%">
        <Stack>
          <Title order={3}>Advisor</Title>
          <Text size="sm" c="dimmed">
            Введите токен доступа к вашему серверу.
          </Text>
          <PasswordInput
            label="Токен"
            value={value}
            onChange={(e) => setValue(e.currentTarget.value)}
            onKeyDown={(e) => e.key === "Enter" && submit()}
            error={error || undefined}
            autoFocus
          />
          <Button onClick={submit} loading={loading} disabled={!value.trim()}>
            Войти
          </Button>
        </Stack>
      </Card>
    </Center>
  );
}
