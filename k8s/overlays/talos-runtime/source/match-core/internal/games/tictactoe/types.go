package tictactoe

type Color string
type ViewerRole string
type GameMode string
type BotLevel string
type GameStatus string

const (
	GameTypeTicTacToe = "tictactoe"

	ColorWhite Color = "w"
	ColorBlack Color = "b"

	ViewerRolePlayer    ViewerRole = "player"
	ViewerRoleSpectator ViewerRole = "spectator"

	GameModePVP     GameMode = "pvp"
	GameModeBotEasy GameMode = "bot_easy"

	BotLevelEasy BotLevel = "easy"

	GameStatusWaiting  GameStatus = "waiting"
	GameStatusActive   GameStatus = "active"
	GameStatusWon      GameStatus = "won"
	GameStatusDraw     GameStatus = "draw"
	GameStatusResigned GameStatus = "resigned"
	GameStatusTimeout  GameStatus = "timeout"
)

type PlayerSeat struct {
	Nickname    string   `json:"nickname"`
	PlayerToken string   `json:"playerToken"`
	Connected   bool     `json:"connected"`
	JoinedAt    int64    `json:"joinedAt"`
	LastSeenAt  int64    `json:"lastSeenAt"`
	IsBot       bool     `json:"isBot,omitempty"`
	BotLevel    BotLevel `json:"botLevel,omitempty"`
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

type RoomState struct {
	GameID       string             `json:"gameId"`
	GameType     string             `json:"gameType"`
	RoomCode     string             `json:"roomCode"`
	Mode         GameMode           `json:"mode"`
	ClockEnabled bool               `json:"clockEnabled"`
	FEN          string             `json:"fen"`
	PGN          string             `json:"pgn"`
	Turn         Color              `json:"turn"`
	Status       GameStatus         `json:"status"`
	Winner       *Color             `json:"winner"`
	EndReason    *string            `json:"endReason"`
	White        *PlayerSeat        `json:"white"`
	Black        *PlayerSeat        `json:"black"`
	Spectators   []SpectatorSession `json:"spectators"`
	Clocks       GameClockState     `json:"clocks"`
	MoveHistory  []MoveRecord       `json:"moveHistory"`
	LastMove     *MoveRecord        `json:"lastMove"`
	DrawOffer    *DrawOffer         `json:"drawOffer"`
	BotMoveDueAt *int64             `json:"botMoveDueAt"`
	StartedAt    *int64             `json:"startedAt"`
	FinishedAt   *int64             `json:"finishedAt"`
	CreatedAt    int64              `json:"createdAt"`
	UpdatedAt    int64              `json:"updatedAt"`
	ExpiresAt    int64              `json:"expiresAt"`
}

type SessionDescriptor struct {
	RoomCode       string     `json:"roomCode"`
	GameType       string     `json:"gameType"`
	Role           ViewerRole `json:"role"`
	Color          *Color     `json:"color"`
	Nickname       string     `json:"nickname"`
	GameID         string     `json:"gameId"`
	Mode           GameMode   `json:"mode"`
	PlayerToken    string     `json:"playerToken,omitempty"`
	SpectatorToken string     `json:"spectatorToken,omitempty"`
}

type SeatSnapshot struct {
	Nickname  *string   `json:"nickname"`
	Connected bool      `json:"connected"`
	IsBot     bool      `json:"isBot"`
	BotLevel  *BotLevel `json:"botLevel"`
}

type RoomSnapshot struct {
	GameID       string         `json:"gameId"`
	GameType     string         `json:"gameType"`
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

type AppError struct {
	Message    string
	Code       string
	StatusCode int32
}

func (e *AppError) Error() string {
	return e.Message
}
