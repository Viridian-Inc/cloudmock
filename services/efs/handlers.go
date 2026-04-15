package efs

import (
	"net/http"
	"strings"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonCreated(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusCreated, Body: body, Format: service.FormatJSON}, nil
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

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		}
	}
	return 0
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getMapList(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, x := range arr {
		if xm, ok := x.(map[string]any); ok {
			out = append(out, xm)
		}
	}
	return out
}

func getStrList(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// parseTagList extracts a list of tag objects under the given key, supporting
// both Title-cased and lower-cased AWS tag shapes.
func parseTagList(m map[string]any, key string) map[string]string {
	out := make(map[string]string)
	for _, t := range getMapList(m, key) {
		k := getStr(t, "Key")
		if k == "" {
			k = getStr(t, "key")
		}
		v := getStr(t, "Value")
		if v == "" {
			v = getStr(t, "value")
		}
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func tagMapToList(tags map[string]string) []map[string]any {
	out := make([]map[string]any, 0, len(tags))
	for k, v := range tags {
		out = append(out, map[string]any{"Key": k, "Value": v})
	}
	return out
}

// pickFromPath returns the segment after `marker/` in the request path, useful
// for EFS REST endpoints like /2015-02-01/file-systems/{FileSystemId}/policy.
func pickFromPath(ctx *service.RequestContext, marker string) string {
	if ctx == nil || ctx.RawRequest == nil {
		return ""
	}
	path := ctx.RawRequest.URL.Path
	idx := strings.Index(path, marker+"/")
	if idx < 0 {
		return ""
	}
	rest := path[idx+len(marker)+1:]
	if slash := strings.Index(rest, "/"); slash >= 0 {
		return rest[:slash]
	}
	return rest
}

func paramOrField(ctx *service.RequestContext, paramKey string, m map[string]any, fieldKey string) string {
	if v := ctx.Params[paramKey]; v != "" {
		return v
	}
	return getStr(m, fieldKey)
}

func rfc3339(t time.Time) string { return t.Format(time.RFC3339) }

// ── Response shapers ────────────────────────────────────────────────────────

func fileSystemToMap(fs *StoredFileSystem) map[string]any {
	tags := tagMapToList(fs.Tags)
	out := map[string]any{
		"OwnerId":              fs.OwnerID,
		"CreationToken":        fs.CreationToken,
		"FileSystemId":         fs.FileSystemID,
		"FileSystemArn":        fs.Arn,
		"CreationTime":         fs.CreationTime.Format(time.RFC3339),
		"LifeCycleState":       fs.LifeCycleState,
		"NumberOfMountTargets": fs.NumberOfMountTargets,
		"PerformanceMode":      fs.PerformanceMode,
		"ThroughputMode":       fs.ThroughputMode,
		"Encrypted":            fs.Encrypted,
		"Tags":                 tags,
		"SizeInBytes": map[string]any{
			"Value":           fs.SizeValue,
			"ValueInIA":       fs.SizeValueInIA,
			"ValueInArchive":  fs.SizeValueInArchive,
			"ValueInStandard": fs.SizeValueInStandard,
			"Timestamp":       fs.CreationTime.Format(time.RFC3339),
		},
		"FileSystemProtection": map[string]any{
			"ReplicationOverwriteProtection": fs.ReplicationOverwriteProtection,
		},
	}
	if fs.Name != "" {
		out["Name"] = fs.Name
	}
	if fs.KmsKeyID != "" {
		out["KmsKeyId"] = fs.KmsKeyID
	}
	if fs.AvailabilityZoneName != "" {
		out["AvailabilityZoneName"] = fs.AvailabilityZoneName
	}
	if fs.AvailabilityZoneID != "" {
		out["AvailabilityZoneId"] = fs.AvailabilityZoneID
	}
	if fs.ProvisionedThroughputInMibps > 0 {
		out["ProvisionedThroughputInMibps"] = fs.ProvisionedThroughputInMibps
	}
	return out
}

func mountTargetToMap(mt *StoredMountTarget) map[string]any {
	return map[string]any{
		"OwnerId":              mt.OwnerID,
		"MountTargetId":        mt.MountTargetID,
		"FileSystemId":         mt.FileSystemID,
		"SubnetId":             mt.SubnetID,
		"LifeCycleState":       mt.LifeCycleState,
		"IpAddress":            mt.IPAddress,
		"Ipv6Address":          mt.IPv6Address,
		"NetworkInterfaceId":   mt.NetworkInterfaceID,
		"AvailabilityZoneName": mt.AvailabilityZoneName,
		"AvailabilityZoneId":   mt.AvailabilityZoneID,
		"VpcId":                mt.VpcID,
	}
}

func accessPointToMap(ap *StoredAccessPoint) map[string]any {
	out := map[string]any{
		"ClientToken":    ap.ClientToken,
		"AccessPointId":  ap.AccessPointID,
		"AccessPointArn": ap.AccessPointArn,
		"FileSystemId":   ap.FileSystemID,
		"OwnerId":        ap.OwnerID,
		"LifeCycleState": ap.LifeCycleState,
		"Tags":           tagMapToList(ap.Tags),
	}
	if ap.Name != "" {
		out["Name"] = ap.Name
	}
	if ap.PosixUser != nil {
		out["PosixUser"] = ap.PosixUser
	}
	if ap.RootDirectory != nil {
		out["RootDirectory"] = ap.RootDirectory
	}
	return out
}

func replicationToMap(r *StoredReplication) map[string]any {
	dests := make([]map[string]any, 0, len(r.Destinations))
	for _, d := range r.Destinations {
		copy := make(map[string]any, len(d))
		for k, v := range d {
			copy[k] = v
		}
		dests = append(dests, copy)
	}
	return map[string]any{
		"SourceFileSystemId":          r.SourceFileSystemID,
		"SourceFileSystemArn":         r.SourceFileSystemArn,
		"SourceFileSystemRegion":      r.SourceFileSystemRegion,
		"OriginalSourceFileSystemArn": r.OriginalSourceArn,
		"SourceFileSystemOwnerId":     r.SourceFileSystemOwnerID,
		"CreationTime":                r.CreationTime.Format(time.RFC3339),
		"Destinations":                dests,
	}
}

// ── File system handlers ────────────────────────────────────────────────────

func handleCreateFileSystem(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req == nil {
		req = map[string]any{}
	}

	creationToken := getStr(req, "CreationToken")
	if creationToken == "" {
		creationToken = newShortID()
	}

	tags := parseTagList(req, "Tags")
	name := getStr(req, "Name")
	if name == "" {
		// AWS surfaces the Name as a tag automatically when one is provided.
		if v, ok := tags["Name"]; ok {
			name = v
		}
	}

	fs := &StoredFileSystem{
		Name:                         name,
		CreationToken:                creationToken,
		PerformanceMode:              getStr(req, "PerformanceMode"),
		ThroughputMode:               getStr(req, "ThroughputMode"),
		ProvisionedThroughputInMibps: getFloat(req, "ProvisionedThroughputInMibps"),
		Encrypted:                    getBool(req, "Encrypted"),
		KmsKeyID:                     getStr(req, "KmsKeyId"),
		AvailabilityZoneName:         getStr(req, "AvailabilityZoneName"),
		Tags:                         tags,
	}
	if getBool(req, "Backup") {
		fs.BackupStatus = "ENABLED"
	}

	created, err := store.CreateFileSystem(fs)
	if err != nil {
		return jsonErr(err)
	}
	return jsonCreated(fileSystemToMap(created))
}

func handleDeleteFileSystem(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "file-systems")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	if err := store.DeleteFileSystem(id); err != nil {
		return jsonErr(err)
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleDescribeFileSystems(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	filterID := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	filterToken := paramOrField(ctx, "CreationToken", req, "CreationToken")

	list := store.ListFileSystems(filterID, filterToken)
	out := make([]map[string]any, 0, len(list))
	for _, fs := range list {
		out = append(out, fileSystemToMap(fs))
	}
	return jsonOK(map[string]any{"FileSystems": out})
}

func handleUpdateFileSystem(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "file-systems")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	fs, err := store.UpdateFileSystem(id, getStr(req, "ThroughputMode"), getFloat(req, "ProvisionedThroughputInMibps"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(fileSystemToMap(fs))
}

func handleUpdateFileSystemProtection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "file-systems")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	fs, err := store.UpdateFileSystemProtection(id, getStr(req, "ReplicationOverwriteProtection"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ReplicationOverwriteProtection": fs.ReplicationOverwriteProtection,
	})
}

// ── File system policy ──────────────────────────────────────────────────────

func handlePutFileSystemPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "file-systems")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	policy := getStr(req, "Policy")
	if policy == "" {
		return jsonErr(service.ErrValidation("Policy is required."))
	}
	fs, err := store.SetFileSystemPolicy(id, policy)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"FileSystemId": fs.FileSystemID,
		"Policy":       fs.Policy,
	})
}

func handleDescribeFileSystemPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "file-systems")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	fs, err := store.GetFileSystem(id)
	if err != nil {
		return jsonErr(err)
	}
	if fs.Policy == "" {
		return jsonErr(service.NewAWSError("PolicyNotFound",
			"No policy associated with "+id, 404))
	}
	return jsonOK(map[string]any{
		"FileSystemId": fs.FileSystemID,
		"Policy":       fs.Policy,
	})
}

func handleDeleteFileSystemPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "file-systems")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	if err := store.DeleteFileSystemPolicy(id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Backup policy ───────────────────────────────────────────────────────────

func handlePutBackupPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "file-systems")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	policy := getMap(req, "BackupPolicy")
	if policy == nil {
		return jsonErr(service.ErrValidation("BackupPolicy is required."))
	}
	status := getStr(policy, "Status")
	if status == "" {
		return jsonErr(service.ErrValidation("BackupPolicy.Status is required."))
	}
	fs, err := store.SetBackupStatus(id, status)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"BackupPolicy": map[string]any{"Status": fs.BackupStatus},
	})
}

func handleDescribeBackupPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "file-systems")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	fs, err := store.GetFileSystem(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"BackupPolicy": map[string]any{"Status": fs.BackupStatus},
	})
}

// ── Lifecycle configuration ─────────────────────────────────────────────────

func handlePutLifecycleConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "file-systems")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	policies := getMapList(req, "LifecyclePolicies")
	fs, err := store.SetLifecyclePolicies(id, policies)
	if err != nil {
		return jsonErr(err)
	}
	out := fs.LifecyclePolicies
	if out == nil {
		out = []map[string]any{}
	}
	return jsonOK(map[string]any{"LifecyclePolicies": out})
}

func handleDescribeLifecycleConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "file-systems")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	fs, err := store.GetFileSystem(id)
	if err != nil {
		return jsonErr(err)
	}
	policies := fs.LifecyclePolicies
	if policies == nil {
		policies = []map[string]any{}
	}
	return jsonOK(map[string]any{"LifecyclePolicies": policies})
}

// ── Mount target handlers ───────────────────────────────────────────────────

func handleCreateMountTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	fileSystemID := getStr(req, "FileSystemId")
	if fileSystemID == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	subnetID := getStr(req, "SubnetId")
	if subnetID == "" {
		return jsonErr(service.ErrValidation("SubnetId is required."))
	}

	mt := &StoredMountTarget{
		FileSystemID:   fileSystemID,
		SubnetID:       subnetID,
		IPAddress:      getStr(req, "IpAddress"),
		IPv6Address:    getStr(req, "Ipv6Address"),
		SecurityGroups: getStrList(req, "SecurityGroups"),
	}
	created, err := store.CreateMountTarget(mt)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(mountTargetToMap(created))
}

func handleDeleteMountTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "MountTargetId", req, "MountTargetId")
	if id == "" {
		id = pickFromPath(ctx, "mount-targets")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("MountTargetId is required."))
	}
	if err := store.DeleteMountTarget(id); err != nil {
		return jsonErr(err)
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleDescribeMountTargets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	fileSystemID := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	mountTargetID := paramOrField(ctx, "MountTargetId", req, "MountTargetId")
	accessPointID := paramOrField(ctx, "AccessPointId", req, "AccessPointId")

	// AccessPointId implies the file system the access point lives on.
	if accessPointID != "" && fileSystemID == "" {
		for _, ap := range store.ListAccessPoints("", accessPointID) {
			fileSystemID = ap.FileSystemID
		}
	}
	if fileSystemID == "" && mountTargetID == "" {
		return jsonErr(service.ErrValidation("FileSystemId or MountTargetId is required."))
	}

	list := store.ListMountTargets(fileSystemID, mountTargetID)
	out := make([]map[string]any, 0, len(list))
	for _, mt := range list {
		out = append(out, mountTargetToMap(mt))
	}
	return jsonOK(map[string]any{"MountTargets": out})
}

func handleDescribeMountTargetSecurityGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "MountTargetId", req, "MountTargetId")
	if id == "" {
		id = pickFromPath(ctx, "mount-targets")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("MountTargetId is required."))
	}
	mt, err := store.GetMountTarget(id)
	if err != nil {
		return jsonErr(err)
	}
	sgs := mt.SecurityGroups
	if sgs == nil {
		sgs = []string{}
	}
	return jsonOK(map[string]any{"SecurityGroups": sgs})
}

func handleModifyMountTargetSecurityGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "MountTargetId", req, "MountTargetId")
	if id == "" {
		id = pickFromPath(ctx, "mount-targets")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("MountTargetId is required."))
	}
	sgs := getStrList(req, "SecurityGroups")
	if err := store.SetMountTargetSecurityGroups(id, sgs); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Access point handlers ───────────────────────────────────────────────────

func handleCreateAccessPoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req == nil {
		req = map[string]any{}
	}
	fileSystemID := getStr(req, "FileSystemId")
	if fileSystemID == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	clientToken := getStr(req, "ClientToken")
	if clientToken == "" {
		clientToken = newShortID()
	}
	tags := parseTagList(req, "Tags")
	name := tags["Name"]

	ap := &StoredAccessPoint{
		ClientToken:   clientToken,
		FileSystemID:  fileSystemID,
		Name:          name,
		PosixUser:     getMap(req, "PosixUser"),
		RootDirectory: getMap(req, "RootDirectory"),
		Tags:          tags,
	}
	created, err := store.CreateAccessPoint(ap)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(accessPointToMap(created))
}

func handleDeleteAccessPoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "AccessPointId", req, "AccessPointId")
	if id == "" {
		id = pickFromPath(ctx, "access-points")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("AccessPointId is required."))
	}
	if err := store.DeleteAccessPoint(id); err != nil {
		return jsonErr(err)
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleDescribeAccessPoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	fileSystemID := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	accessPointID := paramOrField(ctx, "AccessPointId", req, "AccessPointId")

	list := store.ListAccessPoints(fileSystemID, accessPointID)
	out := make([]map[string]any, 0, len(list))
	for _, ap := range list {
		out = append(out, accessPointToMap(ap))
	}
	return jsonOK(map[string]any{"AccessPoints": out})
}

// ── Replication handlers ────────────────────────────────────────────────────

func handleCreateReplicationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	sourceID := paramOrField(ctx, "SourceFileSystemId", req, "SourceFileSystemId")
	if sourceID == "" {
		sourceID = pickFromPath(ctx, "file-systems")
	}
	if sourceID == "" {
		return jsonErr(service.ErrValidation("SourceFileSystemId is required."))
	}
	dests := getMapList(req, "Destinations")
	if len(dests) == 0 {
		return jsonErr(service.ErrValidation("Destinations is required."))
	}

	rep := &StoredReplication{
		SourceFileSystemID: sourceID,
		Destinations:       dests,
	}
	created, err := store.CreateReplication(rep)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(replicationToMap(created))
}

func handleDeleteReplicationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	sourceID := paramOrField(ctx, "SourceFileSystemId", req, "SourceFileSystemId")
	if sourceID == "" {
		sourceID = pickFromPath(ctx, "file-systems")
	}
	if sourceID == "" {
		return jsonErr(service.ErrValidation("SourceFileSystemId is required."))
	}
	if err := store.DeleteReplication(sourceID); err != nil {
		return jsonErr(err)
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleDescribeReplicationConfigurations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	sourceID := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	list := store.ListReplications(sourceID)
	out := make([]map[string]any, 0, len(list))
	for _, r := range list {
		out = append(out, replicationToMap(r))
	}
	return jsonOK(map[string]any{"Replications": out})
}

// ── Tag handlers (file system-scoped legacy API) ────────────────────────────

func handleCreateTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "create-tags")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	tags := parseTagList(req, "Tags")
	if len(tags) == 0 {
		return jsonErr(service.ErrValidation("Tags is required."))
	}
	if err := store.MergeTags(id, tags); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		id = pickFromPath(ctx, "delete-tags")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	keys := getStrList(req, "TagKeys")
	if len(keys) == 0 {
		return jsonErr(service.ErrValidation("TagKeys is required."))
	}
	if err := store.RemoveTags(id, keys); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := paramOrField(ctx, "FileSystemId", req, "FileSystemId")
	if id == "" {
		return jsonErr(service.ErrValidation("FileSystemId is required."))
	}
	fs, err := store.GetFileSystem(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Tags": tagMapToList(fs.Tags)})
}

// ── Generic resource tag handlers ───────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceID := paramOrField(ctx, "ResourceId", req, "ResourceId")
	if resourceID == "" {
		resourceID = pickFromPath(ctx, "resource-tags")
	}
	if resourceID == "" {
		return jsonErr(service.ErrValidation("ResourceId is required."))
	}
	tags := parseTagList(req, "Tags")
	if len(tags) == 0 {
		return jsonErr(service.ErrValidation("Tags is required."))
	}
	if err := store.TagResource(resourceID, tags); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceID := paramOrField(ctx, "ResourceId", req, "ResourceId")
	if resourceID == "" {
		resourceID = pickFromPath(ctx, "resource-tags")
	}
	if resourceID == "" {
		return jsonErr(service.ErrValidation("ResourceId is required."))
	}
	keys := getStrList(req, "tagKeys")
	if len(keys) == 0 {
		keys = getStrList(req, "TagKeys")
	}
	if len(keys) == 0 {
		return jsonErr(service.ErrValidation("tagKeys is required."))
	}
	if err := store.UntagResource(resourceID, keys); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceID := paramOrField(ctx, "ResourceId", req, "ResourceId")
	if resourceID == "" {
		resourceID = pickFromPath(ctx, "resource-tags")
	}
	if resourceID == "" {
		return jsonErr(service.ErrValidation("ResourceId is required."))
	}
	tags, err := store.ListTags(resourceID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Tags": tagMapToList(tags)})
}

// ── Account preferences ─────────────────────────────────────────────────────

func handleDescribeAccountPreferences(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	prefs := store.GetAccountPreferences()
	return jsonOK(map[string]any{
		"ResourceIdPreference": map[string]any{
			"ResourceIdType": prefs.ResourceIDType,
			"Resources":      prefs.Resources,
		},
	})
}

func handlePutAccountPreferences(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceIDType := getStr(req, "ResourceIdType")
	if resourceIDType == "" {
		return jsonErr(service.ErrValidation("ResourceIdType is required."))
	}
	if resourceIDType != "LONG_ID" && resourceIDType != "SHORT_ID" {
		return jsonErr(service.NewAWSError("BadRequest",
			"ResourceIdType must be LONG_ID or SHORT_ID.", http.StatusBadRequest))
	}
	prefs := store.PutAccountPreferences(resourceIDType)
	return jsonOK(map[string]any{
		"ResourceIdPreference": map[string]any{
			"ResourceIdType": prefs.ResourceIDType,
			"Resources":      prefs.Resources,
		},
	})
}
