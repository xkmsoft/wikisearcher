package tcpserver

import (
	"encoding/binary"
	"encoding/json"
	"github.com/xkmsoft/wikisearcher/pkg/engine"
)

func SearchResultsToJSONString(results engine.SearchResults) (string, error) {
	if bytes, err := json.Marshal(results); err != nil {
		return "", err
	} else {
		return string(bytes), nil
	}
}

func BytesToUint32(bytes []byte) uint32 {
	return binary.BigEndian.Uint32(bytes)
}
