import { GoogleSignInButton } from "./GoogleSignInButton";
import { AuthSession } from "../types";

interface HomeViewProps {
  busy: boolean;
  connectionLabel: string;
  onAuthSuccess: (session: AuthSession) => void;
  onAuthError: (message: string) => void;
  onContinueAsGuest: () => void;
}

export const HomeView = ({
  busy,
  connectionLabel,
  onAuthSuccess,
  onAuthError,
  onContinueAsGuest
}: HomeViewProps) => (
  <section className="panel home-panel">
    <div className="home-hero">
      <div className="home-copy">
        <div className="eyebrow">Arcade Local</div>
        <h1>Entre no hub de jogos</h1>
        <p className="lead">
          A nova página inicial separa autenticação, menu de jogos e acesso ao
          xadrez. O fluxo fica mais próximo de um portal e prepara o caminho para
          novos jogos no mesmo cluster.
        </p>

        <div className="feature-grid">
          <article className="feature-card">
            <strong>Login Google</strong>
            <span>Identidade do usuário antes de entrar no lobby.</span>
          </article>
          <article className="feature-card">
            <strong>Menu central</strong>
            <span>Entrada única para xadrez e jogos futuros.</span>
          </article>
          <article className="feature-card">
            <strong>Backend autoritativo</strong>
            <span>As regras do xadrez continuam isoladas do cliente.</span>
          </article>
        </div>
      </div>

      <aside className="panel auth-panel">
        <div className="eyebrow">Acesso</div>
        <h2>Faça login</h2>
        <p className="lead">
          Use sua conta Google para abrir o menu de jogos. Em ambiente local, você
          ainda pode continuar como visitante para testes.
        </p>

        <div className="connection-badge">{connectionLabel}</div>

        <GoogleSignInButton busy={busy} onSuccess={onAuthSuccess} onError={onAuthError} />

        <div className="divider-line" />

        <button className="button button-ghost" disabled={busy} onClick={onContinueAsGuest}>
          Continuar como visitante
        </button>
      </aside>
    </div>
  </section>
);
