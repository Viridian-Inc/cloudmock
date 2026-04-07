package tier1

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type kmsSuite struct{}

func NewKMSSuite() harness.Suite { return &kmsSuite{} }
func (s *kmsSuite) Name() string { return "kms" }
func (s *kmsSuite) Tier() int    { return 1 }

func (s *kmsSuite) Operations() []harness.Operation {
	var keyID string

	newClient := func(endpoint string) (*kms.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return kms.NewFromConfig(cfg, func(o *kms.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createKey := func(ctx context.Context, endpoint string) (string, error) {
		client, err := newClient(endpoint)
		if err != nil {
			return "", err
		}
		out, err := client.CreateKey(ctx, &kms.CreateKeyInput{
			KeySpec:  kmstypes.KeySpecSymmetricDefault,
			KeyUsage: kmstypes.KeyUsageTypeEncryptDecrypt,
		})
		if err != nil {
			return "", err
		}
		return aws.ToString(out.KeyMetadata.KeyId), nil
	}

	return []harness.Operation{
		{
			Name: "CreateKey",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.CreateKey(ctx, &kms.CreateKeyInput{
					KeySpec:  kmstypes.KeySpecSymmetricDefault,
					KeyUsage: kmstypes.KeyUsageTypeEncryptDecrypt,
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateKeyOutput")}
			},
		},
		{
			Name: "Encrypt",
			Setup: func(ctx context.Context, endpoint string) error {
				id, err := createKey(ctx, endpoint)
				if err != nil {
					return err
				}
				keyID = id
				return nil
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				if keyID == "" {
					id, err := createKey(ctx, endpoint)
					if err != nil {
						return nil, err
					}
					keyID = id
				}
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.Encrypt(ctx, &kms.EncryptInput{
					KeyId:     aws.String(keyID),
					Plaintext: []byte("benchmark plaintext"),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "EncryptOutput")}
			},
		},
		{
			Name: "Decrypt",
			Setup: func(ctx context.Context, endpoint string) error {
				id, err := createKey(ctx, endpoint)
				if err != nil {
					return err
				}
				keyID = id
				return nil
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				if keyID == "" {
					id, err := createKey(ctx, endpoint)
					if err != nil {
						return nil, err
					}
					keyID = id
				}
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				encOut, err := client.Encrypt(ctx, &kms.EncryptInput{
					KeyId:     aws.String(keyID),
					Plaintext: []byte("benchmark plaintext"),
				})
				if err != nil {
					return nil, err
				}
				return client.Decrypt(ctx, &kms.DecryptInput{
					CiphertextBlob: encOut.CiphertextBlob,
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DecryptOutput")}
			},
		},
		{
			Name: "ListKeys",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListKeys(ctx, &kms.ListKeysInput{})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListKeysOutput")}
			},
		},
	}
}
