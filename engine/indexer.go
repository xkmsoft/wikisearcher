package engine

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/RoaringBitmap/roaring"
	"github.com/tamerh/xml-stream-parser"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	XmlStreamBufferSize = 1024 * 1024 * 1 // 1MB
	DocumentCapacity    = 524288          // 2^19
	PageSize            = 25
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
	CurrentPage     int            `json:"current_page"`
	NumberOfPages   int            `json:"number_of_pages"`
	Results         []SearchResult `json:"results"`
}

type WikiXMLDoc struct {
	Index    uint32 `xml:"index" json:"index"`
	Title    string `xml:"title" json:"title"`
	Url      string `xml:"url" json:"url"`
	Abstract string `xml:"abstract" json:"abstract"`
}

type IndexerInterface interface {
	DownloadWikimediaDump(path string, url string) error
	UncompressWikimediaDump(path string) error
	LoadWikimediaDump(path string, save bool, indexPath string, dataPath string) error
	LoadIndexDump(path string) error
	LoadDataDump(path string) error
	SaveIndexDump(path string) error
	SaveDataDump(path string) error
	IsFileExists(path string) bool
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

func (i *Indexer) LoadWikimediaDump(path string, save bool, indexPath string, dataPath string) error {

	t0 := time.Now()
	defer func(t0 time.Time) {
		fmt.Printf("Whole process took %f seconds\n", time.Since(t0).Seconds())
	}(t0)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			fmt.Printf("Closing xml file failed: %s\n", err.Error())
		}
	}(f)

	// Phase 1: Parsing the XML file
	t1 := time.Now()
	buffer := bufio.NewReaderSize(f, XmlStreamBufferSize)
	parser := xmlparser.NewXMLParser(buffer, "doc")
	documents := make([]WikiXMLDoc, 0, DocumentCapacity)
	index := uint32(0)

	for xmlElement := range parser.Stream() {
		if xmlElement.Name == "doc" {
			doc := WikiXMLDoc{
				Index:    index,
				Title:    xmlElement.Childs["title"][0].InnerText,
				Url:      xmlElement.Childs["url"][0].InnerText,
				Abstract: xmlElement.Childs["abstract"][0].InnerText,
			}
			documents = append(documents, doc)
			i.Data[index] = doc
			index++
		}
	}
	fmt.Printf("Parsing XML file took %f seconds\n", time.Since(t1).Seconds())

	// Phase 2: Creating indexes concurrently
	t2 := time.Now()
	var chunks [][]WikiXMLDoc
	var wg sync.WaitGroup

	numberOfDocuments := len(documents)
	fmt.Printf("There are %d documents in the file %s\n", numberOfDocuments, path)

	workers := i.Cores * i.Multiplier
	runtime.GOMAXPROCS(workers)
	chunkSize := (numberOfDocuments + workers - 1) / workers

	for i := 0; i < numberOfDocuments; i += chunkSize {
		end := i + chunkSize
		if end > numberOfDocuments {
			end = numberOfDocuments
		}
		chunks = append(chunks, documents[i:end])
	}
	wg.Add(len(chunks))
	for idx := range chunks {
		go i.AddIndexesAsync(chunks[idx], &wg)
	}
	wg.Wait()
	fmt.Printf("Indexing documents took %f seconds\n", time.Since(t2).Seconds())

	if save {
		// Phase 3: Saving concurrently the index and data dump into files
		workers := 2
		done := make(chan bool)
		errors := make(chan error)

		go func() {
			if err := i.SaveIndexDump(indexPath); err != nil {
				errors <- err
			} else {
				done <- true
			}
		}()

		go func() {
			if err := i.SaveDataDump(dataPath); err != nil {
				errors <- err
			} else {
				done <- true
			}
		}()

		count := 0
		for {
			select {
			case err := <-errors:
				return err
			case <-done:
				count++
				if count == workers {
					return nil
				}
			}
		}
	}
	return nil
}

func (i *Indexer) LoadIndexDump(path string) error {
	t0 := time.Now()
	defer func(t0 time.Time) {
		fmt.Printf("Loading indexes dump took %f seconds\n", time.Since(t0).Seconds())
	}(t0)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			fmt.Printf("Error closing json file: %s\n", err.Error())
		}
	}(f)

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	var indexes map[string][]uint32
	if err = json.Unmarshal(bytes, &indexes); err != nil {
		return err
	}

	for token, idx := range indexes {
		i.Indexes[token] = roaring.BitmapOf(idx...)
	}
	return nil
}

func (i *Indexer) LoadDataDump(path string) error {
	t0 := time.Now()
	defer func(t0 time.Time) {
		fmt.Printf("Loading data dump took %f seconds\n", time.Since(t0).Seconds())
	}(t0)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			fmt.Printf("Error closing json file: %s\n", err.Error())
		}
	}(f)

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	var data map[uint32]WikiXMLDoc
	if err = json.Unmarshal(bytes, &data); err != nil {
		return err
	}
	i.Data = data
	return nil
}

func (i *Indexer) IsFileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func (i *Indexer) SaveIndexDump(path string) error {
	t0 := time.Now()
	defer func(t0 time.Time) {
		fmt.Printf("Saving indexes dump into the file took %f seconds\n", time.Since(t0).Seconds())
	}(t0)

	indexes := make(map[string][]uint32, 0)
	for token, idx := range i.Indexes {
		indexes[token] = idx.ToArray()
	}

	bytes, err := json.Marshal(&indexes)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(path, bytes, 0644); err != nil {
		return err
	}
	return nil
}

func (i *Indexer) SaveDataDump(path string) error {
	t0 := time.Now()
	defer func(t0 time.Time) {
		fmt.Printf("Saving data dump into the file took %f seconds\n", time.Since(t0).Seconds())
	}(t0)

	bytes, err := json.Marshal(&i.Data)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(path, bytes, 0644); err != nil {
		return err
	}
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
		if indexes, exists := i.Indexes[token]; exists {
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

func (i *Indexer) Search(s string, page uint32) SearchResults {
	t0 := time.Now()

	searchResults := make([]SearchResult, 0, int(math.Pow(2, 8)))
	rb := roaring.NewBitmap()
	tokens := i.Analyze(s)

	for idx := range tokens {
		token := tokens[idx]
		if indexes, exists := i.Indexes[token]; exists {
			if rb.IsEmpty() {
				rb = indexes.Clone()
			}
			// Parallel ANDing to find the intersection
			rb = roaring.ParAnd(i.Cores, rb, indexes)
		}
	}

	for _, index := range rb.ToArray() {
		if doc, ok := i.Data[index]; ok {
			searchResults = append(searchResults, SearchResult{
				Url:      doc.Url,
				Rank:     1,
				Title:    doc.Title,
				Abstract: doc.Abstract,
			})
		}
	}

	totalResults := len(searchResults)
	numberOfPages := GetNumberOfPages(totalResults, PageSize)
	paginationResults := SliceSearchResults(searchResults, int(page))
	if int(page) > numberOfPages {
		page = uint32(numberOfPages)
	}

	var duration float64
	elapsed := time.Since(t0)
	microseconds := elapsed.Microseconds()
	milliseconds := elapsed.Milliseconds()

	if microseconds > 1000 {
		duration = float64(milliseconds)
	} else {
		duration = float64(microseconds) / 1000.0
	}

	fmt.Printf("%d results returned out of (%d documents) in %f milliseconds for phrase: %s\n", len(paginationResults), len(searchResults), duration, s)
	return SearchResults{
		Processed: Processed{
			Duration: duration,
			Unit:     "milliseconds",
		},
		NumberOfResults: len(searchResults),
		Results:         paginationResults,
		CurrentPage:     int(page),
		NumberOfPages:   numberOfPages,
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

func (i *Indexer) DownloadWikimediaDump(path string, url string) error {
	t0 := time.Now()
	defer func(t0 time.Time) {
		fmt.Printf("Downloading wikimedia dump on %s took %f seconds\n", url, time.Since(t0).Seconds())
	}(t0)

	fmt.Printf("Downloading the file from %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func(b io.ReadCloser) {
		if err := b.Close(); err != nil {
			fmt.Printf("Error closng get body %s\n", err.Error())
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err.Error())
		}
	}(f)

	if _, err = io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func (i *Indexer) UncompressWikimediaDump(path string) error {
	t0 := time.Now()
	defer func(t0 time.Time) {
		fmt.Printf("Uncompressing the file took %f seconds\n", time.Since(t0).Seconds())
	}(t0)

	fmt.Printf("Uncompressing the file: %s\n", path)

	f, err := os.Open(path)
	if err != nil {
		return err
	}

	r, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func(r *gzip.Reader) {
		if err := r.Close(); err != nil {
			fmt.Printf("Error closing reader: %s\n", err.Error())
		}
	}(r)
	dir, file := filepath.Split(path)
	out, err := os.Create(filepath.Join(dir, strings.TrimSuffix(file, ".gz")))
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		if err := out.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err.Error())
		}
	}(out)

	if _, err = io.Copy(out, r); err != nil {
		return err
	}
	return nil
}
