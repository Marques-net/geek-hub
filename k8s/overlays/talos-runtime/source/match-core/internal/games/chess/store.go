package chess

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	client *redis.Client
	config Config
}

func NewStore(config Config) *Store {
	return &Store{
		config: config,
		client: redis.NewClient(&redis.Options{
			Addr:            fmt.Sprintf("%s:%d", config.RedisHost, config.RedisPort),
			Password:        config.RedisPassword,
			MaxRetries:      1,
			MinRetryBackoff: 50 * time.Millisecond,
		}),
	}
}

func (s *Store) Connect(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *Store) Close() error {
	return s.client.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *Store) GetRoom(ctx context.Context, roomCode string) (*RoomState, error) {
	raw, err := s.client.Get(ctx, s.roomKey(roomCode)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var state RoomState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, err
	}
	normalizeStateDefaults(&state)

	return &state, nil
}

func (s *Store) SaveRoom(ctx context.Context, state *RoomState) error {
	nowMs := now()
	state.UpdatedAt = nowMs
	state.ExpiresAt = nowMs + int64(s.config.GameTTLSeconds)*1000

	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, s.roomKey(state.RoomCode), payload, time.Duration(s.config.GameTTLSeconds)*time.Second).Err()
}

func (s *Store) ListRooms(ctx context.Context) ([]*RoomState, error) {
	pattern := fmt.Sprintf("%s:room:*", s.config.RedisKeyPrefix)
	cursor := uint64(0)
	rooms := make([]*RoomState, 0)

	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			raw, err := s.client.Get(ctx, key).Result()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				return nil, err
			}

			var state RoomState
			if err := json.Unmarshal([]byte(raw), &state); err != nil {
				return nil, err
			}
			normalizeStateDefaults(&state)
			rooms = append(rooms, &state)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return rooms, nil
}

func (s *Store) DeleteRoom(ctx context.Context, roomCode string) error {
	return s.client.Del(ctx, s.roomKey(roomCode)).Err()
}

func (s *Store) roomKey(roomCode string) string {
	return fmt.Sprintf("%s:room:%s", s.config.RedisKeyPrefix, roomCode)
}
