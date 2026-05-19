package migrate

import (
	"encoding/json"
	"fmt"
)

var v1KeyRenames = map[string]string{
	"ID":           "id",
	"Name":         "name",
	"Status":       "status",
	"BasePath":     "base_path",
	"ContainerIDs": "container_ids",
	"NetworkID":    "network_id",
	"CreatedAt":    "created_at",
	"StartedAt":    "started_at",
	"FinishedAt":   "finished_at",
}

func v1ToV2(data []byte) ([]byte, error) {
	var raw struct {
		Version    int
		Workspaces map[string]json.RawMessage
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	for wsID, wsRaw := range raw.Workspaces {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(wsRaw, &fields); err != nil {
			return nil, fmt.Errorf("workspace %s: %w", wsID, err)
		}
		renamed := make(map[string]json.RawMessage, len(fields))
		for k, v := range fields {
			if newKey, ok := v1KeyRenames[k]; ok {
				renamed[newKey] = v
			} else {
				renamed[k] = v
			}
		}
		out, err := json.Marshal(renamed)
		if err != nil {
			return nil, fmt.Errorf("workspace %s: %w", wsID, err)
		}
		raw.Workspaces[wsID] = out
	}

	result := struct {
		Version    int                        `json:"version"`
		Workspaces map[string]json.RawMessage `json:"workspaces"`
	}{
		Version:    2,
		Workspaces: raw.Workspaces,
	}
	return json.MarshalIndent(result, "", "  ")
}
