package dynamodb

import "encoding/json"

// ddbState is the serialised form of all DynamoDB state.
type ddbState struct {
	Tables []ddbTableState `json:"tables"`
}

type ddbTableState struct {
	Name                  string                 `json:"name"`
	KeySchema             []KeySchemaElement     `json:"key_schema"`
	AttributeDefinitions  []AttributeDefinition  `json:"attribute_definitions"`
	BillingMode           string                 `json:"billing_mode"`
	ProvisionedThroughput *ProvisionedThroughput `json:"provisioned_throughput,omitempty"`
	GSIs                  []GSI                  `json:"gsis,omitempty"`
	LSIs                  []LSI                  `json:"lsis,omitempty"`
	Items                 []Item                 `json:"items"`
}

// ExportState returns a JSON snapshot of all DynamoDB tables and items.
func (s *DynamoDBService) ExportState() (json.RawMessage, error) {
	state := ddbState{Tables: make([]ddbTableState, 0)}

	for _, name := range s.store.ListTables() {
		table, awsErr := s.store.DescribeTable(name)
		if awsErr != nil {
			continue
		}

		table.mu.RLock()
		items := table.scanAll(0)
		copied := make([]Item, len(items))
		for i, item := range items {
			copied[i] = copyItem(item)
		}
		table.mu.RUnlock()

		ts := ddbTableState{
			Name:                  table.Name,
			KeySchema:             table.KeySchema,
			AttributeDefinitions:  table.AttributeDefinitions,
			BillingMode:           table.BillingMode,
			ProvisionedThroughput: table.ProvisionedThroughput,
			GSIs:                  table.GSIs,
			LSIs:                  table.LSIs,
			Items:                 copied,
		}
		state.Tables = append(state.Tables, ts)
	}

	return json.Marshal(state)
}

// ImportState restores DynamoDB state from a JSON snapshot.
func (s *DynamoDBService) ImportState(data json.RawMessage) error {
	var state ddbState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	for _, ts := range state.Tables {
		// Create table (ignore if already exists).
		s.store.CreateTable(
			ts.Name,
			ts.KeySchema,
			ts.AttributeDefinitions,
			ts.BillingMode,
			ts.ProvisionedThroughput,
			ts.GSIs,
			ts.LSIs,
			nil, // no stream spec for imported tables
		)
		for _, item := range ts.Items {
			s.store.PutItem(ts.Name, item)
		}
	}
	return nil
}
