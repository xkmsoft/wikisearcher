package engine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
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
	jsonFile, err := os.Open("./data/stop_words.json")
	if err != nil {
		return nil, err
	}
	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			fmt.Printf("Error closing json file: %s\n", err.Error())
		}
	}(jsonFile)

	bytes, _ := ioutil.ReadAll(jsonFile)

	var words = []StopWord{}
	var stopWords = map[string]int{}

	err = json.Unmarshal(bytes, &words)
	if err != nil {
		return nil, err
	}
	for _, w := range words {
		stopWords[w.Word] = w.Rank
	}
	return &Filterer{ StopWords: stopWords}, nil
}

func (f *Filterer) Lowercase(tokens []string) []string {
	for i, token := range tokens {
		tokens[i] = strings.ToLower(token)
	}
	return tokens
}

func (f *Filterer) RemoveStopWords(tokens []string) []string {
	newTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, exist := f.StopWords[token]; !exist {
			newTokens = append(newTokens, token)
		}
	}
	return newTokens
}