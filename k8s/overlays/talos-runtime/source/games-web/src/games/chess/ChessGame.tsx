import { useEffect, useRef, useState } from "react";
import { io, Socket } from "socket.io-client";

import { GameView } from "../../components/GameView";
import { Lobby } from "../../components/Lobby";
import { getBackendUrl } from "../../lib/config";
import { getClientTelemetry } from "../../lib/client-context";
import { clearSession, readAuthSession, readSession, writeSession } from "../../lib/storage";
import {
  AuthSession,
  ChatMessage,
  ChatTransport,
  ClientSession,
  RoomSnapshot,
  SnapshotEnvelope,
  SocketAck,
  WebRTCSignalPayload
} from "../../types";

const PORTAL_PATH = "/";
const GAME_TYPE = "chess" as const;
const socketUrl = getBackendUrl();
const clientTelemetry = getClientTelemetry();
const rtcConfiguration: RTCConfiguration = {
  iceServers: [{ urls: ["stun:stun.l.google.com:19302"] }]
};

const toFriendlyError = (message?: string): string =>
  message ?? "Nao foi possivel concluir a operacao.";

const applyAckState = (
  ack: SocketAck,
  setSnapshot: (value: SnapshotEnvelope | null) => void,
  setSession: (value: ClientSession | null) => void
): void => {
  if (ack.snapshot) {
    setSnapshot({
      snapshot: ack.snapshot,
      receivedAt: Date.now()
    });
  }

  if (ack.session) {
    writeSession(ack.session);
    setSession(ack.session);
  }
};

const deriveNickname = (auth: AuthSession | null): string => {
  if (!auth) {
    return "";
  }

  const source =
    auth.givenName.trim() ||
    auth.name.trim() ||
    auth.email?.split("@")[0]?.trim() ||
    "jogador";

  return source.slice(0, 24);
};

const createMessageId = (): string =>
  `chat-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

const mergeMessages = (current: ChatMessage[], incoming: ChatMessage[]): ChatMessage[] => {
  const indexed = new Map<string, ChatMessage>();

  current.forEach((message) => {
    indexed.set(message.id, message);
  });
  incoming.forEach((message) => {
    indexed.set(message.id, message);
  });

  return [...indexed.values()].sort(
    (left, right) => left.createdAt - right.createdAt || left.id.localeCompare(right.id)
  );
};

const normalizeChatTransport = (message: ChatMessage, transport: ChatTransport): ChatMessage => ({
  ...message,
  transport
});

const readCurrentGameSession = (): ClientSession | null => {
  const saved = readSession();
  if (!saved) {
    return null;
  }

  if (!saved.gameType || saved.gameType === GAME_TYPE) {
    return saved;
  }

  return null;
};

export function ChessGame() {
  const [authSession] = useState<AuthSession | null>(() => readAuthSession());
  const [nickname, setNickname] = useState(() => {
    const savedSession = readCurrentGameSession();
    const savedAuth = readAuthSession();
    return savedSession?.nickname ?? deriveNickname(savedAuth);
  });
  const [roomCode, setRoomCode] = useState(() => readCurrentGameSession()?.roomCode ?? "");
  const [playWithoutClock, setPlayWithoutClock] = useState(false);
  const [session, setSession] = useState<ClientSession | null>(() => readCurrentGameSession());
  const [snapshot, setSnapshot] = useState<SnapshotEnvelope | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<string | null>(null);
  const [connectionLabel, setConnectionLabel] = useState("Conectando ao backend...");
  const [isConnected, setIsConnected] = useState(false);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [chatDraft, setChatDraft] = useState("");
  const [chatBusy, setChatBusy] = useState(false);
  const [chatStatus, setChatStatus] = useState("Entre em uma sala para habilitar o chat.");

  const socketRef = useRef<Socket | null>(null);
  const sessionRef = useRef<ClientSession | null>(session);
  const snapshotRef = useRef<SnapshotEnvelope | null>(snapshot);
  const chatIdsRef = useRef<Set<string>>(new Set());
  const peerConnectionRef = useRef<RTCPeerConnection | null>(null);
  const dataChannelRef = useRef<RTCDataChannel | null>(null);
  const offerRoomCodeRef = useRef<string | null>(null);
  const activeRoomRef = useRef<string | null>(session?.roomCode ?? null);

  const appendChatMessages = (incoming: ChatMessage[]) => {
    if (incoming.length === 0) {
      return;
    }

    setChatMessages((current) => {
      const next = mergeMessages(current, incoming);
      chatIdsRef.current = new Set(next.map((message) => message.id));
      return next;
    });
  };

  const replaceChatMessages = (incoming: ChatMessage[]) => {
    const next = mergeMessages([], incoming);
    chatIdsRef.current = new Set(next.map((message) => message.id));
    setChatMessages(next);
  };

  const resetPeerConnection = (nextStatus: string) => {
    offerRoomCodeRef.current = null;

    if (dataChannelRef.current) {
      try {
        dataChannelRef.current.close();
      } catch {
        // ignore best-effort close
      }
    }
    dataChannelRef.current = null;

    if (peerConnectionRef.current) {
      try {
        peerConnectionRef.current.close();
      } catch {
        // ignore best-effort close
      }
    }
    peerConnectionRef.current = null;
    setChatStatus(nextStatus);
  };

  const resetRoomChatState = (nextStatus: string) => {
    chatIdsRef.current = new Set();
    setChatMessages([]);
    setChatDraft("");
    setChatBusy(false);
    resetPeerConnection(nextStatus);
  };

  const attachDataChannel = (channel: RTCDataChannel) => {
    dataChannelRef.current = channel;

    channel.onopen = () => {
      setChatStatus("Chat P2P conectado via WebRTC.");
    };

    channel.onerror = () => {
      setChatStatus("Falha no canal WebRTC; usando relay do servidor.");
    };

    channel.onclose = () => {
      if (snapshotRef.current?.snapshot.mode === "pvp") {
        setChatStatus("Canal WebRTC fechado; usando relay do servidor.");
      }
    };

    channel.onmessage = (event) => {
      try {
        const parsed = JSON.parse(event.data) as ChatMessage;
        appendChatMessages([normalizeChatTransport(parsed, "webrtc")]);
      } catch {
        setChatStatus("Mensagem P2P invalida recebida; usando relay do servidor.");
      }
    };
  };

  const emitSignal = (payload: WebRTCSignalPayload) => {
    const socket = socketRef.current;
    if (!socket?.connected) {
      return;
    }

    socket.emit("webrtc_signal", payload);
  };

  const ensurePeerConnection = async (): Promise<RTCPeerConnection | null> => {
    const currentSession = sessionRef.current;
    const currentSnapshot = snapshotRef.current?.snapshot;
    if (!currentSession || !currentSnapshot || currentSnapshot.mode !== "pvp") {
      return null;
    }

    if (typeof window === "undefined" || typeof window.RTCPeerConnection === "undefined") {
      setChatStatus("WebRTC nao esta disponivel neste navegador; usando relay do servidor.");
      return null;
    }

    if (peerConnectionRef.current) {
      return peerConnectionRef.current;
    }

    const peer = new RTCPeerConnection(rtcConfiguration);

    peer.onicecandidate = (event) => {
      if (!event.candidate || !sessionRef.current) {
        return;
      }

      emitSignal({
        gameType: GAME_TYPE,
        roomCode: sessionRef.current.roomCode,
        kind: "ice_candidate",
        candidate: event.candidate.toJSON()
      });
    };

    peer.onconnectionstatechange = () => {
      switch (peer.connectionState) {
        case "connected":
          setChatStatus("Chat P2P conectado via WebRTC.");
          break;
        case "connecting":
          setChatStatus("Negociando chat WebRTC...");
          break;
        case "disconnected":
        case "failed":
          setChatStatus("Chat WebRTC instavel; relay do servidor disponivel.");
          break;
        case "closed":
          setChatStatus("Canal WebRTC fechado; relay do servidor disponivel.");
          break;
        default:
          break;
      }
    };

    peer.ondatachannel = (event) => {
      attachDataChannel(event.channel);
    };

    if (currentSession.role === "player" && currentSession.color === "w") {
      attachDataChannel(peer.createDataChannel("room-chat", { ordered: true }));
    }

    peerConnectionRef.current = peer;
    return peer;
  };

  const startOffer = async () => {
    const currentSession = sessionRef.current;
    const currentSnapshot = snapshotRef.current?.snapshot;
    if (
      !currentSession ||
      !currentSnapshot ||
      currentSnapshot.mode !== "pvp" ||
      currentSession.role !== "player" ||
      currentSession.color !== "w"
    ) {
      return;
    }

    const peer = await ensurePeerConnection();
    if (!peer || peer.signalingState !== "stable") {
      return;
    }

    try {
      const offer = await peer.createOffer();
      await peer.setLocalDescription(offer);
      offerRoomCodeRef.current = currentSession.roomCode;
      setChatStatus("Negociando chat WebRTC...");
      emitSignal({
        gameType: GAME_TYPE,
        roomCode: currentSession.roomCode,
        kind: "offer",
        description: offer
      });
    } catch {
      setChatStatus("Nao foi possivel abrir o chat WebRTC; relay do servidor disponivel.");
    }
  };

  const handleIncomingSignal = async (signal: WebRTCSignalPayload) => {
    const currentSession = sessionRef.current;
    const currentSnapshot = snapshotRef.current?.snapshot;
    const socket = socketRef.current;
    if (!currentSession || !currentSnapshot || currentSnapshot.mode !== "pvp") {
      return;
    }
    if (signal.roomCode !== currentSession.roomCode) {
      return;
    }
    if (signal.senderId && socket?.id && signal.senderId === socket.id) {
      return;
    }

    const peer = await ensurePeerConnection();
    if (!peer) {
      return;
    }

    try {
      switch (signal.kind) {
        case "offer": {
          if (!signal.description) {
            return;
          }
          if (peer.signalingState !== "stable") {
            resetPeerConnection("Reiniciando negociacao WebRTC...");
            const freshPeer = await ensurePeerConnection();
            if (!freshPeer) {
              return;
            }
            await freshPeer.setRemoteDescription(signal.description);
            const answer = await freshPeer.createAnswer();
            await freshPeer.setLocalDescription(answer);
            emitSignal({
              gameType: GAME_TYPE,
              roomCode: currentSession.roomCode,
              kind: "answer",
              description: answer
            });
            return;
          }

          await peer.setRemoteDescription(signal.description);
          const answer = await peer.createAnswer();
          await peer.setLocalDescription(answer);
          emitSignal({
            gameType: GAME_TYPE,
            roomCode: currentSession.roomCode,
            kind: "answer",
            description: answer
          });
          break;
        }
        case "answer":
          if (signal.description && !peer.currentRemoteDescription) {
            await peer.setRemoteDescription(signal.description);
          }
          break;
        case "ice_candidate":
          if (signal.candidate) {
            await peer.addIceCandidate(signal.candidate);
          }
          break;
        default:
          break;
      }
    } catch {
      setChatStatus("Falha na negociacao WebRTC; relay do servidor disponivel.");
    }
  };

  useEffect(() => {
    sessionRef.current = session;
  }, [session]);

  useEffect(() => {
    snapshotRef.current = snapshot;
  }, [snapshot]);

  useEffect(() => {
    if (!session || !snapshot) {
      return;
    }

    const nextGameId = snapshot.snapshot.gameId;
    if (session.gameId === nextGameId) {
      return;
    }

    const nextSession: ClientSession = {
      ...session,
      gameId: nextGameId,
      mode: snapshot.snapshot.mode,
      gameType: snapshot.snapshot.gameType
    };
    writeSession(nextSession);
    setSession(nextSession);
  }, [session, snapshot]);

  useEffect(() => {
    if (nickname.trim() || !authSession) {
      return;
    }

    setNickname(deriveNickname(authSession));
  }, [authSession, nickname]);

  useEffect(() => {
    const socket = io(socketUrl, {
      autoConnect: false,
      transports: ["websocket"],
      path: "/socket.io",
      reconnection: true
    });

    socketRef.current = socket;

    socket.on("connect", () => {
      setIsConnected(true);
      setConnectionLabel("Conectado");
      setError(null);

      const currentSession = sessionRef.current;
      if (!currentSession) {
        return;
      }

      void new Promise<void>((resolve) => {
        socket.timeout(5000).emit(
          "sync_state",
          {
            gameType: GAME_TYPE,
            roomCode: currentSession.roomCode,
            playerToken: currentSession.playerToken,
            spectatorToken: currentSession.spectatorToken
          },
          (err: unknown, ack: SocketAck) => {
            if (err || !ack?.ok) {
              resolve();
              return;
            }

            applyAckState(ack, setSnapshot, setSession);
            resolve();
          }
        );
      });
    });

    socket.on("disconnect", () => {
      setIsConnected(false);
      setConnectionLabel("Desconectado, tentando reconectar...");
      resetPeerConnection("Reconectando chat...");
    });

    socket.on("connect_error", () => {
      setIsConnected(false);
      setConnectionLabel("Falha de conexao com o backend");
      resetPeerConnection("Falha na conexao de chat; tentando novamente...");
    });

    socket.on("state_updated", (nextSnapshot: RoomSnapshot) => {
      setSnapshot({
        snapshot: nextSnapshot,
        receivedAt: Date.now()
      });
    });

    socket.on("chat_history", (history: ChatMessage[]) => {
      replaceChatMessages(history);
    });

    socket.on("chat_server_message", (message: ChatMessage) => {
      appendChatMessages([message]);
    });

    socket.on("webrtc_signal", (signal: WebRTCSignalPayload) => {
      void handleIncomingSignal(signal);
    });

    socket.connect();

    return () => {
      socket.removeAllListeners();
      socket.disconnect();
      socketRef.current = null;
      resetPeerConnection("Chat encerrado.");
    };
  }, []);

  useEffect(() => {
    if (!session) {
      return;
    }

    const socket = socketRef.current;
    if (!socket || !socket.connected) {
      return;
    }

    const heartbeatId = window.setInterval(() => {
      socket.timeout(4000).emit(
        "heartbeat",
        {
          gameType: GAME_TYPE,
          roomCode: session.roomCode,
          playerToken: session.playerToken,
          spectatorToken: session.spectatorToken
        },
        (_err: unknown, ack: SocketAck) => {
          if (ack?.ok && ack.snapshot) {
            setSnapshot({
              snapshot: ack.snapshot,
              receivedAt: Date.now()
            });
          }
        }
      );
    }, 15000);

    return () => {
      window.clearInterval(heartbeatId);
    };
  }, [isConnected, session]);

  useEffect(() => {
    const currentRoomCode = session?.roomCode ?? null;
    if (activeRoomRef.current === currentRoomCode) {
      return;
    }

    activeRoomRef.current = currentRoomCode;
    resetRoomChatState(
      currentRoomCode
        ? "Sincronizando chat da sala..."
        : "Entre em uma sala para habilitar o chat."
    );
  }, [session?.roomCode]);

  useEffect(() => {
    const currentSession = session;
    const currentSnapshot = snapshot?.snapshot;

    if (!currentSession || !currentSnapshot) {
      if (!session) {
        resetPeerConnection("Entre em uma sala para habilitar o chat.");
      }
      return;
    }

    if (currentSnapshot.mode === "bot_easy") {
      resetPeerConnection("Chat com a maquina easy ativo nesta sala.");
      return;
    }

    if (currentSession.role !== "player") {
      resetPeerConnection("Espectadores usam relay pelo servidor.");
      return;
    }

    if (!isConnected) {
      resetPeerConnection("Reconectando chat...");
      return;
    }

    if (!currentSnapshot.white.connected || !currentSnapshot.black.connected) {
      resetPeerConnection("O chat WebRTC inicia quando os dois jogadores estiverem conectados.");
      return;
    }

    void ensurePeerConnection().then(() => {
      if (currentSession.color === "w" && offerRoomCodeRef.current !== currentSession.roomCode) {
        void startOffer();
        return;
      }

      setChatStatus((currentStatus) =>
        currentStatus.includes("conectado")
          ? currentStatus
          : "Os jogadores estao conectados. Tentando canal WebRTC..."
      );
    });
  }, [
    isConnected,
    session,
    snapshot?.snapshot.mode,
    snapshot?.snapshot.white.connected,
    snapshot?.snapshot.black.connected
  ]);

  const emitWithAck = async (event: string, payload: Record<string, unknown>): Promise<SocketAck> => {
    const socket = socketRef.current;
    if (!socket) {
      throw new Error("Socket indisponivel.");
    }

    setError(null);

    return new Promise<SocketAck>((resolve, reject) => {
      socket.timeout(5000).emit(event, payload, (err: unknown, ack: SocketAck) => {
        if (err) {
          reject(new Error("Timeout na comunicacao com o servidor."));
          return;
        }

        resolve(ack);
      });
    });
  };

  const runAction = async (
    label: string,
    action: () => Promise<SocketAck>,
    afterSuccess?: (ack: SocketAck) => void
  ) => {
    try {
      setPending(label);
      const ack = await action();
      if (!ack.ok) {
        throw new Error(toFriendlyError(ack.message));
      }

      applyAckState(ack, setSnapshot, setSession);
      afterSuccess?.(ack);
    } catch (actionError) {
      setError(actionError instanceof Error ? actionError.message : "Erro inesperado.");
    } finally {
      setPending(null);
    }
  };

  const handleCreateRoom = async () => {
    await runAction("Criando sala...", () =>
      emitWithAck("create_room", {
        nickname,
        gameType: GAME_TYPE,
        mode: "pvp",
        clockControl: playWithoutClock ? "untimed" : "timed",
        client: clientTelemetry
      })
    );
  };

  const handleCreateBotRoom = async () => {
    await runAction("Criando treino contra a maquina...", () =>
      emitWithAck("create_room", {
        nickname,
        gameType: GAME_TYPE,
        mode: "bot_easy",
        clockControl: playWithoutClock ? "untimed" : "timed",
        client: clientTelemetry
      })
    );
  };

  const handleJoinRoom = async () => {
    const saved = readCurrentGameSession();
    const shouldTryReconnect =
      saved &&
      saved.roomCode === roomCode.trim().toUpperCase() &&
      saved.nickname === nickname.trim();

    await runAction("Entrando na sala...", () =>
      emitWithAck("join_room", {
        nickname,
        gameType: GAME_TYPE,
        roomCode,
        playerToken: shouldTryReconnect ? saved.playerToken : undefined,
        spectatorToken: shouldTryReconnect ? saved.spectatorToken : undefined,
        client: clientTelemetry
      })
    );
  };

  const handleReconnectSaved = async () => {
    const saved = readCurrentGameSession();
    if (!saved) {
      setError("Nenhuma sessao salva encontrada.");
      return;
    }

    setNickname(saved.nickname);
    setRoomCode(saved.roomCode);
    setSession(saved);

    await runAction("Sincronizando sessao...", () =>
      emitWithAck("sync_state", {
        roomCode: saved.roomCode,
        gameType: GAME_TYPE,
        playerToken: saved.playerToken,
        spectatorToken: saved.spectatorToken
      })
    );
  };

  const handleDiscardSavedSession = () => {
    clearSession();
    setSession(null);
    setSnapshot(null);
    resetRoomChatState("Entre em uma sala para habilitar o chat.");
  };

  const handleMove = async (from: string, to: string) => {
    if (!session?.playerToken) {
      return;
    }

    await runAction("Enviando lance...", () =>
      emitWithAck("submit_action", {
        gameType: GAME_TYPE,
        roomCode: session.roomCode,
        playerToken: session.playerToken,
        actionType: "move",
        actionPayloadJson: JSON.stringify({
          from,
          to
        })
      })
    );
  };

  const handleRestartGame = async () => {
    if (!session?.playerToken) {
      return;
    }

    await runAction("Reiniciando partida...", () =>
      emitWithAck("restart_game", {
        gameType: GAME_TYPE,
        roomCode: session.roomCode,
        playerToken: session.playerToken
      })
    );
  };

  const handleResign = async () => {
    if (!session?.playerToken) {
      return;
    }

    await runAction("Encerrando partida...", () =>
      emitWithAck("resign", {
        gameType: GAME_TYPE,
        roomCode: session.roomCode,
        playerToken: session.playerToken
      })
    );
  };

  const handleOfferDraw = async () => {
    if (!session?.playerToken) {
      return;
    }

    await runAction("Enviando oferta de empate...", () =>
      emitWithAck("offer_draw", {
        gameType: GAME_TYPE,
        roomCode: session.roomCode,
        playerToken: session.playerToken
      })
    );
  };

  const handleAcceptDraw = async () => {
    if (!session?.playerToken) {
      return;
    }

    await runAction("Aceitando empate...", () =>
      emitWithAck("accept_draw", {
        gameType: GAME_TYPE,
        roomCode: session.roomCode,
        playerToken: session.playerToken
      })
    );
  };

  const handleDeclineDraw = async () => {
    if (!session?.playerToken) {
      return;
    }

    await runAction("Recusando empate...", () =>
      emitWithAck("decline_draw", {
        gameType: GAME_TYPE,
        roomCode: session.roomCode,
        playerToken: session.playerToken
      })
    );
  };

  const handleLeaveRoom = async () => {
    if (!session) {
      return;
    }

    await runAction(
      "Saindo da sala...",
      () =>
        emitWithAck("leave_room", {
          gameType: GAME_TYPE,
          roomCode: session.roomCode,
          playerToken: session.playerToken,
          spectatorToken: session.spectatorToken
        }),
      () => {
        clearSession();
        setSession(null);
        setSnapshot(null);
        resetRoomChatState("Entre em uma sala para habilitar o chat.");
      }
    );
  };

  const handleSendChat = async () => {
    const currentSession = sessionRef.current;
    const currentSnapshot = snapshotRef.current?.snapshot;
    if (!currentSession || !currentSnapshot) {
      return;
    }

    const text = chatDraft.trim();
    if (!text) {
      return;
    }

    const messageId = createMessageId();
    const senderName = currentSession.nickname || "Jogador";
    const localMessage: ChatMessage = {
      id: messageId,
      roomCode: currentSession.roomCode,
      senderName,
      senderType: currentSession.role === "spectator" ? "spectator" : "player",
      senderColor: currentSession.color ?? null,
      text,
      transport: "server",
      createdAt: Date.now()
    };

    setChatBusy(true);
    setChatDraft("");

    try {
      const canUsePeerChannel =
        currentSnapshot.mode === "pvp" &&
        currentSession.role === "player" &&
        dataChannelRef.current?.readyState === "open";

      if (canUsePeerChannel) {
        const peerMessage = normalizeChatTransport(localMessage, "webrtc");
        appendChatMessages([peerMessage]);
        dataChannelRef.current?.send(JSON.stringify(peerMessage));

        const ack = await emitWithAck("chat_message", {
          gameType: GAME_TYPE,
          roomCode: currentSession.roomCode,
          playerToken: currentSession.playerToken,
          spectatorToken: currentSession.spectatorToken,
          messageId,
          text,
          mirrorOnly: true
        });
        if (!ack.ok) {
          throw new Error(toFriendlyError(ack.message));
        }
        setChatStatus("Chat P2P conectado via WebRTC.");
        return;
      }

      const ack = await emitWithAck("chat_message", {
        gameType: GAME_TYPE,
        roomCode: currentSession.roomCode,
        playerToken: currentSession.playerToken,
        spectatorToken: currentSession.spectatorToken,
        messageId,
        text,
        relay: true
      });
      if (!ack.ok) {
        throw new Error(toFriendlyError(ack.message));
      }

      setChatStatus(
        currentSnapshot.mode === "bot_easy"
          ? "Chat com a maquina easy ativo nesta sala."
          : "Mensagem enviada pelo relay do servidor."
      );
    } catch (chatError) {
      setChatDraft(text);
      setError(chatError instanceof Error ? chatError.message : "Nao foi possivel enviar a mensagem.");
    } finally {
      setChatBusy(false);
    }
  };

  const handleBackToMenu = () => {
    window.location.assign(PORTAL_PATH);
  };

  const savedSession = session ?? readCurrentGameSession();
  const showSessionSync = Boolean(session && !snapshot);
  const canSendChat = Boolean(session && snapshot && !chatBusy);
  const chatPlaceholder =
    snapshot?.snapshot.mode === "bot_easy"
      ? "Pergunte sobre o ultimo lance, regras ou ideia da jogada..."
      : "Escreva uma mensagem para a sala...";

  return (
    <main className="app-shell">
      <div className="ambient ambient-one" />
      <div className="ambient ambient-two" />

      {error ? <div className="toast toast-error">{error}</div> : null}
      {pending ? <div className="toast toast-info">{pending}</div> : null}

      {session && snapshot ? (
        <GameView
          session={session}
          state={snapshot}
          busy={Boolean(pending)}
          chatMessages={chatMessages}
          chatDraft={chatDraft}
          chatStatus={chatStatus}
          chatBusy={chatBusy}
          canSendChat={canSendChat}
          chatPlaceholder={chatPlaceholder}
          onMove={handleMove}
          onRestartGame={handleRestartGame}
          onResign={handleResign}
          onOfferDraw={handleOfferDraw}
          onAcceptDraw={handleAcceptDraw}
          onDeclineDraw={handleDeclineDraw}
          onLeaveRoom={handleLeaveRoom}
          onChatDraftChange={setChatDraft}
          onSendChat={handleSendChat}
        />
      ) : showSessionSync ? (
        <section className="panel lobby-panel status-panel">
          <div className="eyebrow">Xadrez</div>
          <h1>Retomando a sua sala</h1>
          <p className="lead">
            Estamos sincronizando a partida {session?.roomCode}. Se a reconexao falhar, voce
            pode limpar a sessao salva e voltar ao portal.
          </p>
          <div className="connection-badge">{connectionLabel}</div>
          <div className="form-actions">
            <button className="button button-secondary" onClick={() => void handleReconnectSaved()}>
              Tentar novamente
            </button>
            <button
              className="button button-ghost"
              onClick={() => {
                handleDiscardSavedSession();
                handleBackToMenu();
              }}
            >
              Limpar sessao e voltar
            </button>
          </div>
        </section>
      ) : (
        <Lobby
          gameLabel="Xadrez Online"
          gameTitle="Sala, treino ou reconexão"
          gameDescription="Escolha como entrar no xadrez. O frontend só renderiza; a regra continua autoritativa no backend."
          nicknamePlaceholder="Ex.: torre42"
          trainingButtonLabel="Treinar vs máquina"
          whiteSeatLabel="brancas"
          blackSeatLabel="pretas"
          authSession={authSession}
          nickname={nickname}
          roomCode={roomCode}
          playWithoutClock={playWithoutClock}
          savedSession={savedSession}
          busy={Boolean(pending)}
          connectionLabel={connectionLabel}
          onNicknameChange={setNickname}
          onRoomCodeChange={setRoomCode}
          onPlayWithoutClockChange={setPlayWithoutClock}
          onCreateRoom={handleCreateRoom}
          onCreateBotRoom={handleCreateBotRoom}
          onJoinRoom={handleJoinRoom}
          onReconnectSaved={handleReconnectSaved}
          onDiscardSavedSession={handleDiscardSavedSession}
          onBackToMenu={handleBackToMenu}
        />
      )}
    </main>
  );
}
