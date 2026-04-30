import { FormEvent, KeyboardEvent, useEffect, useRef, useState } from "react";

import { ChatMessage, ClientSession, Color, RoomSnapshot, SnapshotEnvelope } from "../types";

interface TicTacToeViewProps {
  session: ClientSession;
  state: SnapshotEnvelope;
  busy: boolean;
  chatMessages: ChatMessage[];
  chatDraft: string;
  chatStatus: string;
  chatBusy: boolean;
  canSendChat: boolean;
  chatPlaceholder: string;
  onMove: (from: string, to: string) => Promise<void>;
  onRestartGame: () => Promise<void>;
  onResign: () => Promise<void>;
  onOfferDraw: () => Promise<void>;
  onAcceptDraw: () => Promise<void>;
  onDeclineDraw: () => Promise<void>;
  onLeaveRoom: () => Promise<void>;
  onChatDraftChange: (value: string) => void;
  onSendChat: () => Promise<void>;
}

const CELL_ROWS = [
  ["a1", "b1", "c1"],
  ["a2", "b2", "c2"],
  ["a3", "b3", "c3"]
] as const;

const CELL_ORDER = CELL_ROWS.flat() as string[];

const WINNING_LINES = [
  [0, 1, 2],
  [3, 4, 5],
  [6, 7, 8],
  [0, 3, 6],
  [1, 4, 7],
  [2, 5, 8],
  [0, 4, 8],
  [2, 4, 6]
] as const;

const MIN_BOARD_SIZE = 240;
const BOARD_VIEWPORT_GUTTER = 20;

const formatClock = (ms: number): string => {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
};

const markLabel = (color: Color | null | undefined): string => {
  if (color === "w") {
    return "X";
  }
  if (color === "b") {
    return "O";
  }
  return "-";
};

const describeStatus = (snapshot: RoomSnapshot): string => {
  switch (snapshot.status) {
    case "waiting":
      return "Aguardando oponente";
    case "active":
      if (
        snapshot.clockEnabled &&
        snapshot.moveHistory.length === 0 &&
        snapshot.clocks.turnStartedAt === null
      ) {
        return "Partida ativa, relógio aguardando a primeira jogada";
      }
      return snapshot.turn === "w" ? "Vez do X" : "Vez do O";
    case "won":
      return snapshot.winner === "w" ? "X venceu" : "O venceu";
    case "draw":
      return "Partida empatada";
    case "resigned":
      return snapshot.winner === "w" ? "O desistiu" : "X desistiu";
    case "timeout":
      return snapshot.winner === "w" ? "Vitória do X no relógio" : "Vitória do O no relógio";
    default:
      return "Estado desconhecido";
  }
};

const describeClockMode = (snapshot: RoomSnapshot): string => {
  if (!snapshot.clockEnabled) {
    return "Sem relógio. A partida não expira por tempo.";
  }

  const baseClock = formatClock(snapshot.clocks.initialMs);
  if (snapshot.status === "active" && snapshot.moveHistory.length === 0 && snapshot.clocks.turnStartedAt === null) {
    return `Relógio de ${baseClock} por jogador. Ele começa na primeira jogada do X.`;
  }

  return `Relógio de ${baseClock} por jogador.`;
};

const describeSeatState = (seat: RoomSnapshot["white"]): string => {
  if (seat.isBot) {
    return seat.botLevel === "easy" ? "Máquina easy" : "Máquina";
  }

  return seat.connected ? "Conectado" : "Desconectado";
};

const getLiveClock = (snapshot: RoomSnapshot, receivedAt: number, targetColor: Color): number => {
  const baseValue = targetColor === "w" ? snapshot.clocks.whiteMs : snapshot.clocks.blackMs;

  if (snapshot.clocks.activeColor !== targetColor || snapshot.clocks.turnStartedAt === null) {
    return baseValue;
  }

  const estimatedServerNow = snapshot.serverNow + (Date.now() - receivedAt);
  const elapsed = Math.max(0, estimatedServerNow - snapshot.clocks.turnStartedAt);
  return Math.max(0, baseValue - elapsed);
};

const normalizedBoard = (value: string): string[] => {
  const trimmed = value.trim().toLowerCase();
  if (trimmed.length !== 9) {
    return Array.from("---------");
  }
  return Array.from(trimmed).map((cell) => (cell === "x" || cell === "o" ? cell : "-"));
};

const findWinningIndexes = (board: string[]): Set<number> => {
  for (const line of WINNING_LINES) {
    const [first, second, third] = line;
    const mark = board[first];
    if (mark !== "-" && mark === board[second] && mark === board[third]) {
      return new Set(line);
    }
  }

  return new Set<number>();
};

const formatChatTime = (timestamp: number): string =>
  new Intl.DateTimeFormat("pt-BR", {
    hour: "2-digit",
    minute: "2-digit"
  }).format(timestamp);

const describeChatSender = (message: ChatMessage, currentSession: ClientSession): string => {
  if (message.senderType === "bot") {
    return "Máquina easy";
  }
  if (message.senderType === "system") {
    return "Sistema";
  }
  if (message.senderName === currentSession.nickname) {
    return "Você";
  }
  return message.senderName;
};

export const TicTacToeView = ({
  session,
  state,
  busy,
  chatMessages,
  chatDraft,
  chatStatus,
  chatBusy,
  canSendChat,
  chatPlaceholder,
  onMove,
  onRestartGame,
  onResign,
  onOfferDraw,
  onAcceptDraw,
  onDeclineDraw,
  onLeaveRoom,
  onChatDraftChange,
  onSendChat
}: TicTacToeViewProps) => {
  const { snapshot, receivedAt } = state;
  const [, setTick] = useState(0);
  const [boardWidth, setBoardWidth] = useState(0);
  const boardStageRef = useRef<HTMLDivElement | null>(null);
  const chatListRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      setTick((value) => value + 1);
    }, 500);

    return () => {
      window.clearInterval(intervalId);
    };
  }, []);

  useEffect(() => {
    if (!chatListRef.current) {
      return;
    }

    chatListRef.current.scrollTop = chatListRef.current.scrollHeight;
  }, [chatMessages]);

  useEffect(() => {
    const measureBoard = () => {
      const boardStage = boardStageRef.current;
      if (!boardStage) {
        return;
      }

      const rect = boardStage.getBoundingClientRect();
      const viewportHeight = window.visualViewport?.height ?? window.innerHeight;
      const availableHeight = Math.max(0, viewportHeight - rect.top - BOARD_VIEWPORT_GUTTER);
      const availableWidth = boardStage.clientWidth;
      const nextBoardWidth = Math.floor(
        Math.max(MIN_BOARD_SIZE, Math.min(availableWidth, availableHeight))
      );

      setBoardWidth((currentWidth) => (currentWidth === nextBoardWidth ? currentWidth : nextBoardWidth));
    };

    const scheduleMeasurement = () => {
      window.requestAnimationFrame(measureBoard);
    };

    const boardStage = boardStageRef.current;
    const resizeObserver =
      typeof ResizeObserver !== "undefined" && boardStage
        ? new ResizeObserver(scheduleMeasurement)
        : null;
    const visualViewport = window.visualViewport ?? null;

    resizeObserver?.observe(boardStage as Element);
    window.addEventListener("resize", scheduleMeasurement);
    window.addEventListener("orientationchange", scheduleMeasurement);
    visualViewport?.addEventListener("resize", scheduleMeasurement);
    scheduleMeasurement();

    return () => {
      resizeObserver?.disconnect();
      window.removeEventListener("resize", scheduleMeasurement);
      window.removeEventListener("orientationchange", scheduleMeasurement);
      visualViewport?.removeEventListener("resize", scheduleMeasurement);
    };
  }, []);

  const board = normalizedBoard(snapshot.fen);
  const winningIndexes = findWinningIndexes(board);
  const whiteClock = formatClock(getLiveClock(snapshot, receivedAt, "w"));
  const blackClock = formatClock(getLiveClock(snapshot, receivedAt, "b"));
  const whiteClockLabel = snapshot.clockEnabled ? whiteClock : "Sem tempo";
  const blackClockLabel = snapshot.clockEnabled ? blackClock : "Sem tempo";
  const isPlayer = session.role === "player";
  const isTurn = isPlayer && !busy && session.color === snapshot.turn && snapshot.status === "active";
  const canRestartGame =
    isPlayer && ["won", "draw", "resigned", "timeout"].includes(snapshot.status);
  const drawOfferedByOpponent =
    snapshot.drawOffer &&
    session.color &&
    snapshot.drawOffer.offeredBy !== session.color &&
    snapshot.status === "active";
  const isBotGame = snapshot.mode === "bot_easy";

  const handleCellClick = (cell: string) => {
    if (!isTurn) {
      return;
    }

    const position = board[CELL_ORDER.indexOf(cell)];
    if (position !== "-") {
      return;
    }

    void onMove(cell, cell);
  };

  const handleChatSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void onSendChat();
  };

  const handleChatKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      void onSendChat();
    }
  };

  return (
    <section className="game-layout">
      <div className="panel board-panel">
        <div className="board-header">
          <div>
            <div className="eyebrow">Sala {snapshot.roomCode}</div>
            <h2>Game ID {snapshot.gameId.slice(0, 8)}</h2>
          </div>
          <button className="button button-ghost" disabled={busy} onClick={() => void onLeaveRoom()}>
            Sair da sala
          </button>
        </div>

        <div className="board-stage" ref={boardStageRef}>
          <div className="board-frame" style={boardWidth > 0 ? { width: `${boardWidth}px` } : undefined}>
            <div className="ttt-grid">
              {CELL_ORDER.map((cell, index) => {
                const value = board[index];
                const symbol = value === "x" ? "X" : value === "o" ? "O" : "";
                const isWinningCell = winningIndexes.has(index);

                return (
                  <button
                    key={cell}
                    className={`ttt-cell ${
                      value === "x" ? "ttt-cell-x" : value === "o" ? "ttt-cell-o" : ""
                    } ${isWinningCell ? "ttt-cell-winning" : ""}`}
                    disabled={!isTurn || value !== "-" || busy}
                    onClick={() => handleCellClick(cell)}
                    type="button"
                  >
                    <span>{symbol}</span>
                    <small>{cell.toUpperCase()}</small>
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      <div className="side-column">
        <div className="panel">
          <div className="eyebrow">Status</div>
          <h2>{describeStatus(snapshot)}</h2>
          <p className="lead">
            {isBotGame
              ? isPlayer
                ? "Modo treino contra a máquina easy. Você joga com X e o backend responde com jogadas simples para iniciantes."
                : "Sala de treino contra a máquina easy. Você está assistindo como espectador."
              : isPlayer
                ? `Você joga com ${markLabel(session.color)}.`
                : "Você está assistindo como espectador."}
          </p>
          <p className="lead">{describeClockMode(snapshot)}</p>
        </div>

        <div className="panel players-panel">
          <div className={`player-card ${snapshot.turn === "b" ? "player-card-active" : ""}`}>
            <span>O</span>
            <strong>{snapshot.black.nickname ?? "Aguardando..."}</strong>
            <small>{describeSeatState(snapshot.black)}</small>
            <b>{blackClockLabel}</b>
          </div>

          <div className={`player-card ${snapshot.turn === "w" ? "player-card-active" : ""}`}>
            <span>X</span>
            <strong>{snapshot.white.nickname ?? "Aguardando..."}</strong>
            <small>{describeSeatState(snapshot.white)}</small>
            <b>{whiteClockLabel}</b>
          </div>
        </div>

        <div className="panel actions-panel">
          <div className="eyebrow">Ações</div>
          <div className="action-grid">
            <button className="button" disabled={!canRestartGame || busy} onClick={() => void onRestartGame()}>
              Nova partida
            </button>
            <button
              className="button button-danger"
              disabled={!isPlayer || (snapshot.status !== "active" && snapshot.status !== "waiting") || busy}
              onClick={() => void onResign()}
            >
              Desistir
            </button>
            <button
              className="button button-secondary"
              disabled={!isPlayer || snapshot.status !== "active" || busy || isBotGame}
              onClick={() => void onOfferDraw()}
            >
              Oferecer empate
            </button>
            {drawOfferedByOpponent ? (
              <>
                <button className="button" disabled={busy} onClick={() => void onAcceptDraw()}>
                  Aceitar empate
                </button>
                <button className="button button-ghost" disabled={busy} onClick={() => void onDeclineDraw()}>
                  Recusar empate
                </button>
              </>
            ) : null}
          </div>
        </div>

        <div className="panel moves-panel">
          <div className="eyebrow">Jogadas</div>
          <div className="moves-list">
            {snapshot.moveHistory.length === 0 ? (
              <p>Nenhuma jogada ainda.</p>
            ) : (
              snapshot.moveHistory.map((move, index) => (
                <div key={`${move.at}-${move.lan}`} className="move-pill">
                  <span>{index + 1}.</span>
                  <strong>{move.san}</strong>
                  <small>{move.color === "w" ? "X" : "O"}</small>
                </div>
              ))
            )}
          </div>
        </div>

        <div className="panel chat-panel">
          <div className="chat-panel-header">
            <div>
              <div className="eyebrow">Chat da sala</div>
              <h2>{isBotGame ? "Orientação com a máquina" : "Chat entre jogadores"}</h2>
            </div>
            <span className="chat-status-pill">{chatStatus}</span>
          </div>

          <div className="chat-messages" ref={chatListRef}>
            {chatMessages.length === 0 ? (
              <p className="chat-empty-state">Nenhuma mensagem ainda.</p>
            ) : (
              chatMessages.map((message) => {
                const isOwnMessage =
                  message.senderType !== "bot" &&
                  message.senderType !== "system" &&
                  message.senderName === session.nickname;

                return (
                  <article
                    key={message.id}
                    className={`chat-message ${
                      isOwnMessage ? "chat-message-self" : ""
                    } chat-message-${message.senderType}`}
                  >
                    <div className="chat-message-meta">
                      <strong>{describeChatSender(message, session)}</strong>
                      <span>{formatChatTime(message.createdAt)}</span>
                    </div>
                    <p>{message.text}</p>
                    <small>{message.transport === "webrtc" ? "via WebRTC" : "via servidor"}</small>
                  </article>
                );
              })
            )}
          </div>

          <form className="chat-form" onSubmit={handleChatSubmit}>
            <textarea
              value={chatDraft}
              onChange={(event) => onChatDraftChange(event.target.value)}
              onKeyDown={handleChatKeyDown}
              placeholder={chatPlaceholder}
              disabled={!canSendChat || chatBusy}
              rows={3}
            />
            <div className="chat-form-footer">
              <small>
                {isBotGame
                  ? "A máquina responde pelo backend e pode comentar regras, bloqueios e jogadas."
                  : "O chat tenta WebRTC entre jogadores e usa relay do servidor quando necessário."}
              </small>
              <button className="button" type="submit" disabled={!canSendChat || chatBusy || !chatDraft.trim()}>
                {chatBusy ? "Enviando..." : "Enviar"}
              </button>
            </div>
          </form>
        </div>
      </div>
    </section>
  );
};
