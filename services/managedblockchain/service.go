package managedblockchain

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ManagedBlockchainService is the cloudmock implementation of the Amazon Managed Blockchain API.
type ManagedBlockchainService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new ManagedBlockchainService for the given AWS account ID and region.
func New(accountID, region string) *ManagedBlockchainService {
	return &ManagedBlockchainService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *ManagedBlockchainService) Name() string { return "managedblockchain" }

// Actions returns the list of Managed Blockchain API actions supported by this service.
func (s *ManagedBlockchainService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateNetwork", Method: http.MethodPost, IAMAction: "managedblockchain:CreateNetwork"},
		{Name: "GetNetwork", Method: http.MethodGet, IAMAction: "managedblockchain:GetNetwork"},
		{Name: "ListNetworks", Method: http.MethodGet, IAMAction: "managedblockchain:ListNetworks"},
		{Name: "GetMember", Method: http.MethodGet, IAMAction: "managedblockchain:GetMember"},
		{Name: "ListMembers", Method: http.MethodGet, IAMAction: "managedblockchain:ListMembers"},
		{Name: "CreateNode", Method: http.MethodPost, IAMAction: "managedblockchain:CreateNode"},
		{Name: "GetNode", Method: http.MethodGet, IAMAction: "managedblockchain:GetNode"},
		{Name: "ListNodes", Method: http.MethodGet, IAMAction: "managedblockchain:ListNodes"},
		{Name: "DeleteNode", Method: http.MethodDelete, IAMAction: "managedblockchain:DeleteNode"},
		{Name: "CreateProposal", Method: http.MethodPost, IAMAction: "managedblockchain:CreateProposal"},
		{Name: "GetProposal", Method: http.MethodGet, IAMAction: "managedblockchain:GetProposal"},
		{Name: "ListProposals", Method: http.MethodGet, IAMAction: "managedblockchain:ListProposals"},
	}
}

// HealthCheck always returns nil.
func (s *ManagedBlockchainService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Managed Blockchain request to the appropriate handler.
func (s *ManagedBlockchainService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	// /networks
	if path == "/networks" {
		switch method {
		case http.MethodPost:
			return handleCreateNetwork(params, s.store)
		case http.MethodGet:
			return handleListNetworks(s.store)
		}
	}

	parts := strings.Split(strings.TrimPrefix(path, "/networks/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
	}

	networkID := parts[0]

	// /networks/{networkID}
	if len(parts) == 1 && method == http.MethodGet {
		return handleGetNetwork(networkID, s.store)
	}

	if len(parts) >= 2 {
		sub := parts[1]
		switch sub {
		case "members":
			if len(parts) == 2 && method == http.MethodGet {
				return handleListMembers(networkID, s.store)
			}
			if len(parts) == 3 {
				memberID := parts[2]
				if method == http.MethodGet {
					return handleGetMember(networkID, memberID, s.store)
				}
				// /networks/{id}/members/{memberId}/nodes
				if len(parts) >= 4 && parts[3] == "nodes" {
					if len(parts) == 4 {
						switch method {
						case http.MethodPost:
							return handleCreateNode(networkID, memberID, params, s.store)
						case http.MethodGet:
							return handleListNodes(networkID, s.store)
						}
					}
					if len(parts) == 5 {
						nodeID := parts[4]
						switch method {
						case http.MethodGet:
							return handleGetNode(networkID, nodeID, s.store)
						case http.MethodDelete:
							return handleDeleteNode(networkID, nodeID, s.store)
						}
					}
				}
			}
		case "nodes":
			if len(parts) == 2 && method == http.MethodGet {
				return handleListNodes(networkID, s.store)
			}
			if len(parts) == 3 {
				nodeID := parts[2]
				switch method {
				case http.MethodGet:
					return handleGetNode(networkID, nodeID, s.store)
				case http.MethodDelete:
					return handleDeleteNode(networkID, nodeID, s.store)
				}
			}
		case "proposals":
			if len(parts) == 2 {
				switch method {
				case http.MethodPost:
					return handleCreateProposal(networkID, params, s.store)
				case http.MethodGet:
					return handleListProposals(networkID, s.store)
				}
			}
			if len(parts) == 3 {
				proposalID := parts[2]
				if method == http.MethodGet {
					return handleGetProposal(networkID, proposalID, s.store)
				}
			}
			// /networks/{id}/proposals/{proposalId}/votes
			if len(parts) == 4 && parts[3] == "votes" && method == http.MethodPost {
				return handleVoteOnProposal(networkID, parts[2], params, s.store)
			}
		}
	}

	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}
