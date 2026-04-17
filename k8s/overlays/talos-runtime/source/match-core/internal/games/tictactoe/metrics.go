package tictactoe

import (
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	mu                 sync.Mutex
	roomStates         map[string]roomMetricState
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
			Name: "tictactoe_game_core_active_rooms",
			Help: "Current tracked tic-tac-toe rooms by mode.",
		}, []string{"mode"}),
		activeMatches: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_active_matches",
			Help: "Current active tic-tac-toe matches by mode.",
		}, []string{"mode"}),
		finishedGamesTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "tictactoe_game_core_finished_games_total",
			Help: "Finished tic-tac-toe games grouped by mode, final status and end reason.",
		}, []string{"mode", "status", "end_reason"}),
		roomInfo: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_room_info",
			Help: "Current tic-tac-toe room labels for ongoing monitoring.",
		}, []string{"room_code", "game_id", "mode", "status", "turn", "bot_enabled"}),
		roomViewers: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_room_viewers",
			Help: "Connected viewers by tic-tac-toe room.",
		}, []string{"room_code", "mode"}),
		roomSpectators: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_room_spectators",
			Help: "Connected spectators by tic-tac-toe room.",
		}, []string{"room_code", "mode"}),
		roomMoves: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_room_moves",
			Help: "Applied moves per tic-tac-toe room.",
		}, []string{"room_code", "mode"}),
		roomPlayersConnected: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_room_players_connected",
			Help: "Connected player seats by tic-tac-toe room and color.",
		}, []string{"room_code", "mode", "color"}),
		roomClockSeconds: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_room_clock_seconds",
			Help: "Remaining clock in seconds by tic-tac-toe room and color.",
		}, []string{"room_code", "mode", "color"}),
		roomStartedSeconds: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_room_started_timestamp_seconds",
			Help: "Match start timestamp by tic-tac-toe room.",
		}, []string{"room_code", "mode"}),
		roomFinishedSeconds: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_room_finished_timestamp_seconds",
			Help: "Match finish timestamp by tic-tac-toe room. Zero means not finished.",
		}, []string{"room_code", "mode"}),
		roomFinished: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_room_finished",
			Help: "Whether the tracked tic-tac-toe match is finished by room.",
		}, []string{"room_code", "mode"}),
		roomOpen: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tictactoe_game_core_room_open",
			Help: "Whether the tic-tac-toe room is still open in the game core.",
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
			turn:       string(state.Turn),
			botEnabled: strconv.FormatBool(state.Mode == GameModeBotEasy),
			endReason:  labelOrNone(state.EndReason),
		},
		viewers:        float64(viewerCount(state)),
		spectators:     spectators,
		moves:          float64(len(state.MoveHistory)),
		whiteConnected: whiteConnected,
		blackConnected: blackConnected,
		whiteClock:     float64(state.Clocks.WhiteMs) / 1000,
		blackClock:     float64(state.Clocks.BlackMs) / 1000,
		startedAt:      timestampSeconds(state.StartedAt),
		finishedAt:     timestampSeconds(state.FinishedAt),
		finished:       boolAsFloat(isFinishedStatus(state.Status)),
		open:           1,
	}
}

func countConnectedSpectators(values []SpectatorSession) int {
	total := 0
	for _, spectator := range values {
		if spectator.Connected {
			total++
		}
	}
	return total
}

func connectedSeatValue(seat *PlayerSeat) float64 {
	if seat == nil || !seat.Connected {
		return 0
	}
	return 1
}

func viewerCount(state *RoomState) int {
	total := len(state.Spectators)
	if state.White != nil {
		total++
	}
	if state.Black != nil {
		total++
	}
	return total
}

func timestampSeconds(value *int64) float64 {
	if value == nil {
		return 0
	}
	return float64(*value) / 1000
}

func boolAsFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func labelOrNone(value *string) string {
	if value == nil || *value == "" {
		return "none"
	}
	return *value
}

func isFinishedMetricStatus(status string) bool {
	switch GameStatus(status) {
	case GameStatusWon, GameStatusDraw, GameStatusResigned, GameStatusTimeout:
		return true
	default:
		return false
	}
}
