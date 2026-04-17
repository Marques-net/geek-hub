import { useMemo, useState } from "react";

import { GameMenu } from "./components/GameMenu";
import { HomeView } from "./components/HomeView";
import { getClientTelemetry } from "./lib/client-context";
import {
  clearAuthSession,
  clearSession,
  readAuthSession,
  readSession,
  writeAuthSession
} from "./lib/storage";
import { AuthSession, ClientSession, GameType } from "./types";

type PortalScreen = "home" | "menu";

const CHESS_PATH = "/games/chess/";
const TICTACTOE_PATH = "/games/tictactoe/";

const createGuestAuthSession = (): AuthSession => ({
  provider: "guest",
  sub: "guest-local",
  name: "Visitante local",
  givenName: "Visitante",
  email: null,
  picture: null
});

const resolveInitialScreen = (authSession: AuthSession | null): PortalScreen =>
  authSession ? "menu" : "home";

const resolveGamePath = (gameType: GameType | undefined): string => {
  if (gameType === "tictactoe") {
    return TICTACTOE_PATH;
  }

  return CHESS_PATH;
};

const openGameFrontend = (gameType: GameType): void => {
  window.location.assign(resolveGamePath(gameType));
};

const persistGoogleLogin = async (authSession: AuthSession): Promise<void> => {
  const client = await getClientTelemetry();
  const response = await fetch("/api/auth/logins", {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      provider: authSession.provider,
      sub: authSession.sub,
      name: authSession.name,
      email: authSession.email,
      client
    })
  });

  if (response.ok) {
    return;
  }

  let message = "Nao foi possivel registrar o login Google.";

  try {
    const payload = (await response.json()) as { message?: string };
    if (typeof payload.message === "string" && payload.message.trim()) {
      message = payload.message.trim();
    }
  } catch {
    // noop
  }

  throw new Error(message);
};

export default function App() {
  const [authSession, setAuthSession] = useState<AuthSession | null>(() => readAuthSession());
  const [screen, setScreen] = useState<PortalScreen>(() => resolveInitialScreen(readAuthSession()));
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<string | null>(null);

  const savedSession = useMemo(() => readSession(), [authSession, screen]);

  const handleResumeSavedGame = (session: ClientSession | null) => {
    openGameFrontend(session?.gameType ?? "chess");
  };

  const handleAuthSuccess = async (nextAuth: AuthSession) => {
    setPending("Registrando login Google...");
    setError(null);

    try {
      await persistGoogleLogin(nextAuth);
      writeAuthSession(nextAuth);
      setAuthSession(nextAuth);
      setScreen("menu");
    } catch (error) {
      setError(error instanceof Error ? error.message : "Falha ao registrar o login Google.");
    } finally {
      setPending(null);
    }
  };

  const handleContinueAsGuest = () => {
    const guestSession = createGuestAuthSession();
    writeAuthSession(guestSession);
    setAuthSession(guestSession);
    setError(null);
    setScreen("menu");
  };

  const handleLogout = () => {
    window.google?.accounts.id.disableAutoSelect?.();
    clearAuthSession();
    clearSession();
    setAuthSession(null);
    setError(null);
    setPending(null);
    setScreen("home");
  };

  return (
    <main className={screen === "home" ? "app-shell app-shell-home" : "app-shell"}>
      <div className="ambient ambient-one" />
      <div className="ambient ambient-two" />

      {error ? <div className="toast toast-error">{error}</div> : null}
      {pending ? <div className="toast toast-info">{pending}</div> : null}

      {screen === "home" ? (
        <HomeView
          busy={Boolean(pending)}
          onAuthSuccess={handleAuthSuccess}
          onAuthError={setError}
          onContinueAsGuest={handleContinueAsGuest}
        />
      ) : (
        <GameMenu
          authSession={authSession}
          savedSession={savedSession}
          busy={Boolean(pending)}
          connectionLabel="Portal disponível"
          onOpenChess={() => openGameFrontend("chess")}
          onOpenTicTacToe={() => openGameFrontend("tictactoe")}
          onResumeSavedGame={() => handleResumeSavedGame(savedSession)}
          onLogout={handleLogout}
        />
      )}
    </main>
  );
}
