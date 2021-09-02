package tcpserver

import (
	"encoding/json"
	"wikisearcher/engine"
)

func SearchResultsToJSONString(results engine.SearchResults) (string, error) {
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
