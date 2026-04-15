import { useMemo, useState } from "react";

import { GameMenu } from "./components/GameMenu";
import { HomeView } from "./components/HomeView";
import {
  clearAuthSession,
  clearSession,
  readAuthSession,
  readSession,
  writeAuthSession
} from "./lib/storage";
import { AuthSession } from "./types";

type PortalScreen = "home" | "menu";

const CHESS_PATH = "/games/chess/";

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

const openChessFrontend = (): void => {
  window.location.assign(CHESS_PATH);
};

export default function App() {
  const [authSession, setAuthSession] = useState<AuthSession | null>(() => readAuthSession());
  const [screen, setScreen] = useState<PortalScreen>(() => resolveInitialScreen(readAuthSession()));
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<string | null>(null);

  const savedSession = useMemo(() => readSession(), [authSession, screen]);

  const handleAuthSuccess = (nextAuth: AuthSession) => {
    writeAuthSession(nextAuth);
    setAuthSession(nextAuth);
    setError(null);
    setScreen("menu");
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
    <main className="app-shell">
      <div className="ambient ambient-one" />
      <div className="ambient ambient-two" />

      {error ? <div className="toast toast-error">{error}</div> : null}
      {pending ? <div className="toast toast-info">{pending}</div> : null}

      {screen === "home" ? (
        <HomeView
          busy={Boolean(pending)}
          connectionLabel="Portal disponível"
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
          onOpenChess={openChessFrontend}
          onResumeSavedChess={openChessFrontend}
          onLogout={handleLogout}
        />
      )}
    </main>
  );
}
