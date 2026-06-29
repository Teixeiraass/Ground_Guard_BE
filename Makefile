postgres:
	docker run --name postgres16 -p 5433:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:16.13-alpine

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

server: 
	go run cmd/server/main.go

mock: 
	mockgen -package mockdb -destination db/mock/store.go github.com/Teixeiraass/ground_guard_be/db/sqlc Store
	mockgen -package mockmqtt -destination mqtt/mock/client.go github.com/Teixeiraass/ground_guard_be/mqtt Client

db_docs: 
	dbdocs build docs/db.dbml

db_schema: 
	dbml2sql --postgres -o docs/schema.sql docs/db.dbml

swag:
	swag init -g cmd/server/main.go

.PHONY: postgres createdb dropdb migrateup migrateup1 migratedown migratedown1 sqlc test server mock