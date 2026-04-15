package gateway

type Color string

type ViewerRole string

type GameMode string

type BotLevel string

type GameStatus string
type GameType string

const (
	ColorWhite Color = "w"
	ColorBlack Color = "b"
)

type PlayerSeat struct {
	Nickname    string    `json:"nickname"`
	PlayerToken string    `json:"playerToken"`
	Connected   bool      `json:"connected"`
	JoinedAt    int64     `json:"joinedAt"`
	LastSeenAt  int64     `json:"lastSeenAt"`
	IsBot       bool      `json:"isBot,omitempty"`
	BotLevel    *BotLevel `json:"botLevel,omitempty"`
}

type SpectatorSession struct {
	Nickname       string `json:"nickname"`
	SpectatorToken string `json:"spectatorToken"`
	Connected      bool   `json:"connected"`
	JoinedAt       int64  `json:"joinedAt"`
	LastSeenAt     int64  `json:"lastSeenAt"`
}

type GameClockState struct {
	InitialMs     int64  `json:"initialMs"`
	WhiteMs       int64  `json:"whiteMs"`
	BlackMs       int64  `json:"blackMs"`
	ActiveColor   *Color `json:"activeColor"`
	TurnStartedAt *int64 `json:"turnStartedAt"`
}

type MoveRecord struct {
	SAN       string `json:"san"`
	LAN       string `json:"lan"`
	From      string `json:"from"`
	To        string `json:"to"`
	Color     Color  `json:"color"`
	FEN       string `json:"fen"`
	At        int64  `json:"at"`
	Promotion string `json:"promotion,omitempty"`
}

type DrawOffer struct {
	OfferedBy Color `json:"offeredBy"`
	CreatedAt int64 `json:"createdAt"`
}

type ChatSenderType string

type ChatTransport string

const (
	ChatSenderPlayer    ChatSenderType = "player"
	ChatSenderSpectator ChatSenderType = "spectator"
	ChatSenderBot       ChatSenderType = "bot"
	ChatSenderSystem    ChatSenderType = "system"

	ChatTransportWebRTC ChatTransport = "webrtc"
	ChatTransportServer ChatTransport = "server"
)

type ChatMessage struct {
	ID         string         `json:"id"`
	RoomCode   string         `json:"roomCode"`
	SenderName string         `json:"senderName"`
	SenderType ChatSenderType `json:"senderType"`
	SenderColor *Color        `json:"senderColor,omitempty"`
	Text       string         `json:"text"`
	Transport  ChatTransport  `json:"transport"`
	CreatedAt  int64          `json:"createdAt"`
}

type SeatSnapshot struct {
	Nickname  *string   `json:"nickname"`
	Connected bool      `json:"connected"`
	IsBot     bool      `json:"isBot"`
	BotLevel  *BotLevel `json:"botLevel"`
}

type RoomSnapshot struct {
	GameID       string         `json:"gameId"`
	GameType     GameType       `json:"gameType"`
	RoomCode     string         `json:"roomCode"`
	Mode         GameMode       `json:"mode"`
	ClockEnabled bool           `json:"clockEnabled"`
	FEN          string         `json:"fen"`
	PGN          string         `json:"pgn"`
	Turn         Color          `json:"turn"`
	Status       GameStatus     `json:"status"`
	Winner       *Color         `json:"winner"`
	EndReason    *string        `json:"endReason"`
	White        SeatSnapshot   `json:"white"`
	Black        SeatSnapshot   `json:"black"`
	ViewerCount  int            `json:"viewerCount"`
	Clocks       GameClockState `json:"clocks"`
	MoveHistory  []MoveRecord   `json:"moveHistory"`
	LastMove     *MoveRecord    `json:"lastMove"`
	DrawOffer    *DrawOffer     `json:"drawOffer"`
	ServerNow    int64          `json:"serverNow"`
}

type SessionDescriptor struct {
	RoomCode       string     `json:"roomCode"`
	GameType       GameType   `json:"gameType"`
	Role           ViewerRole `json:"role"`
	Color          *Color     `json:"color"`
	Nickname       string     `json:"nickname"`
	GameID         string     `json:"gameId"`
	Mode           GameMode   `json:"mode"`
	PlayerToken    string     `json:"playerToken,omitempty"`
	SpectatorToken string     `json:"spectatorToken,omitempty"`
}

type ClientTelemetry struct {
	DeviceType string `json:"deviceType,omitempty"`
	Platform   string `json:"platform,omitempty"`
	Browser    string `json:"browser,omitempty"`
	Region     string `json:"region,omitempty"`
}

type CreateRoomPayload struct {
	GameType     GameType         `json:"gameType"`
	Nickname     string           `json:"nickname"`
	Mode         GameMode         `json:"mode,omitempty"`
	ClockControl string           `json:"clockControl,omitempty"`
	Client       *ClientTelemetry `json:"client,omitempty"`
}

type JoinRoomPayload struct {
	GameType       GameType         `json:"gameType"`
	RoomCode       string           `json:"roomCode"`
	Nickname       string           `json:"nickname"`
	PlayerToken    string           `json:"playerToken,omitempty"`
	SpectatorToken string           `json:"spectatorToken,omitempty"`
	Client         *ClientTelemetry `json:"client,omitempty"`
}

type LeaveRoomPayload struct {
	GameType       GameType `json:"gameType,omitempty"`
	RoomCode       string `json:"roomCode"`
	PlayerToken    string `json:"playerToken,omitempty"`
	SpectatorToken string `json:"spectatorToken,omitempty"`
}

type SubmitActionPayload struct {
	GameType          GameType `json:"gameType"`
	RoomCode          string   `json:"roomCode"`
	PlayerToken       string   `json:"playerToken"`
	ActionType        string   `json:"actionType"`
	ActionPayloadJson string   `json:"actionPayloadJson"`
}

type SessionPayload struct {
	GameType       GameType `json:"gameType,omitempty"`
	RoomCode       string `json:"roomCode"`
	PlayerToken    string `json:"playerToken,omitempty"`
	SpectatorToken string `json:"spectatorToken,omitempty"`
}

type HeartbeatPayload = SessionPayload

type ChatMessagePayload struct {
	GameType       GameType `json:"gameType,omitempty"`
	RoomCode       string `json:"roomCode"`
	PlayerToken    string `json:"playerToken,omitempty"`
	SpectatorToken string `json:"spectatorToken,omitempty"`
	MessageID      string `json:"messageId,omitempty"`
	Text           string `json:"text"`
	Relay          bool   `json:"relay,omitempty"`
	MirrorOnly     bool   `json:"mirrorOnly,omitempty"`
}

type WebRTCSignalPayload struct {
	GameType    GameType `json:"gameType,omitempty"`
	RoomCode    string `json:"roomCode"`
	Kind        string `json:"kind"`
	Description any    `json:"description,omitempty"`
	Candidate   any    `json:"candidate,omitempty"`
	SenderID    string `json:"senderId,omitempty"`
	SenderColor *Color `json:"senderColor,omitempty"`
}

type socketSession struct {
	ClientID       string
	RoomCode       string
	PlayerToken    string
	SpectatorToken string
	Nickname       string
	Role           ViewerRole
	Color          *Color
	GameType       GameType
	Mode           GameMode
}

type ackResponse map[string]any

type AppError struct {
	Message    string
	Code       string
	StatusCode int
}

func (e *AppError) Error() string {
	return e.Message
}
