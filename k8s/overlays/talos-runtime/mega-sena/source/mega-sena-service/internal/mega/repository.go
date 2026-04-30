package mega

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ResultRecord struct {
	Concurso     int
	DataSorteio  time.Time
	Dezenas      []int
	OrdemSorteio []int
	Source       string
	OfficialURL  string
	AtualizadoEm time.Time
}

type SimulationNumberRecord struct {
	Position int
	Number   int
	DrawnAt  time.Time
	Date     string
	Hour     int
	Minute   int
	Second   int
}

type SimulationRecord struct {
	Numbers      []SimulationNumberRecord
	RegisteredAt time.Time
	Source       string
}

type Repository struct {
	client               *mongo.Client
	resultCollection     *mongo.Collection
	simulationCollection *mongo.Collection
	counterCollection    *mongo.Collection
}

func NewRepository(ctx context.Context, cfg Config) (*Repository, error) {
	connectCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.MongoTimeoutMs)*time.Millisecond)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(cfg.MongoURI()))
	if err != nil {
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}

	repository := &Repository{
		client:               client,
		resultCollection:     client.Database(cfg.MongoDatabase).Collection(cfg.MongoMegaSenaCollection),
		simulationCollection: client.Database(cfg.MongoDatabase).Collection(cfg.MongoSimulationCollection),
		counterCollection:    client.Database(cfg.MongoDatabase).Collection(cfg.MongoCounterCollection),
	}

	if err := repository.Ready(connectCtx); err != nil {
		_ = repository.Close(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}
	if err := repository.ensureIndexes(connectCtx); err != nil {
		_ = repository.Close(context.Background())
		return nil, fmt.Errorf("ensure mongodb indexes: %w", err)
	}

	return repository, nil
}

func (r *Repository) Ready(ctx context.Context) error {
	if r == nil || r.client == nil {
		return errors.New("mega-sena repository unavailable")
	}
	return r.client.Ping(ctx, nil)
}

func (r *Repository) Close(ctx context.Context) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Disconnect(ctx)
}

func (r *Repository) UpsertResult(ctx context.Context, record ResultRecord) error {
	if r == nil || r.resultCollection == nil {
		return errors.New("mega-sena repository unavailable")
	}
	if record.Concurso <= 0 {
		return errors.New("invalid mega-sena concurso")
	}
	if len(record.Dezenas) != 6 || len(record.OrdemSorteio) != 6 {
		return errors.New("mega-sena result requires six numbers")
	}
	if err := validateNumbers(record.Dezenas); err != nil {
		return err
	}
	if err := validateNumbers(record.OrdemSorteio); err != nil {
		return err
	}

	sortedDezenas := append([]int(nil), record.Dezenas...)
	sort.Ints(sortedDezenas)
	orderedSet := make(map[int]struct{}, len(record.OrdemSorteio))
	for _, value := range record.OrdemSorteio {
		orderedSet[value] = struct{}{}
	}
	for _, value := range sortedDezenas {
		if _, exists := orderedSet[value]; !exists {
			return errors.New("mega-sena sorted and draw-order numbers do not match")
		}
	}

	atualizadoEm := record.AtualizadoEm.UTC()
	if atualizadoEm.IsZero() {
		atualizadoEm = time.Now().UTC()
	}

	_, err := r.resultCollection.UpdateOne(
		ctx,
		bson.M{"concurso": record.Concurso},
		bson.M{
			"$set": bson.M{
				"concurso":     record.Concurso,
				"dataSorteio":  record.DataSorteio.UTC(),
				"dezenas":      sortedDezenas,
				"ordemSorteio": record.OrdemSorteio,
				"source":       strings.TrimSpace(record.Source),
				"officialUrl":  strings.TrimSpace(record.OfficialURL),
				"atualizadoEm": atualizadoEm,
			},
			"$setOnInsert": bson.M{"criadoEm": atualizadoEm},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *Repository) RecordSimulation(ctx context.Context, record SimulationRecord) (int64, error) {
	if r == nil || r.simulationCollection == nil || r.counterCollection == nil {
		return 0, errors.New("mega-sena repository unavailable")
	}
	if len(record.Numbers) != 6 {
		return 0, errors.New("simulation requires six numbers")
	}

	positionSet := make(map[int]struct{}, len(record.Numbers))
	numberSet := make(map[int]struct{}, len(record.Numbers))
	sort.Slice(record.Numbers, func(left, right int) bool {
		return record.Numbers[left].Position < record.Numbers[right].Position
	})

	resultByPosition := bson.M{}
	numbers := make([]bson.M, 0, len(record.Numbers))
	for _, item := range record.Numbers {
		if item.Position < 1 || item.Position > 6 {
			return 0, errors.New("invalid simulation position")
		}
		if item.Number < 1 || item.Number > 60 {
			return 0, errors.New("invalid simulation number")
		}
		if _, exists := positionSet[item.Position]; exists {
			return 0, errors.New("duplicated simulation position")
		}
		if _, exists := numberSet[item.Number]; exists {
			return 0, errors.New("duplicated simulation number")
		}

		positionSet[item.Position] = struct{}{}
		numberSet[item.Number] = struct{}{}

		itemDocument := bson.M{
			"posicao":    item.Position,
			"dezena":     item.Number,
			"data":       strings.TrimSpace(item.Date),
			"hora":       item.Hour,
			"minuto":     item.Minute,
			"segundo":    item.Second,
			"sorteadoEm": item.DrawnAt.UTC(),
		}
		resultByPosition[fmt.Sprintf("dezena%02d", item.Position)] = itemDocument
		numbers = append(numbers, itemDocument)
	}

	registeredAt := record.RegisteredAt.UTC()
	if registeredAt.IsZero() {
		registeredAt = time.Now().UTC()
	}

	var counter struct {
		Seq int64 `bson:"seq"`
	}
	if err := r.counterCollection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": "mega_sena_simulacoes"},
		bson.M{"$inc": bson.M{"seq": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&counter); err != nil {
		return 0, err
	}

	document := bson.M{
		"sequencial":   counter.Seq,
		"registradoEm": registeredAt,
		"source":       strings.TrimSpace(record.Source),
		"resultado":    resultByPosition,
		"dezenas":      numbers,
	}
	_, err := r.simulationCollection.InsertOne(ctx, document)
	if err != nil {
		return 0, err
	}
	return counter.Seq, nil
}

func (r *Repository) ensureIndexes(ctx context.Context) error {
	if _, err := r.resultCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "concurso", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_concurso"),
		},
		{
			Keys:    bson.D{{Key: "dataSorteio", Value: -1}},
			Options: options.Index().SetName("data_sorteio_desc"),
		},
	}); err != nil {
		return err
	}

	_, err := r.simulationCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "sequencial", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "registradoEm", Value: -1}},
		},
	})
	return err
}

func validateNumbers(values []int) error {
	seen := make(map[int]struct{}, len(values))
	for _, value := range values {
		if value < 1 || value > 60 {
			return errors.New("invalid mega-sena number")
		}
		if _, exists := seen[value]; exists {
			return errors.New("duplicated mega-sena number")
		}
		seen[value] = struct{}{}
	}
	return nil
}
