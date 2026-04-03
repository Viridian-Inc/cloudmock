package route53

import "encoding/json"

// r53State is the serialised form of all Route 53 state.
type r53State struct {
	HostedZones []r53ZoneState `json:"hosted_zones"`
}

type r53ZoneState struct {
	Name        string `json:"name"`
	Comment     string `json:"comment,omitempty"`
	PrivateZone bool   `json:"private_zone,omitempty"`
}

// ExportState returns a JSON snapshot of all Route 53 hosted zones.
func (s *Route53Service) ExportState() (json.RawMessage, error) {
	state := r53State{HostedZones: make([]r53ZoneState, 0)}

	for _, z := range s.store.ListZones() {
		state.HostedZones = append(state.HostedZones, r53ZoneState{
			Name:        z.Name,
			Comment:     z.Config.Comment,
			PrivateZone: z.Config.PrivateZone,
		})
	}

	return json.Marshal(state)
}

// ImportState restores Route 53 state from a JSON snapshot.
func (s *Route53Service) ImportState(data json.RawMessage) error {
	var state r53State
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	for _, zs := range state.HostedZones {
		s.store.CreateZone(zs.Name, zs.Name, ZoneConfig{
			Comment:     zs.Comment,
			PrivateZone: zs.PrivateZone,
		})
	}
	return nil
}
