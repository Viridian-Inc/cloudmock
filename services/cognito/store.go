package cognito

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// userStatus represents the lifecycle status of a Cognito user.
type userStatus string

const (
	userStatusUnconfirmed       userStatus = "UNCONFIRMED"
	userStatusConfirmed         userStatus = "CONFIRMED"
	userStatusForceChangePassword userStatus = "FORCE_CHANGE_PASSWORD"
)

// UserPool holds all metadata for a Cognito User Pool.
type UserPool struct {
	Id                   string
	Name                 string
	Arn                  string
	CreationDate         time.Time
	Status               string
	Policies             map[string]any
	AutoVerifiedAttributes []string
	Schema               []map[string]any
	Clients              map[string]*UserPoolClient // keyed by ClientId
	Users                map[string]*User           // keyed by Username (case-insensitive: stored lowercase)
}

// UserPoolClient holds all metadata for a Cognito User Pool App Client.
type UserPoolClient struct {
	ClientId          string
	ClientName        string
	ClientSecret      string
	UserPoolId        string
	ExplicitAuthFlows []string
}

// User holds all data for a Cognito user.
type User struct {
	Username       string
	PasswordHash   []byte
	Attributes     map[string]string // Name -> Value
	UserCreateDate time.Time
	UserStatus     userStatus
	Enabled        bool
	Sub            string
}

// Store is the in-memory store for Cognito resources.
type Store struct {
	mu        sync.RWMutex
	pools     map[string]*UserPool // keyed by UserPoolId
	accountID string
	region    string
}

// NewStore creates an empty Cognito Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		pools:     make(map[string]*UserPool),
		accountID: accountID,
		region:    region,
	}
}

// newUUID returns a random UUID v4 string.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// newPoolID generates a Cognito-style pool ID: us-east-1_XxXxXxXx
func newPoolID(region string) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return fmt.Sprintf("%s_%s", region, string(b))
}

// newClientSecret generates a random client secret string (base64url-ish).
func newClientSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	out := make([]byte, 44)
	for i := range out {
		out[i] = chars[int(b[i%32])%len(chars)]
	}
	return string(out)
}

// buildPoolArn constructs an ARN for a User Pool.
func (s *Store) buildPoolArn(poolID string) string {
	return fmt.Sprintf("arn:aws:cognito-idp:%s:%s:userpool/%s", s.region, s.accountID, poolID)
}

// hashPassword hashes a plaintext password using SHA-256.
// NOTE: This is a mock implementation — not suitable for production use.
func hashPassword(password string) ([]byte, error) {
	h := sha256.Sum256([]byte(password))
	encoded := hex.EncodeToString(h[:])
	return []byte(encoded), nil
}

// checkPassword compares a plaintext password against a SHA-256 hash.
func checkPassword(hash []byte, password string) bool {
	h := sha256.Sum256([]byte(password))
	encoded := hex.EncodeToString(h[:])
	return string(hash) == encoded
}

// ---- UserPool operations ----

// CreateUserPool creates a new user pool and returns it.
func (s *Store) CreateUserPool(name string, policies map[string]any, autoVerifiedAttributes []string, schema []map[string]any) (*UserPool, *service.AWSError) {
	poolID := newPoolID(s.region)
	pool := &UserPool{
		Id:                   poolID,
		Name:                 name,
		Arn:                  s.buildPoolArn(poolID),
		CreationDate:         time.Now().UTC(),
		Status:               "Active",
		Policies:             policies,
		AutoVerifiedAttributes: autoVerifiedAttributes,
		Schema:               schema,
		Clients:              make(map[string]*UserPoolClient),
		Users:                make(map[string]*User),
	}

	s.mu.Lock()
	s.pools[poolID] = pool
	s.mu.Unlock()

	return pool, nil
}

// GetUserPool retrieves a user pool by ID.
func (s *Store) GetUserPool(poolID string) (*UserPool, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pool, ok := s.pools[poolID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool %s does not exist.", poolID), http.StatusBadRequest)
	}
	return pool, nil
}

// DeleteUserPool removes a user pool by ID.
func (s *Store) DeleteUserPool(poolID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[poolID]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool %s does not exist.", poolID), http.StatusBadRequest)
	}
	delete(s.pools, poolID)
	return nil
}

// ListUserPools returns a snapshot of all user pools, limited by maxResults (0 = unlimited).
func (s *Store) ListUserPools(maxResults int) []*UserPool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*UserPool, 0, len(s.pools))
	for _, p := range s.pools {
		out = append(out, p)
		if maxResults > 0 && len(out) >= maxResults {
			break
		}
	}
	return out
}

// ---- UserPoolClient operations ----

// CreateUserPoolClient creates a new app client in the given pool.
func (s *Store) CreateUserPoolClient(poolID, clientName string, explicitAuthFlows []string, generateSecret bool) (*UserPoolClient, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool %s does not exist.", poolID), http.StatusBadRequest)
	}

	clientID := newUUID()
	client := &UserPoolClient{
		ClientId:          clientID,
		ClientName:        clientName,
		UserPoolId:        poolID,
		ExplicitAuthFlows: explicitAuthFlows,
	}
	if generateSecret {
		client.ClientSecret = newClientSecret()
	}

	pool.Clients[clientID] = client
	return client, nil
}

// GetUserPoolClient retrieves a client by pool and client ID.
func (s *Store) GetUserPoolClient(poolID, clientID string) (*UserPoolClient, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool %s does not exist.", poolID), http.StatusBadRequest)
	}
	client, ok := pool.Clients[clientID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool client %s does not exist.", clientID), http.StatusBadRequest)
	}
	return client, nil
}

// ListUserPoolClients returns a snapshot of all clients in a pool.
func (s *Store) ListUserPoolClients(poolID string) ([]*UserPoolClient, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool %s does not exist.", poolID), http.StatusBadRequest)
	}
	out := make([]*UserPoolClient, 0, len(pool.Clients))
	for _, c := range pool.Clients {
		out = append(out, c)
	}
	return out, nil
}

// findPoolByClientID looks up the pool that owns a given client ID (for SignUp / InitiateAuth).
func (s *Store) findPoolByClientID(clientID string) (*UserPool, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, pool := range s.pools {
		if _, ok := pool.Clients[clientID]; ok {
			return pool, nil
		}
	}
	return nil, service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("User pool client %s does not exist.", clientID), http.StatusBadRequest)
}

// ---- User operations ----

// AdminCreateUser creates a new user in the pool with a temporary password.
func (s *Store) AdminCreateUser(poolID, username, temporaryPassword string, userAttributes map[string]string) (*User, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool %s does not exist.", poolID), http.StatusBadRequest)
	}

	lowerUsername := username
	if _, exists := pool.Users[lowerUsername]; exists {
		return nil, service.NewAWSError("UsernameExistsException",
			fmt.Sprintf("User account already exists: %s", username), http.StatusBadRequest)
	}

	var hash []byte
	var err error
	if temporaryPassword != "" {
		hash, err = hashPassword(temporaryPassword)
		if err != nil {
			return nil, service.NewAWSError("InternalErrorException",
				"Failed to hash password.", http.StatusInternalServerError)
		}
	}

	sub := newUUID()
	attrs := make(map[string]string)
	for k, v := range userAttributes {
		attrs[k] = v
	}
	if _, ok := attrs["sub"]; !ok {
		attrs["sub"] = sub
	}

	status := userStatusForceChangePassword
	if temporaryPassword == "" {
		status = userStatusUnconfirmed
	}

	user := &User{
		Username:       username,
		PasswordHash:   hash,
		Attributes:     attrs,
		UserCreateDate: time.Now().UTC(),
		UserStatus:     status,
		Enabled:        true,
		Sub:            sub,
	}

	pool.Users[lowerUsername] = user
	return user, nil
}

// GetUser retrieves a user from the pool.
func (s *Store) GetUser(poolID, username string) (*User, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool %s does not exist.", poolID), http.StatusBadRequest)
	}
	user, ok := pool.Users[username]
	if !ok {
		return nil, service.NewAWSError("UserNotFoundException",
			fmt.Sprintf("User does not exist: %s", username), http.StatusBadRequest)
	}
	return user, nil
}

// DeleteUser removes a user from the pool.
func (s *Store) DeleteUser(poolID, username string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool %s does not exist.", poolID), http.StatusBadRequest)
	}
	if _, ok := pool.Users[username]; !ok {
		return service.NewAWSError("UserNotFoundException",
			fmt.Sprintf("User does not exist: %s", username), http.StatusBadRequest)
	}
	delete(pool.Users, username)
	return nil
}

// SetUserPassword updates a user's password. If permanent is true, status is set to CONFIRMED.
func (s *Store) SetUserPassword(poolID, username, password string, permanent bool) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool %s does not exist.", poolID), http.StatusBadRequest)
	}
	user, ok := pool.Users[username]
	if !ok {
		return service.NewAWSError("UserNotFoundException",
			fmt.Sprintf("User does not exist: %s", username), http.StatusBadRequest)
	}

	hash, err := hashPassword(password)
	if err != nil {
		return service.NewAWSError("InternalErrorException",
			"Failed to hash password.", http.StatusInternalServerError)
	}
	user.PasswordHash = hash
	if permanent {
		user.UserStatus = userStatusConfirmed
	}
	return nil
}

// ConfirmUser sets a user's status to CONFIRMED.
func (s *Store) ConfirmUser(poolID, username string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("User pool %s does not exist.", poolID), http.StatusBadRequest)
	}
	user, ok := pool.Users[username]
	if !ok {
		return service.NewAWSError("UserNotFoundException",
			fmt.Sprintf("User does not exist: %s", username), http.StatusBadRequest)
	}
	user.UserStatus = userStatusConfirmed
	return nil
}

// SignUp creates a new user via the client-facing SignUp API.
func (s *Store) SignUp(clientID, username, password string, userAttributes map[string]string) (*User, *service.AWSError) {
	pool, awsErr := s.findPoolByClientID(clientID)
	if awsErr != nil {
		return nil, awsErr
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := pool.Users[username]; exists {
		return nil, service.NewAWSError("UsernameExistsException",
			fmt.Sprintf("User account already exists: %s", username), http.StatusBadRequest)
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, service.NewAWSError("InternalErrorException",
			"Failed to hash password.", http.StatusInternalServerError)
	}

	sub := newUUID()
	attrs := make(map[string]string)
	for k, v := range userAttributes {
		attrs[k] = v
	}
	if _, ok := attrs["sub"]; !ok {
		attrs["sub"] = sub
	}

	user := &User{
		Username:       username,
		PasswordHash:   hash,
		Attributes:     attrs,
		UserCreateDate: time.Now().UTC(),
		UserStatus:     userStatusUnconfirmed,
		Enabled:        true,
		Sub:            sub,
	}

	pool.Users[username] = user
	return user, nil
}

// AuthResult holds the tokens returned by InitiateAuth.
type AuthResult struct {
	AccessToken  string
	IdToken      string
	RefreshToken string
	ExpiresIn    int
	TokenType    string
}

// InitiateAuth authenticates a user and returns tokens.
// The KeyStore is required to sign JWTs; pass nil to skip token generation (test only).
func (s *Store) InitiateAuth(clientID, username, password string, keys *KeyStore) (*AuthResult, *service.AWSError) {
	pool, awsErr := s.findPoolByClientID(clientID)
	if awsErr != nil {
		return nil, awsErr
	}

	s.mu.RLock()
	user, ok := pool.Users[username]
	poolID := pool.Id
	s.mu.RUnlock()

	if !ok {
		return nil, service.NewAWSError("UserNotFoundException",
			fmt.Sprintf("User does not exist: %s", username), http.StatusBadRequest)
	}
	if !user.Enabled {
		return nil, service.NewAWSError("UserNotConfirmedException",
			"User is disabled.", http.StatusBadRequest)
	}
	if user.UserStatus == userStatusUnconfirmed {
		return nil, service.NewAWSError("UserNotConfirmedException",
			"User is not confirmed.", http.StatusBadRequest)
	}
	if !checkPassword(user.PasswordHash, password) {
		return nil, service.NewAWSError("NotAuthorizedException",
			"Incorrect username or password.", http.StatusBadRequest)
	}

	now := time.Now().UTC()
	iss := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", s.region, poolID)

	accessClaims := map[string]any{
		"sub":       user.Sub,
		"iss":       iss,
		"client_id": clientID,
		"token_use": "access",
		"scope":     "openid email profile",
		"auth_time": now.Unix(),
		"username":  user.Username,
		"iat":       now.Unix(),
		"exp":       now.Add(time.Hour).Unix(),
	}
	idClaims := map[string]any{
		"sub":               user.Sub,
		"iss":               iss,
		"aud":               clientID,
		"token_use":         "id",
		"auth_time":         now.Unix(),
		"cognito:username":  user.Username,
		"email":             user.Attributes["email"],
		"email_verified":    true,
		"iat":               now.Unix(),
		"exp":               now.Add(time.Hour).Unix(),
	}
	refreshClaims := map[string]any{
		"sub":       user.Sub,
		"iss":       iss,
		"token_use": "refresh",
		"iat":       now.Unix(),
		"exp":       now.Add(30 * 24 * time.Hour).Unix(),
	}

	accessToken, err := signJWT(keys, accessClaims)
	if err != nil {
		return nil, service.NewAWSError("InternalErrorException",
			"Failed to sign access token.", http.StatusInternalServerError)
	}
	idToken, err := signJWT(keys, idClaims)
	if err != nil {
		return nil, service.NewAWSError("InternalErrorException",
			"Failed to sign id token.", http.StatusInternalServerError)
	}
	refreshToken, err := signJWT(keys, refreshClaims)
	if err != nil {
		return nil, service.NewAWSError("InternalErrorException",
			"Failed to sign refresh token.", http.StatusInternalServerError)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		IdToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}, nil
}
