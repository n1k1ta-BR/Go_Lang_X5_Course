package handler

import "example.com/orders/internal/model"

type Service interface {
	Create(customer string) (model.Order, error)
}
