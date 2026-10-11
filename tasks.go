package lesson2

// FizzBuzz возвращает FizzBuzz, Fizz, Buzz или пустую строку для обычного числа.
// Для n <= 0 возвращает invalid.
func FizzBuzz(n int) string {
	switch {
	case n <= 0:
		return "invalid"
	case n%3 == 0 && n%5 == 0:
		return "FizzBuzz"
	case n%3 == 0:
		return "Fizz"
	case n%5 == 0:
		return "Buzz"
	default:
		return ""
	}
}

// Sum возвращает сумму. Для пустого среза и nil возвращает 0.
func Sum(values []int64) int64 {
	if values == nil {
		return 0
	}
	var res int64
	for _, val := range values {
		res += val
	}
	return res
}

// Min возвращает минимум. Для пустого среза и nil возвращает 0 по условию задачи.
func Min(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, val := range values {
		if val < min {
			min = val
		}
	}
	return min
}

// Max возвращает максимум. Для пустого среза и nil возвращает 0 по условию задачи.
func Max(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, val := range values {
		if val > max {
			max = val
		}
	}
	return max
}

// Average возвращает среднее как float64. Для пустого среза и nil возвращает 0.
func Average(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	return float64(Sum(values)) / float64(int64(len(values)))
}

// Category возвращает invalid, small, regular или large по сумме в копейках.
func Category(amount int64) string {
	switch {
	case amount < 0:
		return "invalid"
	case 0 <= amount && amount <= 9999:
		return "small"
	case 10000 <= amount && amount < 50000:
		return "regular"
	case amount >= 50000:
		return "large"
	}
	return ""
}

// RuneCount считает кодовые точки Unicode в строке через range.
func RuneCount(text string) int {
	count := 0
	for range text {
		count++
	}
	return count
}
