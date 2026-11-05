postgres:
	docker run --name postgres16 -p 5433:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:16.13-alpine

mosquitto: 
	docker run --name mosquitto -p 1883:1883 -p 9001:9001 -d eclipse-mosquitto:2

createdb: 
	docker exec -it postgres16 createdb --username=root --owner=root ground_guard

dropdb:
	docker exec -it postgres16 dropdb ground_guard

migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5433/ground_guard?sslmode=disable" -verbose up

migrateup1:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5433/ground_guard?sslmode=disable" -verbose up 1

migratedown: 
	migrate -path db/migration -database "postgresql://root:secret@localhost:5433/ground_guard?sslmode=disable" -verbose down

migratedown1: 
	migrate -path db/migration -database "postgresql://root:secret@localhost:5433/ground_guard?sslmode=disable" -verbose down 1

sqlc: 
	sqlc generate

test:
	go test -v -cover ./...

# ---- test pyramid -----------------------------------------------------------
UNIT_PKGS = ./internal/... ./util/... ./mqtt/... ./token/... ./push/... ./websocket/...
# Coverage counts only hand-written code (not sqlc output, mocks or swagger docs).
COVER_PKGS = ./internal/...,./util/...,./mqtt/client/...,./mqtt/subscriber/...,./token/...,./push/...,./websocket/...
E2E_DB = postgresql://root:secret@localhost:5433/ground_guard_e2e?sslmode=disable

test-unit:
	go test -count=1 $(UNIT_PKGS)

# Real Postgres (dev database from app.env).
test-integration:
	go test -count=1 ./db/...

# Fresh database, every migration applied, real API + Postgres + MQTT broker.
e2e-db:
	docker exec postgres16 psql -U root -d postgres -c "DROP DATABASE IF EXISTS ground_guard_e2e" -c "CREATE DATABASE ground_guard_e2e OWNER root"
	migrate -path db/migration -database "$(E2E_DB)" up

e2e: e2e-db
	DB_SOURCE="$(E2E_DB)" MQTT_CLIENT_ID=ground-guard-e2e go test -tags=e2e -count=1 -v ./e2e/...

coverage:
	mkdir -p coverage
	go test -count=1 -coverpkg=$(COVER_PKGS) -coverprofile=coverage/unit.out ./... >/dev/null
	go tool cover -func=coverage/unit.out | tail -1
	go tool cover -html=coverage/unit.out -o coverage/index.html

# (Re)create the API account on the broker. Usage:
#   make mqtt-api-user MQTT_PASSWORD=<same value as MQTT_PASSWORD in app.env/.env>
# On a Linux server the files must be readable by the container's mosquitto user.
mqtt-api-user:
	@test -n "$(MQTT_PASSWORD)" || (echo "MQTT_PASSWORD is required" && exit 1)
	docker run --rm -e PW="$(MQTT_PASSWORD)" eclipse-mosquitto:2 sh -c 'mosquitto_passwd -b -c /tmp/passwd ground-guard-api "$$PW" >/dev/null 2>&1 && cat /tmp/passwd' > mqtt/passwd
	chmod 600 mqtt/passwd mqtt/acl.conf
	-chown 1883:1883 mqtt/passwd mqtt/acl.conf 2>/dev/null
	docker compose restart mosquitto

# Mutation testing: how many deliberately injected bugs the tests catch.
mutation:
	gremlins unleash --workers 4 --timeout-coefficient 3 ./internal/service
	gremlins unleash --workers 4 --timeout-coefficient 3 ./internal/middleware
	gremlins unleash --workers 4 --timeout-coefficient 3 ./util

server: 
	go run cmd/server/main.go

mock: 
	mockgen -package mockdb -destination db/mock/store.go github.com/Teixeiraass/ground_guard_be/db/sqlc Store
	mockgen -package mockmqtt -destination mqtt/mock/client.go github.com/Teixeiraass/ground_guard_be/mqtt/client Client

db_docs: 
	dbdocs build docs/db.dbml

db_schema: 
	dbml2sql --postgres -o docs/schema.sql docs/db.dbml

swag:
	swag init -g cmd/server/main.go

.PHONY: postgres createdb dropdb migrateup migrateup1 migratedown migratedown1 sqlc test server mock db_docs db_schema swag mosquitto test-unit test-integration e2e-db e2e coverage mutation mqtt-api-user
