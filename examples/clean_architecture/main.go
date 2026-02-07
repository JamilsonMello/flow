package main

import (
	"database/sql"
	"flow-tool/examples/clean_architecture/domain"
	"flow-tool/examples/clean_architecture/infra"
	"flow-tool/examples/clean_architecture/usecase"
	"fmt"

	_ "github.com/lib/pq"
)

type InMemoryRepo struct{}

func (r *InMemoryRepo) Save(o domain.Order) error { return nil }

func main() {
	connStr := "user=user password=password dbname=flow_db sslmode=disable host=127.0.0.1 port=5432"
	db, _ := sql.Open("postgres", connStr)

	flowObserver, _ := infra.NewFlowOrderObserver(db, "CleanOrderService")
	repo := &InMemoryRepo{}

	uc := usecase.NewCreateOrderUseCase(repo, flowObserver)

	fmt.Println("Running Clean Architecture Example...")
	uc.Execute("ORDER-CLEAN-999", 500.00)
}
