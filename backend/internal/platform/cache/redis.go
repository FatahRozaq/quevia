package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionStore handles refresh token allow-lists in Redis.
type SessionStore struct {
	client *redis.Client
}

// NewSessionStore creates a Redis session store.
func NewSessionStore(redisURL string) (*SessionStore, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parsing redis url: %w", err)
	}

	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	return &SessionStore{client: client}, nil
}

// StoreRefreshToken saves a refresh token in Redis with a TTL.
func (s *SessionStore) StoreRefreshToken(ctx context.Context, token string, userGUID string, ttl time.Duration) error {
	key := fmt.Sprintf("rt:%s", token)
	if err := s.client.Set(ctx, key, userGUID, ttl).Err(); err != nil {
		return fmt.Errorf("saving refresh token: %w", err)
	}
	return nil
}

// ValidateRefreshToken checks if a refresh token exists in Redis.
func (s *SessionStore) ValidateRefreshToken(ctx context.Context, token string) (string, error) {
	key := fmt.Sprintf("rt:%s", token)
	userGUID, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("token not found or expired: %w", err)
	}
	return userGUID, nil
}

// RevokeRefreshToken deletes a refresh token from Redis.
func (s *SessionStore) RevokeRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("rt:%s", token)
	return s.client.Del(ctx, key).Err()
}
