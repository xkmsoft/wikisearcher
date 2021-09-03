package main

import (
	"flag"
	"log"
	"wikisearcher/tcpserver"
)

func main() {
	hostFlag := flag.String("host", "localhost", "hostname")
	portFlag := flag.String("port", "3333", "port")
	networkFlag := flag.String("network", "tcp", "Network should be tcp, tcp4, tcp6, unix or unixpacket")
	flag.Parse()

	host := *hostFlag
	port := *portFlag
	network := *networkFlag

	tcpServer, err := tcpserver.NewServer(host, port, network)
	if err != nil {
		log.Fatal(err)
	}

	if err = tcpServer.InitializeServer(); err != nil {
		log.Fatal(err)
	}

	if err = tcpServer.AcceptConnections(); err != nil {
		log.Fatal(err)
	}
}
