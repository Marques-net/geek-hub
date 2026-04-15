package chess

import (
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	mu               sync.Mutex
	roomStates       map[string]roomMetricState
	archivedRoomStates map[string]roomMetricState

	activeRooms          *prometheus.GaugeVec
	activeMatches        *prometheus.GaugeVec
	finishedGamesTotal   *prometheus.CounterVec
	roomInfo             *prometheus.GaugeVec
	roomViewers          *prometheus.GaugeVec
	roomSpectators       *prometheus.GaugeVec
	roomMoves            *prometheus.GaugeVec
	roomPlayersConnected *prometheus.GaugeVec
	roomClockSeconds     *prometheus.GaugeVec
	roomStartedSeconds   *prometheus.GaugeVec
	roomFinishedSeconds  *prometheus.GaugeVec
	roomFinished         *prometheus.GaugeVec
	roomOpen             *prometheus.GaugeVec
}

type roomMetricState struct {
	info           roomInfoLabels
	viewers        float64
	spectators     float64
	moves          float64
	whiteConnected float64
	blackConnected float64
	whiteClock     float64
	blackClock     float64
	startedAt      float64
	finishedAt     float64
	finished       float64
	open           float64
}

type roomInfoLabels struct {
	roomCode   string
	gameID     string
	mode       string
	status     string
	turn       string
	botEnabled string
	endReason  string
}

func NewMetrics() *Metrics {
	metrics := &Metrics{
		roomStates:         make(map[string]roomMetricState),
		archivedRoomStates: make(map[string]roomMetricState),
		activeRooms: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_active_rooms",
			Help: "Current tracked rooms by mode.",
		}, []string{"mode"}),
		activeMatches: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_active_matches",
			Help: "Current active matches by mode.",
		}, []string{"mode"}),
		finishedGamesTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "chess_game_core_finished_games_total",
			Help: "Finished games grouped by mode, final status and end reason.",
		}, []string{"mode", "status", "end_reason"}),
		roomInfo: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_room_info",
			Help: "Current room labels for ongoing monitoring.",
		}, []string{"room_code", "game_id", "mode", "status", "turn", "bot_enabled"}),
		roomViewers: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_room_viewers",
			Help: "Connected viewers by room.",
		}, []string{"room_code", "mode"}),
		roomSpectators: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_room_spectators",
			Help: "Connected spectators by room.",
		}, []string{"room_code", "mode"}),
		roomMoves: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_room_moves",
			Help: "Applied moves per room.",
		}, []string{"room_code", "mode"}),
		roomPlayersConnected: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_room_players_connected",
			Help: "Connected player seats by room and color.",
		}, []string{"room_code", "mode", "color"}),
		roomClockSeconds: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_room_clock_seconds",
			Help: "Remaining clock in seconds by room and color.",
		}, []string{"room_code", "mode", "color"}),
		roomStartedSeconds: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_room_started_timestamp_seconds",
			Help: "Match start timestamp by room.",
		}, []string{"room_code", "mode"}),
		roomFinishedSeconds: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_room_finished_timestamp_seconds",
			Help: "Match finish timestamp by room. Zero means not finished.",
		}, []string{"room_code", "mode"}),
		roomFinished: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_room_finished",
			Help: "Whether the tracked match is finished by room.",
		}, []string{"room_code", "mode"}),
		roomOpen: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chess_game_core_room_open",
			Help: "Whether the room is still open in the game core.",
		}, []string{"room_code", "mode"}),
	}

	metrics.recomputeModeGaugesLocked()
	return metrics
}

func (m *Metrics) UpdateRoom(state *RoomState) {
	if state == nil {
		return
	}

	current := buildRoomMetricState(state)

	m.mu.Lock()
	defer m.mu.Unlock()

	if archived, ok := m.archivedRoomStates[state.RoomCode]; ok {
		m.deleteRoomSeriesLocked(archived)
		delete(m.archivedRoomStates, state.RoomCode)
	}

	previous, hadPrevious := m.roomStates[state.RoomCode]
	if hadPrevious {
		m.deleteRoomSeriesLocked(previous)
	}

	if hadPrevious && previous.info.status != current.info.status && isFinishedMetricStatus(current.info.status) {
		m.finishedGamesTotal.WithLabelValues(current.info.mode, current.info.status, current.info.endReason).Inc()
	}

	m.setRoomSeriesLocked(current)
	m.roomStates[state.RoomCode] = current
	m.recomputeModeGaugesLocked()
}

func (m *Metrics) DeleteRoom(roomCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	previous, ok := m.roomStates[roomCode]
	if !ok {
		return
	}

	previous.open = 0
	m.setRoomSeriesLocked(previous)
	m.archivedRoomStates[roomCode] = previous
	delete(m.roomStates, roomCode)
	m.recomputeModeGaugesLocked()
}

func (m *Metrics) deleteRoomSeriesLocked(state roomMetricState) {
	m.roomInfo.DeleteLabelValues(
		state.info.roomCode,
		state.info.gameID,
		state.info.mode,
		state.info.status,
		state.info.turn,
		state.info.botEnabled,
	)
	m.roomViewers.DeleteLabelValues(state.info.roomCode, state.info.mode)
	m.roomSpectators.DeleteLabelValues(state.info.roomCode, state.info.mode)
	m.roomMoves.DeleteLabelValues(state.info.roomCode, state.info.mode)
	m.roomPlayersConnected.DeleteLabelValues(state.info.roomCode, state.info.mode, "white")
	m.roomPlayersConnected.DeleteLabelValues(state.info.roomCode, state.info.mode, "black")
	m.roomClockSeconds.DeleteLabelValues(state.info.roomCode, state.info.mode, "white")
	m.roomClockSeconds.DeleteLabelValues(state.info.roomCode, state.info.mode, "black")
}

func (m *Metrics) setRoomSeriesLocked(state roomMetricState) {
	m.roomInfo.WithLabelValues(
		state.info.roomCode,
		state.info.gameID,
		state.info.mode,
		state.info.status,
		state.info.turn,
		state.info.botEnabled,
	).Set(1)
	m.roomViewers.WithLabelValues(state.info.roomCode, state.info.mode).Set(state.viewers)
	m.roomSpectators.WithLabelValues(state.info.roomCode, state.info.mode).Set(state.spectators)
	m.roomMoves.WithLabelValues(state.info.roomCode, state.info.mode).Set(state.moves)
	m.roomPlayersConnected.WithLabelValues(state.info.roomCode, state.info.mode, "white").Set(state.whiteConnected)
	m.roomPlayersConnected.WithLabelValues(state.info.roomCode, state.info.mode, "black").Set(state.blackConnected)
	m.roomClockSeconds.WithLabelValues(state.info.roomCode, state.info.mode, "white").Set(state.whiteClock)
	m.roomClockSeconds.WithLabelValues(state.info.roomCode, state.info.mode, "black").Set(state.blackClock)
	m.roomStartedSeconds.WithLabelValues(state.info.roomCode, state.info.mode).Set(state.startedAt)
	m.roomFinishedSeconds.WithLabelValues(state.info.roomCode, state.info.mode).Set(state.finishedAt)
	m.roomFinished.WithLabelValues(state.info.roomCode, state.info.mode).Set(state.finished)
	m.roomOpen.WithLabelValues(state.info.roomCode, state.info.mode).Set(state.open)
}

func (m *Metrics) recomputeModeGaugesLocked() {
	counts := map[string]float64{
		string(GameModePVP):     0,
		string(GameModeBotEasy): 0,
	}
	active := map[string]float64{
		string(GameModePVP):     0,
		string(GameModeBotEasy): 0,
	}

	for _, room := range m.roomStates {
		counts[room.info.mode]++
		if room.info.status == string(GameStatusActive) {
			active[room.info.mode]++
		}
	}

	for mode, value := range counts {
		m.activeRooms.WithLabelValues(mode).Set(value)
	}
	for mode, value := range active {
		m.activeMatches.WithLabelValues(mode).Set(value)
	}
}

func buildRoomMetricState(state *RoomState) roomMetricState {
	spectators := float64(countConnectedSpectators(state.Spectators))
	whiteConnected := connectedSeatValue(state.White)
	blackConnected := connectedSeatValue(state.Black)
	mode := string(state.Mode)
	if mode == "" {
		mode = string(GameModePVP)
	}

	return roomMetricState{
		info: roomInfoLabels{
			roomCode:   state.RoomCode,
			gameID:     state.GameID,
			mode:       mode,
			status:     string(state.Status),
			turn:       metricColor(state.Turn),
			botEnabled: strconv.FormatBool(isBotEnabled(state)),
			endReason:  metricEndReason(state.EndReason),
		},
		viewers:        spectators + whiteConnected + blackConnected,
		spectators:     spectators,
		moves:          float64(len(state.MoveHistory)),
		whiteConnected: whiteConnected,
		blackConnected: blackConnected,
		whiteClock:     float64(state.Clocks.WhiteMs) / 1000,
		blackClock:     float64(state.Clocks.BlackMs) / 1000,
		startedAt:      metricTimestampSeconds(state.StartedAt),
		finishedAt:     metricTimestampSeconds(state.FinishedAt),
		finished:       metricFinishedValue(state.Status),
		open:           1,
	}
}

func connectedSeatValue(seat *PlayerSeat) float64 {
	if seat == nil || !seat.Connected {
		return 0
	}

	return 1
}

func countConnectedSpectators(spectators []SpectatorSession) int {
	total := 0
	for _, spectator := range spectators {
		if spectator.Connected {
			total++
		}
	}
	return total
}

func isBotEnabled(state *RoomState) bool {
	return (state.White != nil && state.White.IsBot) || (state.Black != nil && state.Black.IsBot)
}

func metricColor(color Color) string {
	switch color {
	case ColorWhite:
		return "white"
	case ColorBlack:
		return "black"
	default:
		return "none"
	}
}

func metricEndReason(value *string) string {
	if value == nil || *value == "" {
		return "none"
	}

	return *value
}

func metricTimestampSeconds(value *int64) float64 {
	if value == nil || *value <= 0 {
		return 0
	}

	return float64(*value) / 1000
}

func metricFinishedValue(status GameStatus) float64 {
	if isFinishedStatus(status) {
		return 1
	}

	return 0
}

func isFinishedMetricStatus(status string) bool {
	return isFinishedStatus(GameStatus(status))
}
