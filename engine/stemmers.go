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
	for _, token := range tokens {
		stemmed, err := snowball.Stem(token, "english", false)
		if err == nil {
			newTokens = append(newTokens, stemmed)
		}
	}
	return newTokens
}