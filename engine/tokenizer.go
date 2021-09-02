package engine

import (
	"strings"
	"unicode"
)

type TokenizerInterface interface {
	Tokenize(s string) []string
}

type Tokenizer struct {}

func NewTokenizer() *Tokenizer {
	return &Tokenizer{}
}

func (t *Tokenizer) Tokenize(s string) []string {
	tokens := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	return tokens
}