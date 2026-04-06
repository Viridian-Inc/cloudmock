package dynamodb

import (
	"strconv"
	"time"
)

// TTLSpecification describes the TTL configuration for a table.
type TTLSpecification struct {
	AttributeName string `json:"AttributeName"`
	Enabled       bool   `json:"Enabled"`
}

// startTTLReaper starts a background goroutine that periodically removes expired items.
// It runs every 5 seconds and stops when the done channel is closed.
func (s *TableStore) startTTLReaper(done <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				s.reapExpiredItems()
			}
		}
	}()
}

// reapExpiredItems scans all tables and deletes items whose TTL attribute is before now.
func (s *TableStore) reapExpiredItems() {
	now := time.Now().Unix()

	// Snapshot the table list from the sync.Map.
	type tableInfo struct {
		name  string
		table *Table
	}
	var tablesWithTTL []tableInfo
	s.tables.Range(func(key, value any) bool {
		name := key.(string)
		table := value.(*Table)
		if table.TTL != nil && table.TTL.Enabled {
			tablesWithTTL = append(tablesWithTTL, tableInfo{name: name, table: table})
		}
		return true
	})

	for _, ti := range tablesWithTTL {
		table := ti.table

		table.mu.Lock()
		attrName := table.TTL.AttributeName

		// Scan all items to find expired ones.
		allItems := table.scanAll(0)
		var toDelete []Item
		for _, item := range allItems {
			av, ok := item[attrName]
			if !ok {
				continue
			}
			nStr, ok := av["N"].(string)
			if !ok {
				continue
			}
			ttlVal, err := strconv.ParseInt(nStr, 10, 64)
			if err != nil {
				continue
			}
			if ttlVal < now {
				keyItem := make(Item)
				for _, ks := range table.KeySchema {
					if v, exists := item[ks.AttributeName]; exists {
						keyItem[ks.AttributeName] = v
					}
				}
				toDelete = append(toDelete, keyItem)
			}
		}

		// Delete expired items.
		for _, key := range toDelete {
			s.deleteFromTable(table, key)
		}
		table.mu.Unlock()
	}
}
