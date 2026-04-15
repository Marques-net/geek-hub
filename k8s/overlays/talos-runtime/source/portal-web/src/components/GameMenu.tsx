import { AuthSession, ClientSession } from "../types";

interface GameMenuProps {
  authSession: AuthSession | null;
  savedSession: ClientSession | null;
  busy: boolean;
  connectionLabel: string;
  onOpenChess: () => void;
  onResumeSavedChess: () => void;
  onLogout: () => void;
}

const renderProfileCaption = (authSession: AuthSession | null): string => {
  if (!authSession) {
    return "Visitante local";
  }

  if (authSession.provider === "guest") {
    return "Sessão local";
  }

  return authSession.email ?? authSession.name;
};

export const GameMenu = ({
  authSession,
  savedSession,
  busy,
  connectionLabel,
  onOpenChess,
  onResumeSavedChess,
  onLogout
}: GameMenuProps) => (
  <section className="menu-shell">
    <header className="panel menu-header">
      <div className="menu-profile">
        <div className="profile-chip profile-chip-large">
          {authSession?.picture ? (
            <img src={authSession.picture} alt={authSession.name} />
          ) : (
            <span>{(authSession?.name ?? "Visitante").slice(0, 1).toUpperCase()}</span>
          )}
          <div>
            <strong>{authSession?.name ?? "Visitante"}</strong>
            <small>{renderProfileCaption(authSession)}</small>
          </div>
        </div>

        <div>
          <div className="eyebrow">Menu de Jogos</div>
          <h1>Escolha o que jogar</h1>
          <p className="lead">
            O hub prepara a aplicação para múltiplos jogos. O xadrez já está
            disponível; os demais entram como extensões do mesmo portal.
          </p>
        </div>
      </div>

      <div className="topbar-actions">
        <div className="connection-badge">{connectionLabel}</div>
        <button className="button button-ghost" onClick={onLogout}>
          Sair
        </button>
      </div>
    </header>

    <div className="menu-grid">
      <article className="panel game-card game-card-active">
        <div className="game-card-copy">
          <span className="eyebrow">Disponível</span>
          <h2>Xadrez Online</h2>
          <p className="lead">
            Multiplayer, treino contra máquina, relógio opcional e observabilidade
            já integrada ao cluster.
          </p>
        </div>
        <div className="action-grid">
          <button className="button" disabled={busy} onClick={onOpenChess}>
            Abrir xadrez
          </button>
          {savedSession ? (
            <button className="button button-secondary" disabled={busy} onClick={onResumeSavedChess}>
              Retomar sala {savedSession.roomCode}
            </button>
          ) : null}
        </div>
      </article>

      <article className="panel game-card game-card-disabled">
        <div className="game-status-pill">Em breve</div>
        <h2>Damas</h2>
        <p className="lead">
          Carta reservada para um segundo jogo de tabuleiro com reutilização do
          mesmo portal e da mesma telemetria.
        </p>
        <button className="button button-ghost" disabled>
          Em breve
        </button>
      </article>

      <article className="panel game-card game-card-disabled">
        <div className="game-status-pill">Em breve</div>
        <h2>Jogo da Velha</h2>
        <p className="lead">
          Opção rápida para validar novas jornadas de autenticação, menu e
          analytics sem mexer no xadrez.
        </p>
        <button className="button button-ghost" disabled>
          Em breve
        </button>
      </article>
    </div>
  </section>
);
