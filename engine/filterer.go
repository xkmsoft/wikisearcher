package engine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	DataDirectory = "data"
	StopWords     = "stop_words.json"
	StopWordsSize = 100
)

type FilterInterface interface {
	Lowercase(tokens []string) []string
	RemoveStopWords(tokens []string) []string
}

type StopWord struct {
	Rank int    `json:"rank"`
	Word string `json:"word"`
}

type Filterer struct {
	StopWords map[string]int
}

func NewFilterer() (*Filterer, error) {
	f, err := os.Open(filepath.Join(DataDirectory, StopWords))
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			fmt.Printf("Error closing json file: %s\n", err.Error())
		}
	}(f)

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var words = make([]StopWord, 0, StopWordsSize)
	var stopWords = make(map[string]int, 0)

	if err = json.Unmarshal(bytes, &words); err != nil {
		return nil, err
	}
	for idx := range words {
		w := words[idx]
		stopWords[w.Word] = w.Rank
	}
	return &Filterer{
		StopWords: stopWords,
	}, nil
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
