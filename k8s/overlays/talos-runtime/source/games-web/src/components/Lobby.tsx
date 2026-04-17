import { FormEvent } from "react";

import { AuthSession, ClientSession } from "../types";

interface LobbyProps {
  gameLabel: string;
  gameTitle: string;
  gameDescription: string;
  nicknamePlaceholder: string;
  trainingButtonLabel: string;
  whiteSeatLabel: string;
  blackSeatLabel: string;
  authSession: AuthSession | null;
  nickname: string;
  roomCode: string;
  playWithoutClock: boolean;
  savedSession: ClientSession | null;
  busy: boolean;
  connectionLabel: string;
  onNicknameChange: (value: string) => void;
  onRoomCodeChange: (value: string) => void;
  onPlayWithoutClockChange: (value: boolean) => void;
  onCreateRoom: () => Promise<void>;
  onCreateBotRoom: () => Promise<void>;
  onJoinRoom: () => Promise<void>;
  onReconnectSaved: () => Promise<void>;
  onDiscardSavedSession: () => void;
  onBackToMenu: () => void;
}

const renderAuthLabel = (authSession: AuthSession | null): string => {
  if (!authSession) {
    return "Acesso local";
  }

  if (authSession.provider === "guest") {
    return "Visitante local";
  }

  return authSession.email ?? authSession.name;
};

export const Lobby = ({
  gameLabel,
  gameTitle,
  gameDescription,
  nicknamePlaceholder,
  trainingButtonLabel,
  whiteSeatLabel,
  blackSeatLabel,
  authSession,
  nickname,
  roomCode,
  playWithoutClock,
  savedSession,
  busy,
  connectionLabel,
  onNicknameChange,
  onRoomCodeChange,
  onPlayWithoutClockChange,
  onCreateRoom,
  onCreateBotRoom,
  onJoinRoom,
  onReconnectSaved,
  onDiscardSavedSession,
  onBackToMenu
}: LobbyProps) => {
  const handleCreate = async (event: FormEvent) => {
    event.preventDefault();
    await onCreateRoom();
  };

  const handleJoin = async (event: FormEvent) => {
    event.preventDefault();
    await onJoinRoom();
  };

  return (
    <section className="panel lobby-panel">
      <div className="lobby-topbar">
        <div>
          <div className="eyebrow">{gameLabel}</div>
          <h1>{gameTitle}</h1>
          <p className="lead">{gameDescription}</p>
        </div>
        <div className="topbar-actions">
          <div className="profile-chip">
            {authSession?.picture ? (
              <img src={authSession.picture} alt={authSession.name} />
            ) : (
              <span>{renderAuthLabel(authSession).slice(0, 1).toUpperCase()}</span>
            )}
            <div>
              <strong>{authSession?.name ?? "Visitante"}</strong>
              <small>{renderAuthLabel(authSession)}</small>
            </div>
          </div>
          <button className="button button-ghost" onClick={onBackToMenu}>
            Voltar ao menu
          </button>
        </div>
      </div>

      <div className="connection-badge">{connectionLabel}</div>

      {savedSession ? (
        <div className="saved-session">
          <div>
            <strong>Sessão salva</strong>
            <span>
              Sala {savedSession.roomCode} como{" "}
              {savedSession.color === "w"
                ? whiteSeatLabel
                : savedSession.color === "b"
                  ? blackSeatLabel
                  : "espectador"}
              {savedSession.mode === "bot_easy" ? " em treino vs máquina" : ""}
            </span>
          </div>
          <div className="saved-session-actions">
            <button className="button button-secondary" onClick={() => void onReconnectSaved()}>
              Retomar sala
            </button>
            <button className="button button-ghost" onClick={onDiscardSavedSession}>
              Limpar sessão
            </button>
          </div>
        </div>
      ) : null}

      <form className="form-card" onSubmit={handleCreate}>
        <label>
          Nickname
          <input
            value={nickname}
            maxLength={24}
            placeholder={nicknamePlaceholder}
            onChange={(event) => onNicknameChange(event.target.value)}
          />
        </label>

        <label className="toggle-field">
          <span>Controle de tempo</span>
          <span className="toggle-control">
            <input
              type="checkbox"
              checked={playWithoutClock}
              onChange={(event) => onPlayWithoutClockChange(event.target.checked)}
            />
            <span className="toggle-copy">
              <strong>Jogar sem relógio</strong>
              <small>
                Quando desmarcado, a partida usa 10 minutos por jogador e o relógio
                só começa após o primeiro lance das brancas.
              </small>
            </span>
          </span>
        </label>

        <div className="form-actions">
          <button className="button" disabled={busy}>
            Criar sala
          </button>
          <button
            type="button"
            className="button button-secondary"
            disabled={busy}
            onClick={() => void onCreateBotRoom()}
          >
            {trainingButtonLabel}
          </button>
        </div>
      </form>

      <form className="form-card" onSubmit={handleJoin}>
        <label>
          Código da sala
          <input
            value={roomCode}
            maxLength={6}
            placeholder="AB12CD"
            onChange={(event) => onRoomCodeChange(event.target.value.toUpperCase())}
          />
        </label>

        <div className="form-actions">
          <button className="button button-secondary" disabled={busy}>
            Entrar na sala
          </button>
        </div>
      </form>
    </section>
  );
};
