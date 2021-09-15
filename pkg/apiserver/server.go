package apiserver

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/xkmsoft/wikisearcher/pkg/tcpclient"
	"io"
	"net/http"
	"strings"
)

const (
	Ip      = "localhost"
	Port    = "3333"
	Network = "tcp"
)

type QueryParams struct {
	Query string `json:"query"`
	Page  int    `json:"page"`
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func MakeGzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accepts := r.Header.Get("Accept-Encoding")
		if !strings.Contains(accepts, "gzip") {
			// Client does not support gzip encoding; Returning the original handler
			fn(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
		if err != nil {
			// Failed to set the compression level: Returning the original handler
			fn(w, r)
			return
		}
		defer func(gz *gzip.Writer) {
			if err := gz.Close(); err != nil {
				fmt.Printf("Error closing gz writer: %s\n", err.Error())
			}
		}(gz)
		// Setting content-encoding as gzip
		w.Header().Set("Content-Encoding", "gzip")
		fn(gzipResponseWriter{
			Writer:         gz,
			ResponseWriter: w,
		}, r)
	}
}

func HandleQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var params QueryParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := tcpclient.NewTCPClient(Ip, Port, Network)
	clientResponse, err := client.Query(params.Query, uint32(params.Page))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.NewEncoder(w).Encode(clientResponse); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
