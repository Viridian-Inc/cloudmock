package stub_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/neureaux/cloudmock/pkg/stub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testModel returns a ServiceModel for a fictional "widgetfactory" service
// with full CRUD, tagging, and a secondary resource type.
func testModel() *stub.ServiceModel {
	return &stub.ServiceModel{
		ServiceName:  "widgetfactory",
		Protocol:     "json",
		TargetPrefix: "WidgetFactory",
		Actions: map[string]stub.Action{
			"CreateWidget": {
				Name:         "CreateWidget",
				Type:         "create",
				ResourceType: "widget",
				InputFields: []stub.Field{
					{Name: "Name", Type: "string", Required: true},
					{Name: "Color", Type: "string", Required: false},
				},
				OutputFields: []stub.Field{
					{Name: "Name", Type: "string"},
					{Name: "Color", Type: "string"},
				},
				IdField: "WidgetId",
			},
			"DescribeWidget": {
				Name:         "DescribeWidget",
				Type:         "describe",
				ResourceType: "widget",
				InputFields: []stub.Field{
					{Name: "WidgetId", Type: "string", Required: true},
				},
				IdField: "WidgetId",
			},
			"ListWidgets": {
				Name:         "ListWidgets",
				Type:         "list",
				ResourceType: "widget",
			},
			"DeleteWidget": {
				Name:         "DeleteWidget",
				Type:         "delete",
				ResourceType: "widget",
				InputFields: []stub.Field{
					{Name: "WidgetId", Type: "string", Required: true},
				},
				IdField: "WidgetId",
			},
			"UpdateWidget": {
				Name:         "UpdateWidget",
				Type:         "update",
				ResourceType: "widget",
				InputFields: []stub.Field{
					{Name: "WidgetId", Type: "string", Required: true},
					{Name: "Color", Type: "string", Required: false},
				},
				IdField: "WidgetId",
			},
			"TagResource": {
				Name:         "TagResource",
				Type:         "tag",
				ResourceType: "widget",
				InputFields: []stub.Field{
					{Name: "ResourceArn", Type: "string", Required: true},
					{Name: "Tags", Type: "list", Required: true},
				},
			},
			"UntagResource": {
				Name:         "UntagResource",
				Type:         "untag",
				ResourceType: "widget",
				InputFields: []stub.Field{
					{Name: "ResourceArn", Type: "string", Required: true},
					{Name: "TagKeys", Type: "list", Required: true},
				},
			},
			"ListTagsForResource": {
				Name:         "ListTagsForResource",
				Type:         "listTags",
				ResourceType: "widget",
				InputFields: []stub.Field{
					{Name: "ResourceArn", Type: "string", Required: true},
				},
			},
			"RebootWidget": {
				Name:         "RebootWidget",
				Type:         "other",
				ResourceType: "widget",
			},
		},
		ResourceTypes: map[string]stub.ResourceType{
			"widget": {
				Name:       "Widget",
				IdField:    "WidgetId",
				ArnPattern: "arn:aws:widgetfactory:{region}:{account}:widget/{id}",
				Fields: []stub.Field{
					{Name: "Name", Type: "string"},
					{Name: "Color", Type: "string"},
				},
			},
		},
	}
}

// queryModel returns a model that uses the "query" (XML) protocol.
func queryModel() *stub.ServiceModel {
	return &stub.ServiceModel{
		ServiceName:  "gadgetservice",
		Protocol:     "query",
		TargetPrefix: "",
		Actions: map[string]stub.Action{
			"CreateGadget": {
				Name:         "CreateGadget",
				Type:         "create",
				ResourceType: "gadget",
				InputFields: []stub.Field{
					{Name: "Name", Type: "string", Required: true},
				},
				OutputFields: []stub.Field{
					{Name: "Name", Type: "string"},
				},
				IdField: "GadgetId",
			},
			"DescribeGadget": {
				Name:         "DescribeGadget",
				Type:         "describe",
				ResourceType: "gadget",
				InputFields: []stub.Field{
					{Name: "GadgetId", Type: "string", Required: true},
				},
				IdField: "GadgetId",
			},
		},
		ResourceTypes: map[string]stub.ResourceType{
			"gadget": {
				Name:       "Gadget",
				IdField:    "GadgetId",
				ArnPattern: "arn:aws:gadgetservice:{region}:{account}:gadget/{id}",
				Fields: []stub.Field{
					{Name: "Name", Type: "string"},
				},
			},
		},
	}
}

func jsonBody(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func decodeResponse(t *testing.T, resp *service.Response) map[string]interface{} {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

// --- CRUD Lifecycle ---

func TestStubService_CRUDLifecycle(t *testing.T) {
	svc := stub.NewStubService(testModel(), "123456789012", "us-east-1")

	// Verify Name().
	assert.Equal(t, "widgetfactory", svc.Name())

	// Create
	createResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "CreateWidget",
		Body:   jsonBody(t, map[string]string{"Name": "sprocket", "Color": "blue"}),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, createResp.StatusCode)
	assert.Equal(t, service.FormatJSON, createResp.Format)

	created := decodeResponse(t, createResp)
	widgetID, ok := created["WidgetId"].(string)
	require.True(t, ok, "expected WidgetId in create response")
	assert.NotEmpty(t, widgetID)

	arn, ok := created["Arn"].(string)
	require.True(t, ok, "expected Arn in create response")
	assert.Contains(t, arn, "arn:aws:widgetfactory:us-east-1:123456789012:widget/")

	// Output fields should be echoed back.
	assert.Equal(t, "sprocket", created["Name"])
	assert.Equal(t, "blue", created["Color"])

	// Describe
	descResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "DescribeWidget",
		Body:   jsonBody(t, map[string]string{"WidgetId": widgetID}),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, descResp.StatusCode)

	described := decodeResponse(t, descResp)
	assert.Equal(t, widgetID, described["WidgetId"])
	assert.Equal(t, "sprocket", described["Name"])
	assert.Equal(t, "blue", described["Color"])

	// Update
	updateResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "UpdateWidget",
		Body:   jsonBody(t, map[string]string{"WidgetId": widgetID, "Color": "red"}),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)

	updated := decodeResponse(t, updateResp)
	assert.Equal(t, "red", updated["Color"])
	assert.Equal(t, "sprocket", updated["Name"]) // Name unchanged.

	// List
	listResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "ListWidgets",
		Body:   []byte("{}"),
	})
	require.NoError(t, err)
	listed := decodeResponse(t, listResp)
	widgets, ok := listed["Widgets"].([]interface{})
	require.True(t, ok, "expected Widgets list")
	assert.Len(t, widgets, 1)

	// Delete
	deleteResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "DeleteWidget",
		Body:   jsonBody(t, map[string]string{"WidgetId": widgetID}),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)

	// Describe after delete should fail.
	_, err = svc.HandleRequest(&service.RequestContext{
		Action: "DescribeWidget",
		Body:   jsonBody(t, map[string]string{"WidgetId": widgetID}),
	})
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
}

// --- Tagging ---

func TestStubService_Tagging(t *testing.T) {
	svc := stub.NewStubService(testModel(), "123456789012", "us-east-1")

	// Create a widget to get an ARN.
	createResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "CreateWidget",
		Body:   jsonBody(t, map[string]string{"Name": "tagged-widget"}),
	})
	require.NoError(t, err)
	created := decodeResponse(t, createResp)
	arn := created["Arn"].(string)

	// Tag
	_, err = svc.HandleRequest(&service.RequestContext{
		Action: "TagResource",
		Body: jsonBody(t, map[string]interface{}{
			"ResourceArn": arn,
			"Tags": []map[string]string{
				{"Key": "env", "Value": "prod"},
				{"Key": "team", "Value": "platform"},
			},
		}),
	})
	require.NoError(t, err)

	// ListTags
	listTagsResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "ListTagsForResource",
		Body:   jsonBody(t, map[string]string{"ResourceArn": arn}),
	})
	require.NoError(t, err)
	tagResult := decodeResponse(t, listTagsResp)
	tags, ok := tagResult["Tags"].([]interface{})
	require.True(t, ok)
	assert.Len(t, tags, 2)

	// Untag
	_, err = svc.HandleRequest(&service.RequestContext{
		Action: "UntagResource",
		Body: jsonBody(t, map[string]interface{}{
			"ResourceArn": arn,
			"TagKeys":     []string{"team"},
		}),
	})
	require.NoError(t, err)

	// Verify only one tag remains.
	listTagsResp2, err := svc.HandleRequest(&service.RequestContext{
		Action: "ListTagsForResource",
		Body:   jsonBody(t, map[string]string{"ResourceArn": arn}),
	})
	require.NoError(t, err)
	tagResult2 := decodeResponse(t, listTagsResp2)
	tags2, ok := tagResult2["Tags"].([]interface{})
	require.True(t, ok)
	assert.Len(t, tags2, 1)
}

// --- Unknown Action ---

func TestStubService_UnknownAction(t *testing.T) {
	svc := stub.NewStubService(testModel(), "123456789012", "us-east-1")

	_, err := svc.HandleRequest(&service.RequestContext{
		Action: "NonExistentAction",
		Body:   []byte("{}"),
	})
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
	assert.Equal(t, http.StatusBadRequest, awsErr.StatusCode())
}

// --- Missing Required Field ---

func TestStubService_MissingRequiredField(t *testing.T) {
	svc := stub.NewStubService(testModel(), "123456789012", "us-east-1")

	_, err := svc.HandleRequest(&service.RequestContext{
		Action: "CreateWidget",
		Body:   jsonBody(t, map[string]string{"Color": "green"}), // missing Name
	})
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "MissingParameter", awsErr.Code)
	assert.Contains(t, awsErr.Message, "Name")
}

// --- "other" Action Type ---

func TestStubService_OtherActionType(t *testing.T) {
	svc := stub.NewStubService(testModel(), "123456789012", "us-east-1")

	resp, err := svc.HandleRequest(&service.RequestContext{
		Action: "RebootWidget",
		Body:   []byte("{}"),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- XML Protocol ---

func TestStubService_XMLProtocol(t *testing.T) {
	svc := stub.NewStubService(queryModel(), "123456789012", "eu-west-1")

	assert.Equal(t, "gadgetservice", svc.Name())

	// Create using form-encoded body.
	createResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "CreateGadget",
		Body:   []byte("Name=turbine"),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, createResp.StatusCode)
	assert.Equal(t, service.FormatXML, createResp.Format)

	created := decodeResponse(t, createResp)
	gadgetID, ok := created["GadgetId"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, gadgetID)

	// Describe
	descResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "DescribeGadget",
		Body:   []byte("GadgetId=" + gadgetID),
	})
	require.NoError(t, err)
	assert.Equal(t, service.FormatXML, descResp.Format)

	described := decodeResponse(t, descResp)
	assert.Equal(t, gadgetID, described["GadgetId"])
	assert.Equal(t, "turbine", described["Name"])
}

// --- Actions() ---

func TestStubService_Actions(t *testing.T) {
	svc := stub.NewStubService(testModel(), "123456789012", "us-east-1")
	actions := svc.Actions()

	// Should have one action per model action.
	assert.Len(t, actions, len(testModel().Actions))

	// Check that IAM actions are prefixed with the service name.
	for _, a := range actions {
		assert.Contains(t, a.IAMAction, "widgetfactory:")
		assert.Equal(t, http.MethodPost, a.Method)
	}
}

// --- HealthCheck ---

func TestStubService_HealthCheck(t *testing.T) {
	svc := stub.NewStubService(testModel(), "123456789012", "us-east-1")
	assert.NoError(t, svc.HealthCheck())
}

// --- Registry ---

func TestStubRegistry(t *testing.T) {
	reg := stub.NewStubRegistry()
	reg.Register(testModel())
	reg.Register(queryModel())

	services := reg.ListServices()
	assert.Len(t, services, 2)

	svc, err := reg.CreateService("widgetfactory", "123456789012", "us-east-1")
	require.NoError(t, err)
	assert.Equal(t, "widgetfactory", svc.Name())

	_, err = reg.CreateService("nonexistent", "123456789012", "us-east-1")
	assert.Error(t, err)
}

// --- Delete non-existent resource ---

func TestStubService_DeleteNotFound(t *testing.T) {
	svc := stub.NewStubService(testModel(), "123456789012", "us-east-1")

	_, err := svc.HandleRequest(&service.RequestContext{
		Action: "DeleteWidget",
		Body:   jsonBody(t, map[string]string{"WidgetId": "widget-nonexistent"}),
	})
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
	assert.Equal(t, http.StatusNotFound, awsErr.StatusCode())
}

// --- Update non-existent resource ---

func TestStubService_UpdateNotFound(t *testing.T) {
	svc := stub.NewStubService(testModel(), "123456789012", "us-east-1")

	_, err := svc.HandleRequest(&service.RequestContext{
		Action: "UpdateWidget",
		Body:   jsonBody(t, map[string]string{"WidgetId": "widget-nonexistent", "Color": "red"}),
	})
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
}

// --- Empty body ---

func TestStubService_EmptyBody(t *testing.T) {
	svc := stub.NewStubService(testModel(), "123456789012", "us-east-1")

	// ListWidgets with empty body should work.
	resp, err := svc.HandleRequest(&service.RequestContext{
		Action: "ListWidgets",
		Body:   nil,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- ResourceStore unit tests ---

func TestResourceStore_CreateAndGet(t *testing.T) {
	store := stub.NewResourceStore()
	id := store.Create("widget", "wgt", map[string]interface{}{"Name": "test"})
	assert.Equal(t, "wgt-00000001", id)

	fields, err := store.Get("widget", id)
	require.NoError(t, err)
	assert.Equal(t, "test", fields["Name"])
}

func TestResourceStore_SequentialIDs(t *testing.T) {
	store := stub.NewResourceStore()
	id1 := store.Create("widget", "wgt", nil)
	id2 := store.Create("widget", "wgt", nil)
	assert.Equal(t, "wgt-00000001", id1)
	assert.Equal(t, "wgt-00000002", id2)
}

func TestResourceStore_TagOperations(t *testing.T) {
	store := stub.NewResourceStore()
	arn := "arn:aws:test:us-east-1:123456789012:thing/abc"

	store.Tag(arn, map[string]string{"env": "prod", "team": "eng"})
	tags := store.ListTags(arn)
	assert.Equal(t, "prod", tags["env"])
	assert.Equal(t, "eng", tags["team"])

	store.Untag(arn, []string{"team"})
	tags = store.ListTags(arn)
	assert.Len(t, tags, 1)
	assert.Equal(t, "prod", tags["env"])
}

func TestBuildARN(t *testing.T) {
	arn := stub.BuildARN("arn:aws:ec2:{region}:{account}:vpc/{id}", "us-west-2", "111122223333", "vpc-abc")
	assert.Equal(t, "arn:aws:ec2:us-west-2:111122223333:vpc/vpc-abc", arn)
}
