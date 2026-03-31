package managedblockchain

import (
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func jsonNoContent() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func str(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func handleCreateNetwork(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required"))
	}
	framework := str(params, "Framework")
	if framework == "" {
		framework = "HYPERLEDGER_FABRIC"
	}
	frameworkVersion := str(params, "FrameworkVersion")
	if frameworkVersion == "" {
		frameworkVersion = "2.2"
	}

	net, member, _ := store.CreateNetwork(name, str(params, "Description"), framework, frameworkVersion)
	return jsonOK(map[string]any{
		"NetworkId": net.ID,
		"MemberId":  member.ID,
	})
}

func handleGetNetwork(networkID string, store *Store) (*service.Response, error) {
	net, ok := store.GetNetwork(networkID)
	if !ok {
		return jsonErr(service.ErrNotFound("Network", networkID))
	}
	return jsonOK(map[string]any{
		"Network": map[string]any{
			"Id":               net.ID,
			"Name":             net.Name,
			"Description":      net.Description,
			"Framework":        net.Framework,
			"FrameworkVersion": net.FrameworkVersion,
			"Status":           net.Status,
			"CreationDate":     net.CreationDate.Format(time.RFC3339),
		},
	})
}

func handleListNetworks(store *Store) (*service.Response, error) {
	networks := store.ListNetworks()
	out := make([]map[string]any, 0, len(networks))
	for _, n := range networks {
		out = append(out, map[string]any{
			"Id":        n.ID,
			"Name":      n.Name,
			"Framework": n.Framework,
			"Status":    n.Status,
		})
	}
	return jsonOK(map[string]any{"Networks": out})
}

func handleGetMember(networkID, memberID string, store *Store) (*service.Response, error) {
	m, ok := store.GetMember(networkID, memberID)
	if !ok {
		return jsonErr(service.ErrNotFound("Member", memberID))
	}
	return jsonOK(map[string]any{
		"Member": map[string]any{
			"Id":           m.ID,
			"NetworkId":    m.NetworkID,
			"Name":         m.Name,
			"Status":       m.Status,
			"CreationDate": m.CreationDate.Format(time.RFC3339),
		},
	})
}

func handleListMembers(networkID string, store *Store) (*service.Response, error) {
	members := store.ListMembers(networkID)
	out := make([]map[string]any, 0, len(members))
	for _, m := range members {
		out = append(out, map[string]any{
			"Id":     m.ID,
			"Name":   m.Name,
			"Status": m.Status,
		})
	}
	return jsonOK(map[string]any{"Members": out})
}

func handleCreateNode(networkID, memberID string, params map[string]any, store *Store) (*service.Response, error) {
	nodeConfig, _ := params["NodeConfiguration"].(map[string]any)
	instanceType := str(nodeConfig, "InstanceType")
	if instanceType == "" {
		instanceType = "bc.t3.small"
	}
	az := str(nodeConfig, "AvailabilityZone")

	node, err := store.CreateNode(networkID, memberID, instanceType, az)
	if err != nil {
		return jsonErr(service.ErrNotFound("Network", networkID))
	}
	return jsonOK(map[string]any{"NodeId": node.ID})
}

func handleGetNode(networkID, nodeID string, store *Store) (*service.Response, error) {
	node, ok := store.GetNode(networkID, nodeID)
	if !ok {
		return jsonErr(service.ErrNotFound("Node", nodeID))
	}
	return jsonOK(map[string]any{
		"Node": map[string]any{
			"Id":               node.ID,
			"NetworkId":        node.NetworkID,
			"MemberId":         node.MemberID,
			"InstanceType":     node.InstanceType,
			"AvailabilityZone": node.AvailabilityZone,
			"Status":           node.Status,
			"CreationDate":     node.CreationDate.Format(time.RFC3339),
		},
	})
}

func handleListNodes(networkID string, store *Store) (*service.Response, error) {
	nodes := store.ListNodes(networkID)
	out := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, map[string]any{
			"Id":           n.ID,
			"InstanceType": n.InstanceType,
			"Status":       n.Status,
		})
	}
	return jsonOK(map[string]any{"Nodes": out})
}

func handleDeleteNode(networkID, nodeID string, store *Store) (*service.Response, error) {
	if !store.DeleteNode(networkID, nodeID) {
		return jsonErr(service.ErrNotFound("Node", nodeID))
	}
	return jsonNoContent()
}

func handleCreateProposal(networkID string, params map[string]any, store *Store) (*service.Response, error) {
	memberID := str(params, "MemberId")
	description := str(params, "Description")
	proposal, err := store.CreateProposal(networkID, description, memberID)
	if err != nil {
		return jsonErr(service.ErrNotFound("Network", networkID))
	}
	return jsonOK(map[string]any{"ProposalId": proposal.ProposalID})
}

func handleGetProposal(networkID, proposalID string, store *Store) (*service.Response, error) {
	p, ok := store.GetProposal(networkID, proposalID)
	if !ok {
		return jsonErr(service.ErrNotFound("Proposal", proposalID))
	}
	return jsonOK(map[string]any{
		"Proposal": map[string]any{
			"ProposalId":         p.ProposalID,
			"NetworkId":          p.NetworkID,
			"Description":        p.Description,
			"ProposedByMemberId": p.ProposedByMemberID,
			"Status":             p.Status,
			"CreationDate":       p.CreationDate.Format(time.RFC3339),
			"ExpirationDate":     p.ExpirationDate.Format(time.RFC3339),
		},
	})
}

func handleListProposals(networkID string, store *Store) (*service.Response, error) {
	proposals := store.ListProposals(networkID)
	out := make([]map[string]any, 0, len(proposals))
	for _, p := range proposals {
		out = append(out, map[string]any{
			"ProposalId": p.ProposalID,
			"Status":     p.Status,
		})
	}
	return jsonOK(map[string]any{"Proposals": out})
}
