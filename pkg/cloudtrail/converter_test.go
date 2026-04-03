package cloudtrail

import (
	"encoding/json"
	"io"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert_DynamoDB_CreateTable(t *testing.T) {
	event := CloudTrailEvent{
		EventSource: "dynamodb.amazonaws.com",
		EventName:   "CreateTable",
		AWSRegion:   "us-east-1",
		RequestParameters: map[string]any{
			"tableName": "Users",
			"keySchema": []any{
				map[string]any{"attributeName": "id", "keyType": "HASH"},
			},
		},
	}

	req, err := ConvertToRequest(event, "http://localhost:4566")
	require.NoError(t, err)

	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "DynamoDB_20120810.CreateTable", req.Header.Get("X-Amz-Target"))
	assert.Equal(t, "application/x-amz-json-1.0", req.Header.Get("Content-Type"))
	assert.Contains(t, req.Header.Get("Authorization"), "dynamodb")

	body, _ := io.ReadAll(req.Body)
	var params map[string]any
	err = json.Unmarshal(body, &params)
	require.NoError(t, err)
	assert.Equal(t, "Users", params["tableName"])
}

func TestConvert_S3_CreateBucket(t *testing.T) {
	event := CloudTrailEvent{
		EventSource:       "s3.amazonaws.com",
		EventName:         "CreateBucket",
		AWSRegion:         "us-east-1",
		RequestParameters: map[string]any{"bucketName": "my-bucket"},
	}

	req, err := ConvertToRequest(event, "http://localhost:4566")
	require.NoError(t, err)

	assert.Equal(t, "PUT", req.Method)
	assert.Equal(t, "http://localhost:4566/my-bucket", req.URL.String())
	assert.Contains(t, req.Header.Get("Authorization"), "s3")
}

func TestConvert_SQS_CreateQueue(t *testing.T) {
	event := CloudTrailEvent{
		EventSource:       "sqs.amazonaws.com",
		EventName:         "CreateQueue",
		AWSRegion:         "us-east-1",
		RequestParameters: map[string]any{"QueueName": "tasks"},
	}

	req, err := ConvertToRequest(event, "http://localhost:4566")
	require.NoError(t, err)

	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))

	body, _ := io.ReadAll(req.Body)
	form, err := url.ParseQuery(string(body))
	require.NoError(t, err)
	assert.Equal(t, "CreateQueue", form.Get("Action"))
	assert.Equal(t, "tasks", form.Get("QueueName"))
}

func TestConvert_SNS_CreateTopic(t *testing.T) {
	event := CloudTrailEvent{
		EventSource:       "sns.amazonaws.com",
		EventName:         "CreateTopic",
		AWSRegion:         "us-east-1",
		RequestParameters: map[string]any{"Name": "alerts"},
	}

	req, err := ConvertToRequest(event, "http://localhost:4566")
	require.NoError(t, err)

	assert.Equal(t, "POST", req.Method)

	body, _ := io.ReadAll(req.Body)
	form, err := url.ParseQuery(string(body))
	require.NoError(t, err)
	assert.Equal(t, "CreateTopic", form.Get("Action"))
	assert.Equal(t, "alerts", form.Get("Name"))
}

func TestConvert_IAM_CreateRole(t *testing.T) {
	event := CloudTrailEvent{
		EventSource: "iam.amazonaws.com",
		EventName:   "CreateRole",
		AWSRegion:   "us-east-1",
		RequestParameters: map[string]any{
			"roleName":                 "my-role",
			"assumeRolePolicyDocument": `{"Version":"2012-10-17"}`,
		},
	}

	req, err := ConvertToRequest(event, "http://localhost:4566")
	require.NoError(t, err)

	assert.Equal(t, "POST", req.Method)

	body, _ := io.ReadAll(req.Body)
	form, err := url.ParseQuery(string(body))
	require.NoError(t, err)
	assert.Equal(t, "CreateRole", form.Get("Action"))
	assert.Equal(t, "my-role", form.Get("roleName"))
}

func TestConvert_UnknownAction(t *testing.T) {
	event := CloudTrailEvent{
		EventSource: "redshift.amazonaws.com",
		EventName:   "CreateCluster",
		AWSRegion:   "us-east-1",
	}

	_, err := ConvertToRequest(event, "http://localhost:4566")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}
