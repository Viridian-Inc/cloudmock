package azurearm

import (
	"fmt"
	"net/url"
	"strings"
)

// ResourceID is the parsed form of a canonical Azure Resource Manager ID.
type ResourceID struct {
	SubscriptionID string
	ResourceGroup  string
	Provider       string
	Types          []string
	Names          []string
}

// ParseID parses subscription-scope and resource-group-scope ARM IDs.
func ParseID(raw string) (ResourceID, error) {
	parts, err := splitIDPath(raw)
	if err != nil {
		return ResourceID{}, err
	}
	if len(parts) < 2 || !strings.EqualFold(parts[0], "subscriptions") || parts[1] == "" {
		return ResourceID{}, fmt.Errorf("azurearm: invalid resource ID %q: expected /subscriptions/{subscriptionId}", raw)
	}

	id := ResourceID{SubscriptionID: parts[1]}
	i := 2

	if i < len(parts) && strings.EqualFold(parts[i], "resourceGroups") {
		if i+1 >= len(parts) || parts[i+1] == "" {
			return ResourceID{}, fmt.Errorf("azurearm: invalid resource ID %q: missing resource group name", raw)
		}
		id.ResourceGroup = parts[i+1]
		i += 2
	}

	if i == len(parts) {
		return id, nil
	}
	if !strings.EqualFold(parts[i], "providers") {
		return ResourceID{}, fmt.Errorf("azurearm: invalid resource ID %q: expected providers segment", raw)
	}
	if i+1 >= len(parts) || parts[i+1] == "" {
		return ResourceID{}, fmt.Errorf("azurearm: invalid resource ID %q: missing provider namespace", raw)
	}

	id.Provider = parts[i+1]
	i += 2
	remaining := parts[i:]
	if len(remaining)%2 != 0 {
		return ResourceID{}, fmt.Errorf("azurearm: invalid resource ID %q: provider resources must have type/name pairs", raw)
	}
	for j := 0; j < len(remaining); j += 2 {
		if remaining[j] == "" || remaining[j+1] == "" {
			return ResourceID{}, fmt.Errorf("azurearm: invalid resource ID %q: empty resource type or name", raw)
		}
		id.Types = append(id.Types, remaining[j])
		id.Names = append(id.Names, remaining[j+1])
	}

	return id, nil
}

func (id ResourceID) String() string {
	if id.SubscriptionID == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString("/subscriptions/")
	b.WriteString(id.SubscriptionID)
	if id.ResourceGroup != "" {
		b.WriteString("/resourceGroups/")
		b.WriteString(id.ResourceGroup)
	}
	if id.Provider != "" {
		b.WriteString("/providers/")
		b.WriteString(id.Provider)
	}
	for i := range id.Types {
		b.WriteByte('/')
		b.WriteString(id.Types[i])
		if i < len(id.Names) {
			b.WriteByte('/')
			b.WriteString(id.Names[i])
		}
	}
	return b.String()
}

func splitIDPath(raw string) ([]string, error) {
	raw = strings.Trim(raw, "/")
	if raw == "" {
		return nil, fmt.Errorf("azurearm: empty resource ID")
	}

	segments := strings.Split(raw, "/")
	parts := segments[:0]
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		decoded, err := url.PathUnescape(segment)
		if err != nil {
			return nil, fmt.Errorf("azurearm: invalid resource ID segment %q: %w", segment, err)
		}
		parts = append(parts, decoded)
	}
	return parts, nil
}
