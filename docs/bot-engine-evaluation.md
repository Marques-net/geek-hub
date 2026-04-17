# Bot Engine Evaluation

## Status

A engine de bot foi separada do nucleo da partida e roda como um servico dedicado multi-jogo:

- contrato entre componentes em protobuf: [`proto/strategy_engine.proto`](../proto/strategy_engine.proto)
- serviço dedicado da engine: [`services/bot-engine/cmd/bot-engine/main.go`](../services/bot-engine/cmd/bot-engine/main.go)
- clientes gRPC por runtime: [`services/match-core/internal/games/chess/bot_client.go`](../services/match-core/internal/games/chess/bot_client.go) e [`services/match-core/internal/games/tictactoe/bot_client.go`](../services/match-core/internal/games/tictactoe/bot_client.go)
- nucleo autoritativo multi-jogo: [`services/match-core/cmd/match-core/main.go`](../services/match-core/cmd/match-core/main.go)
- estrategias atuais: [`services/bot-engine/internal/games/chess/easy.go`](../services/bot-engine/internal/games/chess/easy.go) e [`services/bot-engine/internal/games/tictactoe/easy.go`](../services/bot-engine/internal/games/tictactoe/easy.go)

## Boundary

O `match-core` continua responsavel por:

- validar as regras de cada jogo no runtime correto
- controlar turno, relógio e término da partida
- persistir estado no Redis
- publicar snapshots por Socket.IO

A bot engine passa a ser responsavel apenas por:

- receber o estado atual da partida serializado em `state_json`
- escolher uma acao compativel com o `game_type` e o modo solicitado
- devolver a sugestao de acao em protobuf/gRPC

Se a engine devolver um lance inválido, o backend rejeita a resposta e não aplica o movimento.

## Communication

O contrato atual é:

- `GetActionRequest`
  - `game_type`
  - `room_code`
  - `game_id`
  - `state_json`
  - `mode`
  - `recent_actions`
  - `move_count`
- `GetActionResponse`
  - `found`
  - `action_type`
  - `action_payload_json`
  - `coach_message`

Isso mantem o acoplamento baixo e permite adicionar novos jogos sem reescrever o backend nem criar um novo contrato por jogo.

## Cobertura Atual

- `chess`
  - entrada esperada em `state_json`: FEN atual
  - resposta easy: movimento com `from`, `to` e `promotion`
  - coaching: principios de desenvolvimento, centro e seguranca do rei
- `tictactoe`
  - entrada esperada em `state_json`: `board` e `turn`
  - resposta easy: movimento com `cell`
  - coaching: bloqueio, linha de vitoria, centro, cantos e laterais

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
- o servico ja atende `chess` e `tictactoe` pelo mesmo endpoint gRPC
- logs estruturados são emitidos em `stdout` para coleta pelo `promtail`/`loki`
- traces OTLP são enviados para o `tempo`

Rust só passa a ser a melhor escolha se a prioridade mudar de "serviço de treino simples e operável" para "engine significativamente mais forte e otimizada".
