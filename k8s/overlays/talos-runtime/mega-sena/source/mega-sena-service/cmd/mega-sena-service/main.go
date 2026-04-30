package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Marques-net/geek-hub/services/mega-sena-service/internal/mega"
	"github.com/Marques-net/geek-hub/services/mega-sena-service/internal/observability"
)

func main() {
	cfg := mega.LoadConfig()
	logger := observability.NewLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := observability.SetupTracing(ctx, logger)
	if err != nil {
		logger.Error("failed to setup tracing", "error", err.Error())
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(shutdownCtx); err != nil {
			logger.Error("failed to shutdown tracing", "error", err.Error())
		}
	}()

	repository, err := mega.NewRepository(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := repository.Close(shutdownCtx); err != nil {
			logger.Error("failed to close repository", "error", err.Error())
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]any{"status": "ok", "service": "mega-sena-service"})
	})
	mux.HandleFunc("/health/ready", func(writer http.ResponseWriter, request *http.Request) {
		checkCtx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
		defer cancel()
		if err := repository.Ready(checkCtx); err != nil {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]any{
				"status":  "error",
				"mongo":   "unavailable",
				"message": err.Error(),
			})
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"status": "ok", "mongo": "ready"})
	})
	mux.HandleFunc("/api/mega-sena/ultimo-resultado", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]any{"status": "error", "message": "Metodo nao suportado."})
			return
		}

		updateCtx, cancel := context.WithTimeout(request.Context(), 10*time.Second)
		defer cancel()

		result, err := fetchOfficialMegaSenaResult(updateCtx, 0)
		if err != nil {
			logger.Error("failed to fetch official mega-sena result", "error", err.Error())
			writeJSON(writer, http.StatusBadGateway, map[string]any{
				"status":  "error",
				"message": "Nao foi possivel consultar o resultado oficial da Mega-Sena.",
			})
			return
		}

		record := result.toRecord(time.Now().UTC())
		if err := repository.UpsertResult(updateCtx, record); err != nil {
			logger.Error("failed to upsert mega-sena result", "error", err.Error(), "concurso", record.Concurso)
			writeJSON(writer, http.StatusServiceUnavailable, map[string]any{
				"status":  "error",
				"message": "Nao foi possivel atualizar o resultado da Mega-Sena no MongoDB.",
			})
			return
		}

		writeJSON(writer, http.StatusOK, map[string]any{
			"status":       "ok",
			"concurso":     record.Concurso,
			"dataSorteio":  record.DataSorteio.Format("2006-01-02"),
			"dezenas":      record.Dezenas,
			"ordemSorteio": record.OrdemSorteio,
			"atualizadoEm": record.AtualizadoEm.Format(time.RFC3339),
			"source":       record.Source,
		})
	})
	mux.HandleFunc("/api/mega-sena/simulacoes", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]any{"status": "error", "message": "Metodo nao suportado."})
			return
		}

		record, err := decodeSimulationRequest(request.Body)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]any{"status": "error", "message": err.Error()})
			return
		}

		record.RegisteredAt = time.Now().UTC()
		record.Source = "mega-sena-web:simulacao"
		registerCtx, cancel := context.WithTimeout(request.Context(), 3*time.Second)
		defer cancel()

		sequencial, err := repository.RecordSimulation(registerCtx, *record)
		if err != nil {
			logger.Error("failed to persist mega-sena simulation", "error", err.Error())
			writeJSON(writer, http.StatusServiceUnavailable, map[string]any{
				"status":  "error",
				"message": "Nao foi possivel registrar a simulacao da Mega-Sena.",
			})
			return
		}

		writeJSON(writer, http.StatusCreated, map[string]any{
			"status":       "ok",
			"sequencial":   sequencial,
			"registradoEm": record.RegisteredAt.Format(time.RFC3339),
		})
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("http shutdown failed", "error", err.Error())
		}
	}()

	logger.Info("mega-sena-service started", "port", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload map[string]any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

type officialMegaSenaResult struct {
	DataApuracao                 string   `json:"dataApuracao"`
	DezenasSorteadasOrdemSorteio []string `json:"dezenasSorteadasOrdemSorteio"`
	ListaDezenas                 []string `json:"listaDezenas"`
	Numero                       int      `json:"numero"`
}

func fetchOfficialMegaSenaResult(ctx context.Context, concurso int) (*officialMegaSenaResult, error) {
	url := "https://servicebus2.caixa.gov.br/portaldeloterias/api/megasena"
	if concurso > 0 {
		url += "/" + strconv.Itoa(concurso)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "geek-hub-mega-sena-service/1.0")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, errors.New("fonte oficial retornou status inesperado")
	}

	var result officialMegaSenaResult
	decoder := json.NewDecoder(io.LimitReader(response.Body, 64<<10))
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	if result.Numero <= 0 || len(result.ListaDezenas) != 6 || len(result.DezenasSorteadasOrdemSorteio) != 6 {
		return nil, errors.New("resultado oficial incompleto")
	}
	return &result, nil
}

func (r officialMegaSenaResult) toRecord(updatedAt time.Time) mega.ResultRecord {
	drawDate, _ := time.Parse("02/01/2006", strings.TrimSpace(r.DataApuracao))
	dezenas := parseOfficialNumbers(r.ListaDezenas)
	ordemSorteio := parseOfficialNumbers(r.DezenasSorteadasOrdemSorteio)
	sort.Ints(dezenas)
	return mega.ResultRecord{
		Concurso:     r.Numero,
		DataSorteio:  drawDate,
		Dezenas:      dezenas,
		OrdemSorteio: ordemSorteio,
		Source:       "caixa:loterias",
		OfficialURL:  "https://servicebus2.caixa.gov.br/portaldeloterias/api/megasena/" + strconv.Itoa(r.Numero),
		AtualizadoEm: updatedAt,
	}
}

func parseOfficialNumbers(values []string) []int {
	numbers := make([]int, 0, len(values))
	for _, value := range values {
		number, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			numbers = append(numbers, number)
		}
	}
	return numbers
}

type simulationRequest struct {
	Dezenas []simulationNumberRequest `json:"dezenas"`
}

type simulationNumberRequest struct {
	Posicao    int    `json:"posicao"`
	Dezena     int    `json:"dezena"`
	SorteadoEm string `json:"sorteadoEm"`
	Data       string `json:"data"`
	Hora       int    `json:"hora"`
	Minuto     int    `json:"minuto"`
	Segundo    int    `json:"segundo"`
}

func decodeSimulationRequest(body io.ReadCloser) (*mega.SimulationRecord, error) {
	defer body.Close()

	var payload simulationRequest
	decoder := json.NewDecoder(io.LimitReader(body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return nil, errors.New("Payload da simulacao invalido.")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("Payload da simulacao invalido.")
	}
	if len(payload.Dezenas) != 6 {
		return nil, errors.New("A simulacao precisa de exatamente seis dezenas.")
	}

	positionSet := make(map[int]struct{}, len(payload.Dezenas))
	numberSet := make(map[int]struct{}, len(payload.Dezenas))
	numbers := make([]mega.SimulationNumberRecord, 0, len(payload.Dezenas))

	for _, item := range payload.Dezenas {
		switch {
		case item.Posicao < 1 || item.Posicao > 6:
			return nil, errors.New("Cada dezena precisa informar uma posicao de 1 a 6.")
		case item.Dezena < 1 || item.Dezena > 60:
			return nil, errors.New("Cada dezena precisa estar entre 1 e 60.")
		case item.Hora < 0 || item.Hora > 23:
			return nil, errors.New("Hora invalida na simulacao.")
		case item.Minuto < 0 || item.Minuto > 59:
			return nil, errors.New("Minuto invalido na simulacao.")
		case item.Segundo < 0 || item.Segundo > 59:
			return nil, errors.New("Segundo invalido na simulacao.")
		}
		if _, exists := positionSet[item.Posicao]; exists {
			return nil, errors.New("Nao e permitido repetir a mesma posicao na simulacao.")
		}
		if _, exists := numberSet[item.Dezena]; exists {
			return nil, errors.New("Nao e permitido repetir dezenas na simulacao.")
		}

		drawnAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(item.SorteadoEm))
		if err != nil {
			return nil, errors.New("Data e hora de sorteio invalida para uma das dezenas.")
		}
		data := strings.TrimSpace(item.Data)
		if _, err := time.Parse("2006-01-02", data); err != nil {
			return nil, errors.New("Data invalida para uma das dezenas.")
		}
		positionSet[item.Posicao] = struct{}{}
		numberSet[item.Dezena] = struct{}{}
		numbers = append(numbers, mega.SimulationNumberRecord{
			Position: item.Posicao,
			Number:   item.Dezena,
			DrawnAt:  drawnAt,
			Date:     data,
			Hour:     item.Hora,
			Minute:   item.Minuto,
			Second:   item.Segundo,
		})
	}
	sort.Slice(numbers, func(left, right int) bool {
		return numbers[left].Position < numbers[right].Position
	})
	return &mega.SimulationRecord{Numbers: numbers}, nil
}
