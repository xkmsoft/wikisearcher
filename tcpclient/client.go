package tcpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/xkmsoft/wikisearcher/engine"
	"io"
	"net"
)

type ClientInterface interface {
	Query(s string) (*engine.SearchResult, error)
	PrepareQuery(s string) string
	Address() string
}

type TCPClient struct {
	Ip      string
	Port    string
	Network string
}

func NewTCPClient(ip string, port string, network string) *TCPClient {
	return &TCPClient{
		Ip:      ip,
		Port:    port,
		Network: network,
	}
}

func (c *TCPClient) PrepareQuery(s string) string {
	return fmt.Sprintf("QUERY %s", s)
}

func (c *TCPClient) Address() string {
	return fmt.Sprintf("%s:%s", c.Ip, c.Port)
}

func (c *TCPClient) Query(s string) (*engine.SearchResult, error) {

	query := c.PrepareQuery(s)
	address := c.Address()

	tcpAddr, err := net.ResolveTCPAddr(c.Network, address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP(c.Network, nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	_, err = conn.Write([]byte(query))
	if err != nil {
		return nil, err
	}
	defer func(conn *net.TCPConn) {
		if err := conn.Close(); err != nil {
			fmt.Printf("Error closing TCP connection: %s\n", err.Error())
		}
	}(conn)

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, conn); err != nil {
		return nil, err
	}
	var searchResults engine.SearchResult
	if err = json.Unmarshal(buffer.Bytes(), &searchResults); err != nil {
		return nil, err
	}

	return &searchResults, nil
}
