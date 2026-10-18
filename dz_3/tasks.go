package homework

// PairInput соответствует объекту nums/target из условия контеста.
type PairInput struct {
	Nums   []int
	Target int
}

// longestPrefix возвращает самый длинный общий префикс строк.
func longestPrefix(words []string) string {
	if len(words) == 0 {
		return ""
	}
	words_runes := [][]rune{}
	for _, word := range words {
		if len(word) == 0 {
			return ""
		}
		words_runes = append(words_runes, []rune(word))
	}

	for i := range len(words_runes[0]) {
		for j := 1; j < len(words_runes); j++ {
			if i >= len(words_runes[j]) || words_runes[j][i] != words_runes[0][i] {
				return string(words_runes[0][:i])
			}
		}
	}
	return words[0]

}

// pairIndices возвращает индексы двух разных элементов по возрастанию.
func pairIndices(input PairInput) [2]int {
	m := make(map[int]int, len(input.Nums))
	for i, num := range input.Nums {
		val := input.Target - num
		j, ok := m[val]
		if ok {
			return [2]int{j, i}
		}
		m[num] = i
	}
	return [2]int{-1, -1}
}

// hasDuplicate сообщает, встречается ли какое-либо значение повторно.
func hasDuplicate(nums []int) bool {
	m := make(map[int]int, len(nums))
	for i, val := range nums {
		_, ok := m[val]
		if ok {
			return true
		}
		m[val] = i
	}
	return false
}

// это решение с мапой, но можно и без нее (быстрее вроде как)
// func compactSorted(nums []int) []int {
// 	if len(nums) == 0 {
// 		return []int{}
// 	}
// 	m := make(map[int]int, len(nums))
// 	index := 0
// 	for _, val := range nums {
// 		_, ok := m[val]
// 		if !ok {
// 			m[val] = index
// 			index++
// 		}
// 	}
// 	res := make([]int, len(m))
// 	for val, index := range m {
// 		res[index] = val
// 	}
// 	return res
// }

func compactSorted(nums []int) []int {
	if len(nums) == 0 {
		return nil
	}
	res := []int{nums[0]} // первый элемент всегда уникальный

	for i, val := range nums[1:] {
		if nums[i] != val { //тут индекс i "спешит" из-за nums[1:]; таким образом мы сравниваем текущий элемент с предыдущим
			res = append(res, val)
		} // либо можно юзать дефолтный for (i := 1, i < ....), но так не интересно)
	}
	return res
}

// balancedBrackets проверяет пары и порядок закрытия скобок.
func balancedBrackets(text string) bool {

	if len(text)%2 != 0 {
		return false
	}

	brackets := map[rune]rune{
		')': '(',
		']': '[',
		'}': '{',
	}

	str := make([]rune, 0, len(text))

	for _, val := range text {
		switch val {
		case '(', '{', '[':
			str = append(str, val)

		case ')', '}', ']':
			if len(str) == 0 || str[len(str)-1] != brackets[val] {
				return false
			}
			str = str[:len(str)-1] //убираем открытую скобку (так как закрыли ее)
		}
	}
	return len(str) == 0 //если строка пустая - все скобки закрыты + проверка на входную ""
}
