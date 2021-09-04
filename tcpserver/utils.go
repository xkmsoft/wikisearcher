package tcpserver

import (
	"encoding/json"
	"github.com/xkmsoft/wikisearcher/engine"
)

func SearchResultsToJSONString(results engine.SearchResults) (string, error) {
	if bytes, err := json.Marshal(results); err != nil {
		return "", err
	} else {
		return string(bytes), nil
	}
}
