package engine

func IndexExists(indexes []int, index int) bool {
	for i := range indexes {
		if indexes[i] == index {
			return true
		}
	}
	return false
}

func FindMax(frequency map[int]int) int {
	max := 0
	for i := range frequency {
		if frequency[i] > max {
			max = frequency[i]
		}
	}
	return max
}
