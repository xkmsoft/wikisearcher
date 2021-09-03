package main

import (
	"flag"
	"log"
	"wikisearcher/tcpserver"
)

func main() {
	host := flag.String("host", "localhost", "hostname")
	port := flag.String("port", "3333", "port")
	network := flag.String("network", "tcp", "Network should be tcp, tcp4, tcp6, unix or unixpacket")
	index := flag.Int("index", 2, "Abstract index [0, 27]")
	clean := flag.Bool("clean", false, "Cleans all files within the data directory if set")
	flag.Parse()

	if *index < 0 || *index > 27 {
		log.Fatalf("Wrong index: %d Index should be [0, 27]", *index)
	}

	tcpServer, err := tcpserver.NewServer(*host, *port, *network, *index, *clean)
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
