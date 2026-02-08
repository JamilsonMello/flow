package infra

import (
	"database/sql"
	"flow-tool/examples/clean_architecture/domain"
	"flow-tool/pkg/flow"
	"fmt"
)

type FlowOrderObserver struct {
	client *flow.FlowClient
}

func NewFlowOrderObserver(db *sql.DB, serviceName string) (*FlowOrderObserver, error) {
	client, err := flow.NewClient(db, flow.FlowConfig{
		ServiceName:   serviceName,
		IsProduction:  false,
		MaxExecutions: 2,
	})
	if err != nil {
		return nil, err
	}
	return &FlowOrderObserver{client: client}, nil
}

func (o *FlowOrderObserver) OnOrderCreated(order domain.Order) {
	fmt.Printf("[Infra] Flow Adapter intercepting Order %s\n", order.ID)

	f, err := o.client.Start("Order Flow", order.ID)
	if err != nil {
		fmt.Printf("Error starting flow: %v\n", err)
		return
	}

	err = f.CreatePoint("Order Created", map[string]interface{}{
		"id":     order.ID,
		"amount": order.Amount,
		"status": order.Status,
	})
	if err != nil {
		fmt.Printf("Error creating point: %v\n", err)
	}
}
