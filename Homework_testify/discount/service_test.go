package discount

import (
	"testing"

	"github.com/go-playground/assert"
	"github.com/stretchr/testify/require"
)

var tests_percent = []struct {
	name    string
	percent int
	want    error
}{
	{"Zero", 0, nil},
	{"Ten", 10, nil},
	{"Hundred", 100, nil},
	{"Negative", -1, ErrInvalidPercent},
	{"Overflow", 101, ErrInvalidPercent},
}

func TestValidatePercent(t *testing.T) {
	for _, tt := range tests_percent {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePercent(tt.percent)
			assert.Equal(t, tt.want, got)
		})
	}
}

var tests_apply = []struct {
	name         string
	price        int
	percent      int
	wanted_price int
	err          error
}{
	{"Default_discount", 100, 10, 90, nil},
	{"Zero_discount", 100, 0, 100, nil},
	{"100%_discount", 100, 100, 0, nil},
	{"Negative_price", -100, 10, 0, ErrNegativePrice},
	{"Negative_percent", 100, -3, 0, ErrInvalidPercent},
	{"Overflow_percent", 100, 101, 0, ErrInvalidPercent},
}

func TestApply(t *testing.T) {
	for _, tt := range tests_apply {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Apply(tt.price, tt.percent)
			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.wanted_price, got)
		})
	}
}

var tests_FinalPrice = []struct {
	name        string
	items       []Item
	total_price int
	err         error
}{
	{"Empty", []Item{}, 0, nil},
	{"Nil", nil, 0, nil},
	{"SeveralItems", []Item{{100, 10}, {200, 25}, {300, 0}}, 540, nil},
	{"FullDiscount", []Item{{100, 100}}, 0, nil},
	{"NegativePrice", []Item{{-100, 10}}, 0, ErrNegativePrice},
	{"NegativeDiscount", []Item{{100, -1}}, 0, ErrInvalidPercent},
	{"DiscountOverflow", []Item{{100, 101}}, 0, ErrInvalidPercent},
	{"InvalidItemAfterValid", []Item{{100, 10}, {200, 101}}, 0, ErrInvalidPercent},
}

func TestFinalPrice(t *testing.T) {
	for _, tt := range tests_FinalPrice {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FinalPrice(tt.items)
			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, got, tt.total_price)
		})
	}
}
