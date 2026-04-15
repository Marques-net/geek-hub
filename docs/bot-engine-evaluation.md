# Bot Engine Evaluation

## Status

A engine de bot foi separada do núcleo da partida e passou a rodar como um serviço dedicado:

- contrato entre componentes em protobuf: [`proto/strategy_engine.proto`](../proto/strategy_engine.proto)
- serviço dedicado da engine: [`services/bot-engine/cmd/bot-engine/main.go`](../services/bot-engine/cmd/bot-engine/main.go)
- cliente gRPC no núcleo da partida: [`services/match-core/internal/games/chess/bot_client.go`](../services/match-core/internal/games/chess/bot_client.go)
- núcleo autoritativo: [`services/match-core/internal/games/chess/service.go`](../services/match-core/internal/games/chess/service.go)

## Boundary

O `match-core` continua responsável por:

- validar regras do xadrez com `chess.js`
- controlar turno, relógio e término da partida
- persistir estado no Redis
- publicar snapshots por Socket.IO

A bot engine passa a ser responsável apenas por:

- receber o estado atual da partida em FEN
- escolher um lance compatível com o modo solicitado
- devolver a sugestão de lance em protobuf/gRPC

Se a engine devolver um lance inválido, o backend rejeita a resposta e não aplica o movimento.

## Communication

O contrato atual é:

- `GetMoveRequest`
  - `room_code`
  - `game_id`
  - `fen`
  - `mode`
  - `recent_sans`
  - `move_count`
- `GetMoveResponse`
  - `found`
  - `from`
  - `to`
  - `promotion`

Isso mantém o acoplamento baixo e permite trocar a implementação da engine sem reescrever o backend.

## Language Evaluation

### Go

Melhor escolha para a próxima versão da engine dedicada.

Pontos fortes:

- excelente suporte a protobuf/gRPC
- binário único e deploy simples em Kubernetes
- latência baixa e previsível
- concorrência simples para múltiplas salas
- curva de manutenção menor que Rust

Trade-off:

- menor controle fino de memória e otimização extrema do que Rust

### Rust

Melhor escolha se o foco principal for máxima eficiência e evolução para uma engine mais forte.

Pontos fortes:

- melhor eficiência de CPU/memória entre as opções avaliadas
- ótimo para busca minimax, alpha-beta, tabelas de transposição e heurísticas pesadas
- alta segurança de memória sem GC

Trade-off:

- maior custo de implementação
- maior tempo de onboarding
- iteração mais lenta para um time que não domina Rust

### C++

Faz sentido apenas se a estratégia for integrar uma engine já madura como Stockfish ou construir algo de nível competitivo.

Pontos fortes:

- ecossistema histórico de engines de xadrez
- performance máxima

Trade-off:

- maior complexidade operacional e de manutenção
- pior experiência de integração/observabilidade do que Go para um serviço interno simples

### TypeScript / Node.js

Adequado como etapa de transição e validação arquitetural, que é o estado atual do projeto.

Pontos fortes:

- mesma stack do backend
- implementação rápida
- reduz risco de refactor inicial

Trade-off:

- pior eficiência por CPU para busca mais pesada
- menos adequado para evolução futura da engine

## Recommendation

Decisão recomendada:

1. manter a separação atual via protobuf/gRPC
2. usar a implementação anterior em TypeScript apenas como referência histórica
3. manter a bot engine em Go como baseline para novos jogos

Go é a linguagem mais eficiente aqui no sentido prático: performance suficiente, baixa complexidade operacional e manutenção mais simples.

Estado atual do projeto:

- a bot engine já foi migrada para Go
- logs estruturados são emitidos em `stdout` para coleta pelo `promtail`/`loki`
- traces OTLP são enviados para o `tempo`

Rust só passa a ser a melhor escolha se a prioridade mudar de "serviço de treino simples e operável" para "engine significativamente mais forte e otimizada".
