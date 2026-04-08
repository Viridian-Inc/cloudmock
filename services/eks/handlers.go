package eks

import (
	"crypto/rand"
	gojson "github.com/goccy/go-json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// EKS uses REST-JSON protocol. Routes are path-based.

// ---- JSON types ----

type tagEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type clusterJSON struct {
	Name                   string            `json:"name"`
	Arn                    string            `json:"arn"`
	Version                string            `json:"version"`
	RoleArn                string            `json:"roleArn,omitempty"`
	Status                 string            `json:"status"`
	Endpoint               string            `json:"endpoint,omitempty"`
	CertificateAuthority   *certAuthJSON     `json:"certificateAuthority,omitempty"`
	Identity               *identityJSON     `json:"identity,omitempty"`
	PlatformVersion        string            `json:"platformVersion,omitempty"`
	ResourcesVpcConfig     *vpcConfigJSON    `json:"resourcesVpcConfig,omitempty"`
	KubernetesNetworkConfig *kubeNetJSON     `json:"kubernetesNetworkConfig,omitempty"`
	CreatedAt              string            `json:"createdAt,omitempty"`
	Tags                   map[string]string `json:"tags,omitempty"`
}

type identityJSON struct {
	Oidc *oidcJSON `json:"oidc,omitempty"`
}

type oidcJSON struct {
	Issuer string `json:"issuer"`
}

type certAuthJSON struct {
	Data string `json:"data"`
}

type vpcConfigJSON struct {
	SubnetIds              []string `json:"subnetIds,omitempty"`
	SecurityGroupIds       []string `json:"securityGroupIds,omitempty"`
	ClusterSecurityGroupId string   `json:"clusterSecurityGroupId,omitempty"`
	VpcId                  string   `json:"vpcId,omitempty"`
}

type kubeNetJSON struct {
	ServiceIpv4Cidr string `json:"serviceIpv4Cidr,omitempty"`
}

type nodegroupJSON struct {
	NodegroupName string                    `json:"nodegroupName"`
	NodegroupArn  string                    `json:"nodegroupArn"`
	ClusterName   string                    `json:"clusterName"`
	Status        string                    `json:"status"`
	NodeRole      string                    `json:"nodeRole,omitempty"`
	InstanceTypes []string                  `json:"instanceTypes,omitempty"`
	AmiType       string                    `json:"amiType,omitempty"`
	DiskSize      int                       `json:"diskSize,omitempty"`
	ScalingConfig *scalingConfigJSON         `json:"scalingConfig,omitempty"`
	Subnets       []string                  `json:"subnets,omitempty"`
	Labels        map[string]string         `json:"labels,omitempty"`
	Taints        []taintJSON               `json:"taints,omitempty"`
	CapacityType  string                    `json:"capacityType,omitempty"`
	CreatedAt     string                    `json:"createdAt,omitempty"`
	Tags          map[string]string         `json:"tags,omitempty"`
}

type scalingConfigJSON struct {
	MinSize     int `json:"minSize"`
	MaxSize     int `json:"maxSize"`
	DesiredSize int `json:"desiredSize"`
}

type taintJSON struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"`
}

type fargateProfileJSON struct {
	FargateProfileName  string                `json:"fargateProfileName"`
	FargateProfileArn   string                `json:"fargateProfileArn"`
	ClusterName         string                `json:"clusterName"`
	PodExecutionRoleArn string                `json:"podExecutionRoleArn,omitempty"`
	Status              string                `json:"status"`
	Subnets             []string              `json:"subnets,omitempty"`
	Selectors           []fargateSelectorJSON `json:"selectors,omitempty"`
	CreatedAt           string                `json:"createdAt,omitempty"`
	Tags                map[string]string     `json:"tags,omitempty"`
}

type fargateSelectorJSON struct {
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type addonJSON struct {
	AddonName             string            `json:"addonName"`
	AddonArn              string            `json:"addonArn"`
	ClusterName           string            `json:"clusterName"`
	AddonVersion          string            `json:"addonVersion,omitempty"`
	Status                string            `json:"status"`
	ServiceAccountRoleArn string            `json:"serviceAccountRoleArn,omitempty"`
	CreatedAt             string            `json:"createdAt,omitempty"`
	ModifiedAt            string            `json:"modifiedAt,omitempty"`
	Tags                  map[string]string `json:"tags,omitempty"`
}

// ---- Conversion helpers ----

func clusterToJSON(c *Cluster) clusterJSON {
	j := clusterJSON{
		Name:            c.Name,
		Arn:             c.ARN,
		Version:         c.Version,
		RoleArn:         c.RoleARN,
		Status:          c.Status,
		Endpoint:        c.Endpoint,
		PlatformVersion: c.PlatformVersion,
		CreatedAt:       c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Tags:            c.Tags,
	}
	if c.CertificateAuthority != "" {
		j.CertificateAuthority = &certAuthJSON{Data: c.CertificateAuthority}
	}
	if c.OIDCIssuer != "" {
		j.Identity = &identityJSON{Oidc: &oidcJSON{Issuer: c.OIDCIssuer}}
	}
	j.ResourcesVpcConfig = &vpcConfigJSON{
		SubnetIds:              c.SubnetIDs,
		SecurityGroupIds:       c.SecurityGroupIDs,
		ClusterSecurityGroupId: c.ClusterSecurityGroupID,
		VpcId:                  c.VPCID,
	}
	if c.ServiceCIDR != "" {
		j.KubernetesNetworkConfig = &kubeNetJSON{ServiceIpv4Cidr: c.ServiceCIDR}
	}
	return j
}

func nodegroupToJSON(ng *Nodegroup) nodegroupJSON {
	j := nodegroupJSON{
		NodegroupName: ng.Name,
		NodegroupArn:  ng.ARN,
		ClusterName:   ng.ClusterName,
		Status:        ng.Status,
		NodeRole:      ng.NodeRole,
		InstanceTypes: ng.InstanceTypes,
		AmiType:       ng.AmiType,
		DiskSize:      ng.DiskSize,
		Subnets:       ng.SubnetIDs,
		Labels:        ng.Labels,
		CapacityType:  ng.CapacityType,
		CreatedAt:     ng.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Tags:          ng.Tags,
	}
	if ng.ScalingConfig != nil {
		j.ScalingConfig = &scalingConfigJSON{
			MinSize:     ng.ScalingConfig.MinSize,
			MaxSize:     ng.ScalingConfig.MaxSize,
			DesiredSize: ng.ScalingConfig.DesiredSize,
		}
	}
	if len(ng.Taints) > 0 {
		taints := make([]taintJSON, 0, len(ng.Taints))
		for _, t := range ng.Taints {
			taints = append(taints, taintJSON{Key: t.Key, Value: t.Value, Effect: t.Effect})
		}
		j.Taints = taints
	}
	return j
}

func fargateProfileToJSON(fp *FargateProfile) fargateProfileJSON {
	selectors := make([]fargateSelectorJSON, 0, len(fp.Selectors))
	for _, s := range fp.Selectors {
		selectors = append(selectors, fargateSelectorJSON{Namespace: s.Namespace, Labels: s.Labels})
	}
	return fargateProfileJSON{
		FargateProfileName:  fp.Name,
		FargateProfileArn:   fp.ARN,
		ClusterName:         fp.ClusterName,
		PodExecutionRoleArn: fp.PodExecutionRoleARN,
		Status:              fp.Status,
		Subnets:             fp.SubnetIDs,
		Selectors:           selectors,
		CreatedAt:           fp.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Tags:                fp.Tags,
	}
}

func addonToJSON(a *Addon) addonJSON {
	return addonJSON{
		AddonName:             a.Name,
		AddonArn:              a.ARN,
		ClusterName:           a.ClusterName,
		AddonVersion:          a.AddonVersion,
		Status:                a.Status,
		ServiceAccountRoleArn: a.ServiceAccountRoleARN,
		CreatedAt:             a.CreatedAt.Format("2006-01-02T15:04:05Z"),
		ModifiedAt:            a.ModifiedAt.Format("2006-01-02T15:04:05Z"),
		Tags:                  a.Tags,
	}
}

// ---- Request routing ----

// HandleRESTRequest routes EKS REST-JSON requests based on path and method.
func HandleRESTRequest(ctx *service.RequestContext, store *Store, locator ServiceLocator) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	// Tag routes: PUT/GET/DELETE /2/tags/{arn}
	const tagPrefix = "/2/tags/"
	if strings.HasPrefix(path, tagPrefix) {
		arn := path[len(tagPrefix):]
		// URL-decode the ARN since it may be encoded.
		if decoded, err := strings.CutPrefix(arn, ""); err {
			arn = decoded
		}
		switch method {
		case http.MethodPost:
			return handleTagResource(ctx, store, arn)
		case http.MethodDelete:
			return handleUntagResource(ctx, store, r, arn)
		case http.MethodGet:
			return handleListTagsForResource(ctx, store, arn)
		}
		return eksNotImplemented()
	}

	// Cluster routes
	const clusterPrefix = "/clusters"
	if !strings.HasPrefix(path, clusterPrefix) {
		return eksNotImplemented()
	}

	rest := path[len(clusterPrefix):]

	// POST /clusters -> CreateCluster
	// GET  /clusters -> ListClusters
	if rest == "" {
		switch method {
		case http.MethodPost:
			return handleCreateCluster(ctx, store)
		case http.MethodGet:
			return handleListClusters(ctx, store)
		}
		return eksNotImplemented()
	}

	parts := strings.SplitN(strings.TrimPrefix(rest, "/"), "/", 2)
	clusterName := parts[0]

	// GET    /clusters/{name} -> DescribeCluster
	// DELETE /clusters/{name} -> DeleteCluster
	if len(parts) == 1 {
		switch method {
		case http.MethodGet:
			return handleDescribeCluster(ctx, store, clusterName)
		case http.MethodDelete:
			return handleDeleteCluster(ctx, store, clusterName)
		}
		return eksNotImplemented()
	}

	subPath := parts[1]

	// PUT /clusters/{name}/update-config -> UpdateClusterConfig
	if subPath == "update-config" && method == http.MethodPost {
		return handleUpdateClusterConfig(ctx, store, clusterName)
	}

	// Nodegroup routes: /clusters/{name}/node-groups
	if strings.HasPrefix(subPath, "node-groups") {
		ngRest := subPath[len("node-groups"):]

		if ngRest == "" {
			switch method {
			case http.MethodPost:
				return handleCreateNodegroup(ctx, store, clusterName)
			case http.MethodGet:
				return handleListNodegroups(ctx, store, clusterName)
			}
			return eksNotImplemented()
		}

		ngParts := strings.SplitN(strings.TrimPrefix(ngRest, "/"), "/", 2)
		ngName := ngParts[0]

		if len(ngParts) == 1 {
			switch method {
			case http.MethodGet:
				return handleDescribeNodegroup(ctx, store, clusterName, ngName)
			case http.MethodDelete:
				return handleDeleteNodegroup(ctx, store, clusterName, ngName)
			}
			return eksNotImplemented()
		}

		if ngParts[1] == "update-config" && method == http.MethodPost {
			return handleUpdateNodegroupConfig(ctx, store, clusterName, ngName)
		}
		return eksNotImplemented()
	}

	// Fargate profile routes: /clusters/{name}/fargate-profiles
	if strings.HasPrefix(subPath, "fargate-profiles") {
		fpRest := subPath[len("fargate-profiles"):]

		if fpRest == "" {
			switch method {
			case http.MethodPost:
				return handleCreateFargateProfile(ctx, store, clusterName)
			case http.MethodGet:
				return handleListFargateProfiles(ctx, store, clusterName)
			}
			return eksNotImplemented()
		}

		fpName := strings.TrimPrefix(fpRest, "/")
		switch method {
		case http.MethodGet:
			return handleDescribeFargateProfile(ctx, store, clusterName, fpName)
		case http.MethodDelete:
			return handleDeleteFargateProfile(ctx, store, clusterName, fpName)
		}
		return eksNotImplemented()
	}

	// Addon routes: /clusters/{name}/addons
	if strings.HasPrefix(subPath, "addons") {
		addonRest := subPath[len("addons"):]

		if addonRest == "" {
			switch method {
			case http.MethodPost:
				return handleCreateAddon(ctx, store, clusterName)
			case http.MethodGet:
				return handleListAddons(ctx, store, clusterName)
			}
			return eksNotImplemented()
		}

		addonName := strings.TrimPrefix(addonRest, "/")
		switch method {
		case http.MethodGet:
			return handleDescribeAddon(ctx, store, clusterName, addonName)
		case http.MethodDelete:
			return handleDeleteAddon(ctx, store, clusterName, addonName)
		}
		return eksNotImplemented()
	}

	return eksNotImplemented()
}

// ---- Cluster handlers ----

type createClusterRequest struct {
	Name               string            `json:"name"`
	Version            string            `json:"version"`
	RoleArn            string            `json:"roleArn"`
	ResourcesVpcConfig *vpcConfigReq     `json:"resourcesVpcConfig"`
	KubernetesNetworkConfig *kubeNetReq  `json:"kubernetesNetworkConfig"`
	Tags               map[string]string `json:"tags"`
}

type vpcConfigReq struct {
	SubnetIds        []string `json:"subnetIds"`
	SecurityGroupIds []string `json:"securityGroupIds"`
}

type kubeNetReq struct {
	ServiceIpv4Cidr string `json:"serviceIpv4Cidr"`
}

func handleCreateCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createClusterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Cluster name is required."))
	}

	var subnetIDs, sgIDs []string
	var vpcID string
	if req.ResourcesVpcConfig != nil {
		subnetIDs = req.ResourcesVpcConfig.SubnetIds
		sgIDs = req.ResourcesVpcConfig.SecurityGroupIds
	}
	var serviceCIDR string
	if req.KubernetesNetworkConfig != nil {
		serviceCIDR = req.KubernetesNetworkConfig.ServiceIpv4Cidr
	}

	c, ok := store.CreateCluster(req.Name, req.Version, req.RoleArn, vpcID, serviceCIDR, subnetIDs, sgIDs, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceInUseException",
			"Cluster already exists with name: "+req.Name, http.StatusConflict))
	}

	return jsonCreated(map[string]any{"cluster": clusterToJSON(c)})
}

func handleDescribeCluster(ctx *service.RequestContext, store *Store, name string) (*service.Response, error) {
	c, ok := store.GetCluster(name)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No cluster found for name: "+name, http.StatusNotFound))
	}

	return jsonOK(map[string]any{"cluster": clusterToJSON(c)})
}

func handleListClusters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	names := store.ListClusters()
	return jsonOK(map[string]any{"clusters": names})
}

func handleDeleteCluster(ctx *service.RequestContext, store *Store, name string) (*service.Response, error) {
	c, ok := store.DeleteCluster(name)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No cluster found for name: "+name, http.StatusNotFound))
	}

	return jsonOK(map[string]any{"cluster": clusterToJSON(c)})
}

type updateClusterConfigRequest struct {
	ResourcesVpcConfig *vpcConfigReq `json:"resourcesVpcConfig"`
	Version            string        `json:"version"`
}

func handleUpdateClusterConfig(ctx *service.RequestContext, store *Store, name string) (*service.Response, error) {
	var req updateClusterConfigRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	var subnetIDs, sgIDs []string
	if req.ResourcesVpcConfig != nil {
		subnetIDs = req.ResourcesVpcConfig.SubnetIds
		sgIDs = req.ResourcesVpcConfig.SecurityGroupIds
	}

	c, ok := store.UpdateClusterConfig(name, req.Version, subnetIDs, sgIDs)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No cluster found for name: "+name, http.StatusNotFound))
	}

	return jsonOK(map[string]any{
		"update": map[string]any{
			"id":     newUUID(),
			"status": "InProgress",
			"type":   "ConfigUpdate",
		},
		"cluster": clusterToJSON(c),
	})
}

// ---- Nodegroup handlers ----

type createNodegroupRequest struct {
	NodegroupName string                 `json:"nodegroupName"`
	NodeRole      string                 `json:"nodeRole"`
	InstanceTypes []string               `json:"instanceTypes"`
	AmiType       string                 `json:"amiType"`
	DiskSize      int                    `json:"diskSize"`
	ScalingConfig *scalingConfigJSON     `json:"scalingConfig"`
	Subnets       []string               `json:"subnets"`
	Labels        map[string]string      `json:"labels"`
	Taints        []taintJSON            `json:"taints"`
	CapacityType  string                 `json:"capacityType"`
	Tags          map[string]string      `json:"tags"`
}

func handleCreateNodegroup(ctx *service.RequestContext, store *Store, clusterName string) (*service.Response, error) {
	var req createNodegroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.NodegroupName == "" {
		return jsonErr(service.ErrValidation("Nodegroup name is required."))
	}

	var scaling *NodegroupScalingConfig
	if req.ScalingConfig != nil {
		scaling = &NodegroupScalingConfig{
			MinSize:     req.ScalingConfig.MinSize,
			MaxSize:     req.ScalingConfig.MaxSize,
			DesiredSize: req.ScalingConfig.DesiredSize,
		}
	}

	var taints []Taint
	for _, t := range req.Taints {
		taints = append(taints, Taint{Key: t.Key, Value: t.Value, Effect: t.Effect})
	}

	ng, ok := store.CreateNodegroup(clusterName, req.NodegroupName, req.NodeRole, req.AmiType,
		req.CapacityType, req.InstanceTypes, req.Subnets, req.DiskSize, scaling, req.Labels, taints, req.Tags)
	if !ok {
		// Check if cluster exists.
		if _, clusterOK := store.GetCluster(clusterName); !clusterOK {
			return jsonErr(service.NewAWSError("ResourceNotFoundException",
				"No cluster found for name: "+clusterName, http.StatusNotFound))
		}
		return jsonErr(service.NewAWSError("ResourceInUseException",
			"Nodegroup already exists with name: "+req.NodegroupName, http.StatusConflict))
	}

	return jsonCreated(map[string]any{"nodegroup": nodegroupToJSON(ng)})
}

func handleDescribeNodegroup(ctx *service.RequestContext, store *Store, clusterName, ngName string) (*service.Response, error) {
	ng, ok := store.GetNodegroup(clusterName, ngName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No managed node group found for name: "+ngName, http.StatusNotFound))
	}

	return jsonOK(map[string]any{"nodegroup": nodegroupToJSON(ng)})
}

func handleListNodegroups(ctx *service.RequestContext, store *Store, clusterName string) (*service.Response, error) {
	names := store.ListNodegroups(clusterName)
	if names == nil {
		names = []string{}
	}
	return jsonOK(map[string]any{"nodegroups": names})
}

func handleDeleteNodegroup(ctx *service.RequestContext, store *Store, clusterName, ngName string) (*service.Response, error) {
	ng, ok := store.DeleteNodegroup(clusterName, ngName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No managed node group found for name: "+ngName, http.StatusNotFound))
	}

	return jsonOK(map[string]any{"nodegroup": nodegroupToJSON(ng)})
}

type updateNodegroupConfigRequest struct {
	ScalingConfig *scalingConfigJSON `json:"scalingConfig"`
	Labels        *labelsUpdateJSON  `json:"labels"`
	Taints        *taintsUpdateJSON  `json:"taints"`
}

type labelsUpdateJSON struct {
	AddOrUpdateLabels map[string]string `json:"addOrUpdateLabels"`
	RemoveLabels      []string          `json:"removeLabels"`
}

type taintsUpdateJSON struct {
	AddOrUpdateTaints []taintJSON `json:"addOrUpdateTaints"`
	RemoveTaints      []taintJSON `json:"removeTaints"`
}

func handleUpdateNodegroupConfig(ctx *service.RequestContext, store *Store, clusterName, ngName string) (*service.Response, error) {
	var req updateNodegroupConfigRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	var scaling *NodegroupScalingConfig
	if req.ScalingConfig != nil {
		scaling = &NodegroupScalingConfig{
			MinSize:     req.ScalingConfig.MinSize,
			MaxSize:     req.ScalingConfig.MaxSize,
			DesiredSize: req.ScalingConfig.DesiredSize,
		}
	}

	// Simplified: just pass updated labels and taints directly.
	var labels map[string]string
	if req.Labels != nil {
		labels = req.Labels.AddOrUpdateLabels
	}

	var taints []Taint
	if req.Taints != nil {
		for _, t := range req.Taints.AddOrUpdateTaints {
			taints = append(taints, Taint{Key: t.Key, Value: t.Value, Effect: t.Effect})
		}
	}

	ng, ok := store.UpdateNodegroupConfig(clusterName, ngName, scaling, labels, taints)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No managed node group found for name: "+ngName, http.StatusNotFound))
	}

	return jsonOK(map[string]any{
		"update": map[string]any{
			"id":     newUUID(),
			"status": "InProgress",
			"type":   "ConfigUpdate",
		},
		"nodegroup": nodegroupToJSON(ng),
	})
}

// ---- Fargate Profile handlers ----

type createFargateProfileRequest struct {
	FargateProfileName  string                `json:"fargateProfileName"`
	PodExecutionRoleArn string                `json:"podExecutionRoleArn"`
	Subnets             []string              `json:"subnets"`
	Selectors           []fargateSelectorJSON `json:"selectors"`
	Tags                map[string]string     `json:"tags"`
}

func handleCreateFargateProfile(ctx *service.RequestContext, store *Store, clusterName string) (*service.Response, error) {
	var req createFargateProfileRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.FargateProfileName == "" {
		return jsonErr(service.ErrValidation("Fargate profile name is required."))
	}

	selectors := make([]FargateSelector, 0, len(req.Selectors))
	for _, s := range req.Selectors {
		selectors = append(selectors, FargateSelector{Namespace: s.Namespace, Labels: s.Labels})
	}

	fp, ok := store.CreateFargateProfile(clusterName, req.FargateProfileName, req.PodExecutionRoleArn,
		req.Subnets, selectors, req.Tags)
	if !ok {
		if _, clusterOK := store.GetCluster(clusterName); !clusterOK {
			return jsonErr(service.NewAWSError("ResourceNotFoundException",
				"No cluster found for name: "+clusterName, http.StatusNotFound))
		}
		return jsonErr(service.NewAWSError("ResourceInUseException",
			"Fargate profile already exists with name: "+req.FargateProfileName, http.StatusConflict))
	}

	return jsonCreated(map[string]any{"fargateProfile": fargateProfileToJSON(fp)})
}

func handleDescribeFargateProfile(ctx *service.RequestContext, store *Store, clusterName, profileName string) (*service.Response, error) {
	fp, ok := store.GetFargateProfile(clusterName, profileName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No Fargate profile found for name: "+profileName, http.StatusNotFound))
	}

	return jsonOK(map[string]any{"fargateProfile": fargateProfileToJSON(fp)})
}

func handleListFargateProfiles(ctx *service.RequestContext, store *Store, clusterName string) (*service.Response, error) {
	names := store.ListFargateProfiles(clusterName)
	if names == nil {
		names = []string{}
	}
	return jsonOK(map[string]any{"fargateProfileNames": names})
}

func handleDeleteFargateProfile(ctx *service.RequestContext, store *Store, clusterName, profileName string) (*service.Response, error) {
	fp, ok := store.DeleteFargateProfile(clusterName, profileName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No Fargate profile found for name: "+profileName, http.StatusNotFound))
	}

	return jsonOK(map[string]any{"fargateProfile": fargateProfileToJSON(fp)})
}

// ---- Addon handlers ----

type createAddonRequest struct {
	AddonName             string            `json:"addonName"`
	AddonVersion          string            `json:"addonVersion"`
	ServiceAccountRoleArn string            `json:"serviceAccountRoleArn"`
	Tags                  map[string]string `json:"tags"`
}

func handleCreateAddon(ctx *service.RequestContext, store *Store, clusterName string) (*service.Response, error) {
	var req createAddonRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.AddonName == "" {
		return jsonErr(service.ErrValidation("Addon name is required."))
	}

	addon, ok := store.CreateAddon(clusterName, req.AddonName, req.AddonVersion, req.ServiceAccountRoleArn, req.Tags)
	if !ok {
		if _, clusterOK := store.GetCluster(clusterName); !clusterOK {
			return jsonErr(service.NewAWSError("ResourceNotFoundException",
				"No cluster found for name: "+clusterName, http.StatusNotFound))
		}
		return jsonErr(service.NewAWSError("ResourceInUseException",
			"Addon already exists with name: "+req.AddonName, http.StatusConflict))
	}

	return jsonCreated(map[string]any{"addon": addonToJSON(addon)})
}

func handleDescribeAddon(ctx *service.RequestContext, store *Store, clusterName, addonName string) (*service.Response, error) {
	addon, ok := store.GetAddon(clusterName, addonName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No addon found for name: "+addonName, http.StatusNotFound))
	}

	return jsonOK(map[string]any{"addon": addonToJSON(addon)})
}

func handleListAddons(ctx *service.RequestContext, store *Store, clusterName string) (*service.Response, error) {
	names := store.ListAddons(clusterName)
	if names == nil {
		names = []string{}
	}
	return jsonOK(map[string]any{"addons": names})
}

func handleDeleteAddon(ctx *service.RequestContext, store *Store, clusterName, addonName string) (*service.Response, error) {
	addon, ok := store.DeleteAddon(clusterName, addonName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No addon found for name: "+addonName, http.StatusNotFound))
	}

	return jsonOK(map[string]any{"addon": addonToJSON(addon)})
}

// ---- Tag handlers ----

type tagResourceRequest struct {
	Tags map[string]string `json:"tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store, arn string) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	if !store.TagResource(arn, req.Tags) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Resource not found: "+arn, http.StatusNotFound))
	}

	return &service.Response{StatusCode: http.StatusOK, Format: service.FormatJSON, Body: map[string]any{}}, nil
}

func handleUntagResource(ctx *service.RequestContext, store *Store, r *http.Request, arn string) (*service.Response, error) {
	// Tag keys come as query parameters: tagKeys=key1&tagKeys=key2
	keys := r.URL.Query()["tagKeys"]

	if !store.UntagResource(arn, keys) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Resource not found: "+arn, http.StatusNotFound))
	}

	return &service.Response{StatusCode: http.StatusOK, Format: service.FormatJSON, Body: map[string]any{}}, nil
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store, arn string) (*service.Response, error) {
	tags, ok := store.ListTagsForResource(arn)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Resource not found: "+arn, http.StatusNotFound))
	}

	return jsonOK(map[string]any{"tags": tags})
}

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonCreated(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusCreated,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func eksNotImplemented() (*service.Response, error) {
	return jsonErr(service.NewAWSError("NotImplemented",
		"This method and path combination is not implemented by cloudmock.", http.StatusNotImplemented))
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
