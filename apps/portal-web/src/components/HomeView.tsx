import { GoogleSignInButton } from "./GoogleSignInButton";
import { AuthSession } from "../types";

interface HomeViewProps {
  busy: boolean;
  onAuthSuccess: (session: AuthSession) => void;
  onAuthError: (message: string) => void;
  onContinueAsGuest: () => void;
}

export const HomeView = ({
  busy,
  onAuthSuccess,
  onAuthError,
  onContinueAsGuest
}: HomeViewProps) => (
  <section className="panel home-panel home-panel-simple">
    <div className="home-simple">
      <div className="home-simple-copy">
        <h1>Portal geek-hub</h1>
        <p className="lead">
          O portal centraliza a entrada dos jogos do ambiente local. Entre com sua
          conta Google ou continue como visitante para abrir o menu principal.
        </p>
      </div>

      <div className="home-actions">
        <GoogleSignInButton busy={busy} onSuccess={onAuthSuccess} onError={onAuthError} />
        <button className="button button-ghost" disabled={busy} onClick={onContinueAsGuest}>
          Continuar como visitante
        </button>
      </div>
    </div>
  </section>
);
