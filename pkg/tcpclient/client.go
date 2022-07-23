package tcpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/xkmsoft/wikisearcher/pkg/engine"
)

const (
	QUERY = byte(0)
)

type ClientInterface interface {
	Query(s string) (*engine.SearchResults, error)
	PrepareQuery(s string, p uint32) []byte
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

func (c *TCPClient) PrepareQuery(s string, p uint32) []byte {
	query := make([]byte, 0)
	query = append(query, GetHeader(QUERY)...)
	query = append(query, Uint32ToBytes(p)...)
	query = append(query, []byte(s)...)
	return query
}

func (c *TCPClient) Address() string {
	return fmt.Sprintf("%s:%s", c.Ip, c.Port)
}

func (c *TCPClient) Query(s string, page uint32) (*engine.SearchResults, error) {

	query := c.PrepareQuery(s, page)
	address := c.Address()

	tcpAddr, err := net.ResolveTCPAddr(c.Network, address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP(c.Network, nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	_, err = conn.Write(query)
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
	var searchResults engine.SearchResults
	if err = json.Unmarshal(buffer.Bytes(), &searchResults); err != nil {
		return nil, err
	}

	return &searchResults, nil
}
