package engine

import "math"

func GetNumberOfPages(total int, pageSize int) int {
	return int(math.Ceil(float64(total) / float64(pageSize)))
}

func SliceSearchResults(results []SearchResult, currentPage int) []SearchResult {
	total := len(results)
	numberOfPages := GetNumberOfPages(total, PageSize)
	low := (currentPage - 1) * PageSize
	high := currentPage * PageSize
	if currentPage > numberOfPages {
		return results[low:]
	}
	return results[low:high]
}