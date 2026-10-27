package main

import (
	"errors"
	"fmt"

	"Notification_serv/practice"
)

func main() {
	email := &practice.EmailNotifier{
		BaseNotifier: practice.BaseNotifier{Channel: "email"},
		Address:      "LatyshevN@example.com",
	}

	push := &practice.PushNotifier{
		BaseNotifier: practice.BaseNotifier{Channel: "push"},
		DeviceID:     "device-1",
	}

	all := []practice.Notifier{email, push}

	// 1. Успешная рассылка.
	fmt.Println("1. Успешная рассылка")

	err := practice.Dispatch(all, "Заказ готов")
	if err != nil {
		fmt.Println("Ошибка:", err)
	} else {
		fmt.Println("Рассылка выполнена")
	}

	fmt.Println("История email:", email.Messages)
	fmt.Println("История push:", push.Messages)

	// 2. Пустой текст.
	fmt.Println("\n2. Пустой текст")

	err = practice.Dispatch(all, "")
	if errors.Is(err, practice.ErrEmptyMessage) {
		fmt.Println("пустое сообщение")
	} else if err != nil {
		fmt.Println("ошибка:", err)
	}

	fmt.Println("История email:", email.Messages)
	fmt.Println("История push:", push.Messages)

	// 3. Push без настройки.
	fmt.Println("\n3. Ошибка настройки")

	unconfigured := &practice.PushNotifier{
		BaseNotifier: practice.BaseNotifier{Channel: "push"},
	}

	err = practice.Dispatch(
		[]practice.Notifier{unconfigured},
		"Заказ готов",
	)

	var sendErr *practice.SendError
	if errors.As(err, &sendErr) {
		fmt.Println("Канал:", sendErr.Channel)
		fmt.Println("Причина:", sendErr.Reason)
	} else if err != nil {
		fmt.Println("ошибка:", err)
	}

	fmt.Println("История push без настройки:", unconfigured.Messages)

	// 4. Продолжение после паники.
	fmt.Println("\n4. Продолжение после паники")

	broken := &practice.BrokenNotifier{
		BaseNotifier: practice.BaseNotifier{Channel: "broken"},
	}

	nextPush := &practice.PushNotifier{
		BaseNotifier: practice.BaseNotifier{Channel: "push"},
		DeviceID:     "device-2",
	}

	errs := practice.SafeDispatch(
		[]practice.Notifier{broken, nextPush},
		"Заказ готов",
	)

	for _, err := range errs {
		var sendErr *practice.SendError

		switch {
		case errors.Is(err, practice.ErrNotifierPanic):
			fmt.Println("Паника:", err)
		case errors.Is(err, practice.ErrEmptyMessage):
			fmt.Println("Пустое сообщение")
		case errors.As(err, &sendErr):
			fmt.Printf("Ошибка настройки %s: %s\n",
				sendErr.Channel, sendErr.Reason)
		default:
			fmt.Println("Ошибка:", err)
		}
	}

	fmt.Println("История push после сломанного канала:", nextPush.Messages)
}
