package engine

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Processed struct {
	Time int64  `json:"time"`
	Unit string `json:"unit"`
}

type SearchResult struct {
	Url      string  `json:"url"`
	Rank     float64 `json:"rank"`
	Title    string  `json:"title"`
	Abstract string  `json:"abstract"`
}

type SearchResults struct {
	Processed Processed      `json:"processed"`
	Results   []SearchResult `json:"results"`
}

type WikiXMLDoc struct {
	Index    int    `xml:"index"`
	Title    string `xml:"title"`
	Url      string `xml:"url"`
	Abstract string `xml:"abstract"`
}

type WikiXMLDump struct {
	Documents []WikiXMLDoc `xml:"doc"`
}

type IndexerInterface interface {
	GetID() int
	DownloadWikimediaDump() error
	LoadWikimediaDump(path string, save bool) error
	LoadIndexDump(path string) error
	LoadDataDump(path string) error
	SaveIndexDump() error
	SaveDataDump() error
	IsIndexesDumped() bool
	IsDataDumped() bool
	Analyze(s string) []string
	AddIndex(tokens []string, index int)
	AddIndexesAsync(documents []WikiXMLDoc, wg *sync.WaitGroup)
	Search(s string) SearchResults
}

type Indexer struct {
	Data      map[int]WikiXMLDoc
	Indexes   map[string][]int
	Tokenizer *Tokenizer
	Filterer  *Filterer
	Stemmer   *Stemmer
	Mutex     sync.Mutex
}

func NewIndexer() (*Indexer, error) {
	filterer, err := NewFilterer()
	if err != nil {
		return nil, err
	}
	return &Indexer{
		Data:      map[int]WikiXMLDoc{},
		Indexes:   map[string][]int{},
		Tokenizer: NewTokenizer(),
		Filterer:  filterer,
		Stemmer:   NewStemmer(),
		Mutex:     sync.Mutex{},
	}, nil
}

func (i *Indexer) GetID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, _ := strconv.Atoi(idField)
	return id
}

func (i *Indexer) LoadWikimediaDump(path string, save bool) error {

	begin := time.Now()
	defer func(begin time.Time) {
		elapsed := time.Since(begin)
		fmt.Printf("Whole process took %f seconds\n", elapsed.Seconds())
	}(begin)

	t0 := time.Now()
	xmlFile, err := os.Open(path)
	if err != nil {
		return err
	}
	t1 := time.Since(t0).Seconds()
	fmt.Printf("Opening file took: %f seconds\n", t1)
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
		docs[idx].Index = idx
		i.Data[idx] = doc
	}
	dump.Documents = docs

	var chunks [][]WikiXMLDoc
	var wg sync.WaitGroup

	numberOfDocuments := len(dump.Documents)
	cores := runtime.NumCPU() * 2
	runtime.GOMAXPROCS(cores)
	chunkSize := (numberOfDocuments + cores - 1) / cores

	fmt.Printf("Number of documents: %d\n", numberOfDocuments)
	fmt.Printf("Number of cores: %d\n", cores)
	fmt.Printf("Chunk size: %d\n", chunkSize)

	for i := 0; i < numberOfDocuments; i += chunkSize {
		end := i + chunkSize
		if end > numberOfDocuments {
			end = numberOfDocuments
		}
		chunks = append(chunks, dump.Documents[i:end])
	}

	for _, docs := range chunks {
		wg.Add(1)
		go i.AddIndexesAsync(docs, &wg)
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

	var indexes map[string][]int

	err = json.Unmarshal(bytes, &indexes)
	if err != nil {
		return err
	}
	i.Indexes = indexes
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

	var data map[int]WikiXMLDoc

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
	file, err := json.Marshal(i.Indexes)
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

func (i *Indexer) AddIndex(tokens []string, index int) {
	for _, token := range tokens {
		i.Mutex.Lock()
		indexes, exists := i.Indexes[token]
		i.Mutex.Unlock()
		if exists {
			if !IndexExists(indexes, index) {
				indexes = append(indexes, index)
				i.Mutex.Lock()
				i.Indexes[token] = indexes
				i.Mutex.Unlock()
			}
		} else {
			i.Mutex.Lock()
			i.Indexes[token] = []int{index}
			i.Mutex.Unlock()
		}
	}
}

func (i *Indexer) Search(s string) SearchResults {
	begin := time.Now()

	results := []SearchResult{}
	frequency := map[int]int{}
	tokens := i.Analyze(s)
	for _, token := range tokens {
		indexes, exists := i.Indexes[token]
		if exists {
			for _, index := range indexes {
				value, ok := frequency[index]
				if ok {
					frequency[index] = value + 1
				} else {
					frequency[index] = 1
				}
			}
		}
	}
	max := FindMax(frequency)
	for index, freq := range frequency {
		rank := float64(freq) / float64(max)
		if rank == 1.0 {
			doc, ok := i.Data[index]
			if ok {
				results = append(results, SearchResult{
					Url:      doc.Url,
					Title:    doc.Title,
					Abstract: doc.Abstract,
					Rank:     rank,
				})
			}
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Rank > results[j].Rank
	})

	elapsed := time.Since(begin)
	microseconds := elapsed.Microseconds()
	milliseconds := elapsed.Milliseconds()

	if microseconds > 1000 {
		fmt.Printf("Search took %d milli seconds for phrase: %s\n", milliseconds, s)
		return SearchResults{
			Processed: Processed{
				Time: milliseconds,
				Unit: "milli seconds",
			},
			Results: results,
		}
	} else {
		fmt.Printf("Search took %d micro seconds for phrase: %s\n", microseconds, s)
		return SearchResults{
			Processed: Processed{
				Time: microseconds,
				Unit: "micro seconds",
			},
			Results: results,
		}
	}
}

func (i *Indexer) AddIndexesAsync(documents []WikiXMLDoc, wg *sync.WaitGroup) {
	defer wg.Done()
	id := i.GetID()
	for idx, doc := range documents {
		abstract := fmt.Sprintf("%s %s", doc.Title, doc.Abstract)
		tokens := i.Analyze(abstract)
		i.AddIndex(tokens, doc.Index)
		if idx%10000 == 0 {
			fmt.Printf("[%d] indexed %dK documents\n", id, idx/1000)
		}
	}
	fmt.Printf("[%d] indexed %d documents successfully. %d routines left\n", id, len(documents), runtime.NumGoroutine())
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
