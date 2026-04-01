package transfer

import (
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func str(params map[string]any, key string) string {
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

func tagsFromParams(params map[string]any) map[string]string {
	tags := make(map[string]string)
	if v, ok := params["Tags"].([]any); ok {
		for _, item := range v {
			if t, ok := item.(map[string]any); ok {
				key, _ := t["Key"].(string)
				val, _ := t["Value"].(string)
				if key != "" {
					tags[key] = val
				}
			}
		}
	}
	return tags
}

func serverResponse(srv *Server) map[string]any {
	return map[string]any{
		"ServerId":             srv.ServerID,
		"Arn":                  srv.Arn,
		"Endpoint":             srv.Endpoint,
		"Domain":               srv.Domain,
		"EndpointType":         srv.EndpointType,
		"IdentityProviderType": srv.IdentityProviderType,
		"LoggingRole":          srv.LoggingRole,
		"Protocols":            srv.Protocols,
		"State":                srv.State,
		"UserCount":            srv.UserCount,
	}
}

func userResponse(u *User) map[string]any {
	keys := make([]map[string]any, 0, len(u.SshPublicKeys))
	for _, k := range u.SshPublicKeys {
		keys = append(keys, map[string]any{
			"SshPublicKeyId":   k.SSHPublicKeyID,
			"SshPublicKeyBody": k.SSHPublicKeyBody,
			"DateImported":     k.DateImported,
		})
	}
	return map[string]any{
		"ServerId":          u.ServerID,
		"UserName":          u.UserName,
		"Arn":               u.Arn,
		"HomeDirectory":     u.HomeDirectory,
		"HomeDirectoryType": u.HomeDirectoryType,
		"Role":              u.Role,
		"SshPublicKeys":     keys,
	}
}

func handleCreateServer(params map[string]any, store *Store) (*service.Response, error) {
	domain := str(params, "Domain")
	if domain == "" {
		domain = "S3"
	}
	endpointType := str(params, "EndpointType")
	if endpointType == "" {
		endpointType = "PUBLIC"
	}
	identityProvider := str(params, "IdentityProviderType")
	if identityProvider == "" {
		identityProvider = "SERVICE_MANAGED"
	}
	protocols := strSlice(params, "Protocols")
	if len(protocols) == 0 {
		protocols = []string{"SFTP"}
	}

	srv, err := store.CreateServer(domain, endpointType, identityProvider,
		str(params, "LoggingRole"), protocols, tagsFromParams(params))
	if err != nil {
		return jsonErr(service.ErrValidation(err.Error()))
	}
	return jsonOK(map[string]any{"ServerId": srv.ServerID})
}

func handleDescribeServer(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ServerId")
	if id == "" {
		return jsonErr(service.ErrValidation("ServerId is required"))
	}
	srv, ok := store.GetServer(id)
	if !ok {
		return jsonErr(service.ErrNotFound("Server", id))
	}
	return jsonOK(map[string]any{"Server": serverResponse(srv)})
}

func handleListServers(store *Store) (*service.Response, error) {
	servers := store.ListServers()
	out := make([]map[string]any, 0, len(servers))
	for _, srv := range servers {
		out = append(out, serverResponse(srv))
	}
	return jsonOK(map[string]any{"Servers": out})
}

func handleStartServer(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ServerId")
	if id == "" {
		return jsonErr(service.ErrValidation("ServerId is required"))
	}
	if err := store.StartServer(id); err != nil {
		return jsonErr(service.NewAWSError("InvalidRequestException", err.Error(), http.StatusBadRequest))
	}
	return jsonOK(map[string]any{})
}

func handleStopServer(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ServerId")
	if id == "" {
		return jsonErr(service.ErrValidation("ServerId is required"))
	}
	if err := store.StopServer(id); err != nil {
		return jsonErr(service.NewAWSError("InvalidRequestException", err.Error(), http.StatusBadRequest))
	}
	return jsonOK(map[string]any{})
}

func handleDeleteServer(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ServerId")
	if id == "" {
		return jsonErr(service.ErrValidation("ServerId is required"))
	}
	if !store.DeleteServer(id) {
		return jsonErr(service.ErrNotFound("Server", id))
	}
	return jsonOK(map[string]any{})
}

func handleCreateUser(params map[string]any, store *Store) (*service.Response, error) {
	serverID := str(params, "ServerId")
	userName := str(params, "UserName")
	if serverID == "" || userName == "" {
		return jsonErr(service.ErrValidation("ServerId and UserName are required"))
	}
	homeDir := str(params, "HomeDirectory")
	homeDirType := str(params, "HomeDirectoryType")
	if homeDirType == "" {
		homeDirType = "PATH"
	}

	user, err := store.CreateUser(serverID, userName, str(params, "Role"), homeDir, homeDirType, tagsFromParams(params))
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "home directory") {
			return jsonErr(service.ErrValidation(errMsg))
		}
		if strings.Contains(errMsg, "not found") {
			return jsonErr(service.ErrNotFound("Server", serverID))
		}
		return jsonErr(service.NewAWSError("ResourceExistsException", errMsg, http.StatusConflict))
	}
	return jsonOK(map[string]any{"ServerId": serverID, "UserName": user.UserName})
}

func handleDescribeUser(params map[string]any, store *Store) (*service.Response, error) {
	serverID := str(params, "ServerId")
	userName := str(params, "UserName")
	if serverID == "" || userName == "" {
		return jsonErr(service.ErrValidation("ServerId and UserName are required"))
	}
	user, ok := store.GetUser(serverID, userName)
	if !ok {
		return jsonErr(service.ErrNotFound("User", userName))
	}
	return jsonOK(map[string]any{"ServerId": serverID, "User": userResponse(user)})
}

func handleListUsers(params map[string]any, store *Store) (*service.Response, error) {
	serverID := str(params, "ServerId")
	if serverID == "" {
		return jsonErr(service.ErrValidation("ServerId is required"))
	}
	users := store.ListUsers(serverID)
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, map[string]any{
			"UserName":          u.UserName,
			"Arn":               u.Arn,
			"HomeDirectory":     u.HomeDirectory,
			"HomeDirectoryType": u.HomeDirectoryType,
			"Role":              u.Role,
		})
	}
	return jsonOK(map[string]any{"ServerId": serverID, "Users": out})
}

func handleDeleteUser(params map[string]any, store *Store) (*service.Response, error) {
	serverID := str(params, "ServerId")
	userName := str(params, "UserName")
	if serverID == "" || userName == "" {
		return jsonErr(service.ErrValidation("ServerId and UserName are required"))
	}
	if !store.DeleteUser(serverID, userName) {
		return jsonErr(service.ErrNotFound("User", userName))
	}
	return jsonOK(map[string]any{})
}

func handleImportSSHPublicKey(params map[string]any, store *Store) (*service.Response, error) {
	serverID := str(params, "ServerId")
	userName := str(params, "UserName")
	keyBody := str(params, "SshPublicKeyBody")
	if serverID == "" || userName == "" || keyBody == "" {
		return jsonErr(service.ErrValidation("ServerId, UserName, and SshPublicKeyBody are required"))
	}

	key, err := store.ImportSSHPublicKey(serverID, userName, keyBody)
	if err != nil {
		return jsonErr(service.ErrNotFound("User", userName))
	}
	return jsonOK(map[string]any{
		"ServerId":       serverID,
		"UserName":       userName,
		"SshPublicKeyId": key.SSHPublicKeyID,
	})
}

func handleDeleteSSHPublicKey(params map[string]any, store *Store) (*service.Response, error) {
	serverID := str(params, "ServerId")
	userName := str(params, "UserName")
	keyID := str(params, "SshPublicKeyId")
	if serverID == "" || userName == "" || keyID == "" {
		return jsonErr(service.ErrValidation("ServerId, UserName, and SshPublicKeyId are required"))
	}

	if err := store.DeleteSSHPublicKey(serverID, userName, keyID); err != nil {
		return jsonErr(service.ErrNotFound("SshPublicKey", keyID))
	}
	return jsonOK(map[string]any{})
}
