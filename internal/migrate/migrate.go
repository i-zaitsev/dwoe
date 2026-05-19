package migrate

import (
	"encoding/json"
	"fmt"
)

const latestVersion = 2

func State(data []byte) ([]byte, int, error) {
	version, err := readVersion(data)
	if err != nil {
		return nil, 0, err
	}
	if version >= latestVersion {
		return data, 0, nil
	}

	applied := 0

	if version == 1 {
		data, err = v1ToV2(data)
		if err != nil {
			return nil, 0, fmt.Errorf("v1→v2: %w", err)
		}
		applied++
	}

	return data, applied, nil
}

func readVersion(data []byte) (int, error) {
	var peek struct {
		Version int
	}
	if err := json.Unmarshal(data, &peek); err != nil {
		return 0, fmt.Errorf("reading version: %w", err)
	}
	return peek.Version, nil
}
