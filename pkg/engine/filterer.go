package engine

import (
	"strings"
)

type FilterInterface interface {
	Lowercase(tokens []string) []string
	RemoveStopWords(tokens []string) []string
}

type Filterer struct {
	StopWords map[string]int
}

func NewFilterer() *Filterer {
	stopWords := map[string]int{
		"the":     1,
		"be":      2,
		"to":      3,
		"of":      4,
		"and":     5,
		"a":       6,
		"in":      7,
		"that":    8,
		"have":    9,
		"I":       10,
		"it":      11,
		"for":     12,
		"not":     13,
		"on":      14,
		"with":    15,
		"he":      16,
		"as":      17,
		"you":     18,
		"do":      19,
		"at":      20,
		"this":    21,
		"but":     22,
		"his":     23,
		"by":      24,
		"from":    25,
		"they":    26,
		"we":      27,
		"say":     28,
		"her":     29,
		"she":     30,
		"or":      31,
		"an":      32,
		"will":    33,
		"my":      34,
		"one":     35,
		"all":     36,
		"would":   37,
		"there":   38,
		"their":   39,
		"what":    40,
		"so":      41,
		"up":      42,
		"out":     43,
		"if":      44,
		"about":   45,
		"who":     46,
		"get":     47,
		"which":   48,
		"go":      49,
		"me":      50,
		"when":    51,
		"make":    52,
		"can":     53,
		"like":    54,
		"time":    55,
		"no":      56,
		"just":    57,
		"him":     58,
		"know":    59,
		"take":    60,
		"people":  61,
		"into":    62,
		"year":    63,
		"your":    64,
		"good":    65,
		"some":    66,
		"could":   67,
		"them":    68,
		"see":     69,
		"other":   70,
		"than":    71,
		"then":    72,
		"now":     73,
		"look":    74,
		"only":    75,
		"come":    76,
		"its":     77,
		"over":    78,
		"think":   79,
		"also":    80,
		"back":    81,
		"after":   82,
		"use":     83,
		"two":     84,
		"how":     85,
		"our":     86,
		"work":    87,
		"first":   88,
		"well":    89,
		"way":     90,
		"even":    91,
		"new":     92,
		"want":    93,
		"because": 94,
		"any":     95,
		"these":   96,
		"give":    97,
		"day":     98,
		"most":    99,
		"us":      100,
	}
	return &Filterer{StopWords: stopWords}
}

func (f *Filterer) Lowercase(tokens []string) []string {
	for idx := range tokens {
		token := tokens[idx]
		tokens[idx] = strings.ToLower(token)
	}
	return tokens
}

func (f *Filterer) RemoveStopWords(tokens []string) []string {
	newTokens := make([]string, 0, len(tokens))
	for idx := range tokens {
		token := tokens[idx]
		if _, exist := f.StopWords[token]; !exist {
			newTokens = append(newTokens, token)
		}
	}
	return newTokens
}
