package tcpserver

import (
	"encoding/json"
	"wikisearcher/engine"
)

func SearchResultsToJSONString(results engine.SearchResults) (string, error) {
	if bytes, err := json.MarshalIndent(results, "", "  "); err != nil {
		return "", err
	} else {
		return string(bytes), nil
	}
}
