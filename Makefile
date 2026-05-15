include .env
export

migrate-up:
	migrate -path ./cmd/migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@localhost:5433/$(DB_NAME)?sslmode=disable" up

migrate-down:
	migrate -path ./cmd/migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@localhost:5433/$(DB_NAME)?sslmode=disable" down
