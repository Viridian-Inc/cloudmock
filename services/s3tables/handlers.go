package s3tables

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

func handleCreateTableBucket(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required"))
	}
	bucket, err := store.CreateTableBucket(name)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("TableBucket", name))
	}
	return jsonOK(map[string]any{"arn": bucket.TableBucketARN})
}

func handleGetTableBucket(arn string, store *Store) (*service.Response, error) {
	bucket, ok := store.GetTableBucket(arn)
	if !ok {
		return jsonErr(service.ErrNotFound("TableBucket", arn))
	}
	return jsonOK(map[string]any{
		"arn":            bucket.TableBucketARN,
		"name":           bucket.Name,
		"ownerAccountId": bucket.OwnerAccountID,
		"createdAt":      bucket.CreatedAt.Format(time.RFC3339),
	})
}

func handleListTableBuckets(store *Store) (*service.Response, error) {
	buckets := store.ListTableBuckets()
	out := make([]map[string]any, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, map[string]any{
			"arn":            b.TableBucketARN,
			"name":           b.Name,
			"ownerAccountId": b.OwnerAccountID,
			"createdAt":      b.CreatedAt.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{"tableBuckets": out})
}

func handleDeleteTableBucket(arn string, store *Store) (*service.Response, error) {
	if !store.DeleteTableBucket(arn) {
		return jsonErr(service.ErrNotFound("TableBucket", arn))
	}
	return jsonNoContent()
}

func handleCreateTable(params map[string]any, bucketARN, namespace, name string, store *Store) (*service.Response, error) {
	format := str(params, "format")
	if format == "" {
		format = "ICEBERG"
	}
	table, err := store.CreateTable(bucketARN, namespace, name, format)
	if err != nil {
		return jsonErr(service.NewAWSError("ConflictException", err.Error(), http.StatusConflict))
	}
	return jsonOK(map[string]any{
		"tableARN": table.TableARN,
		"versionToken": "v1",
	})
}

func handleGetTable(bucketARN, namespace, name string, store *Store) (*service.Response, error) {
	table, ok := store.GetTable(bucketARN, namespace, name)
	if !ok {
		return jsonErr(service.ErrNotFound("Table", namespace+"/"+name))
	}
	return jsonOK(map[string]any{
		"tableARN":       table.TableARN,
		"namespace":      table.Namespace,
		"name":           table.Name,
		"tableBucketARN": table.TableBucketARN,
		"format":         table.Format,
		"type":           table.Type,
		"createdAt":      table.CreatedAt.Format(time.RFC3339),
		"modifiedAt":     table.ModifiedAt.Format(time.RFC3339),
		"versionToken":   "v1",
	})
}

func handleListTables(bucketARN string, store *Store) (*service.Response, error) {
	tables := store.ListTables(bucketARN)
	out := make([]map[string]any, 0, len(tables))
	for _, t := range tables {
		out = append(out, map[string]any{
			"tableARN":       t.TableARN,
			"namespace":      t.Namespace,
			"name":           t.Name,
			"tableBucketARN": t.TableBucketARN,
			"format":         t.Format,
			"type":           t.Type,
			"createdAt":      t.CreatedAt.Format(time.RFC3339),
			"modifiedAt":     t.ModifiedAt.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{"tables": out})
}

func handleDeleteTable(bucketARN, namespace, name string, store *Store) (*service.Response, error) {
	if !store.DeleteTable(bucketARN, namespace, name) {
		return jsonErr(service.ErrNotFound("Table", namespace+"/"+name))
	}
	return jsonNoContent()
}

func handlePutTablePolicy(params map[string]any, tableARN string, store *Store) (*service.Response, error) {
	policy := str(params, "resourcePolicy")
	if policy == "" {
		return jsonErr(service.ErrValidation("resourcePolicy is required"))
	}
	if err := store.PutTablePolicy(tableARN, policy); err != nil {
		return jsonErr(service.ErrNotFound("Table", tableARN))
	}
	return jsonOK(map[string]any{})
}

func handleGetTablePolicy(tableARN string, store *Store) (*service.Response, error) {
	p, ok := store.GetTablePolicy(tableARN)
	if !ok {
		return jsonErr(service.ErrNotFound("TablePolicy", tableARN))
	}
	return jsonOK(map[string]any{
		"resourcePolicy": p.ResourcePolicy,
	})
}

func handleDeleteTablePolicy(tableARN string, store *Store) (*service.Response, error) {
	if !store.DeleteTablePolicy(tableARN) {
		return jsonErr(service.ErrNotFound("TablePolicy", tableARN))
	}
	return jsonNoContent()
}
