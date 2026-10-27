package practice

import (
	"errors"
	"fmt"
)

var ErrEmptyMessage = errors.New("пустое сообщение")

type namedNotifier interface {
	Name() string
}

func channelName(n Notifier) string {
	if named, ok := n.(namedNotifier); ok {
		return named.Name()
	}
	return fmt.Sprintf("%T", n)
}

type SendError struct {
	Channel string
	Reason  string
}

type Notifier interface {
	Send(message string) error
}

type BaseNotifier struct {
	Channel string
}

func (b BaseNotifier) Name() string {
	return b.Channel
}

func (b BaseNotifier) Label(message string) string {
	return "[" + b.Channel + "] " + message
}

type EmailNotifier struct {
	BaseNotifier
	Address  string
	Messages []string
}

type PushNotifier struct {
	BaseNotifier
	DeviceID string
	Messages []string
}

func (n *EmailNotifier) Send(message string) error {
	if message == "" {
		return ErrEmptyMessage
	}
	if n.Address == "" {
		return &SendError{Channel: n.Channel, Reason: "не задан адрес"}
	}
	n.Messages = append(n.Messages, n.Label(message))
	return nil
}

func (n *PushNotifier) Send(message string) error {
	if message == "" {
		return ErrEmptyMessage
	}
	if n.DeviceID == "" {
		return &SendError{Channel: n.Channel, Reason: "не задан device id"}
	}
	n.Messages = append(n.Messages, n.Label(message))
	return nil
}

func Dispatch(all []Notifier, message string) error {
	for _, n := range all {
		err := n.Send(message)
		if err != nil {
			return fmt.Errorf("канал %s: %w", channelName(n), err)
		}
	}
	return nil
}

func EmailAddress(n Notifier) (string, bool) {
	email, ok := n.(*EmailNotifier)
	if !ok || email == nil {
		return "", false
	}

	return email.Address, true
}

func Describe(n Notifier) string {
	switch v := n.(type) {
	case *EmailNotifier:
		if v == nil {
			return "unknown"
		}
		return "email: " + v.Address
	case *PushNotifier:
		if v == nil {
			return "unknown"
		}
		return "push: " + v.DeviceID
	default:
		return "unknown"
	}
}

func (e *SendError) Error() string {
	return fmt.Sprintf("канал %s: %s", e.Channel, e.Reason)
}

type BrokenNotifier struct {
	BaseNotifier
}

func (n *BrokenNotifier) Send(message string) error {
	panic("драйвер канала повреждён")
}

var ErrNotifierPanic = errors.New("паника канала")

func SafeSend(n Notifier, message string) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("%w: %v", ErrNotifierPanic, value)
		}
	}()
	return n.Send(message)
}

func SafeDispatch(all []Notifier, message string) []error {
	var errs []error
	for _, n := range all {
		err := SafeSend(n, message)
		if err != nil {
			errs = append(errs, fmt.Errorf("канал %s: %w", channelName(n), err))
		}
	}
	return errs
}
