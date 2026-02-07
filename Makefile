
run-a:
	go run cmd/service-a/main.go

run-a-conflict:
	go run cmd/service-a/main.go ORDER-CONFLICT-TEST

run-b:
	go run cmd/service-b/main.go

up:
	docker-compose up -d

down:
	docker-compose down

dashboard:
	go run cmd/dashboard/main.go
