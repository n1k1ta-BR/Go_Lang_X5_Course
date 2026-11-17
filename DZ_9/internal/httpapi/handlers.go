package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"example.com/lesson9practice/internal/model"
	"example.com/lesson9practice/internal/repository"
	"example.com/lesson9practice/internal/service"
)

type Orders interface {
	Create(string) (model.Order, error)
	ByID(int) (model.Order, error)
	List() ([]model.Order, error)
	UpdateStatus(int, string) (model.Order, error)
}

type API struct {
	orders Orders
	logger *slog.Logger
}

func New(orders Orders, logger *slog.Logger) *API { return &API{orders: orders, logger: logger} }

// TODO: добавить документацию GET /status.
// @Summary Проверить состояние API
// @Tags status
// @Produce json
// @Success 200 {object} StatusResponse
// @Router /status [get]
func (a *API) status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, StatusResponse{Status: "ok"})
}

// Дополнительно: задокументировать POST /orders.
// @Summary Создать заказ
// @Tags orders
// @Accept json
// @Produce json
// @Param request body CreateOrderRequest true "Данные для создания заказа"
// @Success 201 {object} OrderResponse
// @Header 201 {string} Location "Путь к созданному заказу"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 413 {object} ErrorResponse "Слишком большое тело запроса"
// @Failure 415 {object} ErrorResponse "Неподдерживаемый тип содержимого"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /orders [post]
func (a *API) createOrder(w http.ResponseWriter, r *http.Request) {
	// Указатель позволяет отличить null от объекта. Теги DTO сами не валидируют запрос.
	var in *CreateOrderRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, err.status, err.message, err.code)
		return
	}
	if in == nil {
		writeError(w, 400, "body must be a JSON object", "INVALID_JSON")
		return
	}
	order, err := a.orders.Create(in.Customer)
	if errors.Is(err, service.ErrEmptyCustomer) {
		writeError(w, 400, "customer is required", "VALIDATION_ERROR")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/orders/"+strconv.Itoa(order.ID))
	writeJSON(w, 201, orderResponse(order))
}

// TODO: добавить документацию GET /orders/{id}.
// @Summary Получить заказ по id
// @Tags orders
// @Produce json
// @Param id path int true "Положительный номер заказа" minimum(1)
// @Success 200 {object} OrderResponse
// @Failure 400 {object} ErrorResponse "INVALID_ID"
// @Failure 404 {object} ErrorResponse "ORDER_NOT_FOUND"
// @Failure 500 {object} ErrorResponse "INTERNAL_ERROR"
// @Router /orders/{id} [get]
func (a *API) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, 400, "id must be a positive integer", "INVALID_ID")
		return
	}
	order, err := a.orders.ByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, 404, "order not found", "ORDER_NOT_FOUND")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, orderResponse(order))
}

func (a *API) internalError(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.ErrorContext(r.Context(), "order operation failed", "error", err)
	writeError(w, 500, "internal server error", "INTERNAL_ERROR")
}

// @Summary Получить список заказов
// @Description Возвращает страницу заказов по возрастанию id. Фильтр применяется ко всему списку до пагинации. Пустая страница возвращается как []. Новые заказы имеют статус new, поэтому status=cancelled возвращает пустой массив до первой отмены. Явные пустые, повторные и неизвестные query-параметры запрещены. limit и offset допускают только десятичные цифры в диапазоне int.
// @Tags orders
// @Produce json
// @Param limit query int false "Размер страницы" minimum(1) maximum(100) default(20)
// @Param offset query int false "Количество пропускаемых заказов" minimum(0) default(0)
// @Param status query string false "Фильтр по статусу" Enums(new,cancelled)
// @Success 200 {array} OrderResponse
// @Failure 400 {object} ErrorResponse "INVALID_QUERY"
// @Failure 500 {object} ErrorResponse "INTERNAL_ERROR"
// @Router /orders [get]
func (a *API) listOrders(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r.URL.RawQuery)
	if err != nil {
		writeError(w, 400, "invalid query parameters", "INVALID_QUERY")
		return
	}

	orders, err := a.orders.List()
	if err != nil {
		a.internalError(w, r, err)
		return
	}

	if query.status != "" {
		filtered := make([]model.Order, 0, len(orders))
		for _, order := range orders {
			if order.Status == query.status {
				filtered = append(filtered, order)
			}
		}
		orders = filtered
	}

	if query.offset >= len(orders) {
		writeJSON(w, 200, []OrderResponse{})
		return
	}

	page := orders[query.offset:]
	if len(page) > query.limit {
		page = page[:query.limit]
	}

	response := make([]OrderResponse, len(page))
	for i, order := range page {
		response[i] = orderResponse(order)
	}

	writeJSON(w, 200, response)
}

// @Summary Отменить заказ
// @Description Разрешён переход new в cancelled. Повторная отмена возвращает 409 ORDER_ALREADY_CANCELLED.
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int true "Положительный номер заказа" minimum(1)
// @Param request body UpdateStatusRequest true "Новый статус заказа"
// @Success 200 {object} OrderResponse
// @Failure 400 {object} ErrorResponse "INVALID_ID, INVALID_JSON или VALIDATION_ERROR"
// @Failure 404 {object} ErrorResponse "ORDER_NOT_FOUND"
// @Failure 409 {object} ErrorResponse "ORDER_ALREADY_CANCELLED"
// @Failure 413 {object} ErrorResponse "BODY_TOO_LARGE"
// @Failure 415 {object} ErrorResponse "UNSUPPORTED_MEDIA_TYPE"
// @Failure 500 {object} ErrorResponse "INTERNAL_ERROR"
// @Router /orders/{id}/status [patch]
func (a *API) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, 400, "id must be a positive integer", "INVALID_ID")
		return
	}

	var in *UpdateStatusRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, err.status, err.message, err.code)
		return
	}
	if in == nil {
		writeError(w, 400, "body must be a JSON object", "INVALID_JSON")
		return
	}

	order, err := a.orders.UpdateStatus(id, in.Status)

	switch {
	case errors.Is(err, service.ErrInvalidStatus):
		writeError(w, 400, "status must be cancelled", "VALIDATION_ERROR")

	case errors.Is(err, repository.ErrNotFound):
		writeError(w, 404, "order not found", "ORDER_NOT_FOUND")

	case errors.Is(err, repository.ErrAlreadyCancelled):
		writeError(w, 409, "order already cancelled", "ORDER_ALREADY_CANCELLED")

	case err != nil:
		a.internalError(w, r, err)

	default:
		writeJSON(w, 200, orderResponse(order))
	}
}
