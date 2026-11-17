package service

import (
	"errors"
	"fmt"
	"strings"

	"example.com/lesson9practice/internal/model"
)

var ErrEmptyCustomer = errors.New("customer is required")
var ErrInvalidStatus = errors.New("status must be cancelled")

type Repository interface {
	Create(string) (model.Order, error)
	ByID(int) (model.Order, error)
	List() ([]model.Order, error)
	Cancel(int) (model.Order, error)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Create(customer string) (model.Order, error) {
	customer = strings.TrimSpace(customer)
	if customer == "" {
		return model.Order{}, ErrEmptyCustomer
	}
	return s.repo.Create(customer)
}

func (s *Service) ByID(id int) (model.Order, error) { return s.repo.ByID(id) }

func (s *Service) List() ([]model.Order, error) {
	orders, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("получение списка заказов: %w", err)
	}

	return orders, nil
}

func (s *Service) UpdateStatus(id int, status string) (model.Order, error) {
	if status != "cancelled" {
		return model.Order{}, ErrInvalidStatus
	}

	order, err := s.repo.Cancel(id)
	if err != nil {
		return model.Order{}, fmt.Errorf("отмена заказа %d: %w", id, err)
	}

	return order, nil
}
