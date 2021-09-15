package tcpserver

import (
	"errors"
	"fmt"
	"github.com/xkmsoft/wikisearcher/pkg/engine"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	QUERY = byte(0)
)

const (
	DataDirectory      = "data"
	BaseIndexes        = "indexes%s.json"
	BaseData           = "data%s.json"
	BaseFile           = "enwiki-latest-abstract%s.%s"
	BaseURL            = "https://dumps.wikimedia.org/enwiki/latest/enwiki-latest-abstract%s.xml.gz"
	XMLExtension       = "xml"
	GZExtension        = "xml.gz"
	AbstractFilesCount = 28
)


type ServerInterface interface {
	Address() string
	Signature() string
	InitializeServer() error
	HandleRequest(connection net.Conn)
	HandleResponse(response string, connection net.Conn)
	ParseQuery(query []byte) (*QueryStruct, error)
	AcceptConnections() error
	GetAbstractStruct() *AbstractStruct
	InitializeDataDirectory() error
}

type AbstractStruct struct {
	XMLFileName string
	GZFileName  string
	DataDump    string
	IndexDump   string
	URL         string
}

type Server struct {
	Host       string
	Port       string
	Network    string
	Indexer    *engine.Indexer
	QuitSignal bool
	Abstracts  []*AbstractStruct
	FileIndex  int
	CleanFlag  bool
}

type QueryStruct struct {
	command byte
	page uint32
	phrase string
}

func NewServer(host string, port string, network string, index int, clean bool) *Server {
	abstracts := make([]*AbstractStruct, AbstractFilesCount)
	for i := 0; i < AbstractFilesCount; i++ {
		var index string
		if i == 0 {
			index = ""
		} else {
			index = strconv.Itoa(i)
		}
		abstracts[i] = &AbstractStruct{
			XMLFileName: filepath.Join(DataDirectory, fmt.Sprintf(BaseFile, index, XMLExtension)),
			GZFileName:  filepath.Join(DataDirectory, fmt.Sprintf(BaseFile, index, GZExtension)),
			DataDump:    filepath.Join(DataDirectory, fmt.Sprintf(BaseData, index)),
			IndexDump:   filepath.Join(DataDirectory, fmt.Sprintf(BaseIndexes, index)),
			URL:         fmt.Sprintf(BaseURL, index),
		}
	}
	return &Server{
		Host:       host,
		Port:       port,
		Network:    network,
		Indexer:    engine.NewIndexer(),
		QuitSignal: false,
		Abstracts:  abstracts,
		FileIndex:  index,
		CleanFlag:  clean,
	}
}

func (s *Server) InitializeDataDirectory() error {
	if _, err := os.Stat(DataDirectory); os.IsNotExist(err) {
		if err := os.Mkdir(DataDirectory, 0755); err != nil {
			return err
		}
		return nil
	}
	if s.CleanFlag {
		files, err := os.ReadDir(DataDirectory)
		if err != nil {
			return err
		}
		for _, f := range files {
			if !f.IsDir() {
				if err := os.Remove(filepath.Join(DataDirectory, f.Name())); err != nil {
					fmt.Printf("File %s could not deleted: %s\n", f.Name(), err.Error())
				}
			}
		}
	}
	return nil
}

func (s *Server) GetAbstractStruct() *AbstractStruct {
	return s.Abstracts[s.FileIndex]
}

func (s *Server) Address() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

func (s *Server) Signature() string {
	return fmt.Sprintf("%s %s:%s", s.Network, s.Host, s.Port)
}

func (s *Server) InitializeServer() error {
	fmt.Printf("Initializing the full text search engine and the tcpserver on %s\n", s.Signature())

	t0 := time.Now()
	defer func(t0 time.Time) {
		fmt.Printf("Initializing the server took %f seconds\n", time.Since(t0).Seconds())
	}(t0)

	if err := s.InitializeDataDirectory(); err != nil {
		return err
	}

	abstracts := s.GetAbstractStruct()

	if s.Indexer.IsFileExists(abstracts.IndexDump) && s.Indexer.IsFileExists(abstracts.DataDump) {
		// Loading concurrently the index and data dump files
		workers := 2
		done := make(chan bool)
		errs := make(chan error)

		go func() {
			if err := s.Indexer.LoadIndexDump(abstracts.IndexDump); err != nil {
				errs <- err
			}
			done <- true
		}()

		go func() {
			if err := s.Indexer.LoadDataDump(abstracts.DataDump); err != nil {
				errs <- err
			}
			done <- true
		}()

		count := 0
		for {
			select {
			case err := <-errs:
				return err
			case <-done:
				count++
				if count == workers {
					return nil
				}
			}
		}
	} else {
		if s.Indexer.IsFileExists(abstracts.XMLFileName) {
			if err := s.Indexer.LoadWikimediaDump(abstracts.XMLFileName, true, abstracts.IndexDump, abstracts.DataDump); err != nil {
				return err
			}
		} else {
			// Wiki XML dump does not exists
			if !s.Indexer.IsFileExists(abstracts.GZFileName) {
				// Phase 1: Download from the server
				if err := s.Indexer.DownloadWikimediaDump(abstracts.GZFileName, abstracts.URL); err != nil {
					return err
				}
			}
			// Phase 2: Uncompress the file
			if err := s.Indexer.UncompressWikimediaDump(abstracts.GZFileName); err != nil {
				return err
			}
			// Phase 3: Load file and create indexes
			if err := s.Indexer.LoadWikimediaDump(abstracts.XMLFileName, true, abstracts.IndexDump, abstracts.DataDump); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) HandleRequest(connection net.Conn) {
	buffer := make([]byte, 1024)
	length, err := connection.Read(buffer)
	if err != nil {
		s.HandleResponse(fmt.Sprintf("Error reading connection: %s\n", err.Error()), connection)
		return
	}

	request := buffer[:length]
	queryStruct, err := s.ParseQuery(request)
	if err != nil {
		s.HandleResponse(fmt.Sprintf("Error: %s\n", err.Error()), connection)
		return
	}

	fmt.Printf("Command: %b Page: %d Phrase: %s\n", queryStruct.command, queryStruct.page, queryStruct.phrase)

	query := strings.TrimSpace(queryStruct.phrase)
	results := s.Indexer.Search(query, queryStruct.page)
	str, err := SearchResultsToJSONString(results)
	if err != nil {
		s.HandleResponse(err.Error(), connection)
		return
	}
	s.HandleResponse(str, connection)
}

func (s *Server) ParseQuery(query []byte) (*QueryStruct, error) {
	if len(query) < 5 {
		return nil, errors.New(fmt.Sprintf("invalid length: %d it should be at least 5 bytes", len(query)))
	}
	command := query[0]
	if command != QUERY {
		return nil, errors.New(fmt.Sprintf("invalid header byte %b for query command", command))
	}
	pageBytes := query[1:5]
	page := BytesToUint32(pageBytes)

	phrase := string(query[5:])

	return &QueryStruct{
		command: command,
		page:    page,
		phrase:  phrase,
	}, nil
}

func (s *Server) HandleResponse(response string, connection net.Conn) {
	defer func(c net.Conn) {
		if err := c.Close(); err != nil {
			fmt.Printf("Error closing connection: %s\n", err.Error())
		}
	}(connection)

	if _, err := connection.Write([]byte(response + "\n")); err != nil {
		fmt.Printf("Error writing to the connection: %s\n", err.Error())
	}
}

func (s *Server) AcceptConnections() error {
	listener, err := net.Listen(s.Network, s.Address())
	if err != nil {
		return err
	}
	defer func(l net.Listener) {
		if err := l.Close(); err != nil {
			fmt.Printf("Error closing listener: %s\n", err.Error())
		}
	}(listener)

	fmt.Printf("Accepting connections on %s\n", s.Signature())

	for !s.QuitSignal {
		if con, err := listener.Accept(); err != nil {
			fmt.Printf("Error accepting connection: %s\n", err.Error())
		} else {
			go s.HandleRequest(con)
		}
	}
	fmt.Printf("Server closed on %s\n", s.Signature())
	return nil
}
