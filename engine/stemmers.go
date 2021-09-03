package engine

import "github.com/kljensen/snowball"

type StemmerInterface interface {
	Stem(tokens []string) []string
}

type Stemmer struct {}

func NewStemmer() *Stemmer {
	return &Stemmer{}
}

func (s *Stemmer) Stem(tokens []string) []string {
	newTokens := make([]string, 0, len(tokens))
	for idx := range tokens {
		token := tokens[idx]
		if stemmed, err := snowball.Stem(token, "english", false); err == nil {
			newTokens = append(newTokens, stemmed)
		}
	}
	return newTokens
}
