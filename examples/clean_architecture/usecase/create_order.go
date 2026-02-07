package usecase

import (
	"flow-tool/examples/clean_architecture/domain"
	"fmt"
)

type CreateOrderUseCase struct {
	repo     domain.OrderRepository
	observer domain.OrderObserver // Dependency Inversion
}

func NewCreateOrderUseCase(repo domain.OrderRepository, observer domain.OrderObserver) *CreateOrderUseCase {
	return &CreateOrderUseCase{repo: repo, observer: observer}
}

func (uc *CreateOrderUseCase) Execute(id string, amount float64) error {
	order := domain.Order{ID: id, Amount: amount, Status: "PENDING"}

	// 1. Business Logic
	fmt.Printf("[UseCase] Validating and saving order %s...\n", id)
	if err := uc.repo.Save(order); err != nil {
		return err
	}

	// 2. Notify Observers (Side Effects)
	// The UseCase doesn't know "Flow" exists. It just notifies "something happened".
	if uc.observer != nil {
		uc.observer.OnOrderCreated(order)
	}

	return nil
}
