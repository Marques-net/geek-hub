package login

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserLoginRecord struct {
	Provider        string
	ProviderUserID  string
	Name            string
	Email           *string
	LoggedAt        time.Time
	Source          string
	UserAgent       string
	DeviceType      string
	Platform        string
	PlatformVersion string
	Browser         string
	BrowserVersion  string
	Region          string
	DeviceModel     string
	RawUserAgent    string
}

type LoginRepository struct {
	client          *mongo.Client
	loginCollection *mongo.Collection
}

func NewLoginRepository(ctx context.Context, cfg Config) (*LoginRepository, error) {
	connectCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.MongoTimeoutMs)*time.Millisecond)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(cfg.MongoURI()))
	if err != nil {
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}

	repository := &LoginRepository{
		client:          client,
		loginCollection: client.Database(cfg.MongoDatabase).Collection(cfg.MongoCollection),
	}

	if err := repository.Ready(connectCtx); err != nil {
		_ = repository.Close(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	if err := repository.ensureLoginIndexes(connectCtx); err != nil {
		_ = repository.Close(context.Background())
		return nil, fmt.Errorf("ensure mongodb login indexes: %w", err)
	}

	return repository, nil
}

func (r *LoginRepository) Ready(ctx context.Context) error {
	if r == nil || r.client == nil {
		return errors.New("login repository unavailable")
	}

	return r.client.Ping(ctx, nil)
}

func (r *LoginRepository) Close(ctx context.Context) error {
	if r == nil || r.client == nil {
		return nil
	}

	return r.client.Disconnect(ctx)
}

func (r *LoginRepository) RecordLogin(ctx context.Context, record UserLoginRecord) error {
	if r == nil || r.loginCollection == nil {
		return errors.New("login repository unavailable")
	}

	loggedAt := record.LoggedAt.UTC()
	if loggedAt.IsZero() {
		loggedAt = time.Now().UTC()
	}

	document := bson.M{
		"provider":        strings.TrimSpace(record.Provider),
		"providerUserId":  strings.TrimSpace(record.ProviderUserID),
		"name":            strings.TrimSpace(record.Name),
		"loggedAt":        loggedAt,
		"source":          strings.TrimSpace(record.Source),
		"userAgent":       strings.TrimSpace(record.UserAgent),
		"deviceType":      normalizeDimension(record.DeviceType),
		"platform":        normalizeDimension(record.Platform),
		"platformVersion": strings.TrimSpace(record.PlatformVersion),
		"browser":         normalizeDimension(record.Browser),
		"browserVersion":  strings.TrimSpace(record.BrowserVersion),
		"region":          normalizeDimension(record.Region),
		"deviceModel":     strings.TrimSpace(record.DeviceModel),
		"rawUserAgent":    strings.TrimSpace(record.RawUserAgent),
	}

	if record.Email != nil {
		email := strings.TrimSpace(*record.Email)
		if email != "" {
			document["email"] = email
		}
	}

	_, err := r.loginCollection.InsertOne(ctx, document)
	return err
}

func (r *LoginRepository) ensureLoginIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "loggedAt", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "provider", Value: 1},
				{Key: "providerUserId", Value: 1},
				{Key: "loggedAt", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "email", Value: 1},
				{Key: "loggedAt", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "platform", Value: 1},
				{Key: "browser", Value: 1},
				{Key: "deviceType", Value: 1},
				{Key: "loggedAt", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "deviceModel", Value: 1},
				{Key: "loggedAt", Value: -1},
			},
		},
	}

	_, err := r.loginCollection.Indexes().CreateMany(ctx, indexes)
	return err
}

func normalizeDimension(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "unknown"
	}

	return normalized
}
