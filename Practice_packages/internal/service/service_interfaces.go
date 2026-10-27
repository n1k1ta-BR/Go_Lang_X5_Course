package service

import "example.com/orders/internal/model"

type Repository interface {
	Save(order model.Order) error
	ByID(id int) (model.Order, error)
}
