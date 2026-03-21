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

	s.mu.Lock()
	defer s.mu.Unlock()

	for tableName, table := range s.tables {
		if table.TTL == nil || !table.TTL.Enabled {
			continue
		}
		attrName := table.TTL.AttributeName

		var toDelete []Item
		for _, item := range table.Items {
			av, ok := item[attrName]
			if !ok {
				continue
			}
			// TTL values are stored as N (number) type.
			nStr, ok := av["N"].(string)
			if !ok {
				continue
			}
			ttlVal, err := strconv.ParseInt(nStr, 10, 64)
			if err != nil {
				continue
			}
			if ttlVal < now {
				// Build a key-only item for deletion.
				keyItem := make(Item)
				for _, ks := range table.KeySchema {
					if v, exists := item[ks.AttributeName]; exists {
						keyItem[ks.AttributeName] = v
					}
				}
				toDelete = append(toDelete, keyItem)
			}
		}

		// Delete expired items, recording stream events.
		for _, key := range toDelete {
			// Find the full item before deleting (for stream old image).
			var oldItem Item
			for _, item := range table.Items {
				if table.keyMatchesItem(key, item) {
					oldItem = copyItem(item)
					break
				}
			}

			s.deleteItemLocked(tableName, key)

			// Emit stream record if streams are enabled.
			if table.Stream != nil && oldItem != nil {
				table.Stream.appendRecord("REMOVE", oldItem, nil)
			}
		}
	}
}
