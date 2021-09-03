package tcpserver

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"
	"wikisearcher/engine"
)

const (
	DataDirectory = "data"
	IndexesDump   = "indexes.json"
	DataDump      = "data.json"
	Abstract      = "enwiki-latest-abstract.xml"
	Abstract1     = "enwiki-latest-abstract1.xml"
)

type ServerInterface interface {
	Address () string
	Signature() string
	InitializeServer() error
	HandleRequest(connection net.Conn)
	HandleResponse(response string, connection net.Conn)
	AcceptConnections () error
}

type Server struct {
	Host           string
	Port           string
	Network        string
	Indexer        *engine.Indexer
	QuitSignal     bool
}

func NewServer(host string, port string, network string) (*Server, error) {
	db, err := engine.NewIndexer()
	if err != nil {
		return nil, err
	}
	return &Server{
		Host:           host,
		Port:           port,
		Network:        network,
		Indexer:        db,
		QuitSignal:     false,
	}, nil
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

	abstract := filepath.Join(DataDirectory, Abstract)
	indexDump := filepath.Join(DataDirectory, IndexesDump)
	dataDump := filepath.Join(DataDirectory, DataDump)

	if s.Indexer.IsIndexesDumped(indexDump) && s.Indexer.IsDataDumped(dataDump) {
		// Loading concurrently the index and data dump files
		workers := 2
		done := make(chan bool)
		errors := make(chan error)

		go func() {
			if err := s.Indexer.LoadIndexDump(indexDump); err != nil {
				errors <- err
			}
			done <- true
		}()

		go func() {
			if err := s.Indexer.LoadDataDump(dataDump); err != nil {
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
				if count == workers {
					return nil
				}
			}
		}
	} else {
		if err := s.Indexer.LoadWikimediaDump(abstract, true, indexDump, dataDump); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) HandleRequest(connection net.Conn)  {
	buffer := make([]byte, 1024)
	length, err := connection.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading the connection: %s\n", err.Error())
		return
	}

	request := string(buffer[:length])
	remote := connection.RemoteAddr().String()

	fmt.Printf("Connection from address: %s\n", remote)
	fmt.Printf("Received command: %s\n", request)

	if strings.HasPrefix(request, "QUERY") {
		query := strings.TrimSpace(strings.TrimPrefix(request, "QUERY"))
		results := s.Indexer.Search(query)
		str, err := SearchResultsToJSONString(results)
		if err != nil {
			s.HandleResponse(err.Error(), connection)
		}
		s.HandleResponse(str, connection)
	} else if strings.HasPrefix(request, "QUIT") {
		s.QuitSignal = true
		s.HandleResponse("GOOD BYE", connection)
	} else {
		s.HandleResponse(fmt.Sprintf("UNKNOWN COMMAND: %s\n", request), connection)
	}
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
