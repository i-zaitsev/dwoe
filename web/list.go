// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package web

import (
	"net/http"
	"strings"
)

type batchRef struct {
	ID    string
	Color int
}

type listData struct {
	BatchFilter string
	Items       []workspaceInfo
}

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	list, err := s.workspaces.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := r.URL.Query().Get("q")
	batchFilter := r.URL.Query().Get("batch")
	index := buildBatchIndex(s)

	var infos []workspaceInfo
	for _, ws := range list {
		if q != "" && !strings.Contains(ws.Name, q) {
			continue
		}
		info := toWorkspaceInfo(ws)
		if ref, ok := index[ws.ID]; ok {
			info.BatchID = ref.ID
			info.BatchColor = ref.Color
		}
		if batchFilter != "" && info.BatchID != batchFilter {
			continue
		}
		infos = append(infos, info)
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html")
	writeTemplate(w, "workspace-list", listData{BatchFilter: batchFilter, Items: infos})
}

func buildBatchIndex(s *Server) map[string]batchRef {
	records, err := s.workspaces.BatchRecords()
	if err != nil || len(records) == 0 {
		return nil
	}
	index := make(map[string]batchRef)
	for i, rec := range records {
		color := (i % 4) + 1
		for _, entry := range rec.Entries {
			index[entry.WorkspaceID] = batchRef{
				ID:    rec.ID,
				Color: color,
			}
		}
	}
	return index
}
