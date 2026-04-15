export type Color = "w" | "b";
export type GameType = "chess";
export type ViewerRole = "player" | "spectator";
export type GameMode = "pvp" | "bot_easy";
export type BotLevel = "easy";
export type AuthProvider = "google" | "guest";
export type ChatSenderType = "player" | "spectator" | "bot" | "system";
export type ChatTransport = "webrtc" | "server";
export type GameStatus =
  | "waiting"
  | "active"
  | "checkmate"
  | "stalemate"
  | "draw"
  | "resigned"
  | "timeout";

export interface AuthSession {
  provider: AuthProvider;
  sub: string;
  name: string;
  givenName: string;
  email: string | null;
  picture: string | null;
}

export interface SeatSnapshot {
  nickname: string | null;
  connected: boolean;
  isBot: boolean;
  botLevel: BotLevel | null;
}

export interface GameClockState {
  initialMs: number;
  whiteMs: number;
  blackMs: number;
  activeColor: Color | null;
  turnStartedAt: number | null;
}

export interface MoveRecord {
  san: string;
  lan: string;
  from: string;
  to: string;
  color: Color;
  fen: string;
  at: number;
  promotion?: string;
}

export interface DrawOffer {
  offeredBy: Color;
  createdAt: number;
}

export interface RoomSnapshot {
  gameId: string;
  gameType: GameType;
  roomCode: string;
  mode: GameMode;
  clockEnabled: boolean;
  fen: string;
  pgn: string;
  turn: Color;
  status: GameStatus;
  winner: Color | null;
  endReason: string | null;
  white: SeatSnapshot;
  black: SeatSnapshot;
  viewerCount: number;
  clocks: GameClockState;
  moveHistory: MoveRecord[];
  lastMove: MoveRecord | null;
  drawOffer: DrawOffer | null;
  serverNow: number;
}

export interface ClientSession {
  roomCode: string;
  gameType: GameType;
  role: ViewerRole;
  color: Color | null;
  nickname: string;
  gameId: string;
  mode?: GameMode;
  playerToken?: string;
  spectatorToken?: string;
}

export interface ChatMessage {
  id: string;
  roomCode: string;
  senderName: string;
  senderType: ChatSenderType;
  senderColor?: Color | null;
  text: string;
  transport: ChatTransport;
  createdAt: number;
}

export interface WebRTCSignalPayload {
  gameType?: GameType;
  roomCode: string;
  kind: "offer" | "answer" | "ice_candidate";
  description?: RTCSessionDescriptionInit;
  candidate?: RTCIceCandidateInit;
  senderId?: string;
  senderColor?: Color | null;
}

export interface SocketAck {
  ok: boolean;
  code?: string;
  message?: string;
  snapshot?: RoomSnapshot | null;
  session?: ClientSession | null;
  serverNow?: number;
  left?: boolean;
}

export interface SnapshotEnvelope {
  snapshot: RoomSnapshot;
  receivedAt: number;
}
