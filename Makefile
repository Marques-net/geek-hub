NAMESPACE ?= chess-dev
PORTAL_FRONTEND_IMAGE ?= geek-hub-portal-web:dev
GAMES_WEB_IMAGE ?= geek-hub-games-web:dev
REALTIME_GATEWAY_IMAGE ?= geek-hub-realtime-gateway:dev
MATCH_CORE_IMAGE ?= geek-hub-match-core:dev
BOT_ENGINE_IMAGE ?= geek-hub-bot-engine:dev
VITE_BACKEND_URL ?= http://chess.local

.PHONY: deps portal-web-deps games-web-deps realtime-gateway-deps match-core-deps bot-engine-deps docker-build build-portal-web build-games-web build-realtime-gateway build-match-core build-bot-engine apply delete dev status ingress logs-realtime-gateway logs-match-core logs-portal-web logs-games-web logs-bot-engine

deps: portal-web-deps games-web-deps realtime-gateway-deps match-core-deps bot-engine-deps

portal-web-deps:
	cd apps/portal-web && npm install

games-web-deps:
	cd apps/games-web && npm install

realtime-gateway-deps:
	@echo "realtime-gateway usa Go; build via Docker, Skaffold ou overlay runtime"

match-core-deps:
	@echo "match-core usa Go; build via Docker, Skaffold ou overlay runtime"

bot-engine-deps:
	@echo "bot-engine usa Go; build via Docker, Skaffold ou overlay runtime"

build-portal-web:
	docker build -t $(PORTAL_FRONTEND_IMAGE) --build-arg VITE_BACKEND_URL=$(VITE_BACKEND_URL) apps/portal-web

build-games-web:
	docker build -t $(GAMES_WEB_IMAGE) --build-arg VITE_BACKEND_URL=$(VITE_BACKEND_URL) apps/games-web

build-realtime-gateway:
	docker build -t $(REALTIME_GATEWAY_IMAGE) services/realtime-gateway

build-match-core:
	docker build -t $(MATCH_CORE_IMAGE) services/match-core

build-bot-engine:
	docker build -t $(BOT_ENGINE_IMAGE) services/bot-engine

docker-build: build-portal-web build-games-web build-realtime-gateway build-match-core build-bot-engine

apply:
	kubectl apply -k k8s/overlays/local

delete:
	kubectl delete -k k8s/overlays/local --ignore-not-found

dev:
	skaffold dev

status:
	kubectl get pods,svc,pvc,ingress -n $(NAMESPACE)

ingress:
	kubectl get ingress -n $(NAMESPACE)

logs-realtime-gateway:
	kubectl logs -n $(NAMESPACE) deploy/realtime-gateway -f

logs-match-core:
	kubectl logs -n $(NAMESPACE) deploy/match-core -f

logs-portal-web:
	kubectl logs -n $(NAMESPACE) deploy/portal-web -f

logs-games-web:
	kubectl logs -n $(NAMESPACE) deploy/games-web -f

logs-bot-engine:
	kubectl logs -n $(NAMESPACE) deploy/bot-engine -f
