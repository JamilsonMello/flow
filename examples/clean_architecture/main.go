package main

import (
	"database/sql"
	"flow-tool/examples/clean_architecture/domain"
	"flow-tool/examples/clean_architecture/infra"
	"flow-tool/examples/clean_architecture/usecase"
	"fmt"

	_ "github.com/lib/pq"
)

// Mock Repo
type InMemoryRepo struct{}

func (r *InMemoryRepo) Save(o domain.Order) error { return nil }

func main() {
	// 1. Setup Infrastructure
	connStr := "user=user password=password dbname=flow_db sslmode=disable host=127.0.0.1 port=5432"
	db, _ := sql.Open("postgres", connStr)

	// 2. Create Adapters (Infrastructure Layer)
	flowObserver, _ := infra.NewFlowOrderObserver(db, "CleanOrderService")
	repo := &InMemoryRepo{}

	// 3. Inject Dependencies into UseCase (Domain Layer)
	// NOTICE: UseCase receives 'flowObserver' but sees it only as 'domain.OrderObserver'
	uc := usecase.NewCreateOrderUseCase(repo, flowObserver)

	// 4. Run Application
	fmt.Println("Running Clean Architecture Example...")
	uc.Execute("ORDER-CLEAN-999", 500.00)
}
