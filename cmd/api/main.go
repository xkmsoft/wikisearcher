package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/xkmsoft/wikisearcher/pkg/apiserver"
	"log"
	"net/http"
)

func main() {
	port := flag.Int("port", 3000, "port")
	flag.Parse()
	router := mux.NewRouter()
	router.HandleFunc("/api/query", apiserver.MakeGzipHandler(apiserver.HandleQuery)).Methods("POST")
	fmt.Printf("API listening connection on :%d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), router))
}
