import { AuthSession, ClientSession } from "../types";

interface GameMenuProps {
  authSession: AuthSession | null;
  savedSession: ClientSession | null;
  busy: boolean;
  connectionLabel: string;
  onOpenChess: () => void;
  onOpenTicTacToe: () => void;
  onResumeSavedGame: () => void;
  onLogout: () => void;
}

const resolveSavedSessionGame = (savedSession: ClientSession | null): "chess" | "tictactoe" | null => {
  if (!savedSession) {
    return null;
  }

  return savedSession.gameType ?? "chess";
};

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
  onOpenTicTacToe,
  onResumeSavedGame,
  onLogout
}: GameMenuProps) => {
  const savedSessionGame = resolveSavedSessionGame(savedSession);

  return (
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
              O hub prepara a aplicação para múltiplos jogos. Xadrez e jogo da velha
              dividem o mesmo portal sem quebrar a navegação principal.
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
            {savedSession && savedSessionGame === "chess" ? (
              <button className="button button-secondary" disabled={busy} onClick={onResumeSavedGame}>
                Retomar sala {savedSession.roomCode}
              </button>
            ) : null}
          </div>
        </article>

        <article className="panel game-card game-card-active">
          <div className="game-card-copy">
            <span className="eyebrow">Disponível</span>
            <h2>Jogo da Velha</h2>
            <p className="lead">
              Multiplayer, treino contra máquina, chat da sala, WebRTC entre jogadores
              e observabilidade reaproveitando a mesma malha do xadrez.
            </p>
          </div>
          <div className="action-grid">
            <button className="button" disabled={busy} onClick={onOpenTicTacToe}>
              Abrir jogo da velha
            </button>
            {savedSession && savedSessionGame === "tictactoe" ? (
              <button className="button button-secondary" disabled={busy} onClick={onResumeSavedGame}>
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
      </div>
    </section>
  );
};
