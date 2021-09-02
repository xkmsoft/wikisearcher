package engine

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/RoaringBitmap/roaring"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

type Processed struct {
	Duration float64 `json:"time"`
	Unit     string  `json:"unit"`
}

type SearchResult struct {
	Url      string  `json:"url"`
	Rank     float64 `json:"rank"`
	Title    string  `json:"title"`
	Abstract string  `json:"abstract"`
}

type SearchResults struct {
	Processed       Processed      `json:"processed"`
	NumberOfResults int            `json:"number_of_results"`
	Results         []SearchResult `json:"results"`
}

type WikiXMLDoc struct {
	Index    uint32 `xml:"index"`
	Title    string `xml:"title"`
	Url      string `xml:"url"`
	Abstract string `xml:"abstract"`
}

type WikiXMLDump struct {
	Documents []WikiXMLDoc `xml:"doc"`
}

type IndexerInterface interface {
	DownloadWikimediaDump() error
	LoadWikimediaDump(path string, save bool) error
	LoadIndexDump(path string) error
	LoadDataDump(path string) error
	SaveIndexDump() error
	SaveDataDump() error
	IsIndexesDumped() bool
	IsDataDumped() bool
	Analyze(s string) []string
	AddIndex(tokens []string, index uint32)
	AddIndexesAsync(documents []WikiXMLDoc, wg *sync.WaitGroup)
	Search(s string) SearchResults
}

type Indexer struct {
	Data       map[uint32]WikiXMLDoc
	Indexes    map[string]*roaring.Bitmap
	Tokenizer  *Tokenizer
	Filterer   *Filterer
	Stemmer    *Stemmer
	Mutex      sync.Mutex
	Cores      int
	Multiplier int
}

func NewIndexer() (*Indexer, error) {
	filterer, err := NewFilterer()
	if err != nil {
		return nil, err
	}
	return &Indexer{
		Data:       map[uint32]WikiXMLDoc{},
		Indexes:    map[string]*roaring.Bitmap{},
		Tokenizer:  NewTokenizer(),
		Filterer:   filterer,
		Stemmer:    NewStemmer(),
		Mutex:      sync.Mutex{},
		Cores:      runtime.NumCPU(),
		Multiplier: 2,
	}, nil
}

func (i *Indexer) LoadWikimediaDump(path string, save bool) error {

	begin := time.Now()
	defer func(begin time.Time) {
		elapsed := time.Since(begin)
		fmt.Printf("Whole process took %f seconds\n", elapsed.Seconds())
	}(begin)

	xmlFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(xmlFile *os.File) {
		err := xmlFile.Close()
		if err != nil {
			fmt.Printf("Closing xml file failed: %s\n", err.Error())
		}
	}(xmlFile)

	t2 := time.Now()
	buffer := bufio.NewReaderSize(xmlFile, 1024*1024*2)
	decoder := xml.NewDecoder(buffer)

	var dump WikiXMLDump

	if err := decoder.Decode(&dump); err != nil {
		return err
	}
	t3 := time.Since(t2).Seconds()
	fmt.Printf("Decoding file took %f seconds\n", t3)
	t4 := time.Now()

	docs := dump.Documents
	for idx, doc := range docs {
		docs[idx].Index = uint32(idx)
		i.Data[uint32(idx)] = doc
	}
	dump.Documents = docs

	var chunks [][]WikiXMLDoc
	var wg sync.WaitGroup

	numberOfDocuments := len(dump.Documents)
	workers := i.Cores * i.Multiplier
	runtime.GOMAXPROCS(workers)
	chunkSize := (numberOfDocuments + workers - 1) / workers

	for i := 0; i < numberOfDocuments; i += chunkSize {
		end := i + chunkSize
		if end > numberOfDocuments {
			end = numberOfDocuments
		}
		chunks = append(chunks, dump.Documents[i:end])
	}

	for idx := range chunks {
		wg.Add(1)
		go i.AddIndexesAsync(chunks[idx], &wg)
	}

	wg.Wait()
	t5 := time.Since(t4).Seconds()

	fmt.Printf("Indexing documents took %f seconds\n", t5)

	if save {
		// Saving concurrently the index and data dump into files
		ops := 2
		done := make(chan bool)
		errors := make(chan error)
		go func() {
			err := i.SaveIndexDump()
			if err != nil {
				errors <- err
			}
			done <- true
		}()
		go func() {
			err := i.SaveDataDump()
			if err != nil {
				errors <- err
			}
			done <- true
		}()
		count := 0
		for {
			select {
			case err := <-errors:
				return err
			case <-done:
				count++
				if count == ops {
					return nil
				}
			}
		}
	}
	return nil
}

func (i *Indexer) LoadIndexDump(path string) error {
	fmt.Printf("Loading indexes dump from data...\n")
	begin := time.Now()
	defer func(begin time.Time) {
		elapsed := time.Since(begin)
		fmt.Printf("Loading indexes dump took %f seconds\n", elapsed.Seconds())
	}(begin)

	jsonFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			fmt.Printf("Error closing json file: %s\n", err.Error())
		}
	}(jsonFile)

	bytes, _ := ioutil.ReadAll(jsonFile)

	var indexes map[string][]uint32

	err = json.Unmarshal(bytes, &indexes)
	if err != nil {
		return err
	}
	for token, idx := range indexes {
		i.Indexes[token] = roaring.BitmapOf(idx...)
	}
	fmt.Printf("Indexes loaded from the dump successfully\n")
	return nil
}

func (i *Indexer) LoadDataDump(path string) error {
	fmt.Printf("Loading data dump from data...\n")
	begin := time.Now()
	defer func(begin time.Time) {
		elapsed := time.Since(begin)
		fmt.Printf("Loading data dump took %f seconds\n", elapsed.Seconds())
	}(begin)

	jsonFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			fmt.Printf("Error closing json file: %s\n", err.Error())
		}
	}(jsonFile)

	bytes, _ := ioutil.ReadAll(jsonFile)

	var data map[uint32]WikiXMLDoc

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return err
	}
	i.Data = data
	fmt.Printf("Data dump loaded from the dump file successfully\n")
	return nil
}

func (i *Indexer) IsIndexesDumped() bool {
	if _, err := os.Stat("./data/indexes.json"); os.IsNotExist(err) {
		return false
	}
	return true
}

func (i *Indexer) IsDataDumped() bool {
	if _, err := os.Stat("./data/data.json"); os.IsNotExist(err) {
		return false
	}
	return true
}

func (i *Indexer) SaveIndexDump() error {
	begin := time.Now()
	defer func(begin time.Time) {
		elapsed := time.Since(begin)
		fmt.Printf("Saving indexes dump into the file took %f seconds\n", elapsed.Seconds())
	}(begin)

	indexes := map[string][]uint32{}
	for token, idx := range i.Indexes {
		indexes[token] = idx.ToArray()
	}
	file, err := json.Marshal(indexes)
	if err != nil {
		fmt.Printf("Error marshalling to json the results: %s\n", err.Error())
		return err
	}
	err = ioutil.WriteFile("./data/indexes.json", file, 0644)
	if err != nil {
		fmt.Printf("Error saving the indexes dump into the file: %s\n", err.Error())
		return err
	}
	fmt.Printf("Indexes dump saved successfully into the file\n")
	return nil
}

func (i *Indexer) SaveDataDump() error {
	begin := time.Now()
	defer func(begin time.Time) {
		elapsed := time.Since(begin)
		fmt.Printf("Saving data dump into the file took %f seconds\n", elapsed.Seconds())
	}(begin)
	file, err := json.Marshal(i.Data)
	if err != nil {
		fmt.Printf("Error marshalling to json the data: %s\n", err.Error())
		return err
	}
	err = ioutil.WriteFile("./data/data.json", file, 0644)
	if err != nil {
		fmt.Printf("Error saving the data dump into the file: %s\n", err.Error())
		return err
	}
	fmt.Printf("Data dump saved successfully into the file\n")
	return nil
}

func (i *Indexer) Analyze(s string) []string {
	tokens := i.Tokenizer.Tokenize(s)
	tokens = i.Filterer.Lowercase(tokens)
	tokens = i.Filterer.RemoveStopWords(tokens)
	tokens = i.Stemmer.Stem(tokens)
	return tokens
}

func (i *Indexer) AddIndex(tokens []string, index uint32) {
	for idx := range tokens {
		token := tokens[idx]
		i.Mutex.Lock()
		indexes, exists := i.Indexes[token]
		if exists {
			if !indexes.Contains(index) {
				indexes.Add(index)
				i.Indexes[token] = indexes
			}
		} else {
			i.Indexes[token] = roaring.BitmapOf(index)
		}
		i.Mutex.Unlock()
	}
}

func (i *Indexer) Search(s string) SearchResults {
	begin := time.Now()
	searchResults := make([]SearchResult, 0)
	bitmapResults := roaring.BitmapOf()
	tokens := i.Analyze(s)

	for idx := range tokens {
		token := tokens[idx]
		if indexes, exists := i.Indexes[token]; exists {
			if bitmapResults.IsEmpty() {
				bitmapResults = indexes.Clone()
			}
			// Parallel ANDing to find the intersection
			bitmapResults = roaring.ParAnd(i.Cores, bitmapResults, indexes)
		}
	}

	for _, index := range bitmapResults.ToArray() {
		if doc, ok := i.Data[index]; ok {
			searchResults = append(searchResults, SearchResult{
				Url:      doc.Url,
				Rank:     1,
				Title:    doc.Title,
				Abstract: doc.Abstract,
			})
		}
	}

	elapsed := time.Since(begin)
	microseconds := elapsed.Microseconds()
	milliseconds := elapsed.Milliseconds()

	var duration float64
	if microseconds > 1000 {
		duration = float64(milliseconds)
	} else {
		duration = float64(microseconds) / 1000.0
	}

	fmt.Printf("Search took %f milli seconds for phrase: %s Number of results: %d\n", duration, s, len(searchResults))
	return SearchResults{
		Processed: Processed{
			Duration: duration,
			Unit:     "milli seconds",
		},
		NumberOfResults: len(searchResults),
		Results:         searchResults,
	}
}

func (i *Indexer) AddIndexesAsync(documents []WikiXMLDoc, wg *sync.WaitGroup) {
	defer wg.Done()
	for idx := range documents {
		doc := documents[idx]
		tokens := i.Analyze(fmt.Sprintf("%s %s", doc.Title, doc.Abstract))
		i.AddIndex(tokens, doc.Index)
	}
}

func (i *Indexer) DownloadWikimediaDump() error {

	filePath := "./data/enwiki-latest-abstract1.xml.gz"
	url := "https://dumps.wikimedia.org/enwiki/latest/enwiki-latest-abstract1.xml.gz"

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			fmt.Printf("Error closing file: %s\n", err.Error())
		}
	}(out)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closng get body %s\n", err.Error())
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return err
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
