package main

import (
	"flag"
	"log"
	"strings"

	"github.com/xkmsoft/wikisearcher/pkg/tcpserver"
)

func GetAllowedNetworks(networks map[string]string) string {
	nets := make([]string, 0, 3)
	for n, _ := range networks {
		nets = append(nets, n)
	}
	return strings.Join(nets, ", ")
}

func main() {
	host := flag.String("host", "localhost", "hostname")
	port := flag.String("port", "3333", "port")
	network := flag.String("network", "tcp", "Network should be [tcp, tcp4, tcp6]")
	index := flag.Int("index", 1, "Abstract index [0, 27]")
	clean := flag.Bool("clean", false, "Cleans all files within the data directory if set")
	flag.Parse()

	allowedNetworks := map[string]string{"tcp": "", "tcp4": "", "tcp6": ""}
	if _, ok := allowedNetworks[strings.ToLower(*network)]; !ok {
		log.Fatalf("Not allowed network %s. Network should be: %s\n", strings.ToLower(*network), GetAllowedNetworks(allowedNetworks))
	}

	if *index < 0 || *index > 27 {
		log.Fatalf("Wrong index: %d Index should be [0, 27]", *index)
	}

	tcpServer := tcpserver.NewServer(*host, *port, *network, *index, *clean)

	if err := tcpServer.InitializeServer(); err != nil {
		log.Fatal(err)
	}

	if err := tcpServer.AcceptConnections(); err != nil {
		log.Fatal(err)
	}
}
