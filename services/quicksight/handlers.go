package quicksight

import (
	"fmt"
	"net/http"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Response helpers ─────────────────────────────────────────────────────────

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonStatus(status int, body any) (*service.Response, error) {
	return &service.Response{StatusCode: status, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterValueException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func reqMap(ctx *service.RequestContext) (map[string]any, *service.AWSError) {
	req := map[string]any{}
	if err := parseJSON(ctx.Body, &req); err != nil {
		return nil, err
	}
	// Merge query params so callers using REST URL bindings (?awsAccountId=...) work too.
	for k, v := range ctx.Params {
		if _, ok := req[k]; !ok {
			req[k] = v
		}
	}
	return req, nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return 0
}

func getInt64(m map[string]any, key string) int64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int:
			return int64(n)
		case int64:
			return n
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getBoolPtr(m map[string]any, key string) *bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return &b
		}
	}
	return nil
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

func parseTagList(m map[string]any, key string) map[string]string {
	out := map[string]string{}
	for _, t := range getMapList(m, key) {
		k := getStr(t, "Key")
		v := getStr(t, "Value")
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func parseIdentities(m map[string]any) map[string][]string {
	out := map[string][]string{}
	if id := getMap(m, "Identities"); id != nil {
		for k, v := range id {
			if arr, ok := v.([]any); ok {
				names := []string{}
				for _, x := range arr {
					if s, ok := x.(string); ok {
						names = append(names, s)
					}
				}
				out[k] = names
			}
		}
	}
	return out
}

func rfc3339(t time.Time) string { return t.Format(time.RFC3339) }

// ── Resource → map helpers ───────────────────────────────────────────────────

func userToMap(u *StoredUser) map[string]any {
	return map[string]any{
		"Arn":                                 u.Arn,
		"UserName":                            u.UserName,
		"Email":                               u.Email,
		"Role":                                u.Role,
		"IdentityType":                        u.IdentityType,
		"Active":                              u.Active,
		"PrincipalId":                         u.PrincipalId,
		"CustomPermissionsName":               u.CustomPermissionsName,
		"ExternalLoginFederationProviderType": u.ExternalLoginFederationProviderType,
		"ExternalLoginFederationProviderUrl":  u.ExternalLoginFederationProviderUrl,
		"ExternalLoginId":                     u.ExternalLoginId,
	}
}

func groupToMap(g *StoredGroup) map[string]any {
	return map[string]any{
		"Arn":         g.Arn,
		"GroupName":   g.GroupName,
		"Description": g.Description,
		"PrincipalId": g.PrincipalId,
	}
}

func namespaceToMap(n *StoredNamespace) map[string]any {
	return map[string]any{
		"Name":           n.Name,
		"Arn":            n.Arn,
		"CapacityRegion": n.CapacityRegion,
		"CreationStatus": n.CreationStatus,
		"IdentityStore":  n.IdentityStore,
		"NamespaceError": n.NamespaceError,
	}
}

func dataSourceToMap(d *StoredDataSource) map[string]any {
	return map[string]any{
		"DataSourceId":                  d.DataSourceId,
		"Arn":                           d.Arn,
		"Name":                          d.Name,
		"Type":                          d.Type,
		"Status":                        d.Status,
		"CreatedTime":                   rfc3339(d.CreatedTime),
		"LastUpdatedTime":               rfc3339(d.LastUpdatedTime),
		"DataSourceParameters":          d.DataSourceParameters,
		"AlternateDataSourceParameters": d.AlternateDataSourceParameters,
		"VpcConnectionProperties":       d.VpcConnectionProperties,
		"SslProperties":                 d.SslProperties,
		"ErrorInfo":                     d.ErrorInfo,
		"SecretArn":                     d.SecretArn,
	}
}

func dataSetToMap(d *StoredDataSet) map[string]any {
	return map[string]any{
		"DataSetId":                          d.DataSetId,
		"Arn":                                d.Arn,
		"Name":                               d.Name,
		"CreatedTime":                        rfc3339(d.CreatedTime),
		"LastUpdatedTime":                    rfc3339(d.LastUpdatedTime),
		"PhysicalTableMap":                   d.PhysicalTableMap,
		"LogicalTableMap":                    d.LogicalTableMap,
		"OutputColumns":                      d.OutputColumns,
		"ImportMode":                         d.ImportMode,
		"ConsumedSpiceCapacityInBytes":       d.ConsumedSpiceCapacityInBytes,
		"ColumnGroups":                       d.ColumnGroups,
		"FieldFolders":                       d.FieldFolders,
		"RowLevelPermissionDataSet":          d.RowLevelPermissionDataSet,
		"RowLevelPermissionTagConfiguration": d.RowLevelPermissionTagConfiguration,
		"ColumnLevelPermissionRules":         d.ColumnLevelPermissionRules,
		"DataSetUsageConfiguration":          d.DataSetUsageConfiguration,
		"DatasetParameters":                  d.DatasetParameters,
	}
}

func dataSetSummaryMap(d *StoredDataSet) map[string]any {
	return map[string]any{
		"DataSetId":                    d.DataSetId,
		"Arn":                          d.Arn,
		"Name":                         d.Name,
		"CreatedTime":                  rfc3339(d.CreatedTime),
		"LastUpdatedTime":              rfc3339(d.LastUpdatedTime),
		"ImportMode":                   d.ImportMode,
		"RowLevelPermissionDataSet":    d.RowLevelPermissionDataSet,
		"ColumnLevelPermissionRulesApplied": len(d.ColumnLevelPermissionRules) > 0,
	}
}

func templateToMap(t *StoredTemplate) map[string]any {
	return map[string]any{
		"TemplateId":      t.TemplateId,
		"Arn":             t.Arn,
		"Name":            t.Name,
		"Version":         t.Version,
		"CreatedTime":     rfc3339(t.CreatedTime),
		"LastUpdatedTime": rfc3339(t.LastUpdatedTime),
	}
}

func dashboardToMap(d *StoredDashboard) map[string]any {
	return map[string]any{
		"DashboardId":             d.DashboardId,
		"Arn":                     d.Arn,
		"Name":                    d.Name,
		"Version":                 d.Version,
		"CreatedTime":             rfc3339(d.CreatedTime),
		"LastUpdatedTime":         rfc3339(d.LastUpdatedTime),
		"LastPublishedTime":       rfc3339(d.LastPublishedTime),
		"LinkEntities":            d.LinkEntities,
		"DashboardPublishOptions": d.DashboardPublishOptions,
	}
}

func analysisToMap(a *StoredAnalysis) map[string]any {
	return map[string]any{
		"AnalysisId":      a.AnalysisId,
		"Arn":             a.Arn,
		"Name":            a.Name,
		"Status":          a.Status,
		"Errors":          a.Errors,
		"DataSetArns":     a.DataSetArns,
		"ThemeArn":        a.ThemeArn,
		"CreatedTime":     rfc3339(a.CreatedTime),
		"LastUpdatedTime": rfc3339(a.LastUpdatedTime),
		"Sheets":          a.Sheets,
	}
}

func themeToMap(t *StoredTheme) map[string]any {
	return map[string]any{
		"ThemeId":         t.ThemeId,
		"Arn":             t.Arn,
		"Name":            t.Name,
		"Type":            t.Type,
		"Version":         t.Version,
		"CreatedTime":     rfc3339(t.CreatedTime),
		"LastUpdatedTime": rfc3339(t.LastUpdatedTime),
	}
}

func folderToMap(f *StoredFolder) map[string]any {
	return map[string]any{
		"FolderId":        f.FolderId,
		"Arn":             f.Arn,
		"Name":            f.Name,
		"FolderType":      f.FolderType,
		"FolderPath":      f.FolderPath,
		"CreatedTime":     rfc3339(f.CreatedTime),
		"LastUpdatedTime": rfc3339(f.LastUpdatedTime),
		"SharingModel":    f.SharingModel,
	}
}

func topicToMap(t *StoredTopic) map[string]any {
	return map[string]any{
		"TopicId":               t.TopicId,
		"Arn":                   t.Arn,
		"Name":                  t.Name,
		"Description":           t.Description,
		"UserExperienceVersion": t.UserExperienceVersion,
		"DataSets":              t.DataSets,
		"ConfigOptions":         t.ConfigOptions,
	}
}

func ingestionToMap(i *StoredIngestion) map[string]any {
	return map[string]any{
		"IngestionId":            i.IngestionId,
		"Arn":                    i.Arn,
		"IngestionStatus":        i.IngestionStatus,
		"RowInfo":                i.RowInfo,
		"QueueInfo":              i.QueueInfo,
		"CreatedTime":            rfc3339(i.CreatedTime),
		"IngestionTimeInSeconds": i.IngestionTimeInSeconds,
		"IngestionSizeInBytes":   i.IngestionSizeInBytes,
		"RequestSource":          i.RequestSource,
		"RequestType":            i.RequestType,
		"ErrorInfo":              i.ErrorInfo,
	}
}

func refreshScheduleToMap(r *StoredRefreshSchedule) map[string]any {
	return map[string]any{
		"ScheduleId":         r.ScheduleId,
		"Arn":                r.Arn,
		"ScheduleFrequency":  r.ScheduleFrequency,
		"StartAfterDateTime": rfc3339(r.StartAfterDateTime),
		"RefreshType":        r.RefreshType,
	}
}

func vpcConnectionToMap(v *StoredVPCConnection) map[string]any {
	return map[string]any{
		"VPCConnectionId":    v.VPCConnectionId,
		"Arn":                v.Arn,
		"Name":               v.Name,
		"VPCId":              v.VpcId,
		"SecurityGroupIds":   v.SecurityGroupIds,
		"DnsResolvers":       v.DnsResolvers,
		"Status":             v.Status,
		"AvailabilityStatus": v.AvailabilityStatus,
		"NetworkInterfaces":  v.NetworkInterfaces,
		"RoleArn":            v.RoleArn,
		"CreatedTime":        rfc3339(v.CreatedTime),
		"LastUpdatedTime":    rfc3339(v.LastUpdatedTime),
	}
}

func customPermissionsToMap(c *StoredCustomPermissions) map[string]any {
	return map[string]any{
		"CustomPermissionsName": c.CustomPermissionsName,
		"Arn":                   c.Arn,
		"Capabilities":          c.Capabilities,
	}
}

func brandToMap(b *StoredBrand) map[string]any {
	return map[string]any{
		"BrandId":           b.BrandId,
		"Arn":               b.Arn,
		"BrandName":         b.BrandName,
		"BrandStatus":       b.Status,
		"CreatedTime":       rfc3339(b.CreatedTime),
		"LastUpdatedTime":   rfc3339(b.LastUpdatedTime),
		"BrandDetail":       b.BrandDetail,
		"BrandColorPalette": b.BrandColorPalette,
		"BrandElementStyle": b.BrandElementStyle,
		"VersionStatus":     "CREATE_SUCCEEDED",
	}
}

func iamPolicyAssignmentToMap(a *StoredIAMPolicyAssignment) map[string]any {
	identities := map[string]any{}
	for k, v := range a.Identities {
		arr := make([]any, 0, len(v))
		for _, n := range v {
			arr = append(arr, n)
		}
		identities[k] = arr
	}
	return map[string]any{
		"AssignmentName":   a.AssignmentName,
		"AssignmentId":     a.AssignmentId,
		"AssignmentStatus": a.AssignmentStatus,
		"PolicyArn":        a.PolicyArn,
		"Identities":       identities,
		"AwsAccountId":     a.AwsAccountId,
	}
}

func actionConnectorToMap(a *StoredActionConnector) map[string]any {
	return map[string]any{
		"ActionConnectorId":    a.ActionConnectorId,
		"Arn":                  a.Arn,
		"Name":                 a.Name,
		"Type":                 a.Type,
		"Description":          a.Description,
		"Status":               a.Status,
		"AuthenticationConfig": a.AuthenticationConfig,
		"EnabledActions":       a.EnabledActions,
		"VpcConnectionArn":     a.VpcConnectionArn,
		"CreatedTime":          rfc3339(a.CreatedTime),
		"LastUpdatedTime":      rfc3339(a.LastUpdatedTime),
	}
}

func tagListMap(tags map[string]string) []map[string]any {
	out := make([]map[string]any, 0, len(tags))
	for k, v := range tags {
		out = append(out, map[string]any{"Key": k, "Value": v})
	}
	return out
}

func requestID() string { return "req-" + generateID() }

// ── User handlers ────────────────────────────────────────────────────────────

func handleRegisterUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	namespace := getStr(req, "Namespace")
	userName := getStr(req, "UserName")
	if userName == "" {
		userName = getStr(req, "IamArn") // best-effort
	}
	u, awsErr := store.RegisterUser(namespace, userName,
		getStr(req, "Email"), getStr(req, "UserRole"),
		getStr(req, "IdentityType"), getStr(req, "CustomPermissionsName"),
		getStr(req, "ExternalLoginFederationProviderType"),
		getStr(req, "ExternalLoginFederationProviderUrl"),
		getStr(req, "ExternalLoginId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"User":         userToMap(u),
		"UserInvitationUrl": fmt.Sprintf("https://cloudmock.test/quicksight/invite/%s", u.PrincipalId),
		"RequestId":    requestID(),
		"Status":       http.StatusCreated,
	})
}

func handleDescribeUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	u, awsErr := store.GetUser(getStr(req, "Namespace"), getStr(req, "UserName"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"User": userToMap(u), "RequestId": requestID(), "Status": 200})
}

func handleUpdateUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	u, awsErr := store.UpdateUser(getStr(req, "Namespace"), getStr(req, "UserName"),
		getStr(req, "Email"), getStr(req, "Role"),
		getStr(req, "CustomPermissionsName"),
		getStr(req, "ExternalLoginFederationProviderType"),
		getStr(req, "ExternalLoginFederationProviderUrl"),
		getStr(req, "ExternalLoginId"),
		getBool(req, "UnapplyCustomPermissions"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"User": userToMap(u), "RequestId": requestID(), "Status": 200})
}

func handleDeleteUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteUser(getStr(req, "Namespace"), getStr(req, "UserName")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDeleteUserByPrincipalId(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteUserByPrincipalId(getStr(req, "Namespace"), getStr(req, "PrincipalId")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListUsers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	users := store.ListUsers(getStr(req, "Namespace"))
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, userToMap(u))
	}
	return jsonOK(map[string]any{"UserList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleListUserGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	groups := store.ListGroupsForUser(getStr(req, "Namespace"), getStr(req, "UserName"))
	out := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		out = append(out, groupToMap(g))
	}
	return jsonOK(map[string]any{"GroupList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

// ── Group handlers ───────────────────────────────────────────────────────────

func handleCreateGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	g, awsErr := store.CreateGroup(getStr(req, "Namespace"), getStr(req, "GroupName"), getStr(req, "Description"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonStatus(http.StatusCreated, map[string]any{"Group": groupToMap(g), "RequestId": requestID(), "Status": http.StatusCreated})
}

func handleDescribeGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	g, awsErr := store.GetGroup(getStr(req, "Namespace"), getStr(req, "GroupName"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Group": groupToMap(g), "RequestId": requestID(), "Status": 200})
}

func handleUpdateGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	g, awsErr := store.UpdateGroup(getStr(req, "Namespace"), getStr(req, "GroupName"), getStr(req, "Description"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Group": groupToMap(g), "RequestId": requestID(), "Status": 200})
}

func handleDeleteGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteGroup(getStr(req, "Namespace"), getStr(req, "GroupName")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	groups := store.ListGroups(getStr(req, "Namespace"))
	out := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		out = append(out, groupToMap(g))
	}
	return jsonOK(map[string]any{"GroupList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleSearchGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	groups := store.ListGroups(getStr(req, "Namespace"))
	out := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		out = append(out, groupToMap(g))
	}
	return jsonOK(map[string]any{"GroupList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleCreateGroupMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	_, u, awsErr := store.AddGroupMember(getStr(req, "Namespace"), getStr(req, "GroupName"), getStr(req, "MemberName"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"GroupMember": map[string]any{"Arn": u.Arn, "MemberName": u.UserName},
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleDeleteGroupMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.RemoveGroupMember(getStr(req, "Namespace"), getStr(req, "GroupName"), getStr(req, "MemberName")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDescribeGroupMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	member := getStr(req, "MemberName")
	ok, awsErr := store.IsGroupMember(getStr(req, "Namespace"), getStr(req, "GroupName"), member)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if !ok {
		return jsonErr(errNotFound("GroupMembership", member))
	}
	u, _ := store.GetUser(getStr(req, "Namespace"), member)
	resp := map[string]any{"RequestId": requestID(), "Status": 200}
	if u != nil {
		resp["GroupMember"] = map[string]any{"Arn": u.Arn, "MemberName": u.UserName}
	}
	return jsonOK(resp)
}

func handleListGroupMemberships(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	members, awsErr := store.ListGroupMembers(getStr(req, "Namespace"), getStr(req, "GroupName"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	out := make([]map[string]any, 0, len(members))
	for _, m := range members {
		out = append(out, map[string]any{"Arn": m.Arn, "MemberName": m.UserName})
	}
	return jsonOK(map[string]any{"GroupMemberList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

// ── Namespace handlers ───────────────────────────────────────────────────────

func handleCreateNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	n, awsErr := store.CreateNamespace(getStr(req, "Namespace"), getStr(req, "IdentityStore"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":            n.Arn,
		"Name":           n.Name,
		"CapacityRegion": n.CapacityRegion,
		"CreationStatus": n.CreationStatus,
		"IdentityStore":  n.IdentityStore,
		"RequestId":      requestID(),
		"Status":         http.StatusCreated,
	})
}

func handleDescribeNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	n, awsErr := store.GetNamespace(getStr(req, "Namespace"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Namespace": namespaceToMap(n), "RequestId": requestID(), "Status": 200})
}

func handleDeleteNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteNamespace(getStr(req, "Namespace")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListNamespaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListNamespaces()
	out := make([]map[string]any, 0, len(list))
	for _, n := range list {
		out = append(out, namespaceToMap(n))
	}
	return jsonOK(map[string]any{"Namespaces": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

// ── DataSource handlers ──────────────────────────────────────────────────────

func handleCreateDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DataSourceId")
	d, awsErr := store.CreateDataSource(id, getStr(req, "Name"), getStr(req, "Type"),
		getMap(req, "DataSourceParameters"),
		getMap(req, "VpcConnectionProperties"),
		getMap(req, "SslProperties"),
		getMap(req, "Credentials"),
		getMapList(req, "AlternateDataSourceParameters"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if perms := getMapList(req, "Permissions"); len(perms) > 0 {
		store.UpdateDataSourcePermissions(d.Arn, perms, nil)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(d.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":          d.Arn,
		"DataSourceId": d.DataSourceId,
		"CreationStatus": d.Status,
		"RequestId":    requestID(),
		"Status":       http.StatusCreated,
	})
}

func handleDescribeDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.GetDataSource(getStr(req, "DataSourceId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"DataSource": dataSourceToMap(d), "RequestId": requestID(), "Status": 200})
}

func handleUpdateDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.UpdateDataSource(getStr(req, "DataSourceId"), getStr(req, "Name"),
		getMap(req, "DataSourceParameters"),
		getMap(req, "VpcConnectionProperties"),
		getMap(req, "SslProperties"),
		getMap(req, "Credentials"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":          d.Arn,
		"DataSourceId": d.DataSourceId,
		"UpdateStatus": d.Status,
		"RequestId":    requestID(),
		"Status":       200,
	})
}

func handleDeleteDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DataSourceId")
	d, _ := store.GetDataSource(id)
	if awsErr := store.DeleteDataSource(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"DataSourceId": id,
		"RequestId":    requestID(),
		"Status":       200,
	}
	if d != nil {
		resp["Arn"] = d.Arn
	}
	return jsonOK(resp)
}

func handleListDataSources(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListDataSources()
	out := make([]map[string]any, 0, len(list))
	for _, d := range list {
		out = append(out, dataSourceToMap(d))
	}
	return jsonOK(map[string]any{"DataSources": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleSearchDataSources(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListDataSources()
	out := make([]map[string]any, 0, len(list))
	for _, d := range list {
		out = append(out, map[string]any{
			"DataSourceId": d.DataSourceId,
			"Arn":          d.Arn,
			"Name":         d.Name,
			"Type":         d.Type,
		})
	}
	return jsonOK(map[string]any{"DataSourceSummaries": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleDescribeDataSourcePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DataSourceId")
	d, awsErr := store.GetDataSource(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.DescribeDataSourcePermissions(d.Arn)
	return jsonOK(map[string]any{
		"DataSourceId":   d.DataSourceId,
		"DataSourceArn":  d.Arn,
		"Permissions":    perms,
		"RequestId":      requestID(),
		"Status":         200,
	})
}

func handleUpdateDataSourcePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DataSourceId")
	d, awsErr := store.GetDataSource(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.UpdateDataSourcePermissions(d.Arn, getMapList(req, "GrantPermissions"), getMapList(req, "RevokePermissions"))
	return jsonOK(map[string]any{
		"DataSourceId":   d.DataSourceId,
		"DataSourceArn":  d.Arn,
		"Permissions":    perms,
		"RequestId":      requestID(),
		"Status":         200,
	})
}

// ── DataSet handlers ─────────────────────────────────────────────────────────

func handleCreateDataSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.CreateDataSet(
		getStr(req, "DataSetId"),
		getStr(req, "Name"),
		getStr(req, "ImportMode"),
		getMap(req, "PhysicalTableMap"),
		getMap(req, "LogicalTableMap"),
		getMapList(req, "ColumnGroups"),
		getMap(req, "FieldFolders"),
		getMap(req, "RowLevelPermissionDataSet"),
		getMap(req, "RowLevelPermissionTagConfiguration"),
		getMapList(req, "ColumnLevelPermissionRules"),
		getMap(req, "DataSetUsageConfiguration"),
		getMapList(req, "DatasetParameters"),
		getStrList(req, "FolderArns"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if perms := getMapList(req, "Permissions"); len(perms) > 0 {
		store.UpdateDataSetPermissions(d.Arn, perms, nil)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(d.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":         d.Arn,
		"DataSetId":   d.DataSetId,
		"IngestionArn": s2ingestionPlaceholder(d.Arn),
		"IngestionId":  "initial-" + d.DataSetId,
		"RequestId":    requestID(),
		"Status":       http.StatusCreated,
	})
}

func s2ingestionPlaceholder(datasetArn string) string {
	return datasetArn + "/ingestion/initial"
}

func handleDescribeDataSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.GetDataSet(getStr(req, "DataSetId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"DataSet": dataSetToMap(d), "RequestId": requestID(), "Status": 200})
}

func handleUpdateDataSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.UpdateDataSet(
		getStr(req, "DataSetId"),
		getStr(req, "Name"),
		getStr(req, "ImportMode"),
		getMap(req, "PhysicalTableMap"),
		getMap(req, "LogicalTableMap"),
		getMapList(req, "ColumnGroups"),
		getMap(req, "FieldFolders"),
		getMap(req, "RowLevelPermissionDataSet"),
		getMap(req, "RowLevelPermissionTagConfiguration"),
		getMapList(req, "ColumnLevelPermissionRules"),
		getMap(req, "DataSetUsageConfiguration"),
		getMapList(req, "DatasetParameters"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":       d.Arn,
		"DataSetId": d.DataSetId,
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleDeleteDataSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DataSetId")
	d, _ := store.GetDataSet(id)
	if awsErr := store.DeleteDataSet(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{"DataSetId": id, "RequestId": requestID(), "Status": 200}
	if d != nil {
		resp["Arn"] = d.Arn
	}
	return jsonOK(resp)
}

func handleListDataSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListDataSets()
	out := make([]map[string]any, 0, len(list))
	for _, d := range list {
		out = append(out, dataSetSummaryMap(d))
	}
	return jsonOK(map[string]any{"DataSetSummaries": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleSearchDataSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListDataSets()
	out := make([]map[string]any, 0, len(list))
	for _, d := range list {
		out = append(out, dataSetSummaryMap(d))
	}
	return jsonOK(map[string]any{"DataSetSummaries": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleDescribeDataSetPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DataSetId")
	d, awsErr := store.GetDataSet(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.DescribeDataSetPermissions(d.Arn)
	return jsonOK(map[string]any{
		"DataSetId":   d.DataSetId,
		"DataSetArn":  d.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleUpdateDataSetPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DataSetId")
	d, awsErr := store.GetDataSet(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.UpdateDataSetPermissions(d.Arn, getMapList(req, "GrantPermissions"), getMapList(req, "RevokePermissions"))
	return jsonOK(map[string]any{
		"DataSetId":   d.DataSetId,
		"DataSetArn":  d.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleDescribeDataSetRefreshProperties(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	props, awsErr := store.GetDataSetRefreshProperties(getStr(req, "DataSetId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if props == nil {
		props = map[string]any{}
	}
	return jsonOK(map[string]any{
		"DataSetRefreshProperties": props,
		"RequestId":                requestID(),
		"Status":                   200,
	})
}

func handlePutDataSetRefreshProperties(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.PutDataSetRefreshProperties(getStr(req, "DataSetId"), getMap(req, "DataSetRefreshProperties")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDeleteDataSetRefreshProperties(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteDataSetRefreshProperties(getStr(req, "DataSetId")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

// ── Template handlers ────────────────────────────────────────────────────────

func handleCreateTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "TemplateId")
	version := getMap(req, "SourceEntity")
	if version == nil {
		version = map[string]any{}
	}
	t, awsErr := store.CreateTemplate(id, getStr(req, "Name"), version, getMap(req, "Definition"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if perms := getMapList(req, "Permissions"); len(perms) > 0 {
		store.UpdateTemplatePermissions(t.Arn, perms, nil)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(t.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":              t.Arn,
		"VersionArn":       t.Arn + "/version/1",
		"TemplateId":       t.TemplateId,
		"CreationStatus":   "CREATION_SUCCESSFUL",
		"RequestId":        requestID(),
		"Status":           http.StatusCreated,
	})
}

func handleDescribeTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	t, awsErr := store.GetTemplate(getStr(req, "TemplateId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Template": templateToMap(t), "RequestId": requestID(), "Status": 200})
}

func handleDescribeTemplateDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	t, awsErr := store.GetTemplate(getStr(req, "TemplateId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Name":          t.Name,
		"TemplateId":    t.TemplateId,
		"Definition":    t.Definition,
		"ResourceStatus": "CREATION_SUCCESSFUL",
		"VersionNumber": t.VersionNumber,
		"RequestId":     requestID(),
		"Status":        200,
	})
}

func handleUpdateTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	t, awsErr := store.UpdateTemplate(getStr(req, "TemplateId"), getStr(req, "Name"),
		getMap(req, "SourceEntity"), getMap(req, "Definition"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TemplateId":     t.TemplateId,
		"Arn":            t.Arn,
		"VersionArn":     fmt.Sprintf("%s/version/%d", t.Arn, t.VersionNumber),
		"CreationStatus": "CREATION_SUCCESSFUL",
		"RequestId":      requestID(),
		"Status":         200,
	})
}

func handleDeleteTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "TemplateId")
	t, _ := store.GetTemplate(id)
	if awsErr := store.DeleteTemplate(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{"TemplateId": id, "RequestId": requestID(), "Status": 200}
	if t != nil {
		resp["Arn"] = t.Arn
	}
	return jsonOK(resp)
}

func handleListTemplates(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListTemplates()
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		out = append(out, templateToMap(t))
	}
	return jsonOK(map[string]any{"TemplateSummaryList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleListTemplateVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	t, awsErr := store.GetTemplate(getStr(req, "TemplateId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TemplateVersionSummaryList": t.Versions,
		"NextToken":                  "",
		"RequestId":                  requestID(),
		"Status":                     200,
	})
}

func handleCreateTemplateAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.CreateTemplateAlias(getStr(req, "TemplateId"), getStr(req, "AliasName"), getInt(req, "TemplateVersionNumber"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonStatus(http.StatusCreated, map[string]any{"TemplateAlias": a, "RequestId": requestID(), "Status": http.StatusCreated})
}

func handleDescribeTemplateAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.GetTemplateAlias(getStr(req, "TemplateId"), getStr(req, "AliasName"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"TemplateAlias": a, "RequestId": requestID(), "Status": 200})
}

func handleUpdateTemplateAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.UpdateTemplateAlias(getStr(req, "TemplateId"), getStr(req, "AliasName"), getInt(req, "TemplateVersionNumber"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"TemplateAlias": a, "RequestId": requestID(), "Status": 200})
}

func handleDeleteTemplateAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteTemplateAlias(getStr(req, "TemplateId"), getStr(req, "AliasName")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200, "AliasName": getStr(req, "AliasName")})
}

func handleListTemplateAliases(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	aliases, awsErr := store.ListTemplateAliases(getStr(req, "TemplateId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TemplateAliasList": aliases,
		"NextToken":         "",
		"RequestId":         requestID(),
		"Status":            200,
	})
}

func handleDescribeTemplatePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "TemplateId")
	t, awsErr := store.GetTemplate(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.DescribeTemplatePermissions(t.Arn)
	return jsonOK(map[string]any{
		"TemplateId":  t.TemplateId,
		"TemplateArn": t.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleUpdateTemplatePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "TemplateId")
	t, awsErr := store.GetTemplate(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.UpdateTemplatePermissions(t.Arn, getMapList(req, "GrantPermissions"), getMapList(req, "RevokePermissions"))
	return jsonOK(map[string]any{
		"TemplateId":  t.TemplateId,
		"TemplateArn": t.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

// ── Dashboard handlers ───────────────────────────────────────────────────────

func handleCreateDashboard(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DashboardId")
	version := getMap(req, "SourceEntity")
	if version == nil {
		version = map[string]any{}
	}
	d, awsErr := store.CreateDashboard(id, getStr(req, "Name"),
		version,
		getMap(req, "DashboardPublishOptions"),
		getMap(req, "Definition"),
		getMap(req, "ValidationStrategy"),
		getStrList(req, "LinkEntities"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if perms := getMapList(req, "Permissions"); len(perms) > 0 {
		store.UpdateDashboardPermissions(d.Arn, perms, nil)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(d.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":              d.Arn,
		"VersionArn":       fmt.Sprintf("%s/version/%d", d.Arn, d.VersionNumber),
		"DashboardId":      d.DashboardId,
		"CreationStatus":   "CREATION_SUCCESSFUL",
		"RequestId":        requestID(),
		"Status":           http.StatusCreated,
	})
}

func handleDescribeDashboard(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.GetDashboard(getStr(req, "DashboardId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Dashboard": dashboardToMap(d), "RequestId": requestID(), "Status": 200})
}

func handleDescribeDashboardDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.GetDashboard(getStr(req, "DashboardId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"DashboardId":   d.DashboardId,
		"Name":          d.Name,
		"Definition":    d.Definition,
		"VersionNumber": d.VersionNumber,
		"ResourceStatus": "CREATION_SUCCESSFUL",
		"RequestId":     requestID(),
		"Status":        200,
	})
}

func handleUpdateDashboard(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.UpdateDashboard(getStr(req, "DashboardId"), getStr(req, "Name"),
		getMap(req, "SourceEntity"),
		getMap(req, "DashboardPublishOptions"),
		getMap(req, "Definition"),
		getMap(req, "ValidationStrategy"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":              d.Arn,
		"VersionArn":       fmt.Sprintf("%s/version/%d", d.Arn, d.VersionNumber),
		"DashboardId":      d.DashboardId,
		"CreationStatus":   "CREATION_SUCCESSFUL",
		"RequestId":        requestID(),
		"Status":           200,
	})
}

func handleUpdateDashboardLinks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.UpdateDashboardLinks(getStr(req, "DashboardId"), getStrList(req, "LinkEntities"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"DashboardArn": d.Arn,
		"LinkEntities": d.LinkEntities,
		"RequestId":    requestID(),
		"Status":       200,
	})
}

func handleUpdateDashboardPublishedVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.UpdateDashboardPublishedVersion(getStr(req, "DashboardId"), getInt(req, "VersionNumber"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"DashboardId":  d.DashboardId,
		"DashboardArn": d.Arn,
		"RequestId":    requestID(),
		"Status":       200,
	})
}

func handleDeleteDashboard(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DashboardId")
	d, _ := store.GetDashboard(id)
	if awsErr := store.DeleteDashboard(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{"DashboardId": id, "RequestId": requestID(), "Status": 200}
	if d != nil {
		resp["Arn"] = d.Arn
	}
	return jsonOK(resp)
}

func handleListDashboards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListDashboards()
	out := make([]map[string]any, 0, len(list))
	for _, d := range list {
		out = append(out, dashboardToMap(d))
	}
	return jsonOK(map[string]any{"DashboardSummaryList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleSearchDashboards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListDashboards()
	out := make([]map[string]any, 0, len(list))
	for _, d := range list {
		out = append(out, dashboardToMap(d))
	}
	return jsonOK(map[string]any{"DashboardSummaryList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleListDashboardVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	d, awsErr := store.GetDashboard(getStr(req, "DashboardId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"DashboardVersionSummaryList": d.Versions,
		"NextToken":                   "",
		"RequestId":                   requestID(),
		"Status":                      200,
	})
}

func handleDescribeDashboardPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DashboardId")
	d, awsErr := store.GetDashboard(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.DescribeDashboardPermissions(d.Arn)
	return jsonOK(map[string]any{
		"DashboardId":   d.DashboardId,
		"DashboardArn":  d.Arn,
		"Permissions":   perms,
		"RequestId":     requestID(),
		"Status":        200,
	})
}

func handleUpdateDashboardPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "DashboardId")
	d, awsErr := store.GetDashboard(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.UpdateDashboardPermissions(d.Arn, getMapList(req, "GrantPermissions"), getMapList(req, "RevokePermissions"))
	if linkPerms := getMapList(req, "GrantLinkPermissions"); len(linkPerms) > 0 {
		store.UpdateDashboardPermissions(d.Arn+"/link", linkPerms, nil)
	}
	return jsonOK(map[string]any{
		"DashboardId":          d.DashboardId,
		"DashboardArn":         d.Arn,
		"Permissions":          perms,
		"LinkSharingConfiguration": map[string]any{
			"Permissions": store.DescribeDashboardPermissions(d.Arn + "/link"),
		},
		"RequestId":            requestID(),
		"Status":               200,
	})
}

// ── Analysis handlers ────────────────────────────────────────────────────────

func handleCreateAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.CreateAnalysis(getStr(req, "AnalysisId"), getStr(req, "Name"),
		getMap(req, "SourceEntity"), getMap(req, "Definition"),
		getStr(req, "ThemeArn"), getMap(req, "Parameters"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if perms := getMapList(req, "Permissions"); len(perms) > 0 {
		store.UpdateAnalysisPermissions(a.Arn, perms, nil)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(a.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":            a.Arn,
		"AnalysisId":     a.AnalysisId,
		"CreationStatus": a.Status,
		"RequestId":      requestID(),
		"Status":         http.StatusCreated,
	})
}

func handleDescribeAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.GetAnalysis(getStr(req, "AnalysisId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Analysis": analysisToMap(a), "RequestId": requestID(), "Status": 200})
}

func handleDescribeAnalysisDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.GetAnalysis(getStr(req, "AnalysisId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"AnalysisId":     a.AnalysisId,
		"Name":           a.Name,
		"Definition":     a.Definition,
		"ResourceStatus": a.Status,
		"ThemeArn":       a.ThemeArn,
		"RequestId":      requestID(),
		"Status":         200,
	})
}

func handleUpdateAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.UpdateAnalysis(getStr(req, "AnalysisId"), getStr(req, "Name"),
		getMap(req, "SourceEntity"), getMap(req, "Definition"),
		getStr(req, "ThemeArn"), getMap(req, "Parameters"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":          a.Arn,
		"AnalysisId":   a.AnalysisId,
		"UpdateStatus": a.Status,
		"RequestId":    requestID(),
		"Status":       200,
	})
}

func handleDeleteAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "AnalysisId")
	a, _ := store.GetAnalysis(id)
	if awsErr := store.DeleteAnalysis(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"AnalysisId":         id,
		"DeletionTime":       rfc3339(time.Now().UTC()),
		"RequestId":          requestID(),
		"Status":             200,
	}
	if a != nil {
		resp["Arn"] = a.Arn
	}
	return jsonOK(resp)
}

func handleRestoreAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.RestoreAnalysis(getStr(req, "AnalysisId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"AnalysisId": a.AnalysisId,
		"Arn":        a.Arn,
		"RequestId":  requestID(),
		"Status":     200,
	})
}

func handleListAnalyses(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListAnalyses()
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, analysisToMap(a))
	}
	return jsonOK(map[string]any{"AnalysisSummaryList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleSearchAnalyses(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListAnalyses()
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, analysisToMap(a))
	}
	return jsonOK(map[string]any{"AnalysisSummaryList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleDescribeAnalysisPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "AnalysisId")
	a, awsErr := store.GetAnalysis(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.DescribeAnalysisPermissions(a.Arn)
	return jsonOK(map[string]any{
		"AnalysisId":  a.AnalysisId,
		"AnalysisArn": a.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleUpdateAnalysisPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "AnalysisId")
	a, awsErr := store.GetAnalysis(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.UpdateAnalysisPermissions(a.Arn, getMapList(req, "GrantPermissions"), getMapList(req, "RevokePermissions"))
	return jsonOK(map[string]any{
		"AnalysisId":  a.AnalysisId,
		"AnalysisArn": a.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

// ── Theme handlers ───────────────────────────────────────────────────────────

func handleCreateTheme(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	t, awsErr := store.CreateTheme(getStr(req, "ThemeId"), getStr(req, "Name"),
		getStr(req, "BaseThemeId"), getMap(req, "Configuration"), getMap(req, "VersionDescription"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if perms := getMapList(req, "Permissions"); len(perms) > 0 {
		store.UpdateThemePermissions(t.Arn, perms, nil)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(t.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":              t.Arn,
		"VersionArn":       fmt.Sprintf("%s/version/%d", t.Arn, t.VersionNumber),
		"ThemeId":          t.ThemeId,
		"CreationStatus":   "CREATION_SUCCESSFUL",
		"RequestId":        requestID(),
		"Status":           http.StatusCreated,
	})
}

func handleDescribeTheme(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	t, awsErr := store.GetTheme(getStr(req, "ThemeId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Theme": themeToMap(t), "RequestId": requestID(), "Status": 200})
}

func handleUpdateTheme(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	t, awsErr := store.UpdateTheme(getStr(req, "ThemeId"), getStr(req, "Name"),
		getStr(req, "BaseThemeId"), getMap(req, "Configuration"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"ThemeId":        t.ThemeId,
		"Arn":            t.Arn,
		"VersionArn":     fmt.Sprintf("%s/version/%d", t.Arn, t.VersionNumber),
		"CreationStatus": "CREATION_SUCCESSFUL",
		"RequestId":      requestID(),
		"Status":         200,
	})
}

func handleDeleteTheme(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "ThemeId")
	t, _ := store.GetTheme(id)
	if awsErr := store.DeleteTheme(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{"ThemeId": id, "RequestId": requestID(), "Status": 200}
	if t != nil {
		resp["Arn"] = t.Arn
	}
	return jsonOK(resp)
}

func handleListThemes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListThemes()
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		out = append(out, themeToMap(t))
	}
	return jsonOK(map[string]any{"ThemeSummaryList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleListThemeVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	t, awsErr := store.GetTheme(getStr(req, "ThemeId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"ThemeVersionSummaryList": t.Versions,
		"NextToken":               "",
		"RequestId":               requestID(),
		"Status":                  200,
	})
}

func handleCreateThemeAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.CreateThemeAlias(getStr(req, "ThemeId"), getStr(req, "AliasName"), getInt(req, "ThemeVersionNumber"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonStatus(http.StatusCreated, map[string]any{"ThemeAlias": a, "RequestId": requestID(), "Status": http.StatusCreated})
}

func handleDescribeThemeAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.GetThemeAlias(getStr(req, "ThemeId"), getStr(req, "AliasName"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ThemeAlias": a, "RequestId": requestID(), "Status": 200})
}

func handleUpdateThemeAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.UpdateThemeAlias(getStr(req, "ThemeId"), getStr(req, "AliasName"), getInt(req, "ThemeVersionNumber"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ThemeAlias": a, "RequestId": requestID(), "Status": 200})
}

func handleDeleteThemeAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteThemeAlias(getStr(req, "ThemeId"), getStr(req, "AliasName")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"AliasName": getStr(req, "AliasName"), "RequestId": requestID(), "Status": 200})
}

func handleListThemeAliases(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	aliases, awsErr := store.ListThemeAliases(getStr(req, "ThemeId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ThemeAliasList": aliases, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleDescribeThemePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "ThemeId")
	t, awsErr := store.GetTheme(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.DescribeThemePermissions(t.Arn)
	return jsonOK(map[string]any{
		"ThemeId":     t.ThemeId,
		"ThemeArn":    t.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleUpdateThemePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "ThemeId")
	t, awsErr := store.GetTheme(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.UpdateThemePermissions(t.Arn, getMapList(req, "GrantPermissions"), getMapList(req, "RevokePermissions"))
	return jsonOK(map[string]any{
		"ThemeId":     t.ThemeId,
		"ThemeArn":    t.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

// ── Folder handlers ──────────────────────────────────────────────────────────

func handleCreateFolder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	f, awsErr := store.CreateFolder(getStr(req, "FolderId"), getStr(req, "Name"),
		getStr(req, "FolderType"), getStr(req, "ParentFolderArn"), getStr(req, "SharingModel"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if perms := getMapList(req, "Permissions"); len(perms) > 0 {
		store.UpdateFolderPermissions(f.Arn, perms, nil)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(f.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":       f.Arn,
		"FolderId":  f.FolderId,
		"RequestId": requestID(),
		"Status":    http.StatusCreated,
	})
}

func handleDescribeFolder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	f, awsErr := store.GetFolder(getStr(req, "FolderId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Folder": folderToMap(f), "RequestId": requestID(), "Status": 200})
}

func handleUpdateFolder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	f, awsErr := store.UpdateFolder(getStr(req, "FolderId"), getStr(req, "Name"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"FolderId":  f.FolderId,
		"Arn":       f.Arn,
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleDeleteFolder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "FolderId")
	f, _ := store.GetFolder(id)
	if awsErr := store.DeleteFolder(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{"FolderId": id, "RequestId": requestID(), "Status": 200}
	if f != nil {
		resp["Arn"] = f.Arn
	}
	return jsonOK(resp)
}

func handleListFolders(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListFolders()
	out := make([]map[string]any, 0, len(list))
	for _, f := range list {
		out = append(out, folderToMap(f))
	}
	return jsonOK(map[string]any{"FolderSummaryList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleSearchFolders(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListFolders()
	out := make([]map[string]any, 0, len(list))
	for _, f := range list {
		out = append(out, folderToMap(f))
	}
	return jsonOK(map[string]any{"FolderSummaryList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleCreateFolderMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	m, awsErr := store.CreateFolderMembership(getStr(req, "FolderId"), getStr(req, "MemberId"), getStr(req, "MemberType"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"FolderMember": m,
		"RequestId":    requestID(),
		"Status":       http.StatusCreated,
	})
}

func handleDeleteFolderMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteFolderMembership(getStr(req, "FolderId"), getStr(req, "MemberId")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListFolderMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	members, awsErr := store.ListFolderMembers(getStr(req, "FolderId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"FolderMemberList": members, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleListFoldersForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	folders := store.ListFoldersForResource(getStr(req, "ResourceArn"))
	return jsonOK(map[string]any{"Folders": folders, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleDescribeFolderPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "FolderId")
	f, awsErr := store.GetFolder(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.DescribeFolderPermissions(f.Arn)
	return jsonOK(map[string]any{
		"FolderId":    f.FolderId,
		"Arn":         f.Arn,
		"Permissions": perms,
		"NextToken":   "",
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleDescribeFolderResolvedPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return handleDescribeFolderPermissions(ctx, store)
}

func handleUpdateFolderPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "FolderId")
	f, awsErr := store.GetFolder(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.UpdateFolderPermissions(f.Arn, getMapList(req, "GrantPermissions"), getMapList(req, "RevokePermissions"))
	return jsonOK(map[string]any{
		"FolderId":    f.FolderId,
		"Arn":         f.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

// ── Topic handlers ───────────────────────────────────────────────────────────

func handleCreateTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	topic := getMap(req, "Topic")
	if topic == nil {
		topic = req
	}
	t, awsErr := store.CreateTopic(getStr(req, "TopicId"),
		getStr(topic, "Name"),
		getStr(topic, "Description"),
		getStr(topic, "UserExperienceVersion"),
		getMapList(topic, "DataSets"),
		getMap(topic, "ConfigOptions"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(t.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":             t.Arn,
		"TopicId":         t.TopicId,
		"RefreshArn":      "",
		"RequestId":       requestID(),
		"Status":          http.StatusCreated,
	})
}

func handleDescribeTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	t, awsErr := store.GetTopic(getStr(req, "TopicId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":       t.Arn,
		"TopicId":   t.TopicId,
		"Topic":     topicToMap(t),
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleUpdateTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	topic := getMap(req, "Topic")
	if topic == nil {
		topic = req
	}
	t, awsErr := store.UpdateTopic(getStr(req, "TopicId"),
		getStr(topic, "Name"), getStr(topic, "Description"), getMapList(topic, "DataSets"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TopicId":    t.TopicId,
		"Arn":        t.Arn,
		"RefreshArn": "",
		"RequestId":  requestID(),
		"Status":     200,
	})
}

func handleDeleteTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "TopicId")
	t, _ := store.GetTopic(id)
	if awsErr := store.DeleteTopic(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{"TopicId": id, "RequestId": requestID(), "Status": 200}
	if t != nil {
		resp["Arn"] = t.Arn
	}
	return jsonOK(resp)
}

func handleListTopics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListTopics()
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		out = append(out, topicToMap(t))
	}
	return jsonOK(map[string]any{"TopicsSummaries": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleSearchTopics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListTopics()
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		out = append(out, topicToMap(t))
	}
	return jsonOK(map[string]any{"TopicSummaryList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleCreateTopicRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.SetTopicRefreshSchedule(getStr(req, "TopicId"), getMap(req, "RefreshSchedule")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"TopicId":   getStr(req, "TopicId"),
		"RequestId": requestID(),
		"Status":    http.StatusCreated,
	})
}

func handleDescribeTopicRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	sched, awsErr := store.GetTopicRefreshSchedule(getStr(req, "TopicId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TopicId":         getStr(req, "TopicId"),
		"DatasetArn":      getStr(req, "DatasetArn"),
		"RefreshSchedule": sched,
		"RequestId":       requestID(),
		"Status":          200,
	})
}

func handleUpdateTopicRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.SetTopicRefreshSchedule(getStr(req, "TopicId"), getMap(req, "RefreshSchedule")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"TopicId": getStr(req, "TopicId"), "RequestId": requestID(), "Status": 200})
}

func handleDeleteTopicRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteTopicRefreshSchedule(getStr(req, "TopicId")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListTopicRefreshSchedules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	topicId := getStr(req, "TopicId")
	sched, awsErr := store.GetTopicRefreshSchedule(topicId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	out := []map[string]any{}
	if sched != nil {
		out = append(out, sched)
	}
	return jsonOK(map[string]any{
		"TopicId":                topicId,
		"RefreshSchedules":       out,
		"NextToken":              "",
		"RequestId":              requestID(),
		"Status":                 200,
	})
}

func handleDescribeTopicRefresh(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"RefreshDetails": map[string]any{
			"RefreshArn":    fmt.Sprintf("arn:aws:quicksight:%s:%s:topic/%s/refresh/%s", store.region, store.accountID, getStr(req, "TopicId"), getStr(req, "RefreshId")),
			"RefreshStatus": "COMPLETED",
		},
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleDescribeTopicPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "TopicId")
	t, awsErr := store.GetTopic(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.DescribeTopicPermissions(t.Arn)
	return jsonOK(map[string]any{
		"TopicId":     t.TopicId,
		"TopicArn":    t.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleUpdateTopicPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "TopicId")
	t, awsErr := store.GetTopic(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.UpdateTopicPermissions(t.Arn, getMapList(req, "GrantPermissions"), getMapList(req, "RevokePermissions"))
	return jsonOK(map[string]any{
		"TopicId":     t.TopicId,
		"TopicArn":    t.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleBatchCreateTopicReviewedAnswer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	answers, awsErr := store.AddTopicReviewedAnswer(getStr(req, "TopicId"), getMapList(req, "Answers"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TopicId":               getStr(req, "TopicId"),
		"TopicArn":              "",
		"SucceededAnswers":      answers,
		"InvalidAnswers":        []any{},
		"RequestId":             requestID(),
		"Status":                200,
	})
}

func handleBatchDeleteTopicReviewedAnswer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteTopicReviewedAnswer(getStr(req, "TopicId"), getStrList(req, "AnswerIds")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TopicId":              getStr(req, "TopicId"),
		"TopicArn":             "",
		"SucceededAnswers":     getStrList(req, "AnswerIds"),
		"InvalidAnswers":       []any{},
		"RequestId":            requestID(),
		"Status":               200,
	})
}

func handleListTopicReviewedAnswers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	answers, awsErr := store.ListTopicReviewedAnswers(getStr(req, "TopicId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TopicId":   getStr(req, "TopicId"),
		"Answers":   answers,
		"RequestId": requestID(),
		"Status":    200,
	})
}

// ── IAM Policy Assignment handlers ───────────────────────────────────────────

func handleCreateIAMPolicyAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.CreateIAMPolicyAssignment(getStr(req, "Namespace"), getStr(req, "AssignmentName"),
		getStr(req, "AssignmentStatus"), getStr(req, "PolicyArn"), parseIdentities(req))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := iamPolicyAssignmentToMap(a)
	resp["RequestId"] = requestID()
	resp["Status"] = http.StatusCreated
	return jsonStatus(http.StatusCreated, resp)
}

func handleDescribeIAMPolicyAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.GetIAMPolicyAssignment(getStr(req, "Namespace"), getStr(req, "AssignmentName"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"IAMPolicyAssignment": iamPolicyAssignmentToMap(a), "RequestId": requestID(), "Status": 200})
}

func handleUpdateIAMPolicyAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.UpdateIAMPolicyAssignment(getStr(req, "Namespace"), getStr(req, "AssignmentName"),
		getStr(req, "AssignmentStatus"), getStr(req, "PolicyArn"), parseIdentities(req))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := iamPolicyAssignmentToMap(a)
	resp["RequestId"] = requestID()
	resp["Status"] = 200
	return jsonOK(resp)
}

func handleDeleteIAMPolicyAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteIAMPolicyAssignment(getStr(req, "Namespace"), getStr(req, "AssignmentName")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"AssignmentName": getStr(req, "AssignmentName"), "RequestId": requestID(), "Status": 200})
}

func handleListIAMPolicyAssignments(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	list := store.ListIAMPolicyAssignments(getStr(req, "Namespace"))
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, iamPolicyAssignmentToMap(a))
	}
	return jsonOK(map[string]any{"IAMPolicyAssignments": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleListIAMPolicyAssignmentsForUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	list := store.ListIAMPolicyAssignments(getStr(req, "Namespace"))
	userName := getStr(req, "UserName")
	out := []map[string]any{}
	for _, a := range list {
		for k, names := range a.Identities {
			if k == "User" {
				for _, n := range names {
					if n == userName {
						out = append(out, map[string]any{
							"AssignmentName": a.AssignmentName,
							"PolicyArn":      a.PolicyArn,
						})
					}
				}
			}
		}
	}
	return jsonOK(map[string]any{"ActiveAssignments": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

// ── Ingestion handlers ───────────────────────────────────────────────────────

func handleCreateIngestion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	i, awsErr := store.CreateIngestion(getStr(req, "DataSetId"), getStr(req, "IngestionId"), getStr(req, "IngestionType"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":             i.Arn,
		"IngestionId":     i.IngestionId,
		"IngestionStatus": i.IngestionStatus,
		"RequestId":       requestID(),
		"Status":          http.StatusCreated,
	})
}

func handleDescribeIngestion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	i, awsErr := store.DescribeIngestion(getStr(req, "DataSetId"), getStr(req, "IngestionId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Ingestion": ingestionToMap(i), "RequestId": requestID(), "Status": 200})
}

func handleCancelIngestion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	i, awsErr := store.CancelIngestion(getStr(req, "DataSetId"), getStr(req, "IngestionId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":         i.Arn,
		"IngestionId": i.IngestionId,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleListIngestions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	list := store.ListIngestions(getStr(req, "DataSetId"))
	out := make([]map[string]any, 0, len(list))
	for _, i := range list {
		out = append(out, ingestionToMap(i))
	}
	return jsonOK(map[string]any{"Ingestions": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

// ── RefreshSchedule handlers ─────────────────────────────────────────────────

func handleCreateRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	r, awsErr := store.CreateRefreshSchedule(getStr(req, "DataSetId"), getMap(req, "Schedule"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"ScheduleId": r.ScheduleId,
		"Arn":        r.Arn,
		"RequestId":  requestID(),
		"Status":     http.StatusCreated,
	})
}

func handleDescribeRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	r, awsErr := store.GetRefreshSchedule(getStr(req, "DataSetId"), getStr(req, "ScheduleId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RefreshSchedule": refreshScheduleToMap(r), "Arn": r.Arn, "RequestId": requestID(), "Status": 200})
}

func handleUpdateRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	r, awsErr := store.UpdateRefreshSchedule(getStr(req, "DataSetId"), getMap(req, "Schedule"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"ScheduleId": r.ScheduleId,
		"Arn":        r.Arn,
		"RequestId":  requestID(),
		"Status":     200,
	})
}

func handleDeleteRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "ScheduleId")
	if awsErr := store.DeleteRefreshSchedule(getStr(req, "DataSetId"), id); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"ScheduleId": id,
		"Arn":        store.arnRefreshSchedule(getStr(req, "DataSetId"), id),
		"RequestId":  requestID(),
		"Status":     200,
	})
}

func handleListRefreshSchedules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	list := store.ListRefreshSchedules(getStr(req, "DataSetId"))
	out := make([]map[string]any, 0, len(list))
	for _, r := range list {
		out = append(out, refreshScheduleToMap(r))
	}
	return jsonOK(map[string]any{"RefreshSchedules": out, "RequestId": requestID(), "Status": 200})
}

// ── VPC Connection handlers ──────────────────────────────────────────────────

func handleCreateVPCConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	v, awsErr := store.CreateVPCConnection(getStr(req, "VPCConnectionId"), getStr(req, "Name"),
		getStr(req, "VpcId"), getStrList(req, "SecurityGroupIds"), getStrList(req, "DnsResolvers"),
		getStr(req, "RoleArn"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(v.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":               v.Arn,
		"VPCConnectionId":   v.VPCConnectionId,
		"CreationStatus":    v.Status,
		"AvailabilityStatus": v.AvailabilityStatus,
		"RequestId":         requestID(),
		"Status":            http.StatusCreated,
	})
}

func handleDescribeVPCConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	v, awsErr := store.GetVPCConnection(getStr(req, "VPCConnectionId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"VPCConnection": vpcConnectionToMap(v), "RequestId": requestID(), "Status": 200})
}

func handleUpdateVPCConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	v, awsErr := store.UpdateVPCConnection(getStr(req, "VPCConnectionId"), getStr(req, "Name"),
		getStrList(req, "SecurityGroupIds"), getStrList(req, "DnsResolvers"), getStr(req, "RoleArn"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":             v.Arn,
		"VPCConnectionId": v.VPCConnectionId,
		"UpdateStatus":    "UPDATE_SUCCESSFUL",
		"AvailabilityStatus": v.AvailabilityStatus,
		"RequestId":       requestID(),
		"Status":          200,
	})
}

func handleDeleteVPCConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "VPCConnectionId")
	v, _ := store.GetVPCConnection(id)
	if awsErr := store.DeleteVPCConnection(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"VPCConnectionId": id,
		"DeletionStatus":  "DELETED",
		"AvailabilityStatus": "UNAVAILABLE",
		"RequestId":       requestID(),
		"Status":          200,
	}
	if v != nil {
		resp["Arn"] = v.Arn
	}
	return jsonOK(resp)
}

func handleListVPCConnections(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListVPCConnections()
	out := make([]map[string]any, 0, len(list))
	for _, v := range list {
		out = append(out, vpcConnectionToMap(v))
	}
	return jsonOK(map[string]any{"VPCConnectionSummaries": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

// ── Custom permissions handlers ──────────────────────────────────────────────

func handleCreateCustomPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	c, awsErr := store.CreateCustomPermissions(getStr(req, "CustomPermissionsName"), getMap(req, "Capabilities"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(c.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"CustomPermissions": customPermissionsToMap(c),
		"Arn":               c.Arn,
		"RequestId":         requestID(),
		"Status":            http.StatusCreated,
	})
}

func handleDescribeCustomPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	c, awsErr := store.GetCustomPermissions(getStr(req, "CustomPermissionsName"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"CustomPermissions": customPermissionsToMap(c), "RequestId": requestID(), "Status": 200})
}

func handleUpdateCustomPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	c, awsErr := store.UpdateCustomPermissions(getStr(req, "CustomPermissionsName"), getMap(req, "Capabilities"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"CustomPermissions": customPermissionsToMap(c), "RequestId": requestID(), "Status": 200})
}

func handleDeleteCustomPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteCustomPermissions(getStr(req, "CustomPermissionsName")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListCustomPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListCustomPermissions()
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		out = append(out, customPermissionsToMap(c))
	}
	return jsonOK(map[string]any{"CustomPermissionsList": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

// ── Brand handlers ───────────────────────────────────────────────────────────

func handleCreateBrand(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	def := getMap(req, "BrandDefinition")
	if def == nil {
		def = map[string]any{}
	}
	b, awsErr := store.CreateBrand(getStr(req, "BrandId"), getStr(def, "BrandName"),
		getMap(def, "Description"),
		getMap(def, "ApplicationTheme"),
		getMap(def, "LogoConfiguration"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(b.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"BrandDetail":    brandToMap(b),
		"BrandDefinition": def,
		"RequestId":      requestID(),
		"Status":         http.StatusCreated,
	})
}

func handleDescribeBrand(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	b, awsErr := store.GetBrand(getStr(req, "BrandId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"BrandDetail": brandToMap(b), "RequestId": requestID(), "Status": 200})
}

func handleUpdateBrand(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	def := getMap(req, "BrandDefinition")
	if def == nil {
		def = map[string]any{}
	}
	b, awsErr := store.UpdateBrand(getStr(req, "BrandId"), getStr(def, "BrandName"),
		getMap(def, "Description"),
		getMap(def, "ApplicationTheme"),
		getMap(def, "LogoConfiguration"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"BrandDetail": brandToMap(b), "RequestId": requestID(), "Status": 200})
}

func handleDeleteBrand(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if awsErr := store.DeleteBrand(getStr(req, "BrandId")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListBrands(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListBrands()
	out := make([]map[string]any, 0, len(list))
	for _, b := range list {
		out = append(out, map[string]any{
			"Arn":             b.Arn,
			"BrandId":         b.BrandId,
			"BrandName":       b.BrandName,
			"BrandStatus":     b.Status,
			"CreatedTime":     rfc3339(b.CreatedTime),
			"LastUpdatedTime": rfc3339(b.LastUpdatedTime),
		})
	}
	return jsonOK(map[string]any{"Brands": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleDescribeBrandPublishedVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	b, awsErr := store.GetBrand(getStr(req, "BrandId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"BrandDetail":     brandToMap(b),
		"BrandDefinition": map[string]any{},
		"VersionId":       b.PublishedVersionId,
		"RequestId":       requestID(),
		"Status":          200,
	})
}

func handleUpdateBrandPublishedVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	b, awsErr := store.UpdateBrandPublishedVersion(getStr(req, "BrandId"), getStr(req, "VersionId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"VersionId": b.PublishedVersionId,
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleDescribeBrandAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"BrandArn":  store.GetBrandAssignment(),
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleUpdateBrandAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.UpdateBrandAssignment(getStr(req, "BrandArn"))
	return jsonOK(map[string]any{
		"BrandArn":  store.GetBrandAssignment(),
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleDeleteBrandAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	store.DeleteBrandAssignment()
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

// ── Account handlers ─────────────────────────────────────────────────────────

func handleCreateAccountSubscription(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	sub := store.CreateAccountSubscription(getStr(req, "Edition"), getStr(req, "AuthenticationMethod"),
		getStr(req, "AccountName"), getStr(req, "NotificationEmail"))
	return jsonOK(map[string]any{
		"SignupResponse": map[string]any{
			"IAMUser":         true,
			"userLoginName":   getStr(req, "AdminUserName"),
			"accountName":     getStr(req, "AccountName"),
			"directoryType":   sub.DirectoryType,
		},
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleDescribeAccountSubscription(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	sub := store.GetAccountSubscription()
	settings := store.GetAccountSettings()
	return jsonOK(map[string]any{
		"AccountInfo": map[string]any{
			"AccountName":               settings.AccountName,
			"Edition":                   sub.Edition,
			"AuthenticationType":        sub.AuthenticationType,
			"AccountSubscriptionStatus": sub.AccountSubscriptionStatus,
			"NotificationEmail":         settings.NotificationEmail,
			"IAMIdentityCenterInstanceArn": sub.IAMIdentityCenterInstanceArn,
		},
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleDeleteAccountSubscription(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	store.DeleteAccountSubscription()
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDescribeAccountSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	settings := store.GetAccountSettings()
	return jsonOK(map[string]any{
		"AccountSettings": map[string]any{
			"AccountName":                  settings.AccountName,
			"Edition":                      settings.Edition,
			"DefaultNamespace":             settings.DefaultNamespace,
			"NotificationEmail":            settings.NotificationEmail,
			"PublicSharingEnabled":         settings.PublicSharingEnabled,
			"TerminationProtectionEnabled": settings.TerminationProtectionEnabled,
		},
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleUpdateAccountSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.UpdateAccountSettings(getStr(req, "DefaultNamespace"), getStr(req, "NotificationEmail"), getBoolPtr(req, "TerminationProtectionEnabled"))
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleCreateAccountCustomization(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.PutAccountCustomization(getStr(req, "Namespace"), getMap(req, "AccountCustomization"))
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":                  store.arnNamespace(getStr(req, "Namespace")),
		"AwsAccountId":         store.accountID,
		"Namespace":            getStr(req, "Namespace"),
		"AccountCustomization": store.GetAccountCustomization(getStr(req, "Namespace")),
		"RequestId":            requestID(),
		"Status":               http.StatusCreated,
	})
}

func handleDescribeAccountCustomization(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	custom := store.GetAccountCustomization(getStr(req, "Namespace"))
	return jsonOK(map[string]any{
		"Arn":                  store.arnNamespace(getStr(req, "Namespace")),
		"AwsAccountId":         store.accountID,
		"Namespace":            getStr(req, "Namespace"),
		"AccountCustomization": custom,
		"RequestId":            requestID(),
		"Status":               200,
	})
}

func handleUpdateAccountCustomization(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return handleCreateAccountCustomization(ctx, store)
}

func handleDeleteAccountCustomization(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.DeleteAccountCustomization(getStr(req, "Namespace"))
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleUpdatePublicSharingSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.UpdatePublicSharingSettings(getBool(req, "PublicSharingEnabled"))
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

// ── IP restriction & key registration ────────────────────────────────────────

func handleDescribeIpRestriction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	r := store.GetIpRestriction()
	out := map[string]any{
		"AwsAccountId": store.accountID,
		"RequestId":    requestID(),
		"Status":       200,
	}
	for k, v := range r {
		out[k] = v
	}
	return jsonOK(out)
}

func handleUpdateIpRestriction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.UpdateIpRestriction(req)
	return jsonOK(map[string]any{
		"AwsAccountId": store.accountID,
		"RequestId":    requestID(),
		"Status":       200,
	})
}

func handleDescribeKeyRegistration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	keys := store.GetKeyRegistration()
	return jsonOK(map[string]any{
		"AwsAccountId":    store.accountID,
		"KeyRegistration": keys,
		"RequestId":       requestID(),
		"Status":          200,
	})
}

func handleUpdateKeyRegistration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	keys := getMapList(req, "KeyRegistration")
	store.UpdateKeyRegistration(keys)
	return jsonOK(map[string]any{
		"FailedKeyRegistration":     []any{},
		"SuccessfulKeyRegistration": keys,
		"RequestId":                 requestID(),
		"Status":                    200,
	})
}

// ── Q personalization & search config ────────────────────────────────────────

func handleDescribeQPersonalizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"PersonalizationMode": store.GetQPersonalizationConfiguration(),
		"RequestId":           requestID(),
		"Status":              200,
	})
}

func handleUpdateQPersonalizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	mode := store.UpdateQPersonalizationConfiguration(getStr(req, "PersonalizationMode"))
	return jsonOK(map[string]any{
		"PersonalizationMode": mode,
		"RequestId":           requestID(),
		"Status":              200,
	})
}

func handleDescribeQuickSightQSearchConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"QSearchStatus": store.GetQuickSightQSearchConfiguration(),
		"RequestId":     requestID(),
		"Status":        200,
	})
}

func handleUpdateQuickSightQSearchConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	status := store.UpdateQuickSightQSearchConfiguration(getStr(req, "QSearchStatus"))
	return jsonOK(map[string]any{
		"QSearchStatus": status,
		"RequestId":     requestID(),
		"Status":        200,
	})
}

func handleDescribeDashboardsQAConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"DashboardsQAStatus": store.GetDashboardsQAConfiguration(),
		"RequestId":          requestID(),
		"Status":             200,
	})
}

func handleUpdateDashboardsQAConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	status := store.UpdateDashboardsQAConfiguration(getStr(req, "DashboardsQAStatus"))
	return jsonOK(map[string]any{
		"DashboardsQAStatus": status,
		"RequestId":          requestID(),
		"Status":             200,
	})
}

// ── Default Q business app ───────────────────────────────────────────────────

func handleDescribeDefaultQBusinessApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	app := store.GetDefaultQBusinessApplication()
	if app == nil {
		app = map[string]any{}
	}
	return jsonOK(map[string]any{
		"ApplicationId": getStr(app, "ApplicationId"),
		"RequestId":     requestID(),
		"Status":        200,
	})
}

func handleUpdateDefaultQBusinessApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.SetDefaultQBusinessApplication(req)
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDeleteDefaultQBusinessApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	store.DeleteDefaultQBusinessApplication()
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

// ── Identity propagation ─────────────────────────────────────────────────────

func handleUpdateIdentityPropagationConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	svcName := getStr(req, "Service")
	if svcName == "" {
		return jsonErr(errInvalid("Service is required"))
	}
	store.PutIdentityPropagationConfig(svcName, req)
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDeleteIdentityPropagationConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.DeleteIdentityPropagationConfig(getStr(req, "Service"))
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListIdentityPropagationConfigs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	configs := store.ListIdentityPropagationConfigs()
	return jsonOK(map[string]any{
		"Services":  configs,
		"NextToken": "",
		"RequestId": requestID(),
		"Status":    200,
	})
}

// ── Account custom permission, role ──────────────────────────────────────────

func handleDescribeAccountCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	name := store.GetAccountCustomPermission(getStr(req, "PrincipalArn"))
	return jsonOK(map[string]any{
		"AccountCustomPermissionsName": name,
		"RequestId":                    requestID(),
		"Status":                       200,
	})
}

func handleUpdateAccountCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.PutAccountCustomPermission(getStr(req, "PrincipalArn"), getStr(req, "CustomPermissionsName"))
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDeleteAccountCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.DeleteAccountCustomPermission(getStr(req, "PrincipalArn"))
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDescribeRoleCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	name := store.GetRoleCustomPermission(getStr(req, "Namespace"), getStr(req, "Role"))
	return jsonOK(map[string]any{
		"CustomPermissionsName": name,
		"RequestId":             requestID(),
		"Status":                200,
	})
}

func handleUpdateRoleCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.PutRoleCustomPermission(getStr(req, "Namespace"), getStr(req, "Role"), getStr(req, "CustomPermissionsName"))
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDeleteRoleCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.DeleteRoleCustomPermission(getStr(req, "Namespace"), getStr(req, "Role"))
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleCreateRoleMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.AddRoleMembership(getStr(req, "Namespace"), getStr(req, "Role"), getStr(req, "MemberName"))
	return jsonStatus(http.StatusCreated, map[string]any{"RequestId": requestID(), "Status": http.StatusCreated})
}

func handleDeleteRoleMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.RemoveRoleMembership(getStr(req, "Namespace"), getStr(req, "Role"), getStr(req, "MemberName"))
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListRoleMemberships(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	members := store.ListRoleMemberships(getStr(req, "Namespace"), getStr(req, "Role"))
	return jsonOK(map[string]any{
		"MembersList": members,
		"NextToken":   "",
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleUpdateUserCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if _, awsErr := store.UpdateUser(getStr(req, "Namespace"), getStr(req, "UserName"), "", "", getStr(req, "CustomPermissionsName"), "", "", "", false); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDeleteUserCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	if _, awsErr := store.UpdateUser(getStr(req, "Namespace"), getStr(req, "UserName"), "", "", "", "", "", "", true); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

// ── Tag handlers ─────────────────────────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	tags := parseTagList(req, "Tags")
	store.TagResource(getStr(req, "ResourceArn"), tags)
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.UntagResource(getStr(req, "ResourceArn"), getStrList(req, "TagKeys"))
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	tags := store.ListTagsForResource(getStr(req, "ResourceArn"))
	return jsonOK(map[string]any{"Tags": tagListMap(tags), "RequestId": requestID(), "Status": 200})
}

// ── Embed URL handlers ───────────────────────────────────────────────────────

func makeEmbedURL(prefix string) string {
	return fmt.Sprintf("https://cloudmock.test/embed/quicksight/%s?Code=%s&isauthcode=true", prefix, generateID())
}

func handleGetSessionEmbedUrl(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"EmbedUrl":  makeEmbedURL("session"),
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleGetDashboardEmbedUrl(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"EmbedUrl":  makeEmbedURL("dashboard"),
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleGenerateEmbedUrlForRegisteredUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"EmbedUrl":  makeEmbedURL("registered-user"),
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleGenerateEmbedUrlForRegisteredUserWithIdentity(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"EmbedUrl":  makeEmbedURL("registered-user-identity"),
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleGenerateEmbedUrlForAnonymousUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"EmbedUrl":  makeEmbedURL("anonymous"),
		"RequestId": requestID(),
		"Status":    200,
	})
}

func handleGetIdentityContext(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"Identity":  map[string]any{"Type": "QUICKSIGHT", "AccountId": store.accountID},
		"RequestId": requestID(),
		"Status":    200,
	})
}

// ── Asset bundle export job handlers ─────────────────────────────────────────

func handleStartAssetBundleExportJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	j, awsErr := store.StartAssetBundleExportJob(getStr(req, "AssetBundleExportJobId"),
		getStrList(req, "ResourceArns"), getStr(req, "ExportFormat"),
		getMap(req, "CloudFormationOverridePropertyConfiguration"),
		getBool(req, "IncludePermissions"), getBool(req, "IncludeTags"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":                    j.Arn,
		"AssetBundleExportJobId": j.JobId,
		"RequestId":              requestID(),
		"Status":                 200,
	})
}

func handleDescribeAssetBundleExportJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	j, awsErr := store.DescribeAssetBundleExportJob(getStr(req, "AssetBundleExportJobId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"JobStatus":              j.Status,
		"DownloadUrl":            j.DownloadUrl,
		"Errors":                 j.Errors,
		"Warnings":               j.Warnings,
		"Arn":                    j.Arn,
		"CreatedTime":            rfc3339(j.CreatedTime),
		"AssetBundleExportJobId": j.JobId,
		"AwsAccountId":           j.AwsAccountId,
		"ResourceArns":           j.ResourceArns,
		"IncludeAllDependencies": j.IncludeAllDependencies,
		"ExportFormat":           j.ExportFormat,
		"CloudFormationOverridePropertyConfiguration": j.CloudFormationOverridePropertyConfiguration,
		"IncludePermissions":     j.IncludePermissions,
		"IncludeTags":            j.IncludeTags,
		"ValidationStrategy":     j.ValidationStrategy,
		"RequestId":              requestID(),
		"Status":                 200,
	})
}

func handleListAssetBundleExportJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListAssetBundleExportJobs()
	out := make([]map[string]any, 0, len(list))
	for _, j := range list {
		out = append(out, map[string]any{
			"JobStatus":              j.Status,
			"Arn":                    j.Arn,
			"CreatedTime":            rfc3339(j.CreatedTime),
			"AssetBundleExportJobId": j.JobId,
			"IncludeAllDependencies": j.IncludeAllDependencies,
			"ExportFormat":           j.ExportFormat,
			"IncludePermissions":     j.IncludePermissions,
			"IncludeTags":            j.IncludeTags,
		})
	}
	return jsonOK(map[string]any{
		"AssetBundleExportJobSummaryList": out,
		"NextToken":                       "",
		"RequestId":                       requestID(),
		"Status":                          200,
	})
}

// ── Asset bundle import job handlers ─────────────────────────────────────────

func handleStartAssetBundleImportJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	j, awsErr := store.StartAssetBundleImportJob(getStr(req, "AssetBundleImportJobId"),
		getMap(req, "AssetBundleImportSource"),
		getMap(req, "OverrideParameters"),
		getMap(req, "OverridePermissions"),
		getMap(req, "OverrideTags"),
		getMap(req, "OverrideValidationStrategy"),
		getStr(req, "FailureAction"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":                    j.Arn,
		"AssetBundleImportJobId": j.JobId,
		"RequestId":              requestID(),
		"Status":                 200,
	})
}

func handleDescribeAssetBundleImportJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	j, awsErr := store.DescribeAssetBundleImportJob(getStr(req, "AssetBundleImportJobId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"JobStatus":               j.Status,
		"Errors":                  j.Errors,
		"RollbackErrors":          j.RollbackErrors,
		"Arn":                     j.Arn,
		"CreatedTime":             rfc3339(j.CreatedTime),
		"AssetBundleImportJobId":  j.JobId,
		"AwsAccountId":            j.AwsAccountId,
		"AssetBundleImportSource": j.AssetBundleImportSource,
		"OverrideParameters":      j.OverrideParameters,
		"FailureAction":           j.FailureAction,
		"OverridePermissions":     j.OverridePermissions,
		"OverrideTags":            j.OverrideTags,
		"OverrideValidationStrategy": j.OverrideValidationStrategy,
		"RequestId":               requestID(),
		"Status":                  200,
	})
}

func handleListAssetBundleImportJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListAssetBundleImportJobs()
	out := make([]map[string]any, 0, len(list))
	for _, j := range list {
		out = append(out, map[string]any{
			"JobStatus":              j.Status,
			"Arn":                    j.Arn,
			"CreatedTime":            rfc3339(j.CreatedTime),
			"AssetBundleImportJobId": j.JobId,
			"FailureAction":          j.FailureAction,
		})
	}
	return jsonOK(map[string]any{
		"AssetBundleImportJobSummaryList": out,
		"NextToken":                       "",
		"RequestId":                       requestID(),
		"Status":                          200,
	})
}

// ── Dashboard snapshot job handlers ──────────────────────────────────────────

func handleStartDashboardSnapshotJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	j, awsErr := store.StartDashboardSnapshotJob(getStr(req, "SnapshotJobId"), getStr(req, "DashboardId"),
		getMap(req, "UserConfiguration"), getMap(req, "SnapshotConfiguration"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":           j.Arn,
		"SnapshotJobId": j.JobId,
		"RequestId":     requestID(),
		"Status":        200,
	})
}

func handleStartDashboardSnapshotJobSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleDescribeDashboardSnapshotJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	j, awsErr := store.GetDashboardSnapshotJob(getStr(req, "SnapshotJobId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"AwsAccountId":          j.AwsAccountId,
		"DashboardId":           j.DashboardId,
		"SnapshotJobId":         j.JobId,
		"UserConfiguration":     j.UserConfiguration,
		"SnapshotConfiguration": j.SnapshotConfiguration,
		"Arn":                   j.Arn,
		"JobStatus":             j.JobStatus,
		"CreatedTime":           rfc3339(j.CreatedTime),
		"RequestId":             requestID(),
		"Status":                200,
	})
}

func handleDescribeDashboardSnapshotJobResult(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	j, awsErr := store.GetDashboardSnapshotJob(getStr(req, "SnapshotJobId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":           j.Arn,
		"JobStatus":     j.JobStatus,
		"CreatedTime":   rfc3339(j.CreatedTime),
		"Result":        j.Result,
		"RequestId":     requestID(),
		"Status":        200,
	})
}

// ── Action connector handlers ────────────────────────────────────────────────

func handleCreateActionConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.CreateActionConnector(getStr(req, "ActionConnectorId"), getStr(req, "Name"),
		getStr(req, "Type"), getStr(req, "Description"),
		getMap(req, "AuthenticationConfig"), getStrList(req, "EnabledActions"),
		getStr(req, "VpcConnectionArn"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if perms := getMapList(req, "Permissions"); len(perms) > 0 {
		store.UpdateActionConnectorPermissions(a.Arn, perms, nil)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(a.Arn, tags)
	}
	return jsonStatus(http.StatusCreated, map[string]any{
		"Arn":               a.Arn,
		"ActionConnectorId": a.ActionConnectorId,
		"RequestId":         requestID(),
		"Status":            http.StatusCreated,
	})
}

func handleDescribeActionConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.GetActionConnector(getStr(req, "ActionConnectorId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ActionConnector": actionConnectorToMap(a), "RequestId": requestID(), "Status": 200})
}

func handleUpdateActionConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	a, awsErr := store.UpdateActionConnector(getStr(req, "ActionConnectorId"), getStr(req, "Name"),
		getStr(req, "Description"), getMap(req, "AuthenticationConfig"), getStrList(req, "EnabledActions"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Arn":               a.Arn,
		"ActionConnectorId": a.ActionConnectorId,
		"RequestId":         requestID(),
		"Status":            200,
	})
}

func handleDeleteActionConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "ActionConnectorId")
	a, _ := store.GetActionConnector(id)
	if awsErr := store.DeleteActionConnector(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{"ActionConnectorId": id, "RequestId": requestID(), "Status": 200}
	if a != nil {
		resp["Arn"] = a.Arn
	}
	return jsonOK(resp)
}

func handleListActionConnectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListActionConnectors()
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, actionConnectorToMap(a))
	}
	return jsonOK(map[string]any{"ActionConnectors": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleSearchActionConnectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return handleListActionConnectors(ctx, store)
}

func handleDescribeActionConnectorPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "ActionConnectorId")
	a, awsErr := store.GetActionConnector(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.DescribeActionConnectorPermissions(a.Arn)
	return jsonOK(map[string]any{
		"ActionConnectorId":  a.ActionConnectorId,
		"ActionConnectorArn": a.Arn,
		"Permissions":        perms,
		"RequestId":          requestID(),
		"Status":             200,
	})
}

func handleUpdateActionConnectorPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "ActionConnectorId")
	a, awsErr := store.GetActionConnector(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.UpdateActionConnectorPermissions(a.Arn, getMapList(req, "GrantPermissions"), getMapList(req, "RevokePermissions"))
	return jsonOK(map[string]any{
		"ActionConnectorId":  a.ActionConnectorId,
		"ActionConnectorArn": a.Arn,
		"Permissions":        perms,
		"RequestId":          requestID(),
		"Status":             200,
	})
}

// ── Flow handlers ────────────────────────────────────────────────────────────

func handleListFlows(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListFlows()
	out := make([]map[string]any, 0, len(list))
	for _, f := range list {
		out = append(out, map[string]any{
			"FlowId":          f.FlowId,
			"Arn":             f.Arn,
			"Name":            f.Name,
			"Description":     f.Description,
			"CreatedTime":     rfc3339(f.CreatedTime),
			"LastUpdatedTime": rfc3339(f.LastUpdatedTime),
		})
	}
	return jsonOK(map[string]any{"Flows": out, "NextToken": "", "RequestId": requestID(), "Status": 200})
}

func handleSearchFlows(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return handleListFlows(ctx, store)
}

func handleGetFlowMetadata(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "FlowId")
	f, awsErr := store.GetFlow(id)
	if awsErr != nil {
		// Auto-create a synthetic flow so reads succeed for the mock.
		f = store.CreateFlow(id, "synthetic-"+id, "")
	}
	return jsonOK(map[string]any{
		"FlowId":          f.FlowId,
		"Arn":             f.Arn,
		"Name":            f.Name,
		"Description":     f.Description,
		"CreatedTime":     rfc3339(f.CreatedTime),
		"LastUpdatedTime": rfc3339(f.LastUpdatedTime),
		"RequestId":       requestID(),
		"Status":          200,
	})
}

func handleGetFlowPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "FlowId")
	f, awsErr := store.GetFlow(id)
	if awsErr != nil {
		f = store.CreateFlow(id, "synthetic-"+id, "")
	}
	perms := store.GetFlowPermissions(f.Arn)
	return jsonOK(map[string]any{
		"FlowId":      f.FlowId,
		"FlowArn":     f.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

func handleUpdateFlowPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	id := getStr(req, "FlowId")
	f, awsErr := store.GetFlow(id)
	if awsErr != nil {
		f = store.CreateFlow(id, "synthetic-"+id, "")
	}
	perms := store.UpdateFlowPermissions(f.Arn, getMapList(req, "GrantPermissions"), getMapList(req, "RevokePermissions"))
	return jsonOK(map[string]any{
		"FlowId":      f.FlowId,
		"FlowArn":     f.Arn,
		"Permissions": perms,
		"RequestId":   requestID(),
		"Status":      200,
	})
}

// ── Automation job handlers ──────────────────────────────────────────────────

func handleStartAutomationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	jobId := getStr(req, "JobId")
	if jobId == "" {
		jobId = "job-" + generateID()
	}
	j := store.StartAutomationJob(jobId, req)
	return jsonOK(map[string]any{
		"JobId":     j.JobId,
		"Status":    j.Status,
		"RequestId": requestID(),
	})
}

func handleDescribeAutomationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	j, awsErr := store.GetAutomationJob(getStr(req, "JobId"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"JobId":         j.JobId,
		"Status":        j.Status,
		"CreatedTime":   rfc3339(j.CreatedTime),
		"Configuration": j.Configuration,
		"RequestId":     requestID(),
	})
}

// ── Predict QA results ───────────────────────────────────────────────────────

func handlePredictQAResults(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"PrimaryResult":          map[string]any{},
		"AdditionalResults":      []any{},
		"RequestId":              requestID(),
		"Status":                 200,
	})
}

// ── Self upgrade handlers ────────────────────────────────────────────────────

func handleDescribeSelfUpgradeConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	cfg := store.GetSelfUpgradeConfiguration()
	return jsonOK(map[string]any{
		"SelfUpgradeConfiguration": cfg,
		"RequestId":                requestID(),
		"Status":                   200,
	})
}

func handleUpdateSelfUpgradeConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, err := reqMap(ctx)
	if err != nil {
		return jsonErr(err)
	}
	store.PutSelfUpgradeConfiguration(req)
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleUpdateSelfUpgrade(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleListSelfUpgrades(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"SelfUpgrades": store.ListSelfUpgrades(),
		"NextToken":    "",
		"RequestId":    requestID(),
		"Status":       200,
	})
}

// ── SPICE & app handlers ─────────────────────────────────────────────────────

func handleUpdateSPICECapacityConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}

func handleUpdateApplicationWithTokenExchangeGrant(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"RequestId": requestID(), "Status": 200})
}
