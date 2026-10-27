package handler

import (
	"example.com/orders/internal/model"
)

type Handler struct {
	orders Service
}

func NewHandler(orders Service) *Handler {
	return &Handler{orders: orders}
}

func (h *Handler) Create(customer string) (model.Order, error) {
	return h.orders.Create(customer)
}
