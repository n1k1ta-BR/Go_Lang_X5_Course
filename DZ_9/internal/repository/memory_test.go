package repository

import "testing"

func TestListReturnsSortedCopy(t *testing.T) {
	repo := New()

	for _, customer := range []string{"Алиса", "Борис", "Вера"} {
		if _, err := repo.Create(customer); err != nil {
			t.Fatal(err)
		}
	}

	orders, err := repo.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 3 {
		t.Fatalf("orders=%+v", orders)
	}

	for i, order := range orders {
		if order.ID != i+1 {
			t.Fatalf("orders[%d].ID=%d", i, order.ID)
		}
	}

	orders[0].Customer = "Изменённое имя"

	saved, err := repo.ByID(1)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Customer != "Алиса" {
		t.Fatalf("list changed stored order: %+v", saved)
	}
}
