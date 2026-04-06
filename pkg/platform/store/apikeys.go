package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neureaux/cloudmock/pkg/platform/model"
)

// APIKeyStore handles persistence for APIKey records.
type APIKeyStore struct {
	pool *pgxpool.Pool
}

// NewAPIKeyStore creates an APIKeyStore backed by the given pool.
func NewAPIKeyStore(pool *pgxpool.Pool) *APIKeyStore {
	return &APIKeyStore{pool: pool}
}

// hashKey returns a hex-encoded SHA-256 hash of plaintext.
func hashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// Create generates a new API key, stores the hash, and returns the plaintext (shown once).
// The plaintext has the form "cm_live_<32 random hex bytes>".
func (s *APIKeyStore) Create(ctx context.Context, tenantID, appID, name, role string) (plaintext string, key *model.APIKey, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate key bytes: %w", err)
	}
	plaintext = "cm_live_" + hex.EncodeToString(raw)
	prefix := plaintext[:15] // "cm_live_" + first 7 hex chars

	keyHash := hashKey(plaintext)

	const q = `
		INSERT INTO api_keys (tenant_id, app_id, key_hash, prefix, name, role)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, app_id, key_hash, prefix, name, role,
		          last_used_at, expires_at, revoked_at, created_at`

	key = &model.APIKey{}
	row := s.pool.QueryRow(ctx, q, tenantID, appID, keyHash, prefix, name, role)
	if err = row.Scan(
		&key.ID, &key.TenantID, &key.AppID, &key.KeyHash, &key.Prefix,
		&key.Name, &key.Role, &key.LastUsedAt, &key.ExpiresAt, &key.RevokedAt,
		&key.CreatedAt,
	); err != nil {
		return "", nil, fmt.Errorf("insert api key: %w", err)
	}
	return plaintext, key, nil
}

// GetByPlaintext hashes plaintext and fetches the matching active (non-revoked, non-expired) key.
func (s *APIKeyStore) GetByPlaintext(ctx context.Context, plaintext string) (*model.APIKey, error) {
	keyHash := hashKey(plaintext)

	const q = `
		SELECT id, tenant_id, app_id, key_hash, prefix, name, role,
		       last_used_at, expires_at, revoked_at, created_at
		FROM api_keys
		WHERE key_hash = $1
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > now())`

	key := &model.APIKey{}
	err := s.pool.QueryRow(ctx, q, keyHash).Scan(
		&key.ID, &key.TenantID, &key.AppID, &key.KeyHash, &key.Prefix,
		&key.Name, &key.Role, &key.LastUsedAt, &key.ExpiresAt, &key.RevokedAt,
		&key.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get api key by plaintext: %w", err)
	}
	return key, nil
}

// ListByApp returns all non-revoked keys for the given app with KeyHash cleared.
func (s *APIKeyStore) ListByApp(ctx context.Context, appID string) ([]model.APIKey, error) {
	const q = `
		SELECT id, tenant_id, app_id, prefix, name, role,
		       last_used_at, expires_at, revoked_at, created_at
		FROM api_keys
		WHERE app_id = $1
		  AND revoked_at IS NULL
		ORDER BY created_at ASC`

	rows, err := s.pool.Query(ctx, q, appID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(
			&k.ID, &k.TenantID, &k.AppID, &k.Prefix, &k.Name, &k.Role,
			&k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan api key row: %w", err)
		}
		// KeyHash intentionally left empty
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return keys, nil
}

// Revoke sets revoked_at = now() for the given key.
func (s *APIKeyStore) Revoke(ctx context.Context, id string) error {
	const q = `UPDATE api_keys SET revoked_at = now() WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchLastUsed sets last_used_at = now() for the given key.
func (s *APIKeyStore) TouchLastUsed(ctx context.Context, id string) error {
	const q = `UPDATE api_keys SET last_used_at = now() WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("touch last used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
