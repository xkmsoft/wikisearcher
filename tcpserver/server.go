package tcpserver

import (
	"fmt"
	"net"
	"strings"
	"time"
	"wikisearcher/engine"
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
	begin := time.Now()
	defer func(begin time.Time) {
		elapsed := time.Since(begin)
		fmt.Printf("Initializing the server took %f seconds\n", elapsed.Seconds())
	}(begin)

	fmt.Printf("Initializing the full text search engine and the tcpserver on %s\n", s.Signature())
	if s.Indexer.IsIndexesDumped() && s.Indexer.IsDataDumped() {
		// Loading concurrently the index and data dump files
		ops := 2
		done := make(chan bool)
		errors := make(chan error)
		go func() {
			err := s.Indexer.LoadIndexDump("./data/indexes.json")
			if err != nil {
				errors <- err
			}
			done <- true
		}()
		go func() {
			err := s.Indexer.LoadDataDump("./data/data.json")
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
	} else {
		fmt.Printf("Loading xml dump to index the data...\n")
		err := s.Indexer.LoadWikimediaDump("./data/enwiki-latest-abstract1.xml", true)
		if err != nil {
			return err
		}
		fmt.Printf("Indexes have been created from the xml file successully\n")
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
	defer func(connection net.Conn) {
		err := connection.Close()
		if err != nil {
			fmt.Printf("Error closing connection: %s\n", err.Error())
		}
	}(connection)
	_, err := connection.Write([]byte(response + "\n"))
	if err != nil {
		fmt.Printf("Error writing to the connection: %s\n", err.Error())
	}
}

func (s *Server) AcceptConnections() error {
	listener, err := net.Listen(s.Network, s.Address())
	if err != nil {
		return err
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Printf("Error closing listener: %s\n", err.Error())
		}
	}(listener)

	fmt.Printf("Accepting connections on %s\n", s.Signature())

	for !s.QuitSignal {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %s\n", err.Error())
		}
		go s.HandleRequest(connection)
	}

	fmt.Printf("Server closed on %s\n", s.Signature())
	return nil
}
