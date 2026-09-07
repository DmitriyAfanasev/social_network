import { useEffect, useMemo, useRef, useState } from "react";

import type { AuthResponse, AuthUser } from "../../entities/user/model/user";
import { apiRequest, saveAuthTokens } from "../../shared/api/http";
import { navigate } from "../../shared/lib/navigation";
import { AuthLayout } from "../../shared/ui/AuthLayout";

export function ConfirmRegistrationPage() {
  const token = useMemo(() => new URLSearchParams(window.location.search).get("token"), []);
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [message, setMessage] = useState("Подтверждаем email...");
  const [user, setUser] = useState<AuthUser | null>(null);
  const submittedToken = useRef<string | null>(null);

  useEffect(() => {
    if (!token) {
      setStatus("error");
      setMessage("В ссылке нет token");
      return;
    }
    if (submittedToken.current === token) return;
    submittedToken.current = token;

    apiRequest<AuthResponse>("/registration-confirmations/confirm", {
      method: "POST",
      body: JSON.stringify({ token }),
    })
      .then((result) => {
        saveAuthTokens(result);
        setUser(result.user);
        setStatus("success");
        setMessage("Email подтвержден");
        navigate("/profile/setup");
      })
      .catch((err) => {
        submittedToken.current = null;
        setStatus("error");
        setMessage(err instanceof Error ? err.message : "Не удалось подтвердить email");
      });
  }, [token]);

  return (
    <AuthLayout title="Подтверждение" aside="После подтверждения Go API сразу выдаёт пару токенов для новой сессии.">
      <div className="state-stack">
        <p className={status === "error" ? "error" : "success"}>{message}</p>
        {user && <button onClick={() => navigate("/profile/me")}>Открыть профиль</button>}
      </div>
    </AuthLayout>
  );
}
