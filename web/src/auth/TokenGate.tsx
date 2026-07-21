import { useEffect, useState } from "react";
import {
  Button,
  Card,
  Center,
  PasswordInput,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { authStatus, login, register, setToken } from "../api/client";

// TokenGate — экран входа по логину/паролю. На первом запуске (аккаунта ещё нет)
// показывает создание аккаунта, дальше — вход.
export function TokenGate({ onAuthed }: { onAuthed: () => void }) {
  const [registered, setRegistered] = useState<boolean | null>(null);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    authStatus()
      .then((s) => setRegistered(s.registered))
      .catch(() => setRegistered(true)); // при ошибке считаем, что аккаунт есть → показываем вход
  }, []);

  async function submit() {
    setError("");
    if (!username.trim() || password.length < 6) {
      setError("Введите логин и пароль (не короче 6 символов)");
      return;
    }
    setLoading(true);
    try {
      const fn = registered ? login : register;
      const res = await fn(username.trim(), password);
      setToken(res.token);
      onAuthed();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Ошибка входа";
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  const isRegister = registered === false;

  return (
    <Center h="100vh" p="md">
      <Card withBorder shadow="sm" padding="lg" w={360} maw="100%">
        <Stack>
          <Title order={3}>Advisor</Title>
          <Text size="sm" c="dimmed">
            {isRegister ? "Создайте аккаунт — это первый вход." : "Войдите в аккаунт."}
          </Text>
          <TextInput
            label="Логин"
            value={username}
            onChange={(e) => setUsername(e.currentTarget.value)}
            autoFocus
          />
          <PasswordInput
            label="Пароль"
            value={password}
            onChange={(e) => setPassword(e.currentTarget.value)}
            onKeyDown={(e) => e.key === "Enter" && submit()}
            error={error || undefined}
          />
          <Button onClick={submit} loading={loading || registered === null}>
            {isRegister ? "Создать аккаунт" : "Войти"}
          </Button>
        </Stack>
      </Card>
    </Center>
  );
}
