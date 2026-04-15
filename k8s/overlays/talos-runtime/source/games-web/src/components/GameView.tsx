import { CSSProperties, FormEvent, KeyboardEvent, useEffect, useRef, useState } from "react";

import { Chess, Square } from "chess.js";

import { Chessboard } from "react-chessboard";

import { ChatMessage, ClientSession, Color, RoomSnapshot, SnapshotEnvelope } from "../types";

interface GameViewProps {
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
  onResign: () => Promise<void>;
  onOfferDraw: () => Promise<void>;
  onAcceptDraw: () => Promise<void>;
  onDeclineDraw: () => Promise<void>;
  onLeaveRoom: () => Promise<void>;
  onChatDraftChange: (value: string) => void;
  onSendChat: () => Promise<void>;
}

const FILES = ["a", "b", "c", "d", "e", "f", "g", "h"] as const;

const VALID_SQUARE_STYLE: CSSProperties = {
  backgroundColor: "rgba(70, 179, 92, 0.42)",
  boxShadow: "inset 0 0 0 4px rgba(111, 240, 135, 0.9)"
};

const INVALID_SQUARE_STYLE: CSSProperties = {
  backgroundColor: "rgba(218, 54, 51, 0.44)",
  boxShadow: "inset 0 0 0 4px rgba(255, 131, 125, 0.95)"
};

const PROJECTED_VALID_SQUARE_STYLE: CSSProperties = {
  backgroundColor: "rgba(70, 179, 92, 0.56)",
  boxShadow:
    "inset 0 0 0 4px rgba(111, 240, 135, 1), 0 0 24px rgba(111, 240, 135, 0.52)"
};

const PROJECTED_INVALID_SQUARE_STYLE: CSSProperties = {
  backgroundColor: "rgba(218, 54, 51, 0.56)",
  boxShadow:
    "inset 0 0 0 4px rgba(255, 131, 125, 1), 0 0 24px rgba(255, 131, 125, 0.52)"
};

const CHECK_SQUARE_STYLE: CSSProperties = {
  backgroundColor: "rgba(255, 214, 10, 0.48)",
  boxShadow: "inset 0 0 0 4px rgba(255, 238, 148, 0.95)"
};

const MIN_BOARD_SIZE = 120;
const BOARD_VIEWPORT_GUTTER = 20;

const formatClock = (ms: number): string => {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
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
        return "Partida ativa, relogio aguardando o primeiro lance";
      }
      return snapshot.turn === "w" ? "Vez das brancas" : "Vez das pretas";
    case "checkmate":
      return snapshot.winner === "w" ? "Xeque-mate: brancas vencem" : "Xeque-mate: pretas vencem";
    case "stalemate":
      return "Empate por afogamento";
    case "draw":
      return "Partida empatada";
    case "resigned":
      return snapshot.winner === "w" ? "Pretas desistiram" : "Brancas desistiram";
    case "timeout":
      return snapshot.winner === "w" ? "Vitoria das brancas no relogio" : "Vitoria das pretas no relogio";
    default:
      return "Estado desconhecido";
  }
};

const describeClockMode = (snapshot: RoomSnapshot): string => {
  if (!snapshot.clockEnabled) {
    return "Sem relogio. A partida nao expira por tempo.";
  }

  const baseClock = formatClock(snapshot.clocks.initialMs);
  if (snapshot.status === "active" && snapshot.moveHistory.length === 0 && snapshot.clocks.turnStartedAt === null) {
    return `Relogio de ${baseClock} por jogador. Ele comeca no primeiro lance das brancas.`;
  }

  return `Relogio de ${baseClock} por jogador.`;
};

const describeSeatState = (seat: RoomSnapshot["white"]): string => {
  if (seat.isBot) {
    return seat.botLevel === "easy" ? "Maquina easy" : "Maquina";
  }

  return seat.connected ? "Conectado" : "Desconectado";
};

const getLiveClock = (
  snapshot: RoomSnapshot,
  receivedAt: number,
  targetColor: Color
): number => {
  const baseValue = targetColor === "w" ? snapshot.clocks.whiteMs : snapshot.clocks.blackMs;

  if (snapshot.clocks.activeColor !== targetColor || snapshot.clocks.turnStartedAt === null) {
    return baseValue;
  }

  const estimatedServerNow = snapshot.serverNow + (Date.now() - receivedAt);
  const elapsed = Math.max(0, estimatedServerNow - snapshot.clocks.turnStartedAt);
  return Math.max(0, baseValue - elapsed);
};

const getCheckSquare = (fen: string): Square | null => {
  const chess = new Chess(fen);

  if (!chess.isCheck()) {
    return null;
  }

  const checkedColor = chess.turn();
  const board = chess.board();

  for (let rankIndex = 0; rankIndex < board.length; rankIndex += 1) {
    for (let fileIndex = 0; fileIndex < board[rankIndex].length; fileIndex += 1) {
      const piece = board[rankIndex][fileIndex];

      if (piece?.type === "k" && piece.color === checkedColor) {
        return `${FILES[fileIndex]}${8 - rankIndex}` as Square;
      }
    }
  }

  return null;
};

const getValidTargets = (fen: string, sourceSquare: Square | null): Square[] => {
  if (!sourceSquare) {
    return [];
  }

  const chess = new Chess(fen);
  return chess
    .moves({ square: sourceSquare, verbose: true })
    .map((move) => move.to as Square);
};

const formatChatTime = (timestamp: number): string =>
  new Intl.DateTimeFormat("pt-BR", {
    hour: "2-digit",
    minute: "2-digit"
  }).format(timestamp);

const describeChatSender = (message: ChatMessage, currentSession: ClientSession): string => {
  if (message.senderType === "bot") {
    return "Maquina easy";
  }
  if (message.senderType === "system") {
    return "Sistema";
  }
  if (message.senderName === currentSession.nickname) {
    return "Voce";
  }
  return message.senderName;
};

export const GameView = ({
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
  onResign,
  onOfferDraw,
  onAcceptDraw,
  onDeclineDraw,
  onLeaveRoom,
  onChatDraftChange,
  onSendChat
}: GameViewProps) => {
  const { snapshot, receivedAt } = state;
  const [, setTick] = useState(0);
  const [previewFen, setPreviewFen] = useState<string | null>(null);
  const [dragSource, setDragSource] = useState<Square | null>(null);
  const [projectedSquare, setProjectedSquare] = useState<Square | null>(null);
  const [invalidSquare, setInvalidSquare] = useState<Square | null>(null);
  const [boardWidth, setBoardWidth] = useState(0);
  const invalidResetRef = useRef<number | null>(null);
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
    if (!previewFen) {
      return;
    }

    if (snapshot.fen === previewFen || !busy) {
      setPreviewFen(null);
    }
  }, [busy, previewFen, snapshot.fen]);

  useEffect(() => {
    setDragSource(null);
    setProjectedSquare(null);
    setInvalidSquare(null);
  }, [snapshot.fen]);

  useEffect(() => {
    if (!chatListRef.current) {
      return;
    }

    chatListRef.current.scrollTop = chatListRef.current.scrollHeight;
  }, [chatMessages]);

  useEffect(() => {
    return () => {
      if (invalidResetRef.current) {
        window.clearTimeout(invalidResetRef.current);
      }
    };
  }, []);

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

      setBoardWidth((currentWidth) =>
        currentWidth === nextBoardWidth ? currentWidth : nextBoardWidth
      );
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

  const whiteClock = formatClock(getLiveClock(snapshot, receivedAt, "w"));
  const blackClock = formatClock(getLiveClock(snapshot, receivedAt, "b"));
  const whiteClockLabel = snapshot.clockEnabled ? whiteClock : "Sem tempo";
  const blackClockLabel = snapshot.clockEnabled ? blackClock : "Sem tempo";
  const isPlayer = session.role === "player";
  const isTurn =
    isPlayer &&
    !busy &&
    session.color === snapshot.turn &&
    snapshot.status === "active";
  const drawOfferedByOpponent =
    snapshot.drawOffer &&
    session.color &&
    snapshot.drawOffer.offeredBy !== session.color &&
    snapshot.status === "active";
  const isBotGame = snapshot.mode === "bot_easy";
  const displayFen = previewFen ?? snapshot.fen;
  const validTargets = getValidTargets(snapshot.fen, dragSource);
  const validTargetSet = new Set(validTargets);
  const projectedValidSquare =
    dragSource && projectedSquare && validTargetSet.has(projectedSquare)
      ? projectedSquare
      : null;
  const projectedInvalidSquare =
    dragSource && projectedSquare && !validTargetSet.has(projectedSquare)
      ? projectedSquare
      : null;
  const checkSquare = getCheckSquare(snapshot.fen);
  const customSquareStyles: Record<string, CSSProperties> = {};
  const applySquareStyle = (square: Square, style: CSSProperties) => {
    const existingStyle = customSquareStyles[square] ?? {};
    const mergedBoxShadow = [existingStyle.boxShadow, style.boxShadow].filter(Boolean).join(", ");

    customSquareStyles[square] = {
      ...existingStyle,
      ...style,
      boxShadow: mergedBoxShadow || style.boxShadow
    };
  };

  validTargets.forEach((square) => {
    applySquareStyle(square, VALID_SQUARE_STYLE);
  });

  if (projectedValidSquare) {
    applySquareStyle(projectedValidSquare, PROJECTED_VALID_SQUARE_STYLE);
  }

  if (projectedInvalidSquare) {
    applySquareStyle(projectedInvalidSquare, PROJECTED_INVALID_SQUARE_STYLE);
  } else if (invalidSquare) {
    applySquareStyle(invalidSquare, INVALID_SQUARE_STYLE);
  }

  if (checkSquare) {
    applySquareStyle(checkSquare, CHECK_SQUARE_STYLE);
  }

  const flashInvalidSquare = (square: Square) => {
    if (invalidResetRef.current) {
      window.clearTimeout(invalidResetRef.current);
    }

    setInvalidSquare(square);
    invalidResetRef.current = window.setTimeout(() => {
      setInvalidSquare((current) => (current === square ? null : current));
      invalidResetRef.current = null;
    }, 700);
  };

  const handlePieceDrop = (sourceSquare: string, targetSquare?: string): boolean => {
    if (!targetSquare) {
      return false;
    }

    const from = sourceSquare as Square;
    const to = targetSquare as Square;
    const previewGame = new Chess(snapshot.fen);

    setDragSource(null);
    setProjectedSquare(null);

    if (!isTurn) {
      return false;
    }

    const legalTargets = previewGame
      .moves({ square: from, verbose: true })
      .map((move) => move.to as Square);

    if (!legalTargets.includes(to)) {
      flashInvalidSquare(to);
      return false;
    }

    const move = previewGame.move({
      from,
      to,
      promotion: "q"
    });

    if (!move) {
      flashInvalidSquare(to);
      return false;
    }

    setInvalidSquare(null);
    setPreviewFen(previewGame.fen());
    void onMove(from, to);
    return true;
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
            <Chessboard
              id="geek-hub-chess-board"
              position={displayFen}
              boardWidth={boardWidth || undefined}
              arePiecesDraggable={Boolean(isTurn)}
              allowDragOutsideBoard={false}
              boardOrientation={session.color === "b" ? "black" : "white"}
              snapToCursor={false}
              customDropSquareStyle={{ boxShadow: "none" }}
              customSquareStyles={customSquareStyles}
              onPieceDragBegin={(_piece, sourceSquare) => {
                setInvalidSquare(null);
                setProjectedSquare(null);
                setDragSource(sourceSquare as Square);
              }}
              onPieceDragEnd={() => {
                setDragSource(null);
                setProjectedSquare(null);
              }}
              onDragOverSquare={(square) => {
                if (dragSource) {
                  setProjectedSquare(square as Square);
                }
              }}
              onMouseOutSquare={() => {
                if (dragSource) {
                  setProjectedSquare(null);
                }
              }}
              onPieceDrop={handlePieceDrop}
              customDarkSquareStyle={{ backgroundColor: "#a44c22" }}
              customLightSquareStyle={{ backgroundColor: "#f0d8b6" }}
              customBoardStyle={{
                borderRadius: "24px",
                boxShadow: "0 30px 60px rgba(18, 9, 3, 0.35)"
              }}
            />
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
                ? "Modo treino contra a maquina easy. Voce joga com as brancas e o backend responde com lances simples para iniciantes."
                : "Sala de treino contra a maquina easy. Voce esta assistindo como espectador."
              : isPlayer
                ? `Voce joga com ${session.color === "w" ? "as brancas" : "as pretas"}.`
                : "Voce esta assistindo como espectador."}
          </p>
          <p className="lead">{describeClockMode(snapshot)}</p>
        </div>

        <div className="panel players-panel">
          <div className={`player-card ${snapshot.turn === "b" ? "player-card-active" : ""}`}>
            <span>Pretas</span>
            <strong>{snapshot.black.nickname ?? "Aguardando..."}</strong>
            <small>{describeSeatState(snapshot.black)}</small>
            <b>{blackClockLabel}</b>
          </div>

          <div className={`player-card ${snapshot.turn === "w" ? "player-card-active" : ""}`}>
            <span>Brancas</span>
            <strong>{snapshot.white.nickname ?? "Aguardando..."}</strong>
            <small>{describeSeatState(snapshot.white)}</small>
            <b>{whiteClockLabel}</b>
          </div>
        </div>

        <div className="panel actions-panel">
          <div className="eyebrow">Acoes</div>
          <div className="action-grid">
            <button
              className="button button-danger"
              disabled={
                !isPlayer ||
                (snapshot.status !== "active" && snapshot.status !== "waiting") ||
                busy
              }
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
          <div className="eyebrow">Lances</div>
          <div className="moves-list">
            {snapshot.moveHistory.length === 0 ? (
              <p>Nenhum lance ainda.</p>
            ) : (
              snapshot.moveHistory.map((move, index) => (
                <div key={`${move.at}-${move.lan}`} className="move-pill">
                  <span>{index + 1}.</span>
                  <strong>{move.san}</strong>
                  <small>{move.color === "w" ? "brancas" : "pretas"}</small>
                </div>
              ))
            )}
          </div>
        </div>

        <div className="panel chat-panel">
          <div className="chat-panel-header">
            <div>
              <div className="eyebrow">Chat da sala</div>
              <h2>{isBotGame ? "Orientacao com a maquina" : "Chat entre jogadores"}</h2>
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
                    <small>
                      {message.transport === "webrtc" ? "via WebRTC" : "via servidor"}
                    </small>
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
                  ? "A maquina responde pelo backend e pode comentar regras e jogadas."
                  : "O chat tenta WebRTC entre jogadores e usa relay do servidor quando necessario."}
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
