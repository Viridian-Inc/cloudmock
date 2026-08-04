package storage

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	gojson "encoding/json"
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func (s *StorageService) handleFile(ctx *service.RequestContext, account string) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-file") {
		parts = parts[1:]
	}
	if account == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The file request URI is invalid.")
	}
	if len(parts) == 0 {
		if ctx.RawRequest.Method == http.MethodGet && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "list") {
			return s.listFileShares(account, ctx.RawRequest.URL.Query())
		}
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The file request URI is invalid.")
	}

	shareName := parts[0]
	filePath := strings.Join(parts[1:], "/")
	query := ctx.RawRequest.URL.Query()
	shareSnapshot := query.Get("sharesnapshot")
	if shareSnapshot != "" && !fileShareSnapshotAllowsRequest(ctx.RawRequest.Method, len(parts), query) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "Share snapshots are read-only.")
	}
	if len(parts) == 1 && strings.EqualFold(ctx.RawRequest.URL.Query().Get("restype"), "directory") {
		if ctx.RawRequest.Method == http.MethodGet && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "listhandles") {
			return s.listFileHandles(account, shareName, "", ctx.RawRequest.URL.Query(), true)
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "forceclosehandles") {
			return s.forceCloseFileHandles(account, shareName, "", ctx.RawRequest.URL.Query(), ctx.RawRequest.Header, true)
		}
		if ctx.RawRequest.Method == http.MethodGet && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "list") {
			return s.listDirectory(account, shareName, "", ctx.RawRequest.URL.Query())
		}
	}
	if len(parts) == 1 && strings.EqualFold(ctx.RawRequest.URL.Query().Get("restype"), "share") {
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "lease") {
				return s.leaseFileShare(account, shareName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
			}
			if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "acl") {
				return s.setFileShareACL(account, shareName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header, ctx.Body)
			}
			if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "filepermission") {
				return s.createFilePermission(account, shareName, ctx.RawRequest.URL.Query(), ctx.Body)
			}
			if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "snapshot") {
				return s.snapshotFileShare(account, shareName, ctx.RawRequest.Header)
			}
			if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "metadata") {
				return s.setFileShareMetadata(account, shareName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
			}
			if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "properties") {
				return s.setFileShareProperties(account, shareName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
			}
			return s.createFileShare(account, shareName, ctx.RawRequest.Header)
		case http.MethodGet, http.MethodHead:
			if ctx.RawRequest.Method == http.MethodGet && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "stats") {
				return s.getFileShareStats(account, shareName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
			}
			if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "acl") {
				return s.getFileShareACL(account, shareName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header, ctx.RawRequest.Method == http.MethodHead)
			}
			if ctx.RawRequest.Method == http.MethodGet && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "filepermission") {
				return s.getFilePermission(account, shareName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
			}
			if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "metadata") {
				return s.getFileShareMetadata(account, shareName, ctx.RawRequest.URL.Query())
			}
			return s.getFileShareProperties(account, shareName, ctx.RawRequest.URL.Query())
		case http.MethodDelete:
			return s.deleteFileShare(account, shareName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
		}
	}
	if len(parts) > 1 && strings.EqualFold(ctx.RawRequest.URL.Query().Get("restype"), "directory") {
		if ctx.RawRequest.Method == http.MethodGet && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "listhandles") {
			return s.listFileHandles(account, shareName, filePath, ctx.RawRequest.URL.Query(), true)
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "forceclosehandles") {
			return s.forceCloseFileHandles(account, shareName, filePath, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header, true)
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "rename") {
			return s.renameFileDirectory(account, shareName, filePath, ctx.RawRequest.Header)
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "properties") {
			return s.setFileDirectoryProperties(account, shareName, filePath, ctx.RawRequest.Header)
		}
		if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "metadata") {
			switch ctx.RawRequest.Method {
			case http.MethodPut:
				return s.setFileDirectoryMetadata(account, shareName, filePath, ctx.RawRequest.Header)
			case http.MethodGet, http.MethodHead:
				return s.getFileDirectoryMetadata(account, shareName, filePath)
			}
		}
		if ctx.RawRequest.Method == http.MethodPut {
			return s.createFileDirectory(account, shareName, filePath, ctx.RawRequest.Header)
		}
		if ctx.RawRequest.Method == http.MethodGet && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "list") {
			return s.listDirectory(account, shareName, filePath, ctx.RawRequest.URL.Query())
		}
		if (ctx.RawRequest.Method == http.MethodGet || ctx.RawRequest.Method == http.MethodHead) && ctx.RawRequest.URL.Query().Get("comp") == "" {
			return s.getFileDirectoryProperties(account, shareName, filePath)
		}
		if ctx.RawRequest.Method == http.MethodDelete {
			return s.deleteFileDirectory(account, shareName, filePath)
		}
	}
	if len(parts) > 1 {
		if ctx.RawRequest.Method == http.MethodGet && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "listhandles") {
			return s.listFileHandles(account, shareName, filePath, ctx.RawRequest.URL.Query(), false)
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "forceclosehandles") {
			return s.forceCloseFileHandles(account, shareName, filePath, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header, false)
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "lease") {
			return s.leaseFile(account, shareName, filePath, ctx.RawRequest.Header)
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.TrimSpace(ctx.RawRequest.Header.Get("x-ms-copy-source")) != "" {
			return s.copyFile(account, shareName, filePath, ctx.RawRequest.Header)
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "properties") {
			return s.setFileProperties(account, shareName, filePath, ctx.RawRequest.Header)
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "rename") {
			return s.renameFile(account, shareName, filePath, ctx.RawRequest.Header)
		}
		if ctx.RawRequest.Method == http.MethodGet && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "rangelist") {
			return s.listFileRanges(account, shareName, filePath, ctx.RawRequest.Header)
		}
		if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "metadata") {
			switch ctx.RawRequest.Method {
			case http.MethodPut:
				return s.setFileMetadata(account, shareName, filePath, ctx.RawRequest.Header)
			case http.MethodGet, http.MethodHead:
				return s.getFileMetadata(account, shareName, filePath, ctx.RawRequest.URL.Query())
			}
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "range") {
			return s.putFileRange(account, shareName, filePath, ctx)
		}
		if ctx.RawRequest.Method == http.MethodPut && strings.EqualFold(ctx.RawRequest.Header.Get("x-ms-type"), "file") {
			return s.createFile(account, shareName, filePath, ctx.RawRequest.Header)
		}
		if ctx.RawRequest.Method == http.MethodGet || ctx.RawRequest.Method == http.MethodHead {
			return s.getFile(account, shareName, filePath, ctx.RawRequest)
		}
		if ctx.RawRequest.Method == http.MethodDelete {
			return s.deleteFile(account, shareName, filePath, ctx.RawRequest.Header)
		}
	}

	return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified file share route was not found.")
}

func (s *StorageService) createFileShare(account, shareName string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fileShares[accountKey] == nil {
		s.fileShares[accountKey] = make(map[string]fileShare)
	}
	if _, exists := s.fileShares[accountKey][shareKey]; exists {
		return azurearm.ErrorResponse(http.StatusConflict, "ShareAlreadyExists", "The specified share already exists.")
	}
	now := time.Now().UTC()
	share := fileShare{
		Name:             shareName,
		Metadata:         metadataFromHeaders(header),
		ETag:             s.nextToken("\"share") + "\"",
		LastModified:     now,
		Quota:            strings.TrimSpace(header.Get("x-ms-share-quota")),
		AccessTier:       strings.TrimSpace(header.Get("x-ms-access-tier")),
		EnabledProtocols: strings.TrimSpace(header.Get("x-ms-enabled-protocols")),
		RootSquash:       strings.TrimSpace(header.Get("x-ms-root-squash")),
		SnapshotVDir:     strings.TrimSpace(header.Get("x-ms-enable-snapshot-virtual-directory-access")),
		FilePermissions:  make(map[string]fileSharePermission),
		Directories:      make(map[string]fileDirectory),
		Files:            make(map[string]fileObject),
		Snapshots:        make(map[string]fileShare),
	}
	if share.EnabledProtocols == "" {
		share.EnabledProtocols = "SMB"
	}
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusCreated, fileShareHeaders(share))
}

func (s *StorageService) snapshotFileShare(account, shareName string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	if !fileShareLeaseAllowsWrite(header, share) {
		return fileShareLeaseWriteFailure(header, share)
	}
	if len(share.Snapshots) >= 200 {
		return azurearm.ErrorResponse(http.StatusConflict, "ShareSnapshotCountExceeded", "The share snapshot count limit has been exceeded.")
	}
	now := time.Now().UTC()
	snapshotID := snapshotTime(now)
	for share.Snapshots[snapshotID].Name != "" {
		now = now.Add(time.Nanosecond)
		snapshotID = snapshotTime(now)
	}
	snapshot := cloneFileShare(share)
	snapshot.Snapshots = nil
	if metadataHeadersPresent(header) {
		snapshot.Metadata = metadataFromHeaders(header)
		snapshot.ETag = s.nextToken("\"share-snapshot") + "\""
		snapshot.LastModified = now
	}
	if share.Snapshots == nil {
		share.Snapshots = make(map[string]fileShare)
	}
	share.Snapshots[snapshotID] = snapshot
	s.fileShares[accountKey][shareKey] = share

	headers := fileShareHeaders(snapshot)
	headers["x-ms-snapshot"] = snapshotID
	return emptyResponse(http.StatusCreated, headers)
}

func (s *StorageService) setFileShareMetadata(account, shareName string, query url.Values, header http.Header) (*service.Response, error) {
	if query.Get("sharesnapshot") != "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "Set Share Metadata is not supported for share snapshots.")
	}
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	if !fileShareLeaseAllowsWrite(header, share) {
		return fileShareLeaseWriteFailure(header, share)
	}
	share.Metadata = metadataFromHeaders(header)
	share.ETag = s.nextToken("\"share") + "\""
	share.LastModified = time.Now().UTC()
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusOK, storageHeaders(share.ETag, share.LastModified))
}

func (s *StorageService) setFileShareProperties(account, shareName string, query url.Values, header http.Header) (*service.Response, error) {
	if query.Get("sharesnapshot") != "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "Set Share Properties is not supported for share snapshots.")
	}
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	if !fileShareLeaseAllowsWrite(header, share) {
		return fileShareLeaseWriteFailure(header, share)
	}
	if hasHeader(header, "x-ms-share-quota") {
		share.Quota = strings.TrimSpace(header.Get("x-ms-share-quota"))
	}
	if hasHeader(header, "x-ms-access-tier") {
		share.AccessTier = strings.TrimSpace(header.Get("x-ms-access-tier"))
	}
	if hasHeader(header, "x-ms-root-squash") {
		share.RootSquash = strings.TrimSpace(header.Get("x-ms-root-squash"))
	}
	if hasHeader(header, "x-ms-enable-snapshot-virtual-directory-access") {
		share.SnapshotVDir = strings.TrimSpace(header.Get("x-ms-enable-snapshot-virtual-directory-access"))
	}
	share.ETag = s.nextToken("\"share") + "\""
	share.LastModified = time.Now().UTC()
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusOK, fileShareHeaders(share))
}

func (s *StorageService) getFileShareProperties(account, shareName string, query url.Values) (*service.Response, error) {
	s.mu.RLock()
	share, ok := s.fileShareForReadLocked(strings.ToLower(account), strings.ToLower(shareName), query.Get("sharesnapshot"))
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	headers := fileShareHeaders(share)
	if snapshot := query.Get("sharesnapshot"); snapshot != "" {
		headers["x-ms-snapshot"] = snapshot
	}
	return emptyResponse(http.StatusOK, headers)
}

func (s *StorageService) getFileShareMetadata(account, shareName string, query url.Values) (*service.Response, error) {
	s.mu.RLock()
	share, ok := s.fileShareForReadLocked(strings.ToLower(account), strings.ToLower(shareName), query.Get("sharesnapshot"))
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	headers := storageHeaders(share.ETag, share.LastModified)
	for key, value := range share.Metadata {
		headers["x-ms-meta-"+key] = value
	}
	if snapshot := query.Get("sharesnapshot"); snapshot != "" {
		headers["x-ms-snapshot"] = snapshot
	}
	return emptyResponse(http.StatusOK, headers)
}

func (s *StorageService) getFileShareStats(account, shareName string, query url.Values, header http.Header) (*service.Response, error) {
	if query.Get("sharesnapshot") != "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "Statistics for share snapshots cannot be retrieved.")
	}
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	s.mu.RLock()
	share, ok := s.fileShares[accountKey][shareKey]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	if leaseID := strings.TrimSpace(header.Get("x-ms-lease-id")); leaseID != "" && (fileShareLeaseState(share) != "leased" || leaseID != share.LeaseID) {
		resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseIdMismatchWithShareOperation", "The lease ID specified did not match the lease ID for the share.")
		if resp != nil {
			resp.Headers = storageHeaders("", time.Now().UTC())
			resp.Headers["x-ms-error-code"] = "LeaseIdMismatchWithShareOperation"
		}
		return resp, err
	}

	resp, err := xmlResponse(http.StatusOK, fileShareStatsResponse{ShareUsageBytes: fileShareUsageBytes(share)})
	if err != nil {
		return nil, err
	}
	resp.Headers = storageHeaders(share.ETag, share.LastModified)
	return resp, nil
}

func (s *StorageService) getFileShareACL(account, shareName string, query url.Values, header http.Header, headOnly bool) (*service.Response, error) {
	if query.Get("sharesnapshot") != "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "Access policies cannot be retrieved for share snapshots.")
	}
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	s.mu.RLock()
	share, ok := s.fileShares[accountKey][shareKey]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	if leaseID := strings.TrimSpace(header.Get("x-ms-lease-id")); leaseID != "" && (fileShareLeaseState(share) != "leased" || leaseID != share.LeaseID) {
		resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseIdMismatchWithShareOperation", "The lease ID specified did not match the lease ID for the share.")
		if resp != nil {
			resp.Headers = storageHeaders("", time.Now().UTC())
			resp.Headers["x-ms-error-code"] = "LeaseIdMismatchWithShareOperation"
		}
		return resp, err
	}

	headers := storageHeaders(share.ETag, share.LastModified)
	if headOnly {
		return emptyResponse(http.StatusOK, headers)
	}
	resp, err := xmlResponse(http.StatusOK, fileShareACLResponse{SignedIdentifiers: cloneFileShareAccessPolicies(share.AccessPolicies)})
	if err != nil {
		return nil, err
	}
	resp.Headers = headers
	return resp, nil
}

func (s *StorageService) setFileShareACL(account, shareName string, query url.Values, header http.Header, body []byte) (*service.Response, error) {
	if query.Get("sharesnapshot") != "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "Access policies cannot be set for share snapshots.")
	}
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	policies, err := parseFileShareACL(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", err.Error())
	}
	if len(policies) > 5 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", "A share can contain at most five stored access policies.")
	}
	for _, policy := range policies {
		if len(policy.ID) > 64 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", "A stored access policy identifier cannot exceed 64 characters.")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	if !fileShareLeaseAllowsWrite(header, share) {
		return fileShareACLLeaseFailure(header, share)
	}
	share.AccessPolicies = cloneFileShareAccessPolicies(policies)
	share.ETag = s.nextToken("\"share") + "\""
	share.LastModified = time.Now().UTC()
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusOK, storageHeaders(share.ETag, share.LastModified))
}

func (s *StorageService) createFilePermission(account, shareName string, query url.Values, body []byte) (*service.Response, error) {
	if query.Get("sharesnapshot") != "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "File permissions cannot be created for share snapshots.")
	}
	permission, errResp, err := parseFilePermissionCreateBody(body)
	if errResp != nil || err != nil {
		return errResp, err
	}
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	key := filePermissionKey(permission)
	if share.FilePermissions == nil {
		share.FilePermissions = make(map[string]fileSharePermission)
	}
	share.FilePermissions[key] = permission
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusCreated, map[string]string{
		"x-ms-version":             dataPlaneAPIVersion,
		"x-ms-file-permission-key": key,
	})
}

func (s *StorageService) getFilePermission(account, shareName string, query url.Values, header http.Header) (*service.Response, error) {
	if query.Get("sharesnapshot") != "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "File permissions cannot be retrieved for share snapshots.")
	}
	key := strings.TrimSpace(header.Get("x-ms-file-permission-key"))
	if key == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-file-permission-key header is required.")
	}
	requestFormat := strings.ToLower(strings.TrimSpace(header.Get("x-ms-file-permission-format")))
	if requestFormat != "" && requestFormat != "sddl" && requestFormat != "binary" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-file-permission-format header value is invalid.")
	}

	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	s.mu.RLock()
	share, ok := s.fileShares[accountKey][shareKey]
	permission, permissionOK := share.FilePermissions[key]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	if !permissionOK {
		return azurearm.ErrorResponse(http.StatusNotFound, "PermissionNotFound", "The specified file permission key was not found.")
	}
	if requestFormat != "" && requestFormat != permission.Format {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The requested permission format is not available for the specified permission key.")
	}

	body := map[string]string{"permission": permission.Permission}
	if storageVersionAtLeast(header.Get("x-ms-version"), "2024-11-04") || permission.Format == "binary" {
		body["format"] = permission.Format
	}
	resp, err := azurearm.JSONResponse(http.StatusOK, body)
	if err != nil {
		return nil, err
	}
	resp.Headers = map[string]string{"x-ms-version": dataPlaneAPIVersion}
	return resp, nil
}

func (s *StorageService) deleteFileShare(account, shareName string, query url.Values, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	if query.Get("sharesnapshot") == "" && !fileShareLeaseAllowsWrite(header, share) {
		return fileShareLeaseWriteFailure(header, share)
	}
	if snapshot := query.Get("sharesnapshot"); snapshot != "" {
		if strings.TrimSpace(header.Get("x-ms-delete-snapshots")) != "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The x-ms-delete-snapshots header is not valid when deleting an individual share snapshot.")
		}
		snapshotShare, ok := share.Snapshots[snapshot]
		if !ok {
			return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share snapshot does not exist.")
		}
		if !fileShareLeaseAllowsWrite(header, snapshotShare) {
			return fileShareLeaseWriteFailure(header, snapshotShare)
		}
		delete(share.Snapshots, snapshot)
		s.fileShares[accountKey][shareKey] = share
		return emptyResponse(http.StatusAccepted, map[string]string{"x-ms-version": dataPlaneAPIVersion})
	}
	if len(share.Snapshots) > 0 {
		deleteSnapshots := strings.ToLower(strings.TrimSpace(header.Get("x-ms-delete-snapshots")))
		if deleteSnapshots != "include" && deleteSnapshots != "include-leased" {
			return fileShareSnapshotsPresentResponse()
		}
	}
	delete(s.fileShares[accountKey], shareKey)
	return emptyResponse(http.StatusAccepted, map[string]string{"x-ms-version": dataPlaneAPIVersion})
}

func (s *StorageService) leaseFileShare(account, shareName string, query url.Values, header http.Header) (*service.Response, error) {
	action := strings.ToLower(strings.TrimSpace(header.Get("x-ms-lease-action")))
	if action == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-lease-action header is required.")
	}

	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	snapshotID := query.Get("sharesnapshot")
	target := share
	if snapshotID != "" {
		snapshotShare, ok := share.Snapshots[snapshotID]
		if !ok {
			return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share snapshot does not exist.")
		}
		target = snapshotShare
	}
	persist := func(updated fileShare) {
		if snapshotID != "" {
			share.Snapshots[snapshotID] = updated
			s.fileShares[accountKey][shareKey] = share
			return
		}
		s.fileShares[accountKey][shareKey] = updated
	}

	switch action {
	case "acquire":
		if strings.TrimSpace(header.Get("x-ms-lease-duration")) == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-lease-duration header is required.")
		}
		if duration := strings.TrimSpace(header.Get("x-ms-lease-duration")); duration != "-1" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-lease-duration header must be -1 for Azure Files.")
		}
		leaseID := strings.TrimSpace(header.Get("x-ms-proposed-lease-id"))
		if leaseID == "" {
			leaseID = s.nextToken("lease-")
		}
		if fileShareLeaseState(target) == "leased" && target.LeaseID != leaseID {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseAlreadyPresent", "There is already a lease present.")
		}
		target.LeaseID = leaseID
		target.LeaseState = "leased"
		persist(target)
		headers := storageHeaders(target.ETag, target.LastModified)
		headers["x-ms-lease-id"] = leaseID
		return emptyResponse(http.StatusCreated, headers)
	case "renew":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		if leaseID == "" || leaseID != target.LeaseID || fileShareLeaseState(target) != "leased" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the share.")
		}
		headers := storageHeaders(target.ETag, target.LastModified)
		headers["x-ms-lease-id"] = leaseID
		return emptyResponse(http.StatusOK, headers)
	case "change":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		proposed := strings.TrimSpace(header.Get("x-ms-proposed-lease-id"))
		if proposed == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-proposed-lease-id header is required.")
		}
		if leaseID == "" || leaseID != target.LeaseID || fileShareLeaseState(target) != "leased" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the share.")
		}
		target.LeaseID = proposed
		target.LeaseState = "leased"
		persist(target)
		headers := storageHeaders(target.ETag, target.LastModified)
		headers["x-ms-lease-id"] = proposed
		return emptyResponse(http.StatusOK, headers)
	case "release":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		if leaseID == "" || leaseID != target.LeaseID {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the share.")
		}
		target.LeaseID = ""
		target.LeaseState = "available"
		persist(target)
		return emptyResponse(http.StatusOK, storageHeaders(target.ETag, target.LastModified))
	case "break":
		if fileShareLeaseState(target) != "leased" && fileShareLeaseState(target) != "broken" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseNotPresentWithLeaseOperation", "There is currently no lease on the share.")
		}
		target.LeaseState = "broken"
		persist(target)
		headers := storageHeaders(target.ETag, target.LastModified)
		headers["x-ms-lease-time"] = "0"
		return emptyResponse(http.StatusAccepted, headers)
	default:
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-lease-action header value is invalid.")
	}
}

func (s *StorageService) listFileHandles(account, shareName, resourcePath string, query url.Values, directory bool) (*service.Response, error) {
	if query.Get("maxresults") != "" && parsePositiveInt(query.Get("maxresults"), 0) <= 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The maxresults query parameter must be a positive integer.")
	}
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	resourcePath = normalizeFilePath(resourcePath)

	s.mu.RLock()
	_, ok := s.fileHandleTargetLocked(accountKey, shareKey, query.Get("sharesnapshot"), resourcePath, directory)
	s.mu.RUnlock()
	if !ok {
		if query.Get("sharesnapshot") != "" {
			return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share snapshot does not exist.")
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified file resource does not exist.")
	}

	result := fileHandleListResponse{
		Marker:        query.Get("marker"),
		ShareSnapshot: query.Get("sharesnapshot"),
		MaxResults:    query.Get("maxresults"),
		HandleList:    fileHandleListContainer{},
	}
	resp, err := xmlResponse(http.StatusOK, result)
	if err != nil {
		return nil, err
	}
	resp.Headers = map[string]string{"x-ms-version": dataPlaneAPIVersion}
	return resp, nil
}

func (s *StorageService) forceCloseFileHandles(account, shareName, resourcePath string, query url.Values, header http.Header, directory bool) (*service.Response, error) {
	if strings.TrimSpace(header.Get("x-ms-handle-id")) == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-handle-id header is required.")
	}
	if !directory && isTruthy(header.Get("x-ms-recursive")) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-recursive header is valid only for directories.")
	}
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	resourcePath = normalizeFilePath(resourcePath)

	s.mu.RLock()
	_, ok := s.fileHandleTargetLocked(accountKey, shareKey, query.Get("sharesnapshot"), resourcePath, directory)
	s.mu.RUnlock()
	if !ok {
		if query.Get("sharesnapshot") != "" {
			return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share snapshot does not exist.")
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified file resource does not exist.")
	}
	return emptyResponse(http.StatusOK, map[string]string{
		"x-ms-version":                  dataPlaneAPIVersion,
		"x-ms-number-of-handles-closed": "0",
		"x-ms-number-of-handles-failed": "0",
	})
}

func (s *StorageService) fileHandleTargetLocked(accountKey, shareKey, snapshot, resourcePath string, directory bool) (fileShare, bool) {
	share, ok := s.fileShareForReadLocked(accountKey, shareKey, snapshot)
	if !ok {
		return fileShare{}, false
	}
	if directory {
		if resourcePath == "" {
			return share, true
		}
		_, ok := share.Directories[resourcePath]
		return share, ok
	}
	_, ok = share.Files[resourcePath]
	return share, ok
}

func (s *StorageService) listFileShares(account string, query url.Values) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	prefix := query.Get("prefix")
	marker := strings.ToLower(query.Get("marker"))
	maxResults := parsePositiveInt(query.Get("maxresults"), 0)
	includeMetadata := blobListIncludesMetadata(query.Get("include"))
	includeSnapshots := fileListIncludes(query.Get("include"), "snapshots")

	if query.Get("maxresults") != "" && maxResults <= 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The maxresults query parameter must be a positive integer.")
	}

	s.mu.RLock()
	shares := s.fileShares[accountKey]
	names := make([]string, 0, len(shares))
	for key, share := range shares {
		if prefix != "" && !strings.HasPrefix(share.Name, prefix) {
			continue
		}
		if marker != "" && key <= marker {
			continue
		}
		names = append(names, key)
	}
	sort.Strings(names)

	nextMarker := ""
	if maxResults > 0 && len(names) > maxResults {
		nextMarker = names[maxResults]
		names = names[:maxResults]
	}

	items := make([]fileShareListItem, 0, len(names))
	for _, key := range names {
		share := shares[key]
		item := fileShareListItem{
			Name: share.Name,
			Properties: fileShareListProperties{
				LastModified:     share.LastModified.UTC().Format(http.TimeFormat),
				ETag:             share.ETag,
				Quota:            share.Quota,
				AccessTier:       share.AccessTier,
				EnabledProtocols: share.EnabledProtocols,
				RootSquash:       share.RootSquash,
			},
		}
		if includeMetadata && len(share.Metadata) > 0 {
			metadata := blobListMetadata(share.Metadata)
			item.Metadata = &metadata
		}
		items = append(items, item)
		if includeSnapshots && len(share.Snapshots) > 0 {
			snapshotIDs := make([]string, 0, len(share.Snapshots))
			for snapshotID := range share.Snapshots {
				snapshotIDs = append(snapshotIDs, snapshotID)
			}
			sort.Strings(snapshotIDs)
			for _, snapshotID := range snapshotIDs {
				snapshot := share.Snapshots[snapshotID]
				snapshotItem := fileShareListItem{
					Name:     share.Name,
					Snapshot: snapshotID,
					Properties: fileShareListProperties{
						LastModified:     snapshot.LastModified.UTC().Format(http.TimeFormat),
						ETag:             snapshot.ETag,
						Quota:            snapshot.Quota,
						AccessTier:       snapshot.AccessTier,
						EnabledProtocols: snapshot.EnabledProtocols,
						RootSquash:       snapshot.RootSquash,
					},
				}
				if includeMetadata && len(snapshot.Metadata) > 0 {
					metadata := blobListMetadata(snapshot.Metadata)
					snapshotItem.Metadata = &metadata
				}
				items = append(items, snapshotItem)
			}
		}
	}
	s.mu.RUnlock()

	result := fileShareListResponse{
		ServiceEndpoint: "https://" + account + ".file.core.windows.net/",
		Prefix:          prefix,
		Marker:          query.Get("marker"),
		MaxResults:      query.Get("maxresults"),
		Shares:          items,
		NextMarker:      nextMarker,
	}
	resp, err := xmlResponse(http.StatusOK, result)
	if err != nil {
		return nil, err
	}
	resp.Headers = map[string]string{"x-ms-version": dataPlaneAPIVersion}
	return resp, nil
}

func (s *StorageService) createFileDirectory(account, shareName, directoryPath string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	directoryPath = normalizeFilePath(directoryPath)
	if directoryPath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The directory path is required.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	parent := fileParentPath(directoryPath)
	if parent != "" {
		if _, ok := share.Directories[parent]; !ok {
			return azurearm.ErrorResponse(http.StatusPreconditionFailed, "ParentNotFound", "The parent directory does not exist.")
		}
	}
	if _, exists := share.Directories[directoryPath]; exists {
		return azurearm.ErrorResponse(http.StatusConflict, "ResourceAlreadyExists", "The specified resource already exists.")
	}
	if _, exists := share.Files[directoryPath]; exists {
		return azurearm.ErrorResponse(http.StatusConflict, "ResourceAlreadyExists", "The specified resource already exists.")
	}
	now := time.Now().UTC()
	directory := fileDirectory{
		Path:         directoryPath,
		Metadata:     metadataFromHeaders(header),
		Attributes:   "Directory",
		ETag:         s.nextToken("\"dir") + "\"",
		LastModified: now,
	}
	share.Directories[directoryPath] = directory
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusCreated, fileDirectoryHeaders(directory))
}

func (s *StorageService) setFileDirectoryProperties(account, shareName, directoryPath string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	directoryPath = normalizeFilePath(directoryPath)
	if directoryPath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The directory path is required.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	directory, ok := share.Directories[directoryPath]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified directory does not exist.")
	}
	applyFileDirectoryPropertyHeaders(&directory, header)
	directory.ETag = s.nextToken("\"dir") + "\""
	directory.LastModified = time.Now().UTC()
	share.Directories[directoryPath] = directory
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusOK, fileDirectoryHeaders(directory))
}

func (s *StorageService) setFileDirectoryMetadata(account, shareName, directoryPath string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	directoryPath = normalizeFilePath(directoryPath)
	if directoryPath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The directory path is required.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	directory, ok := share.Directories[directoryPath]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified directory does not exist.")
	}
	directory.Metadata = metadataFromHeaders(header)
	directory.ETag = s.nextToken("\"dir") + "\""
	directory.LastModified = time.Now().UTC()
	share.Directories[directoryPath] = directory
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusOK, storageHeaders(directory.ETag, directory.LastModified))
}

func (s *StorageService) getFileDirectoryMetadata(account, shareName, directoryPath string) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	directoryPath = normalizeFilePath(directoryPath)
	if directoryPath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The directory path is required.")
	}

	s.mu.RLock()
	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	directory, ok := share.Directories[directoryPath]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified directory does not exist.")
	}
	return emptyResponse(http.StatusOK, fileMetadataHeaders(directory.ETag, directory.LastModified, directory.Metadata, "Directory"))
}

func (s *StorageService) renameFileDirectory(account, shareName, destinationPath string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	destinationPath = normalizeFilePath(destinationPath)
	if destinationPath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The destination directory path is required.")
	}
	sourceShare, sourcePath, ok := parseFileRenameSource(account, header.Get("x-ms-file-rename-source"))
	if !ok || !strings.EqualFold(sourceShare, shareName) || sourcePath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-file-rename-source header must reference a directory in the same share.")
	}
	if destinationPath == sourcePath || strings.HasPrefix(destinationPath, sourcePath+"/") {
		return azurearm.ErrorResponse(http.StatusConflict, "InvalidRenameSource", "The destination directory cannot be the source directory or its child.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	directory, ok := share.Directories[sourcePath]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified source directory does not exist.")
	}
	parent := fileParentPath(destinationPath)
	if parent != "" {
		if _, ok := share.Directories[parent]; !ok {
			return azurearm.ErrorResponse(http.StatusPreconditionFailed, "ParentNotFound", "The parent directory does not exist.")
		}
	}
	if _, ok := share.Files[destinationPath]; ok {
		return azurearm.ErrorResponse(http.StatusConflict, "ResourceAlreadyExists", "The specified destination path is a file.")
	}
	if _, exists := share.Directories[destinationPath]; exists {
		if !isTruthy(header.Get("x-ms-file-rename-replace-if-exists")) {
			return azurearm.ErrorResponse(http.StatusConflict, "ResourceAlreadyExists", "The specified destination directory already exists.")
		}
		removeFileDirectorySubtree(share, destinationPath)
	}

	movedDirectories := make(map[string]fileDirectory)
	sourcePrefix := sourcePath + "/"
	for path, candidate := range share.Directories {
		if path != sourcePath && !strings.HasPrefix(path, sourcePrefix) {
			continue
		}
		delete(share.Directories, path)
		newPath := renamedFileChildPath(sourcePath, destinationPath, path)
		candidate.Path = newPath
		if path == sourcePath {
			if metadataHeadersPresent(header) {
				candidate.Metadata = metadataFromHeaders(header)
			}
			applyFileDirectoryPropertyHeaders(&candidate, header)
			candidate.ETag = s.nextToken("\"dir") + "\""
			candidate.LastModified = time.Now().UTC()
			directory = candidate
		}
		movedDirectories[newPath] = candidate
	}
	for path, directory := range movedDirectories {
		share.Directories[path] = directory
	}

	movedFiles := make(map[string]fileObject)
	for path, file := range share.Files {
		if !strings.HasPrefix(path, sourcePrefix) {
			continue
		}
		delete(share.Files, path)
		newPath := renamedFileChildPath(sourcePath, destinationPath, path)
		file.Path = newPath
		movedFiles[newPath] = file
	}
	for path, file := range movedFiles {
		share.Files[path] = file
	}

	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusOK, fileDirectoryHeaders(directory))
}

func (s *StorageService) createFile(account, shareName, filePath string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	filePath = normalizeFilePath(filePath)
	if filePath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The file path is required.")
	}
	length, err := strconv.Atoi(strings.TrimSpace(header.Get("x-ms-content-length")))
	if err != nil || length < 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-content-length header must be a nonnegative integer.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	parent := fileParentPath(filePath)
	if parent != "" {
		if _, ok := share.Directories[parent]; !ok {
			return azurearm.ErrorResponse(http.StatusPreconditionFailed, "ParentNotFound", "The parent directory does not exist.")
		}
	}
	if _, exists := share.Directories[filePath]; exists {
		return azurearm.ErrorResponse(http.StatusConflict, "ResourceAlreadyExists", "The specified resource already exists.")
	}
	var leaseID, leaseState string
	if existing, exists := share.Files[filePath]; exists {
		if !fileLeaseAllowsWrite(header, existing) {
			return fileLeaseWriteFailure(header, existing)
		}
		leaseID = existing.LeaseID
		leaseState = fileLeaseState(existing)
	}
	now := time.Now().UTC()
	contentType := strings.TrimSpace(header.Get("x-ms-content-type"))
	if contentType == "" {
		contentType = strings.TrimSpace(header.Get("Content-Type"))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	file := fileObject{
		Path:         filePath,
		Content:      make([]byte, length),
		ContentType:  contentType,
		ContentMD5:   strings.TrimSpace(header.Get("x-ms-content-md5")),
		Metadata:     metadataFromHeaders(header),
		ETag:         s.nextToken("\"file") + "\"",
		LastModified: now,
		LeaseID:      leaseID,
		LeaseState:   leaseState,
	}
	share.Files[filePath] = file
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusCreated, fileObjectHeaders(file))
}

func (s *StorageService) leaseFile(account, shareName, filePath string, header http.Header) (*service.Response, error) {
	action := strings.ToLower(strings.TrimSpace(header.Get("x-ms-lease-action")))
	if action == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-lease-action header is required.")
	}

	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	filePath = normalizeFilePath(filePath)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	file, ok := share.Files[filePath]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified file does not exist.")
	}

	switch action {
	case "acquire":
		if strings.TrimSpace(header.Get("x-ms-lease-duration")) == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-lease-duration header is required.")
		}
		if duration := strings.TrimSpace(header.Get("x-ms-lease-duration")); duration != "-1" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-lease-duration header must be -1 for Azure Files.")
		}
		leaseID := strings.TrimSpace(header.Get("x-ms-proposed-lease-id"))
		if leaseID == "" {
			leaseID = s.nextToken("lease-")
		}
		if fileLeaseState(file) == "leased" && file.LeaseID != leaseID {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseAlreadyPresent", "There is already a lease present.")
		}
		file.LeaseID = leaseID
		file.LeaseState = "leased"
		share.Files[filePath] = file
		s.fileShares[accountKey][shareKey] = share
		headers := storageHeaders(file.ETag, file.LastModified)
		headers["x-ms-lease-id"] = leaseID
		return emptyResponse(http.StatusCreated, headers)
	case "renew":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		if leaseID == "" || leaseID != file.LeaseID || fileLeaseState(file) != "leased" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the file.")
		}
		headers := storageHeaders(file.ETag, file.LastModified)
		headers["x-ms-lease-id"] = leaseID
		return emptyResponse(http.StatusOK, headers)
	case "change":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		proposed := strings.TrimSpace(header.Get("x-ms-proposed-lease-id"))
		if proposed == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-proposed-lease-id header is required.")
		}
		if leaseID == "" || leaseID != file.LeaseID || fileLeaseState(file) != "leased" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the file.")
		}
		file.LeaseID = proposed
		file.LeaseState = "leased"
		share.Files[filePath] = file
		s.fileShares[accountKey][shareKey] = share
		headers := storageHeaders(file.ETag, file.LastModified)
		headers["x-ms-lease-id"] = proposed
		return emptyResponse(http.StatusOK, headers)
	case "release":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		if leaseID == "" || leaseID != file.LeaseID {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the file.")
		}
		file.LeaseID = ""
		file.LeaseState = "available"
		share.Files[filePath] = file
		s.fileShares[accountKey][shareKey] = share
		return emptyResponse(http.StatusOK, storageHeaders(file.ETag, file.LastModified))
	case "break":
		if fileLeaseState(file) != "leased" && fileLeaseState(file) != "broken" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseNotPresentWithLeaseOperation", "There is currently no lease on the file.")
		}
		file.LeaseState = "broken"
		share.Files[filePath] = file
		s.fileShares[accountKey][shareKey] = share
		headers := storageHeaders(file.ETag, file.LastModified)
		headers["x-ms-lease-time"] = "0"
		return emptyResponse(http.StatusAccepted, headers)
	default:
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-lease-action header value is invalid.")
	}
}

func (s *StorageService) copyFile(account, shareName, destinationPath string, header http.Header) (*service.Response, error) {
	sourceURL := strings.TrimSpace(header.Get("x-ms-copy-source"))
	if sourceURL == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-copy-source header is required.")
	}
	source, ok := parseFileCopySource(account, sourceURL)
	if !ok {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-copy-source header must reference a file or blob.")
	}

	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	destinationPath = normalizeFilePath(destinationPath)
	if destinationPath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The destination file path is required.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	parent := fileParentPath(destinationPath)
	if parent != "" {
		if _, ok := share.Directories[parent]; !ok {
			return azurearm.ErrorResponse(http.StatusPreconditionFailed, "ParentNotFound", "The parent directory does not exist.")
		}
	}
	if _, ok := share.Directories[destinationPath]; ok {
		return azurearm.ErrorResponse(http.StatusConflict, "ResourceAlreadyExists", "The specified destination path is a directory.")
	}

	var leaseID, leaseState string
	if destination, exists := share.Files[destinationPath]; exists {
		if !fileLeaseAllowsWrite(header, destination) {
			return fileLeaseWriteFailure(header, destination)
		}
		leaseID = destination.LeaseID
		leaseState = fileLeaseState(destination)
	} else if strings.TrimSpace(header.Get("x-ms-lease-id")) != "" {
		resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseNotPresentWithFileOperation", "There is currently no lease on the destination file.")
		if resp != nil {
			resp.Headers = storageHeaders("", time.Now().UTC())
			resp.Headers["x-ms-error-code"] = "LeaseNotPresentWithFileOperation"
		}
		return resp, err
	}

	content, ranges, contentType, cacheControl, contentEncoding, contentLanguage, contentDisposition, contentMD5, metadata, sourceOK := s.fileCopySourceContent(source)
	if !sourceOK {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified copy source does not exist.")
	}
	if metadataHeadersPresent(header) {
		metadata = metadataFromHeaders(header)
	}
	if leaseState == "broken" && strings.TrimSpace(header.Get("x-ms-lease-id")) == "" {
		leaseID = ""
		leaseState = "available"
	}

	now := time.Now().UTC()
	copyID := s.nextToken("copy-")
	destination := fileObject{
		Path:               destinationPath,
		Content:            append([]byte(nil), content...),
		Ranges:             cloneFileRanges(ranges),
		ContentType:        contentType,
		CacheControl:       cacheControl,
		ContentEncoding:    contentEncoding,
		ContentLanguage:    contentLanguage,
		ContentDisposition: contentDisposition,
		ContentMD5:         contentMD5,
		Metadata:           cloneStringMap(metadata),
		ETag:               s.nextToken("\"file") + "\"",
		LastModified:       now,
		LeaseID:            leaseID,
		LeaseState:         leaseState,
		CopyID:             copyID,
		CopyStatus:         "success",
		CopySource:         sourceURL,
		CopyProgress:       strconv.Itoa(len(content)) + "/" + strconv.Itoa(len(content)),
		CopyCompletionTime: now,
	}
	share.Files[destinationPath] = destination
	s.fileShares[accountKey][shareKey] = share

	headers := storageHeaders(destination.ETag, destination.LastModified)
	headers["x-ms-copy-id"] = copyID
	headers["x-ms-copy-status"] = "success"
	return emptyResponse(http.StatusAccepted, headers)
}

func (s *StorageService) fileCopySourceContent(source fileCopySource) ([]byte, []fileRange, string, string, string, string, string, string, map[string]string, bool) {
	accountKey := strings.ToLower(source.Account)
	containerKey := strings.ToLower(source.Container)
	switch source.Service {
	case "file":
		share, ok := s.fileShares[accountKey][containerKey]
		if !ok {
			return nil, nil, "", "", "", "", "", "", nil, false
		}
		file, ok := share.Files[source.Path]
		if !ok {
			return nil, nil, "", "", "", "", "", "", nil, false
		}
		return file.Content, file.Ranges, file.ContentType, file.CacheControl, file.ContentEncoding, file.ContentLanguage, file.ContentDisposition, file.ContentMD5, file.Metadata, true
	case "blob":
		container, ok := s.containers[accountKey][containerKey]
		if !ok {
			return nil, nil, "", "", "", "", "", "", nil, false
		}
		blob, ok := container.Blobs[source.Path]
		if !ok {
			return nil, nil, "", "", "", "", "", "", nil, false
		}
		return blob.Content, []fileRange{{Start: 0, End: len(blob.Content) - 1}}, blob.ContentType, blob.CacheControl, blob.ContentEncoding, blob.ContentLanguage, blob.ContentDisposition, blob.ContentMD5, blob.Metadata, true
	default:
		return nil, nil, "", "", "", "", "", "", nil, false
	}
}

func (s *StorageService) putFileRange(account, shareName, filePath string, ctx *service.RequestContext) (*service.Response, error) {
	start, end, ok := parseFileRangeHeader(ctx.RawRequest.Header.Get("x-ms-range"))
	if !ok {
		start, end, ok = parseFileRangeHeader(ctx.RawRequest.Header.Get("Range"))
	}
	if !ok {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-range header must be a valid byte range.")
	}

	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	filePath = normalizeFilePath(filePath)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, okShare := s.fileShares[accountKey][shareKey]
	if !okShare {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	file, okFile := share.Files[filePath]
	if !okFile {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified file does not exist.")
	}
	if !fileLeaseAllowsWrite(ctx.RawRequest.Header, file) {
		return fileLeaseWriteFailure(ctx.RawRequest.Header, file)
	}
	if start < 0 || end < start || end >= len(file.Content) {
		return azurearm.ErrorResponse(http.StatusRequestedRangeNotSatisfiable, "InvalidRange", "The specified range is invalid for the file size.")
	}
	writeMode := strings.ToLower(strings.TrimSpace(ctx.RawRequest.Header.Get("x-ms-write")))
	switch writeMode {
	case "update":
		if len(ctx.Body) != end-start+1 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The request body length must match x-ms-range.")
		}
		copy(file.Content[start:end+1], ctx.Body)
		file.Ranges = addFileRange(file.Ranges, fileRange{Start: start, End: end})
	case "clear":
		clear(file.Content[start : end+1])
		file.Ranges = removeFileRange(file.Ranges, fileRange{Start: start, End: end})
	default:
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-write header must be update or clear.")
	}
	if fileLeaseState(file) == "broken" && strings.TrimSpace(ctx.RawRequest.Header.Get("x-ms-lease-id")) == "" {
		file.LeaseID = ""
		file.LeaseState = "available"
	}
	file.ETag = s.nextToken("\"file") + "\""
	file.LastModified = time.Now().UTC()
	share.Files[filePath] = file
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusCreated, fileObjectHeaders(file))
}

func (s *StorageService) setFileProperties(account, shareName, filePath string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	filePath = normalizeFilePath(filePath)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	file, ok := share.Files[filePath]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified file does not exist.")
	}
	if !fileLeaseAllowsWrite(header, file) {
		return fileLeaseWriteFailure(header, file)
	}
	if hasHeader(header, "x-ms-content-length") {
		length, err := strconv.Atoi(strings.TrimSpace(header.Get("x-ms-content-length")))
		if err != nil || length < 0 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-content-length header must be a nonnegative integer.")
		}
		switch {
		case length < len(file.Content):
			file.Content = file.Content[:length]
			file.Ranges = clipFileRanges(file.Ranges, 0, length-1)
		case length > len(file.Content):
			file.Content = append(file.Content, make([]byte, length-len(file.Content))...)
		}
	}
	if hasAnyHeader(header,
		"x-ms-cache-control",
		"x-ms-content-type",
		"x-ms-content-md5",
		"x-ms-content-encoding",
		"x-ms-content-language",
		"x-ms-content-disposition",
	) {
		file.CacheControl = header.Get("x-ms-cache-control")
		file.ContentType = header.Get("x-ms-content-type")
		file.ContentMD5 = header.Get("x-ms-content-md5")
		file.ContentEncoding = header.Get("x-ms-content-encoding")
		file.ContentLanguage = header.Get("x-ms-content-language")
		file.ContentDisposition = header.Get("x-ms-content-disposition")
	}
	file.ETag = s.nextToken("\"file") + "\""
	file.LastModified = time.Now().UTC()
	share.Files[filePath] = file
	s.fileShares[accountKey][shareKey] = share
	headers := storageHeaders(file.ETag, file.LastModified)
	headers["x-ms-file-attributes"] = "Archive"
	return emptyResponse(http.StatusOK, headers)
}

func (s *StorageService) setFileMetadata(account, shareName, filePath string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	filePath = normalizeFilePath(filePath)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	file, ok := share.Files[filePath]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified file does not exist.")
	}
	if !fileLeaseAllowsWrite(header, file) {
		return fileLeaseWriteFailure(header, file)
	}
	file.Metadata = metadataFromHeaders(header)
	file.ETag = s.nextToken("\"file") + "\""
	file.LastModified = time.Now().UTC()
	share.Files[filePath] = file
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusOK, storageHeaders(file.ETag, file.LastModified))
}

func (s *StorageService) getFileMetadata(account, shareName, filePath string, query url.Values) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	filePath = normalizeFilePath(filePath)

	s.mu.RLock()
	share, ok := s.fileShareForReadLocked(accountKey, shareKey, query.Get("sharesnapshot"))
	if !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	file, ok := share.Files[filePath]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified file does not exist.")
	}
	headers := fileMetadataHeaders(file.ETag, file.LastModified, file.Metadata, "File")
	if snapshot := query.Get("sharesnapshot"); snapshot != "" {
		headers["x-ms-snapshot"] = snapshot
	}
	return emptyResponse(http.StatusOK, headers)
}

func (s *StorageService) renameFile(account, shareName, destinationPath string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	destinationPath = normalizeFilePath(destinationPath)
	if destinationPath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The destination file path is required.")
	}
	sourceShare, sourcePath, ok := parseFileRenameSource(account, header.Get("x-ms-file-rename-source"))
	if !ok || !strings.EqualFold(sourceShare, shareName) || sourcePath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-file-rename-source header must reference a file in the same share.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	file, ok := share.Files[sourcePath]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified source file does not exist.")
	}
	if !fileLeaseAllowsWrite(header, file) {
		return fileLeaseWriteFailure(header, file)
	}
	parent := fileParentPath(destinationPath)
	if parent != "" {
		if _, ok := share.Directories[parent]; !ok {
			return azurearm.ErrorResponse(http.StatusPreconditionFailed, "ParentNotFound", "The parent directory does not exist.")
		}
	}
	if _, ok := share.Directories[destinationPath]; ok {
		return azurearm.ErrorResponse(http.StatusConflict, "ResourceAlreadyExists", "The specified destination path is a directory.")
	}
	if existing, exists := share.Files[destinationPath]; exists && !fileLeaseAllowsWrite(header, existing) {
		return fileLeaseWriteFailure(header, existing)
	}
	if _, exists := share.Files[destinationPath]; exists && !isTruthy(header.Get("x-ms-file-rename-replace-if-exists")) {
		return azurearm.ErrorResponse(http.StatusConflict, "ResourceAlreadyExists", "The specified destination file already exists.")
	}
	if metadataHeadersPresent(header) {
		file.Metadata = metadataFromHeaders(header)
	}
	if contentType := strings.TrimSpace(header.Get("x-ms-content-type")); contentType != "" {
		file.ContentType = contentType
	}
	delete(share.Files, sourcePath)
	file.Path = destinationPath
	file.ETag = s.nextToken("\"file") + "\""
	file.LastModified = time.Now().UTC()
	share.Files[destinationPath] = file
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusOK, fileObjectHeaders(file))
}

func (s *StorageService) listFileRanges(account, shareName, filePath string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	filePath = normalizeFilePath(filePath)

	s.mu.RLock()
	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	file, ok := share.Files[filePath]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified file does not exist.")
	}

	ranges := append([]fileRange(nil), file.Ranges...)
	if rangeHeader := fileRequestRangeFromHeader(header); rangeHeader != "" {
		start, end, ok := parseFileRangeHeader(rangeHeader)
		if !ok || start < 0 || end < start || end >= len(file.Content) {
			return azurearm.ErrorResponse(http.StatusRequestedRangeNotSatisfiable, "InvalidRange", "The specified range is invalid for the file size.")
		}
		ranges = clipFileRanges(ranges, start, end)
	}

	result := fileRangeListResponse{Ranges: make([]fileRangeListItem, 0, len(ranges))}
	for _, fileRange := range ranges {
		result.Ranges = append(result.Ranges, fileRangeListItem{
			Start: fileRange.Start,
			End:   fileRange.End,
		})
	}
	resp, err := xmlResponse(http.StatusOK, result)
	if err != nil {
		return nil, err
	}
	resp.Headers = storageHeaders(file.ETag, file.LastModified)
	resp.Headers["x-ms-content-length"] = strconv.Itoa(len(file.Content))
	return resp, nil
}

func (s *StorageService) getFile(account, shareName, filePath string, req *http.Request) (*service.Response, error) {
	filePath = normalizeFilePath(filePath)

	s.mu.RLock()
	snapshot := req.URL.Query().Get("sharesnapshot")
	share, ok := s.fileShareForReadLocked(strings.ToLower(account), strings.ToLower(shareName), snapshot)
	if !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	file, ok := share.Files[filePath]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified file does not exist.")
	}

	body := file.Content
	status := http.StatusOK
	headers := fileObjectHeaders(file)
	if rangeHeader := fileRequestRange(req); rangeHeader != "" {
		start, end, ok := parseFileRangeHeader(rangeHeader)
		if !ok || start < 0 || end < start || end >= len(file.Content) {
			return azurearm.ErrorResponse(http.StatusRequestedRangeNotSatisfiable, "InvalidRange", "The specified range is invalid for the file size.")
		}
		body = file.Content[start : end+1]
		status = http.StatusPartialContent
		headers["Content-Range"] = "bytes " + strconv.Itoa(start) + "-" + strconv.Itoa(end) + "/" + strconv.Itoa(len(file.Content))
		headers["Content-Length"] = strconv.Itoa(len(body))
	}
	if req.Method == http.MethodHead {
		body = nil
	}
	if snapshot != "" {
		headers["x-ms-snapshot"] = snapshot
	}
	return &service.Response{
		StatusCode:     status,
		RawBody:        body,
		RawContentType: file.ContentType,
		Headers:        headers,
	}, nil
}

func (s *StorageService) deleteFile(account, shareName, filePath string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	filePath = normalizeFilePath(filePath)

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	file, ok := share.Files[filePath]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified file does not exist.")
	}
	if !fileLeaseAllowsWrite(header, file) {
		return fileLeaseWriteFailure(header, file)
	}
	delete(share.Files, filePath)
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusAccepted, map[string]string{"x-ms-version": dataPlaneAPIVersion})
}

func (s *StorageService) deleteFileDirectory(account, shareName, directoryPath string) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	directoryPath = normalizeFilePath(directoryPath)
	if directoryPath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The directory path is required.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	if _, ok := share.Directories[directoryPath]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified directory does not exist.")
	}
	prefix := directoryPath + "/"
	for path := range share.Files {
		if strings.HasPrefix(path, prefix) {
			return azurearm.ErrorResponse(http.StatusConflict, "DirectoryNotEmpty", "The specified directory is not empty.")
		}
	}
	for path := range share.Directories {
		if path != directoryPath && strings.HasPrefix(path, prefix) {
			return azurearm.ErrorResponse(http.StatusConflict, "DirectoryNotEmpty", "The specified directory is not empty.")
		}
	}
	delete(share.Directories, directoryPath)
	s.fileShares[accountKey][shareKey] = share
	return emptyResponse(http.StatusAccepted, map[string]string{"x-ms-version": dataPlaneAPIVersion})
}

func (s *StorageService) getFileDirectoryProperties(account, shareName, directoryPath string) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	directoryPath = normalizeFilePath(directoryPath)
	if directoryPath == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The directory path is required.")
	}

	s.mu.RLock()
	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	directory, ok := share.Directories[directoryPath]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified directory does not exist.")
	}
	return emptyResponse(http.StatusOK, fileDirectoryHeaders(directory))
}

func (s *StorageService) listDirectory(account, shareName, directoryPath string, query url.Values) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	shareKey := strings.ToLower(shareName)
	directoryPath = normalizeFilePath(directoryPath)
	maxResults := parsePositiveInt(query.Get("maxresults"), 0)
	if query.Get("maxresults") != "" && maxResults <= 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The maxresults query parameter must be a positive integer.")
	}
	prefix := query.Get("prefix")

	s.mu.RLock()
	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ShareNotFound", "The specified share does not exist.")
	}
	if directoryPath != "" {
		if _, ok := share.Directories[directoryPath]; !ok {
			s.mu.RUnlock()
			return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified directory does not exist.")
		}
	}

	entries := make([]fileDirectoryListEntry, 0)
	for _, directory := range share.Directories {
		name, ok := fileDirectChildName(directoryPath, directory.Path)
		if !ok || (prefix != "" && !strings.HasPrefix(name, prefix)) {
			continue
		}
		entries = append(entries, fileDirectoryListEntry{Directory: &fileDirectoryListDirectory{
			Name: name,
			Properties: fileDirectoryListDirectoryProperties{
				LastModified: directory.LastModified.UTC().Format(http.TimeFormat),
				ETag:         directory.ETag,
			},
		}})
	}
	for _, file := range share.Files {
		name, ok := fileDirectChildName(directoryPath, file.Path)
		if !ok || (prefix != "" && !strings.HasPrefix(name, prefix)) {
			continue
		}
		entries = append(entries, fileDirectoryListEntry{File: &fileDirectoryListFile{
			Name: name,
			Properties: fileDirectoryListFileProperties{
				ContentLength: len(file.Content),
				LastModified:  file.LastModified.UTC().Format(http.TimeFormat),
				ETag:          file.ETag,
			},
		}})
	}
	s.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name() < entries[j].name()
	})
	nextMarker := ""
	if maxResults > 0 && len(entries) > maxResults {
		nextMarker = entries[maxResults].name()
		entries = entries[:maxResults]
	}

	result := fileDirectoryListResponse{
		ServiceEndpoint: "https://" + account + ".file.core.windows.net/",
		ShareName:       shareName,
		DirectoryPath:   directoryPath,
		Prefix:          prefix,
		Marker:          query.Get("marker"),
		MaxResults:      query.Get("maxresults"),
		Entries:         entries,
		NextMarker:      nextMarker,
	}
	resp, err := xmlResponse(http.StatusOK, result)
	if err != nil {
		return nil, err
	}
	resp.Headers = map[string]string{"x-ms-version": dataPlaneAPIVersion}
	return resp, nil
}

func fileShareHeaders(share fileShare) map[string]string {
	headers := storageHeaders(share.ETag, share.LastModified)
	for key, value := range share.Metadata {
		headers["x-ms-meta-"+key] = value
	}
	if share.Quota != "" {
		headers["x-ms-share-quota"] = share.Quota
	}
	if share.AccessTier != "" {
		headers["x-ms-access-tier"] = share.AccessTier
	}
	if share.EnabledProtocols != "" {
		headers["x-ms-enabled-protocols"] = share.EnabledProtocols
	}
	if share.RootSquash != "" {
		headers["x-ms-root-squash"] = share.RootSquash
	}
	if share.SnapshotVDir != "" {
		headers["x-ms-enable-snapshot-virtual-directory-access"] = share.SnapshotVDir
	}
	switch fileShareLeaseState(share) {
	case "leased", "breaking":
		headers["x-ms-lease-status"] = "locked"
		headers["x-ms-lease-state"] = fileShareLeaseState(share)
	case "broken":
		headers["x-ms-lease-status"] = "unlocked"
		headers["x-ms-lease-state"] = "broken"
	default:
		headers["x-ms-lease-status"] = "unlocked"
		headers["x-ms-lease-state"] = "available"
	}
	return headers
}

func (s *StorageService) fileShareForReadLocked(accountKey, shareKey, snapshot string) (fileShare, bool) {
	share, ok := s.fileShares[accountKey][shareKey]
	if !ok {
		return fileShare{}, false
	}
	if snapshot == "" {
		return share, true
	}
	snapshotShare, ok := share.Snapshots[snapshot]
	return snapshotShare, ok
}

func fileShareSnapshotAllowsRequest(method string, partCount int, query url.Values) bool {
	if method == http.MethodGet || method == http.MethodHead {
		return true
	}
	if method == http.MethodPut && partCount == 1 && strings.EqualFold(query.Get("restype"), "share") && strings.EqualFold(query.Get("comp"), "lease") {
		return true
	}
	return method == http.MethodDelete && partCount == 1 && strings.EqualFold(query.Get("restype"), "share")
}

func fileListIncludes(raw, target string) bool {
	for _, value := range strings.Split(raw, ",") {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func fileShareSnapshotsPresentResponse() (*service.Response, error) {
	resp, err := azurearm.ErrorResponse(http.StatusConflict, "ShareHasSnapshots", "This operation is not permitted because the share has snapshots.")
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = "ShareHasSnapshots"
	}
	return resp, err
}

func cloneFileShare(share fileShare) fileShare {
	clone := share
	clone.Metadata = cloneStringMap(share.Metadata)
	clone.AccessPolicies = cloneFileShareAccessPolicies(share.AccessPolicies)
	clone.FilePermissions = cloneFileSharePermissions(share.FilePermissions)
	clone.LeaseID = ""
	clone.LeaseState = "available"
	clone.Directories = make(map[string]fileDirectory, len(share.Directories))
	for path, directory := range share.Directories {
		directory.Metadata = cloneStringMap(directory.Metadata)
		clone.Directories[path] = directory
	}
	clone.Files = make(map[string]fileObject, len(share.Files))
	for path, file := range share.Files {
		clone.Files[path] = cloneFileObject(file)
	}
	clone.Snapshots = nil
	return clone
}

func cloneFileObject(file fileObject) fileObject {
	clone := file
	clone.Content = append([]byte(nil), file.Content...)
	clone.Ranges = cloneFileRanges(file.Ranges)
	clone.Metadata = cloneStringMap(file.Metadata)
	return clone
}

func fileShareUsageBytes(share fileShare) int {
	total := 0
	for _, file := range share.Files {
		total += len(file.Content)
	}
	return total
}

func parseFileShareACL(body []byte) ([]fileShareSignedIdentifier, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, nil
	}
	var acl fileShareACLResponse
	if err := xml.Unmarshal(body, &acl); err != nil {
		return nil, err
	}
	return cloneFileShareAccessPolicies(acl.SignedIdentifiers), nil
}

func cloneFileShareAccessPolicies(in []fileShareSignedIdentifier) []fileShareSignedIdentifier {
	if len(in) == 0 {
		return nil
	}
	out := make([]fileShareSignedIdentifier, len(in))
	copy(out, in)
	return out
}

func cloneFileSharePermissions(in map[string]fileSharePermission) map[string]fileSharePermission {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]fileSharePermission, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func parseFilePermissionCreateBody(body []byte) (fileSharePermission, *service.Response, error) {
	var raw struct {
		Permission string `json:"permission"`
		Format     string `json:"format"`
	}
	if err := gojson.Unmarshal(body, &raw); err != nil {
		resp, respErr := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidJsonDocument", "The file permission request body must be valid JSON.")
		return fileSharePermission{}, resp, respErr
	}
	raw.Permission = strings.TrimSpace(raw.Permission)
	raw.Format = strings.ToLower(strings.TrimSpace(raw.Format))
	if raw.Format == "" {
		raw.Format = "sddl"
	}
	if raw.Permission == "" {
		resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredProperty", "The permission property is required.")
		return fileSharePermission{}, resp, err
	}
	if raw.Format != "sddl" && raw.Format != "binary" {
		resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidJsonDocument", "The permission format must be sddl or binary.")
		return fileSharePermission{}, resp, err
	}
	if raw.Format == "binary" {
		if _, err := base64.StdEncoding.DecodeString(raw.Permission); err != nil {
			resp, respErr := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidJsonDocument", "Binary file permissions must be base64 encoded.")
			return fileSharePermission{}, resp, respErr
		}
	}
	return fileSharePermission{Permission: raw.Permission, Format: raw.Format}, nil, nil
}

func filePermissionKey(permission fileSharePermission) string {
	sum := sha256.Sum256([]byte(permission.Format + "\x00" + permission.Permission))
	return "perm-" + hex.EncodeToString(sum[:])[:32]
}

func storageVersionAtLeast(version, minimum string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}
	return version >= minimum
}

func fileResourceHeaders(etag string, lastModified time.Time, metadata map[string]string, contentLength int, contentType string, directory bool) map[string]string {
	headers := storageHeaders(etag, lastModified)
	headers["Accept-Ranges"] = "bytes"
	if directory {
		headers["x-ms-type"] = "Directory"
		headers["x-ms-file-attributes"] = "Directory"
	} else {
		headers["x-ms-type"] = "File"
		headers["Content-Length"] = strconv.Itoa(contentLength)
		if contentType != "" {
			headers["Content-Type"] = contentType
		}
		headers["x-ms-file-attributes"] = "Archive"
	}
	for key, value := range metadata {
		headers["x-ms-meta-"+key] = value
	}
	return headers
}

func fileObjectHeaders(file fileObject) map[string]string {
	headers := fileResourceHeaders(file.ETag, file.LastModified, file.Metadata, len(file.Content), file.ContentType, false)
	if file.CacheControl != "" {
		headers["Cache-Control"] = file.CacheControl
	}
	if file.ContentEncoding != "" {
		headers["Content-Encoding"] = file.ContentEncoding
	}
	if file.ContentLanguage != "" {
		headers["Content-Language"] = file.ContentLanguage
	}
	if file.ContentDisposition != "" {
		headers["Content-Disposition"] = file.ContentDisposition
	}
	if file.ContentMD5 != "" {
		headers["Content-MD5"] = file.ContentMD5
	}
	if file.CopyID != "" {
		headers["x-ms-copy-id"] = file.CopyID
		headers["x-ms-copy-status"] = file.CopyStatus
		headers["x-ms-copy-source"] = file.CopySource
		headers["x-ms-copy-progress"] = file.CopyProgress
		if !file.CopyCompletionTime.IsZero() {
			headers["x-ms-copy-completion-time"] = file.CopyCompletionTime.UTC().Format(http.TimeFormat)
		}
	}
	switch fileLeaseState(file) {
	case "leased", "breaking":
		headers["x-ms-lease-status"] = "locked"
		headers["x-ms-lease-state"] = fileLeaseState(file)
	case "broken":
		headers["x-ms-lease-status"] = "unlocked"
		headers["x-ms-lease-state"] = "broken"
	default:
		headers["x-ms-lease-status"] = "unlocked"
		headers["x-ms-lease-state"] = "available"
	}
	return headers
}

func fileDirectoryHeaders(directory fileDirectory) map[string]string {
	headers := fileResourceHeaders(directory.ETag, directory.LastModified, directory.Metadata, 0, "", true)
	if directory.Attributes != "" {
		headers["x-ms-file-attributes"] = directory.Attributes
	}
	if directory.CreationTime != "" {
		headers["x-ms-file-creation-time"] = directory.CreationTime
	}
	if directory.LastWriteTime != "" {
		headers["x-ms-file-last-write-time"] = directory.LastWriteTime
	}
	if directory.ChangeTime != "" {
		headers["x-ms-file-change-time"] = directory.ChangeTime
	}
	return headers
}

func fileMetadataHeaders(etag string, lastModified time.Time, metadata map[string]string, resourceType string) map[string]string {
	headers := storageHeaders(etag, lastModified)
	headers["x-ms-type"] = resourceType
	for key, value := range metadata {
		headers["x-ms-meta-"+key] = value
	}
	return headers
}

func fileLeaseAllowsWrite(header http.Header, file fileObject) bool {
	requestLeaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
	switch {
	case fileLeaseState(file) == "leased" || fileLeaseState(file) == "breaking":
		return requestLeaseID == file.LeaseID
	case requestLeaseID != "":
		return false
	default:
		return true
	}
}

func fileLeaseWriteFailure(header http.Header, file fileObject) (*service.Response, error) {
	leaseState := fileLeaseState(file)
	requestLeaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
	if (leaseState == "leased" || leaseState == "breaking") && requestLeaseID == "" {
		resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseIdMissing", "There is currently a lease on the file and no lease ID was specified in the request.")
		if resp != nil {
			resp.Headers = storageHeaders("", time.Now().UTC())
			resp.Headers["x-ms-error-code"] = "LeaseIdMissing"
		}
		return resp, err
	}
	if requestLeaseID != "" && (leaseState == "available" || leaseState == "broken") {
		resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseNotPresentWithFileOperation", "There is currently no lease on the file.")
		if resp != nil {
			resp.Headers = storageHeaders("", time.Now().UTC())
			resp.Headers["x-ms-error-code"] = "LeaseNotPresentWithFileOperation"
		}
		return resp, err
	}
	resp, err := azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithFileOperation", "The lease ID specified did not match the lease ID for the file.")
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = "LeaseIdMismatchWithFileOperation"
	}
	return resp, err
}

func fileLeaseState(file fileObject) string {
	if file.LeaseState != "" {
		return file.LeaseState
	}
	if file.LeaseID != "" {
		return "leased"
	}
	return "available"
}

func fileShareLeaseAllowsWrite(header http.Header, share fileShare) bool {
	requestLeaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
	switch {
	case fileShareLeaseState(share) == "leased" || fileShareLeaseState(share) == "breaking":
		return requestLeaseID == share.LeaseID
	case requestLeaseID != "":
		return false
	default:
		return true
	}
}

func fileShareLeaseWriteFailure(header http.Header, share fileShare) (*service.Response, error) {
	leaseState := fileShareLeaseState(share)
	requestLeaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
	if (leaseState == "leased" || leaseState == "breaking") && requestLeaseID == "" {
		resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseIdMissing", "There is currently a lease on the share and no lease ID was specified in the request.")
		if resp != nil {
			resp.Headers = storageHeaders("", time.Now().UTC())
			resp.Headers["x-ms-error-code"] = "LeaseIdMissing"
		}
		return resp, err
	}
	if requestLeaseID != "" && (leaseState == "available" || leaseState == "broken") {
		resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseNotPresentWithShareOperation", "There is currently no lease on the share.")
		if resp != nil {
			resp.Headers = storageHeaders("", time.Now().UTC())
			resp.Headers["x-ms-error-code"] = "LeaseNotPresentWithShareOperation"
		}
		return resp, err
	}
	resp, err := azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithShareOperation", "The lease ID specified did not match the lease ID for the share.")
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = "LeaseIdMismatchWithShareOperation"
	}
	return resp, err
}

func fileShareACLLeaseFailure(header http.Header, share fileShare) (*service.Response, error) {
	leaseState := fileShareLeaseState(share)
	requestLeaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
	errorCode := "LeaseIdMismatchWithShareOperation"
	message := "The lease ID specified did not match the lease ID for the share."
	if (leaseState == "leased" || leaseState == "breaking") && requestLeaseID == "" {
		errorCode = "LeaseIdMissing"
		message = "There is currently a lease on the share and no lease ID was specified in the request."
	} else if requestLeaseID != "" && (leaseState == "available" || leaseState == "broken") {
		errorCode = "LeaseNotPresentWithShareOperation"
		message = "There is currently no lease on the share."
	}
	resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, errorCode, message)
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = errorCode
	}
	return resp, err
}

func fileShareLeaseState(share fileShare) string {
	if share.LeaseState != "" {
		return share.LeaseState
	}
	if share.LeaseID != "" {
		return "leased"
	}
	return "available"
}

func normalizeFilePath(path string) string {
	return strings.Trim(strings.Trim(path, "/"), " ")
}

func hasAnyHeader(header http.Header, names ...string) bool {
	for _, name := range names {
		if hasHeader(header, name) {
			return true
		}
	}
	return false
}

func hasHeader(header http.Header, name string) bool {
	for key := range header {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}

func metadataHeadersPresent(header http.Header) bool {
	for key := range header {
		if strings.HasPrefix(strings.ToLower(key), "x-ms-meta-") {
			return true
		}
	}
	return false
}

func applyFileDirectoryPropertyHeaders(directory *fileDirectory, header http.Header) {
	if value := strings.TrimSpace(header.Get("x-ms-file-attributes")); value != "" && !strings.EqualFold(value, "preserve") {
		directory.Attributes = value
	}
	if value := strings.TrimSpace(header.Get("x-ms-file-creation-time")); value != "" && !strings.EqualFold(value, "preserve") {
		directory.CreationTime = value
	}
	if value := strings.TrimSpace(header.Get("x-ms-file-last-write-time")); value != "" && !strings.EqualFold(value, "preserve") {
		directory.LastWriteTime = value
	}
	if value := strings.TrimSpace(header.Get("x-ms-file-change-time")); value != "" && !strings.EqualFold(value, "preserve") {
		if strings.EqualFold(value, "now") {
			value = time.Now().UTC().Format(time.RFC3339)
		}
		directory.ChangeTime = value
	}
}

type fileCopySource struct {
	Service   string
	Account   string
	Container string
	Path      string
	Snapshot  string
}

func parseFileCopySource(defaultAccount, raw string) (fileCopySource, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fileCopySource{}, false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fileCopySource{}, false
	}
	host := strings.ToLower(parsed.Hostname())
	path := parsed.EscapedPath()
	if path == "" {
		path = raw
	}
	parts := splitPath(path)
	if strings.Contains(host, ".file.core.windows.net") {
		account := strings.Split(host, ".")[0]
		if len(parts) < 2 {
			return fileCopySource{}, false
		}
		return fileCopySource{
			Service:   "file",
			Account:   account,
			Container: parts[0],
			Path:      normalizeFilePath(strings.Join(parts[1:], "/")),
			Snapshot:  parsed.Query().Get("snapshot"),
		}, true
	}
	if strings.Contains(host, ".blob.core.windows.net") {
		account := strings.Split(host, ".")[0]
		if len(parts) < 2 {
			return fileCopySource{}, false
		}
		return fileCopySource{
			Service:   "blob",
			Account:   account,
			Container: parts[0],
			Path:      normalizeFilePath(strings.Join(parts[1:], "/")),
			Snapshot:  parsed.Query().Get("snapshot"),
		}, true
	}
	if len(parts) < 2 {
		return fileCopySource{}, false
	}
	first := parts[0]
	if strings.HasSuffix(strings.ToLower(first), "-file") {
		if len(parts) < 3 {
			return fileCopySource{}, false
		}
		return fileCopySource{
			Service:   "file",
			Account:   strings.TrimSuffix(first, "-file"),
			Container: parts[1],
			Path:      normalizeFilePath(strings.Join(parts[2:], "/")),
			Snapshot:  parsed.Query().Get("snapshot"),
		}, true
	}
	if strings.HasSuffix(strings.ToLower(first), "-blob") {
		if len(parts) < 3 {
			return fileCopySource{}, false
		}
		return fileCopySource{
			Service:   "blob",
			Account:   strings.TrimSuffix(first, "-blob"),
			Container: parts[1],
			Path:      normalizeFilePath(strings.Join(parts[2:], "/")),
			Snapshot:  parsed.Query().Get("snapshot"),
		}, true
	}
	if strings.EqualFold(first, defaultAccount) && len(parts) >= 3 {
		return fileCopySource{
			Service:   "blob",
			Account:   first,
			Container: parts[1],
			Path:      normalizeFilePath(strings.Join(parts[2:], "/")),
			Snapshot:  parsed.Query().Get("snapshot"),
		}, true
	}
	return fileCopySource{}, false
}

func parseFileRenameSource(account, raw string) (string, string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", false
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = raw
	}
	parts := splitPath(path)
	if len(parts) > 0 && strings.EqualFold(parts[0], account+"-file") {
		parts = parts[1:]
	}
	if len(parts) < 2 {
		return "", "", false
	}
	return parts[0], normalizeFilePath(strings.Join(parts[1:], "/")), true
}

func cloneFileRanges(in []fileRange) []fileRange {
	if len(in) == 0 {
		return nil
	}
	out := make([]fileRange, 0, len(in))
	for _, r := range in {
		if r.Start <= r.End {
			out = append(out, r)
		}
	}
	return out
}

func isTruthy(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func removeFileDirectorySubtree(share fileShare, directoryPath string) {
	delete(share.Directories, directoryPath)
	prefix := directoryPath + "/"
	for path := range share.Directories {
		if strings.HasPrefix(path, prefix) {
			delete(share.Directories, path)
		}
	}
	for path := range share.Files {
		if strings.HasPrefix(path, prefix) {
			delete(share.Files, path)
		}
	}
}

func renamedFileChildPath(sourcePath, destinationPath, childPath string) string {
	if childPath == sourcePath {
		return destinationPath
	}
	return destinationPath + strings.TrimPrefix(childPath, sourcePath)
}

func fileParentPath(path string) string {
	path = normalizeFilePath(path)
	if !strings.Contains(path, "/") {
		return ""
	}
	return path[:strings.LastIndex(path, "/")]
}

func fileRequestRange(req *http.Request) string {
	return fileRequestRangeFromHeader(req.Header)
}

func fileRequestRangeFromHeader(header http.Header) string {
	if value := header.Get("x-ms-range"); value != "" {
		return value
	}
	return header.Get("Range")
}

func parseFileRangeHeader(raw string) (int, int, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(raw), "bytes=") {
		return 0, 0, false
	}
	parts := strings.SplitN(strings.TrimPrefix(raw, "bytes="), "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return start, end, true
}

func fileDirectChildName(parent, child string) (string, bool) {
	parent = normalizeFilePath(parent)
	child = normalizeFilePath(child)
	if child == "" || child == parent {
		return "", false
	}
	if parent != "" {
		prefix := parent + "/"
		if !strings.HasPrefix(child, prefix) {
			return "", false
		}
		child = strings.TrimPrefix(child, prefix)
	}
	if strings.Contains(child, "/") {
		return "", false
	}
	return child, true
}

func addFileRange(ranges []fileRange, next fileRange) []fileRange {
	if next.Start > next.End {
		return ranges
	}
	ranges = append(ranges, next)
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})
	merged := make([]fileRange, 0, len(ranges))
	for _, candidate := range ranges {
		if len(merged) == 0 || candidate.Start > merged[len(merged)-1].End+1 {
			merged = append(merged, candidate)
			continue
		}
		if candidate.End > merged[len(merged)-1].End {
			merged[len(merged)-1].End = candidate.End
		}
	}
	return merged
}

func removeFileRange(ranges []fileRange, clearRange fileRange) []fileRange {
	if clearRange.Start > clearRange.End {
		return ranges
	}
	out := make([]fileRange, 0, len(ranges))
	for _, existing := range ranges {
		if clearRange.End < existing.Start || clearRange.Start > existing.End {
			out = append(out, existing)
			continue
		}
		if clearRange.Start > existing.Start {
			out = append(out, fileRange{Start: existing.Start, End: clearRange.Start - 1})
		}
		if clearRange.End < existing.End {
			out = append(out, fileRange{Start: clearRange.End + 1, End: existing.End})
		}
	}
	return out
}

func clipFileRanges(ranges []fileRange, start, end int) []fileRange {
	if start > end {
		return nil
	}
	out := make([]fileRange, 0, len(ranges))
	for _, existing := range ranges {
		if end < existing.Start || start > existing.End {
			continue
		}
		clipped := fileRange{Start: existing.Start, End: existing.End}
		if clipped.Start < start {
			clipped.Start = start
		}
		if clipped.End > end {
			clipped.End = end
		}
		out = append(out, clipped)
	}
	return out
}

type fileShareListResponse struct {
	XMLName         xml.Name            `xml:"EnumerationResults"`
	ServiceEndpoint string              `xml:"ServiceEndpoint,attr"`
	Prefix          string              `xml:"Prefix,omitempty"`
	Marker          string              `xml:"Marker,omitempty"`
	MaxResults      string              `xml:"MaxResults,omitempty"`
	Shares          []fileShareListItem `xml:"Shares>Share"`
	NextMarker      string              `xml:"NextMarker,omitempty"`
}

type fileShareListItem struct {
	Name       string                  `xml:"Name"`
	Snapshot   string                  `xml:"Snapshot,omitempty"`
	Properties fileShareListProperties `xml:"Properties"`
	Metadata   *blobListMetadata       `xml:"Metadata,omitempty"`
}

type fileShareListProperties struct {
	LastModified     string `xml:"Last-Modified"`
	ETag             string `xml:"Etag"`
	Quota            string `xml:"Quota,omitempty"`
	AccessTier       string `xml:"AccessTier,omitempty"`
	EnabledProtocols string `xml:"EnabledProtocols,omitempty"`
	RootSquash       string `xml:"RootSquash,omitempty"`
}

type fileShareStatsResponse struct {
	XMLName         xml.Name `xml:"ShareStats"`
	ShareUsageBytes int      `xml:"ShareUsageBytes"`
}

type fileShareACLResponse struct {
	XMLName           xml.Name                    `xml:"SignedIdentifiers"`
	SignedIdentifiers []fileShareSignedIdentifier `xml:"SignedIdentifier"`
}

type fileHandleListResponse struct {
	XMLName       xml.Name                `xml:"EnumerationResults"`
	Marker        string                  `xml:"Marker,omitempty"`
	ShareSnapshot string                  `xml:"ShareSnapshot,omitempty"`
	MaxResults    string                  `xml:"MaxResults,omitempty"`
	HandleList    fileHandleListContainer `xml:"HandleList"`
	NextMarker    string                  `xml:"NextMarker,omitempty"`
}

type fileHandleListContainer struct {
	Handles []fileHandleListItem `xml:"Handle"`
}

type fileHandleListItem struct {
	HandleID          string   `xml:"HandleId"`
	Path              string   `xml:"Path"`
	FileID            string   `xml:"FileId,omitempty"`
	ParentID          string   `xml:"ParentId,omitempty"`
	SessionID         string   `xml:"SessionId,omitempty"`
	ClientIP          string   `xml:"ClientIp,omitempty"`
	ClientName        string   `xml:"ClientName,omitempty"`
	OpenTime          string   `xml:"OpenTime,omitempty"`
	LastReconnectTime string   `xml:"LastReconnectTime,omitempty"`
	AccessRights      []string `xml:"AccessRightList>AccessRight,omitempty"`
}

type fileDirectoryListResponse struct {
	XMLName         xml.Name                 `xml:"EnumerationResults"`
	ServiceEndpoint string                   `xml:"ServiceEndpoint,attr"`
	ShareName       string                   `xml:"ShareName,attr"`
	DirectoryPath   string                   `xml:"DirectoryPath,attr,omitempty"`
	Prefix          string                   `xml:"Prefix,omitempty"`
	Marker          string                   `xml:"Marker,omitempty"`
	MaxResults      string                   `xml:"MaxResults,omitempty"`
	Entries         []fileDirectoryListEntry `xml:"Entries"`
	NextMarker      string                   `xml:"NextMarker"`
}

type fileDirectoryListEntry struct {
	File      *fileDirectoryListFile      `xml:"File,omitempty"`
	Directory *fileDirectoryListDirectory `xml:"Directory,omitempty"`
}

func (entry fileDirectoryListEntry) name() string {
	if entry.File != nil {
		return entry.File.Name
	}
	if entry.Directory != nil {
		return entry.Directory.Name
	}
	return ""
}

type fileDirectoryListFile struct {
	Name       string                          `xml:"Name"`
	Properties fileDirectoryListFileProperties `xml:"Properties"`
}

type fileDirectoryListDirectory struct {
	Name       string                               `xml:"Name"`
	Properties fileDirectoryListDirectoryProperties `xml:"Properties,omitempty"`
}

type fileDirectoryListFileProperties struct {
	ContentLength int    `xml:"Content-Length"`
	LastModified  string `xml:"Last-Modified,omitempty"`
	ETag          string `xml:"Etag,omitempty"`
}

type fileDirectoryListDirectoryProperties struct {
	LastModified string `xml:"Last-Modified,omitempty"`
	ETag         string `xml:"Etag,omitempty"`
}

type fileRangeListResponse struct {
	XMLName xml.Name            `xml:"Ranges"`
	Ranges  []fileRangeListItem `xml:"Range"`
}

type fileRangeListItem struct {
	Start int `xml:"Start"`
	End   int `xml:"End"`
}
