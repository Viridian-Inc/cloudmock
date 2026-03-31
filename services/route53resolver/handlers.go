package route53resolver

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
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

func strSlice(params map[string]any, key string) []string {
	if v, ok := params[key].([]any); ok {
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func num(params map[string]any, key string, def int) int {
	if v, ok := params[key].(float64); ok {
		return int(v)
	}
	return def
}

func endpointResponse(ep *ResolverEndpoint) map[string]any {
	return map[string]any{
		"Id":               ep.ID,
		"Arn":              ep.Arn,
		"Name":             ep.Name,
		"Direction":        ep.Direction,
		"IpAddressCount":   ep.IPAddressCount,
		"SecurityGroupIds": ep.SecurityGroupIds,
		"Status":           ep.Status,
		"StatusMessage":    ep.StatusMessage,
		"HostVPCId":        ep.HostVPCId,
		"CreatorRequestId": ep.ID,
	}
}

func ruleResponse(r *ResolverRule) map[string]any {
	targetIPs := make([]map[string]any, 0, len(r.TargetIPs))
	for _, t := range r.TargetIPs {
		targetIPs = append(targetIPs, map[string]any{"Ip": t.IP, "Port": t.Port})
	}
	return map[string]any{
		"Id":                  r.ID,
		"Arn":                 r.Arn,
		"Name":                r.Name,
		"DomainName":          r.DomainName,
		"RuleType":            r.RuleType,
		"ResolverEndpointId":  r.ResolverEndpointID,
		"Status":              r.Status,
		"StatusMessage":       r.StatusMessage,
		"TargetIps":           targetIPs,
	}
}

func handleCreateResolverEndpoint(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "Name")
	direction := str(params, "Direction")
	if direction == "" {
		return jsonErr(service.ErrValidation("Direction is required"))
	}

	secGroupIDs := strSlice(params, "SecurityGroupIds")
	ipCount := 2
	if ips, ok := params["IpAddresses"].([]any); ok {
		ipCount = len(ips)
	}

	ep, _ := store.CreateResolverEndpoint(name, direction, str(params, "HostVPCId"), secGroupIDs, ipCount)
	return jsonOK(map[string]any{"ResolverEndpoint": endpointResponse(ep)})
}

func handleGetResolverEndpoint(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ResolverEndpointId")
	if id == "" {
		return jsonErr(service.ErrValidation("ResolverEndpointId is required"))
	}
	ep, ok := store.GetResolverEndpoint(id)
	if !ok {
		return jsonErr(service.ErrNotFound("ResolverEndpoint", id))
	}
	return jsonOK(map[string]any{"ResolverEndpoint": endpointResponse(ep)})
}

func handleListResolverEndpoints(store *Store) (*service.Response, error) {
	endpoints := store.ListResolverEndpoints()
	out := make([]map[string]any, 0, len(endpoints))
	for _, ep := range endpoints {
		out = append(out, endpointResponse(ep))
	}
	return jsonOK(map[string]any{"ResolverEndpoints": out, "MaxResults": 100})
}

func handleDeleteResolverEndpoint(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ResolverEndpointId")
	if id == "" {
		return jsonErr(service.ErrValidation("ResolverEndpointId is required"))
	}
	ep, ok := store.DeleteResolverEndpoint(id)
	if !ok {
		return jsonErr(service.ErrNotFound("ResolverEndpoint", id))
	}
	return jsonOK(map[string]any{"ResolverEndpoint": endpointResponse(ep)})
}

func handleCreateResolverRule(params map[string]any, store *Store) (*service.Response, error) {
	domainName := str(params, "DomainName")
	ruleType := str(params, "RuleType")
	if domainName == "" || ruleType == "" {
		return jsonErr(service.ErrValidation("DomainName and RuleType are required"))
	}

	var targetIPs []TargetAddress
	if tips, ok := params["TargetIps"].([]any); ok {
		for _, t := range tips {
			if tm, ok := t.(map[string]any); ok {
				targetIPs = append(targetIPs, TargetAddress{
					IP:   str(tm, "Ip"),
					Port: num(tm, "Port", 53),
				})
			}
		}
	}

	rule, _ := store.CreateResolverRule(str(params, "Name"), domainName, ruleType, str(params, "ResolverEndpointId"), targetIPs)
	return jsonOK(map[string]any{"ResolverRule": ruleResponse(rule)})
}

func handleGetResolverRule(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ResolverRuleId")
	if id == "" {
		return jsonErr(service.ErrValidation("ResolverRuleId is required"))
	}
	rule, ok := store.GetResolverRule(id)
	if !ok {
		return jsonErr(service.ErrNotFound("ResolverRule", id))
	}
	return jsonOK(map[string]any{"ResolverRule": ruleResponse(rule)})
}

func handleListResolverRules(store *Store) (*service.Response, error) {
	rules := store.ListResolverRules()
	out := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		out = append(out, ruleResponse(r))
	}
	return jsonOK(map[string]any{"ResolverRules": out, "MaxResults": 100})
}

func handleDeleteResolverRule(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ResolverRuleId")
	if id == "" {
		return jsonErr(service.ErrValidation("ResolverRuleId is required"))
	}
	rule, ok := store.DeleteResolverRule(id)
	if !ok {
		return jsonErr(service.ErrNotFound("ResolverRule", id))
	}
	return jsonOK(map[string]any{"ResolverRule": ruleResponse(rule)})
}

func handleAssociateResolverRule(params map[string]any, store *Store) (*service.Response, error) {
	ruleID := str(params, "ResolverRuleId")
	vpcID := str(params, "VPCId")
	if ruleID == "" || vpcID == "" {
		return jsonErr(service.ErrValidation("ResolverRuleId and VPCId are required"))
	}
	assoc, err := store.AssociateResolverRule(ruleID, vpcID, str(params, "Name"))
	if err != nil {
		return jsonErr(service.ErrNotFound("ResolverRule", ruleID))
	}
	return jsonOK(map[string]any{
		"ResolverRuleAssociation": map[string]any{
			"Id":             assoc.ID,
			"ResolverRuleId": assoc.ResolverRuleID,
			"VPCId":          assoc.VPCId,
			"Name":           assoc.Name,
			"Status":         assoc.Status,
		},
	})
}

func handleGetResolverRuleAssociation(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ResolverRuleAssociationId")
	if id == "" {
		return jsonErr(service.ErrValidation("ResolverRuleAssociationId is required"))
	}
	assoc, ok := store.GetResolverRuleAssociation(id)
	if !ok {
		return jsonErr(service.ErrNotFound("ResolverRuleAssociation", id))
	}
	return jsonOK(map[string]any{
		"ResolverRuleAssociation": map[string]any{
			"Id":             assoc.ID,
			"ResolverRuleId": assoc.ResolverRuleID,
			"VPCId":          assoc.VPCId,
			"Name":           assoc.Name,
			"Status":         assoc.Status,
		},
	})
}

func handleListResolverRuleAssociations(store *Store) (*service.Response, error) {
	assocs := store.ListResolverRuleAssociations()
	out := make([]map[string]any, 0, len(assocs))
	for _, a := range assocs {
		out = append(out, map[string]any{
			"Id":             a.ID,
			"ResolverRuleId": a.ResolverRuleID,
			"VPCId":          a.VPCId,
			"Name":           a.Name,
			"Status":         a.Status,
		})
	}
	return jsonOK(map[string]any{"ResolverRuleAssociations": out, "MaxResults": 100})
}

func handleDisassociateResolverRule(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ResolverRuleAssociationId")
	if id == "" {
		return jsonErr(service.ErrValidation("ResolverRuleAssociationId is required"))
	}
	assoc, ok := store.DisassociateResolverRule(id)
	if !ok {
		return jsonErr(service.ErrNotFound("ResolverRuleAssociation", id))
	}
	return jsonOK(map[string]any{
		"ResolverRuleAssociation": map[string]any{
			"Id":     assoc.ID,
			"Status": "DELETING",
		},
	})
}

func handleCreateQueryLogConfig(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "Name")
	destArn := str(params, "DestinationArn")
	if name == "" || destArn == "" {
		return jsonErr(service.ErrValidation("Name and DestinationArn are required"))
	}
	config, _ := store.CreateQueryLogConfig(name, destArn)
	return jsonOK(map[string]any{
		"ResolverQueryLogConfig": map[string]any{
			"Id":             config.ID,
			"Arn":            config.Arn,
			"Name":           config.Name,
			"DestinationArn": config.DestinationArn,
			"Status":         config.Status,
		},
	})
}

func handleGetQueryLogConfig(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ResolverQueryLogConfigId")
	if id == "" {
		return jsonErr(service.ErrValidation("ResolverQueryLogConfigId is required"))
	}
	config, ok := store.GetQueryLogConfig(id)
	if !ok {
		return jsonErr(service.ErrNotFound("ResolverQueryLogConfig", id))
	}
	return jsonOK(map[string]any{
		"ResolverQueryLogConfig": map[string]any{
			"Id":               config.ID,
			"Arn":              config.Arn,
			"Name":             config.Name,
			"DestinationArn":   config.DestinationArn,
			"Status":           config.Status,
			"AssociationCount": config.AssociationCount,
		},
	})
}

func handleListQueryLogConfigs(store *Store) (*service.Response, error) {
	configs := store.ListQueryLogConfigs()
	out := make([]map[string]any, 0, len(configs))
	for _, c := range configs {
		out = append(out, map[string]any{
			"Id":     c.ID,
			"Name":   c.Name,
			"Status": c.Status,
		})
	}
	return jsonOK(map[string]any{"ResolverQueryLogConfigs": out, "TotalCount": len(configs)})
}

func handleDeleteQueryLogConfig(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ResolverQueryLogConfigId")
	if id == "" {
		return jsonErr(service.ErrValidation("ResolverQueryLogConfigId is required"))
	}
	config, ok := store.DeleteQueryLogConfig(id)
	if !ok {
		return jsonErr(service.ErrNotFound("ResolverQueryLogConfig", id))
	}
	return jsonOK(map[string]any{
		"ResolverQueryLogConfig": map[string]any{
			"Id":     config.ID,
			"Status": "DELETING",
		},
	})
}
