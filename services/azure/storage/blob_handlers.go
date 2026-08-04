package storage

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"hash/crc64"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

var blobRequestIDCounter atomic.Uint64

const maxBlobCORSBytes = 2 * 1024

func (s *StorageService) handleBlob(ctx *service.RequestContext, account string) (resp *service.Response, err error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	host := ctx.RawRequest.URL.Host
	if host == "" {
		host = ctx.RawRequest.Host
	}
	host = strings.ToLower(host)
	if colon := strings.IndexByte(host, ':'); colon >= 0 {
		host = host[:colon]
	}
	if len(parts) > 0 && isLocalStorageHost(host) {
		first := strings.ToLower(parts[0])
		accountKey := strings.ToLower(account)
		if first == accountKey || strings.TrimSuffix(first, "-blob") == accountKey {
			parts = parts[1:]
		}
	}
	if account == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The blob request URI is invalid.")
	}
	defer func() {
		if resp == nil {
			return
		}
		applyBlobResponseHeaders(ctx.RawRequest.Header, resp)
		if ctx.RawRequest.Method != http.MethodOptions {
			s.applyBlobCORSActualHeaders(account, ctx.RawRequest, resp)
		}
	}()
	if ctx.RawRequest.Method == http.MethodOptions {
		if ctx.RawRequest.Header.Get("Origin") == "" || ctx.RawRequest.Header.Get("Access-Control-Request-Method") == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeader", "A CORS preflight request requires Origin and Access-Control-Request-Method headers.")
		}
		if resp, ok := s.blobCORSPreflightResponse(account, ctx.RawRequest.Header); ok {
			return resp, nil
		}
		return azurearm.ErrorResponse(http.StatusForbidden, "CorsNotAllowed", "No CORS rule matches the preflight request.")
	}
	if isBlobAccountInformationRequest(ctx.RawRequest.URL.Query()) {
		switch ctx.RawRequest.Method {
		case http.MethodGet, http.MethodHead:
			return s.getBlobAccountInformation(account, ctx.RawRequest.Header)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if isBlobServiceStatsRequest(ctx.RawRequest.URL.Query()) {
		switch ctx.RawRequest.Method {
		case http.MethodGet:
			return s.getBlobServiceStats(account, ctx.RawRequest.Header)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if isBlobUserDelegationKeyRequest(ctx.RawRequest.URL.Query()) {
		switch ctx.RawRequest.Method {
		case http.MethodPost:
			return s.getBlobUserDelegationKey(account, ctx.Body, ctx.RawRequest.Header)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if isBlobServicePropertiesRequest(ctx.RawRequest.URL.Query()) {
		switch ctx.RawRequest.Method {
		case http.MethodGet:
			return s.getBlobServiceProperties(account, ctx.RawRequest.Header)
		case http.MethodPut:
			return s.setBlobServiceProperties(account, ctx.Body, ctx.RawRequest.Header)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if len(parts) == 0 {
		if ctx.RawRequest.Method == http.MethodGet {
			switch {
			case strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "list"):
				return s.listContainers(account, ctx.RawRequest.URL.Query())
			case strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "blobs"):
				return s.findBlobsByTags(account, ctx.RawRequest.URL.Query())
			}
		}
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The blob request URI is invalid.")
	}

	containerName := parts[0]
	isContainerResource := ctx.RawRequest.URL.Query().Get("restype") == "container"
	if isContainerResource {
		if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "lease") {
			if ctx.RawRequest.Method == http.MethodPut {
				return s.leaseContainer(account, containerName, ctx.RawRequest.Header)
			}
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "metadata") {
			switch ctx.RawRequest.Method {
			case http.MethodPut:
				return s.setContainerMetadata(account, containerName, ctx.RawRequest.Header)
			case http.MethodGet, http.MethodHead:
				return s.getContainerMetadata(account, containerName, ctx.RawRequest.Header)
			default:
				return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
			}
		}
		if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "acl") {
			switch ctx.RawRequest.Method {
			case http.MethodPut:
				return s.setContainerACL(account, containerName, ctx.RawRequest.Header, ctx.Body)
			case http.MethodGet, http.MethodHead:
				return s.getContainerACL(account, containerName, ctx.RawRequest.Header, ctx.RawRequest.Method == http.MethodHead)
			default:
				return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
			}
		}
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.createContainer(account, containerName, ctx.RawRequest.Header)
		case http.MethodGet:
			if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "list") {
				q := ctx.RawRequest.URL.Query()
				maxResults, invalid := parseListMaxResults(q.Get("maxresults"))
				if invalid {
					return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The maxresults query parameter must be greater than 0.")
				}
				return s.listBlobs(account, containerName, q.Get("prefix"), q.Get("delimiter"), q.Get("marker"), maxResults, blobListIncludesMetadata(q.Get("include")))
			}
			if ctx.RawRequest.URL.Query().Get("comp") == "" {
				return s.getContainerProperties(account, containerName, ctx.RawRequest.Header)
			}
		case http.MethodHead:
			if ctx.RawRequest.URL.Query().Get("comp") == "" {
				return s.getContainerProperties(account, containerName, ctx.RawRequest.Header)
			}
		case http.MethodDelete:
			return s.deleteContainer(account, containerName, ctx.RawRequest.Header)
		}
	}

	if len(parts) < 2 {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	blobName := strings.Join(parts[1:], "/")
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "block") {
		if ctx.RawRequest.Method == http.MethodPut {
			if strings.TrimSpace(ctx.RawRequest.Header.Get("x-ms-copy-source")) != "" {
				return s.putBlockFromURL(account, containerName, blobName, ctx.RawRequest.URL.Query().Get("blockid"), ctx.Body, ctx.RawRequest.Header)
			}
			return s.putBlock(account, containerName, blobName, ctx.RawRequest.URL.Query().Get("blockid"), ctx.Body, ctx.RawRequest.Header)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "blocklist") {
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.putBlockList(account, containerName, blobName, ctx)
		case http.MethodGet:
			return s.getBlockList(account, containerName, blobName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "appendblock") {
		if ctx.RawRequest.Method == http.MethodPut {
			return s.appendBlock(account, containerName, blobName, ctx.Body, ctx.RawRequest.Header)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "page") {
		if ctx.RawRequest.Method == http.MethodPut {
			return s.putPage(account, containerName, blobName, ctx.Body, ctx.RawRequest.Header)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "pagelist") {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.getPageRanges(account, containerName, blobName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "copy") {
		if ctx.RawRequest.Method == http.MethodPut {
			return s.abortCopyBlob(account, containerName, blobName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "snapshot") {
		if ctx.RawRequest.Method == http.MethodPut {
			return s.snapshotBlob(account, containerName, blobName, ctx.RawRequest.Header)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "properties") {
		if ctx.RawRequest.Method == http.MethodPut {
			return s.setBlobProperties(account, containerName, blobName, ctx.RawRequest.Header)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "tier") {
		if ctx.RawRequest.Method == http.MethodPut {
			return s.setBlobTier(account, containerName, blobName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "lease") {
		if ctx.RawRequest.Method == http.MethodPut {
			return s.leaseBlob(account, containerName, blobName, ctx.RawRequest.Header)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "tags") {
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.setBlobTags(account, containerName, blobName, ctx.RawRequest.URL.Query(), ctx.Body, ctx.RawRequest.Header)
		case http.MethodGet:
			return s.getBlobTags(account, containerName, blobName, ctx.RawRequest.URL.Query())
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "metadata") {
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.setBlobMetadata(account, containerName, blobName, ctx.RawRequest.Header)
		case http.MethodGet, http.MethodHead:
			return s.getBlobMetadata(account, containerName, blobName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		if strings.TrimSpace(ctx.RawRequest.Header.Get("x-ms-copy-source")) != "" {
			if strings.TrimSpace(ctx.RawRequest.Header.Get("x-ms-blob-type")) != "" {
				return s.putBlobFromURL(account, containerName, blobName, ctx)
			}
			return s.copyBlob(account, containerName, blobName, ctx.RawRequest.Header)
		}
		return s.putBlob(account, containerName, blobName, ctx)
	case http.MethodGet:
		return s.getBlob(account, containerName, blobName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
	case http.MethodHead:
		return s.getBlobProperties(account, containerName, blobName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
	case http.MethodDelete:
		return s.deleteBlob(account, containerName, blobName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *StorageService) createContainer(account, name string, header http.Header) (*service.Response, error) {
	now := time.Now().UTC()
	etag := s.nextToken("\"container")
	etag += "\""

	s.mu.Lock()
	if s.containers[strings.ToLower(account)] != nil {
		if _, exists := s.containers[strings.ToLower(account)][strings.ToLower(name)]; exists {
			s.mu.Unlock()
			return azurearm.ErrorResponse(http.StatusConflict, "ContainerAlreadyExists", "The specified container already exists.")
		}
	}
	if s.containers[strings.ToLower(account)] == nil {
		s.containers[strings.ToLower(account)] = make(map[string]blobContainer)
	}
	s.containers[strings.ToLower(account)][strings.ToLower(name)] = blobContainer{
		Name:         name,
		Metadata:     metadataFromHeaders(header),
		ETag:         etag,
		LastModified: now,
		Blobs:        make(map[string]blobObject),
		StagedBlocks: make(map[string]map[string]blobBlock),
		Snapshots:    make(map[string]map[string]blobObject),
	}
	s.mu.Unlock()

	return emptyResponse(http.StatusCreated, storageHeaders(etag, now))
}

func (s *StorageService) deleteContainer(account, name string, header http.Header) (*service.Response, error) {
	now := time.Now().UTC()
	etag := s.nextToken("\"container-delete") + "\""

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(name)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if !containerLeaseAllowsDelete(header, container) {
		return containerLeaseDeleteFailure(header, container)
	}
	delete(s.containers[accountKey], containerKey)
	return emptyResponse(http.StatusAccepted, storageHeaders(etag, now))
}

func (s *StorageService) getContainerProperties(account, containerName string, header http.Header) (*service.Response, error) {
	s.mu.RLock()
	container, ok := s.containers[strings.ToLower(account)][strings.ToLower(containerName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if !blobConditionsMatch(header, container.ETag) {
		return blobConditionNotMetResponse()
	}
	if !containerLeaseAllowsRead(header, container) {
		return containerLeaseReadFailure()
	}

	headers := storageHeaders(container.ETag, container.LastModified)
	for key, value := range container.Metadata {
		headers["x-ms-meta-"+key] = value
	}
	addContainerLeaseHeaders(headers, container)
	return emptyResponse(http.StatusOK, headers)
}

func (s *StorageService) leaseContainer(account, containerName string, header http.Header) (*service.Response, error) {
	action := strings.ToLower(strings.TrimSpace(header.Get("x-ms-lease-action")))
	if action == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-lease-action header is required.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if !blobConditionsMatch(header, container.ETag) {
		return blobConditionNotMetResponse()
	}

	headers := storageHeaders(container.ETag, container.LastModified)
	switch action {
	case "acquire":
		if strings.TrimSpace(header.Get("x-ms-lease-duration")) == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-lease-duration header is required.")
		}
		leaseID := strings.TrimSpace(header.Get("x-ms-proposed-lease-id"))
		if leaseID == "" {
			leaseID = s.nextToken("lease-")
		}
		if containerLeaseState(container) == "leased" && container.LeaseID != leaseID {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseAlreadyPresent", "There is already a lease present.")
		}
		container.LeaseID = leaseID
		container.LeaseState = "leased"
		if strings.TrimSpace(header.Get("x-ms-lease-duration")) == "-1" {
			container.LeaseDuration = "infinite"
		} else {
			container.LeaseDuration = "fixed"
		}
		headers["x-ms-lease-id"] = leaseID
		s.containers[accountKey][containerKey] = container
		return emptyResponse(http.StatusCreated, headers)
	case "renew":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		if leaseID == "" || leaseID != container.LeaseID || containerLeaseState(container) != "leased" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the container.")
		}
		headers["x-ms-lease-id"] = leaseID
		return emptyResponse(http.StatusOK, headers)
	case "change":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		proposed := strings.TrimSpace(header.Get("x-ms-proposed-lease-id"))
		if proposed == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-proposed-lease-id header is required.")
		}
		if leaseID == "" || leaseID != container.LeaseID || containerLeaseState(container) != "leased" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the container.")
		}
		container.LeaseID = proposed
		container.LeaseState = "leased"
		headers["x-ms-lease-id"] = proposed
		s.containers[accountKey][containerKey] = container
		return emptyResponse(http.StatusOK, headers)
	case "release":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		if leaseID == "" || leaseID != container.LeaseID {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the container.")
		}
		container.LeaseID = ""
		container.LeaseState = "available"
		container.LeaseDuration = ""
		s.containers[accountKey][containerKey] = container
		return emptyResponse(http.StatusOK, headers)
	case "break":
		if containerLeaseState(container) != "leased" && containerLeaseState(container) != "breaking" && containerLeaseState(container) != "broken" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseNotPresentWithLeaseOperation", "There is currently no lease on the container.")
		}
		container.LeaseState = "broken"
		container.LeaseDuration = ""
		headers["x-ms-lease-time"] = "0"
		s.containers[accountKey][containerKey] = container
		return emptyResponse(http.StatusAccepted, headers)
	default:
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-lease-action header value is invalid.")
	}
}

func addContainerLeaseHeaders(headers map[string]string, container blobContainer) {
	switch containerLeaseState(container) {
	case "leased", "breaking":
		headers["x-ms-lease-status"] = "locked"
		headers["x-ms-lease-state"] = containerLeaseState(container)
		if container.LeaseDuration != "" {
			headers["x-ms-lease-duration"] = container.LeaseDuration
		}
	case "broken":
		headers["x-ms-lease-status"] = "unlocked"
		headers["x-ms-lease-state"] = "broken"
	default:
		headers["x-ms-lease-status"] = "unlocked"
		headers["x-ms-lease-state"] = "available"
	}
}

func containerLeaseState(container blobContainer) string {
	if container.LeaseState != "" {
		return container.LeaseState
	}
	if container.LeaseID != "" {
		return "leased"
	}
	return "available"
}

func containerLeaseAllowsDelete(header http.Header, container blobContainer) bool {
	requestLeaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
	switch {
	case containerLeaseState(container) == "leased" || containerLeaseState(container) == "breaking":
		return requestLeaseID == container.LeaseID
	case requestLeaseID != "":
		return false
	default:
		return true
	}
}

func containerLeaseAllowsRead(header http.Header, container blobContainer) bool {
	requestLeaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
	if requestLeaseID == "" {
		return true
	}
	return containerLeaseState(container) == "leased" && requestLeaseID == container.LeaseID
}

func containerLeaseReadFailure() (*service.Response, error) {
	resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseIdMismatchWithContainerOperation", "The lease ID specified did not match the lease ID for the container.")
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = "LeaseIdMismatchWithContainerOperation"
	}
	return resp, err
}

func containerLeaseDeleteFailure(header http.Header, container blobContainer) (*service.Response, error) {
	if (containerLeaseState(container) == "leased" || containerLeaseState(container) == "breaking") && strings.TrimSpace(header.Get("x-ms-lease-id")) == "" {
		resp, err := azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMissing", "There is currently a lease on the container and no lease ID was specified in the request.")
		if resp != nil {
			resp.Headers = storageHeaders("", time.Now().UTC())
			resp.Headers["x-ms-error-code"] = "LeaseIdMissing"
		}
		return resp, err
	}
	resp, err := azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithContainerOperation", "The lease ID specified did not match the lease ID for the container.")
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = "LeaseIdMismatchWithContainerOperation"
	}
	return resp, err
}

func (s *StorageService) setContainerMetadata(account, containerName string, header http.Header) (*service.Response, error) {
	now := time.Now().UTC()
	etag := s.nextToken("\"container") + "\""

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if !blobConditionsMatch(header, container.ETag) {
		return blobConditionNotMetResponse()
	}
	container.Metadata = metadataFromHeaders(header)
	container.ETag = etag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container
	return emptyResponse(http.StatusOK, storageHeaders(etag, now))
}

func (s *StorageService) getContainerACL(account, containerName string, header http.Header, headOnly bool) (*service.Response, error) {
	s.mu.RLock()
	container, ok := s.containers[strings.ToLower(account)][strings.ToLower(containerName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if !containerLeaseAllowsRead(header, container) {
		return containerLeaseReadFailure()
	}
	headers := storageHeaders(container.ETag, container.LastModified)
	if container.PublicAccess != "" {
		headers["x-ms-blob-public-access"] = container.PublicAccess
	}
	if headOnly {
		return emptyResponse(http.StatusOK, headers)
	}
	resp, err := xmlResponse(http.StatusOK, blobContainerACLResponse{SignedIdentifiers: cloneBlobContainerAccessPolicies(container.AccessPolicies)})
	if err != nil {
		return nil, err
	}
	resp.Headers = headers
	return resp, nil
}

func (s *StorageService) setContainerACL(account, containerName string, header http.Header, body []byte) (*service.Response, error) {
	policies, err := parseBlobContainerACL(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", err.Error())
	}
	if len(policies) > 5 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", "A container can contain at most five stored access policies.")
	}
	for _, policy := range policies {
		if len(policy.ID) > 64 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", "A stored access policy identifier cannot exceed 64 characters.")
		}
	}
	publicAccess := strings.TrimSpace(header.Get("x-ms-blob-public-access"))
	if publicAccess != "" && publicAccess != "blob" && publicAccess != "container" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-public-access header value is invalid.")
	}

	now := time.Now().UTC()
	etag := s.nextToken("\"container") + "\""

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if !containerLeaseAllowsRead(header, container) {
		return containerLeaseReadFailure()
	}
	container.AccessPolicies = cloneBlobContainerAccessPolicies(policies)
	container.PublicAccess = publicAccess
	container.ETag = etag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container
	return emptyResponse(http.StatusOK, storageHeaders(etag, now))
}

func parseBlobContainerACL(body []byte) ([]blobContainerSignedIdentifier, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, nil
	}
	var acl blobContainerACLResponse
	if err := xml.Unmarshal(body, &acl); err != nil {
		return nil, err
	}
	return cloneBlobContainerAccessPolicies(acl.SignedIdentifiers), nil
}

func cloneBlobContainerAccessPolicies(in []blobContainerSignedIdentifier) []blobContainerSignedIdentifier {
	if len(in) == 0 {
		return nil
	}
	out := make([]blobContainerSignedIdentifier, len(in))
	copy(out, in)
	return out
}

func isBlobAccountInformationRequest(query url.Values) bool {
	return strings.EqualFold(query.Get("restype"), "account") && strings.EqualFold(query.Get("comp"), "properties")
}

func isBlobServicePropertiesRequest(query url.Values) bool {
	return strings.EqualFold(query.Get("restype"), "service") && strings.EqualFold(query.Get("comp"), "properties")
}

func isBlobServiceStatsRequest(query url.Values) bool {
	return strings.EqualFold(query.Get("restype"), "service") && strings.EqualFold(query.Get("comp"), "stats")
}

func isBlobUserDelegationKeyRequest(query url.Values) bool {
	return strings.EqualFold(query.Get("restype"), "service") && strings.EqualFold(query.Get("comp"), "userdelegationkey")
}

func (s *StorageService) getBlobAccountInformation(account string, header http.Header) (*service.Response, error) {
	skuName, accountKind := s.dataPlaneStorageAccountProperties(account)
	headers := map[string]string{
		"Content-Length":      "0",
		"x-ms-version":        dataPlaneAPIVersion,
		"x-ms-sku-name":       skuName,
		"x-ms-account-kind":   accountKind,
		"x-ms-is-hns-enabled": "false",
	}
	if requestID := header.Get("x-ms-client-request-id"); requestID != "" && len(requestID) <= 1024 {
		headers["x-ms-client-request-id"] = requestID
	}
	return emptyResponse(http.StatusOK, headers)
}

func (s *StorageService) getBlobServiceStats(account string, header http.Header) (*service.Response, error) {
	if !blobSecondaryAccountName(account) {
		return azurearm.ErrorResponse(http.StatusForbidden, "InsufficientAccountPermissions", "The account being accessed does not have sufficient permissions to execute this operation.")
	}
	resp, err := xmlResponse(http.StatusOK, blobServiceStatsResponse{
		GeoReplication: blobGeoReplicationStats{
			Status:       "live",
			LastSyncTime: time.Now().UTC().Format(http.TimeFormat),
		},
	})
	if err != nil {
		return nil, err
	}
	resp.Headers = blobBaseHeadersForRequest(header)
	return resp, nil
}

func blobSecondaryAccountName(account string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(account)), "-secondary")
}

func (s *StorageService) getBlobUserDelegationKey(account string, body []byte, header http.Header) (*service.Response, error) {
	if blobServicePropertiesAPIVersion(header.Get("x-ms-version")) < "2018-11-09" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "FeatureVersionMismatch", "Get User Delegation Key requires Blob service version 2018-11-09 or later.")
	}
	var keyInfo blobUserDelegationKeyInfo
	if err := xml.Unmarshal(body, &keyInfo); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", err.Error())
	}
	if keyInfo.XMLName.Local != "KeyInfo" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", "expected KeyInfo root element")
	}
	startText := strings.TrimSpace(keyInfo.Start)
	expiryText := strings.TrimSpace(keyInfo.Expiry)
	if startText == "" || expiryText == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredXmlNode", "Start and Expiry are required.")
	}
	start, err := time.Parse(time.RFC3339, startText)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlNodeValue", "Start must be a valid ISO date.")
	}
	expiry, err := time.Parse(time.RFC3339, expiryText)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlNodeValue", "Expiry must be a valid ISO date.")
	}
	if !expiry.After(start) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlNodeValue", "Expiry must be later than Start.")
	}
	now := time.Now().UTC()
	if start.Before(now.Add(-7*24*time.Hour)) || start.After(now.Add(7*24*time.Hour)) ||
		expiry.Before(now.Add(-7*24*time.Hour)) || expiry.After(now.Add(7*24*time.Hour)) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlNodeValue", "Start and Expiry must be within seven days of the current time.")
	}

	tenantID := "00000000-0000-0000-0000-000000000001"
	response := blobUserDelegationKeyResponse{
		SignedOid:              "00000000-0000-0000-0000-000000000001",
		SignedTid:              tenantID,
		SignedStart:            startText,
		SignedExpiry:           expiryText,
		SignedService:          "b",
		SignedVersion:          blobServicePropertiesAPIVersion(header.Get("x-ms-version")),
		SignedDelegatedUserTid: strings.TrimSpace(keyInfo.DelegatedUserTid),
		Value:                  base64.StdEncoding.EncodeToString([]byte("cloudmock:userdelegationkey:" + strings.ToLower(account) + ":" + startText + ":" + expiryText)),
	}
	resp, err := xmlResponse(http.StatusOK, response)
	if err != nil {
		return nil, err
	}
	resp.Headers = blobBaseHeadersForRequest(header)
	return resp, nil
}

func (s *StorageService) getBlobServiceProperties(account string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)

	s.mu.RLock()
	stored := append([]byte(nil), s.blobProps[accountKey]...)
	s.mu.RUnlock()

	if len(stored) == 0 {
		stored = defaultBlobServiceProperties()
	}
	var err error
	stored, err = blobServicePropertiesForVersion(stored, header.Get("x-ms-version"))
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     http.StatusOK,
		Headers:        blobBaseHeadersForRequest(header),
		RawBody:        stored,
		RawContentType: "application/xml",
	}, nil
}

func (s *StorageService) setBlobServiceProperties(account string, body []byte, header http.Header) (*service.Response, error) {
	update, err := parseBlobServiceProperties(body, header.Get("x-ms-version"))
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", err.Error())
	}

	accountKey := strings.ToLower(account)
	s.mu.Lock()
	current := append([]byte(nil), s.blobProps[accountKey]...)
	if len(current) == 0 {
		current = defaultBlobServiceProperties()
	}
	merged, err := mergeBlobServiceProperties(current, update)
	if err != nil {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", err.Error())
	}
	s.blobProps[accountKey] = merged
	s.mu.Unlock()

	return emptyResponse(http.StatusAccepted, blobBaseHeadersForRequest(header))
}

func (s *StorageService) blobCORSPreflightResponse(account string, header http.Header) (*service.Response, bool) {
	origin := header.Get("Origin")
	method := header.Get("Access-Control-Request-Method")
	if origin == "" || method == "" {
		return nil, false
	}

	rules, ok := s.blobCORSRules(account)
	if !ok {
		return nil, false
	}
	requestedHeaders := queueCORSHeaderValues(header.Get("Access-Control-Request-Headers"))
	for _, rule := range rules {
		if !queueCORSOriginMatches(rule.AllowedOrigins, origin) ||
			!queueCORSMethodMatches(rule.AllowedMethods, method) ||
			!queueCORSHeadersMatch(rule.AllowedHeaders, requestedHeaders) {
			continue
		}
		headers := blobBaseHeadersForRequest(header)
		headers["Access-Control-Allow-Origin"] = origin
		headers["Access-Control-Allow-Methods"] = strings.ToUpper(strings.TrimSpace(method))
		if header.Get("Access-Control-Request-Headers") != "" {
			headers["Access-Control-Allow-Headers"] = header.Get("Access-Control-Request-Headers")
		}
		headers["Access-Control-Max-Age"] = strings.TrimSpace(rule.MaxAgeInSeconds)
		return &service.Response{StatusCode: http.StatusOK, Headers: headers}, true
	}
	return nil, false
}

func (s *StorageService) applyBlobCORSActualHeaders(account string, req *http.Request, resp *service.Response) {
	origin := req.Header.Get("Origin")
	if origin == "" {
		return
	}
	rule, wildcardOrigin, ok := s.blobCORSActualRule(account, origin, req.Method, req.Header)
	if !ok {
		if (req.Method == http.MethodGet || req.Method == http.MethodHead) && s.blobCORSRulesEnabled(account) {
			if resp.Headers == nil {
				resp.Headers = make(map[string]string)
			}
			resp.Headers["Vary"] = "Origin"
		}
		return
	}
	if resp.Headers == nil {
		resp.Headers = make(map[string]string)
	}
	if wildcardOrigin {
		resp.Headers["Access-Control-Allow-Origin"] = "*"
	} else {
		resp.Headers["Access-Control-Allow-Origin"] = origin
		if req.Method == http.MethodGet || req.Method == http.MethodHead {
			resp.Headers["Vary"] = "Origin"
		}
	}
	if exposed := strings.TrimSpace(rule.ExposedHeaders); exposed != "" {
		resp.Headers["Access-Control-Expose-Headers"] = exposed
	}
}

func (s *StorageService) blobCORSActualRule(account, origin, method string, header http.Header) (queueServicePropertiesCorsRule, bool, bool) {
	rules, ok := s.blobCORSRules(account)
	if !ok {
		return queueServicePropertiesCorsRule{}, false, false
	}
	requestedHeaders := queueCORSActualRequestHeaders(header)
	for _, rule := range rules {
		wildcardOrigin, originMatched := queueCORSOriginMatchType(rule.AllowedOrigins, origin)
		if originMatched && queueCORSMethodMatches(rule.AllowedMethods, method) && queueCORSHeadersMatch(rule.AllowedHeaders, requestedHeaders) {
			return rule, wildcardOrigin, true
		}
	}
	return queueServicePropertiesCorsRule{}, false, false
}

func (s *StorageService) blobCORSRulesEnabled(account string) bool {
	rules, ok := s.blobCORSRules(account)
	return ok && len(rules) > 0
}

func (s *StorageService) blobCORSRules(account string) ([]queueServicePropertiesCorsRule, bool) {
	accountKey := strings.ToLower(account)
	s.mu.RLock()
	stored := append([]byte(nil), s.blobProps[accountKey]...)
	s.mu.RUnlock()
	if len(stored) == 0 {
		return nil, false
	}
	var properties queueServicePropertiesDocument
	if err := xml.Unmarshal(stored, &properties); err != nil || properties.Cors == nil || len(properties.Cors.Rules) == 0 {
		return nil, false
	}
	return properties.Cors.Rules, true
}

func (s *StorageService) dataPlaneStorageAccountProperties(account string) (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, stored := range s.accounts {
		if strings.EqualFold(stored.Name, account) {
			skuName := stored.SKU.Name
			if skuName == "" {
				skuName = "Standard_LRS"
			}
			accountKind := stored.Kind
			if accountKind == "" {
				accountKind = "StorageV2"
			}
			return skuName, accountKind
		}
	}
	return "Standard_LRS", "StorageV2"
}

var blobServicePropertyRootOrder = []string{"Logging", "Metrics", "HourMetrics", "MinuteMetrics", "Cors", "DefaultServiceVersion", "DeleteRetentionPolicy", "StaticWebsite"}

type blobServicePropertiesRawDocument struct {
	XMLName               xml.Name                          `xml:"StorageServiceProperties"`
	Logging               *queueServicePropertiesRawElement `xml:"Logging"`
	Metrics               *queueServicePropertiesRawElement `xml:"Metrics"`
	HourMetrics           *queueServicePropertiesRawElement `xml:"HourMetrics"`
	MinuteMetrics         *queueServicePropertiesRawElement `xml:"MinuteMetrics"`
	Cors                  *queueServicePropertiesRawElement `xml:"Cors"`
	DefaultServiceVersion *queueServicePropertiesRawElement `xml:"DefaultServiceVersion"`
	DeleteRetentionPolicy *queueServicePropertiesRawElement `xml:"DeleteRetentionPolicy"`
	StaticWebsite         *queueServicePropertiesRawElement `xml:"StaticWebsite"`
}

type blobServicePropertiesDocument struct {
	XMLName               xml.Name                       `xml:"StorageServiceProperties"`
	Logging               *queueServicePropertiesLogging `xml:"Logging"`
	Metrics               *queueServicePropertiesMetrics `xml:"Metrics"`
	HourMetrics           *queueServicePropertiesMetrics `xml:"HourMetrics"`
	MinuteMetrics         *queueServicePropertiesMetrics `xml:"MinuteMetrics"`
	Cors                  *queueServicePropertiesCors    `xml:"Cors"`
	DefaultServiceVersion string                         `xml:"DefaultServiceVersion"`
	DeleteRetentionPolicy *blobServiceDeleteRetention    `xml:"DeleteRetentionPolicy"`
	StaticWebsite         *blobServiceStaticWebsite      `xml:"StaticWebsite"`
}

type blobServiceStatsResponse struct {
	XMLName        xml.Name                `xml:"StorageServiceStats"`
	GeoReplication blobGeoReplicationStats `xml:"GeoReplication"`
}

type blobGeoReplicationStats struct {
	Status       string `xml:"Status"`
	LastSyncTime string `xml:"LastSyncTime"`
}

type blobUserDelegationKeyInfo struct {
	XMLName          xml.Name `xml:"KeyInfo"`
	Start            string   `xml:"Start"`
	Expiry           string   `xml:"Expiry"`
	DelegatedUserTid string   `xml:"DelegatedUserTid"`
}

type blobUserDelegationKeyResponse struct {
	XMLName                xml.Name `xml:"UserDelegationKey"`
	SignedOid              string   `xml:"SignedOid"`
	SignedTid              string   `xml:"SignedTid"`
	SignedStart            string   `xml:"SignedStart"`
	SignedExpiry           string   `xml:"SignedExpiry"`
	SignedService          string   `xml:"SignedService"`
	SignedVersion          string   `xml:"SignedVersion"`
	SignedDelegatedUserTid string   `xml:"SignedDelegatedUserTid,omitempty"`
	Value                  string   `xml:"Value"`
}

type blobServiceDeleteRetention struct {
	Enabled              string `xml:"Enabled"`
	Days                 string `xml:"Days"`
	AllowPermanentDelete string `xml:"AllowPermanentDelete"`
}

type blobServiceStaticWebsite struct {
	Enabled                  string `xml:"Enabled"`
	IndexDocument            string `xml:"IndexDocument"`
	DefaultIndexDocumentPath string `xml:"DefaultIndexDocumentPath"`
	ErrorDocument404Path     string `xml:"ErrorDocument404Path"`
}

func parseBlobServiceProperties(body []byte, apiVersion string) ([]byte, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, &xml.SyntaxError{Msg: "missing StorageServiceProperties"}
	}
	var properties blobServicePropertiesDocument
	if err := xml.Unmarshal([]byte(trimmed), &properties); err != nil {
		return nil, err
	}
	if properties.XMLName.Local != "StorageServiceProperties" {
		return nil, fmt.Errorf("expected StorageServiceProperties root element")
	}
	if err := validateBlobServicePropertiesVersion(properties, apiVersion); err != nil {
		return nil, err
	}
	if err := validateBlobServiceRootProperties(properties); err != nil {
		return nil, err
	}
	if properties.Cors != nil {
		if err := validateBlobServiceCORSProperties(properties.Cors); err != nil {
			return nil, err
		}
	}
	elements, err := blobServicePropertyRootElements([]byte(trimmed))
	if err != nil {
		return nil, err
	}
	if len(elements) == 0 {
		return nil, fmt.Errorf("StorageServiceProperties must include at least one property root element")
	}
	if blobServicePropertiesUsesLegacyMetrics(apiVersion) {
		return modernBlobServicePropertiesFromLegacy([]byte(trimmed))
	}
	return []byte(trimmed), nil
}

func validateBlobServicePropertiesVersion(properties blobServicePropertiesDocument, apiVersion string) error {
	effectiveVersion := blobServicePropertiesAPIVersion(apiVersion)
	if strings.TrimSpace(properties.DefaultServiceVersion) != "" && effectiveVersion < "2011-08-18" {
		return fmt.Errorf("DefaultServiceVersion is supported for Blob service version 2011-08-18 and later")
	}
	if effectiveVersion < "2013-08-15" {
		if properties.HourMetrics != nil || properties.MinuteMetrics != nil || properties.Cors != nil {
			return fmt.Errorf("HourMetrics, MinuteMetrics, and Cors are supported for Blob service version 2013-08-15 and later")
		}
		return nil
	}
	if properties.Metrics != nil {
		return fmt.Errorf("modern Blob service properties do not support the legacy Metrics root element")
	}
	if properties.DeleteRetentionPolicy != nil && effectiveVersion < "2017-07-29" {
		return fmt.Errorf("DeleteRetentionPolicy is supported for Blob service version 2017-07-29 and later")
	}
	if properties.StaticWebsite != nil && effectiveVersion < "2018-03-28" {
		return fmt.Errorf("StaticWebsite is supported for Blob service version 2018-03-28 and later")
	}
	if properties.DeleteRetentionPolicy != nil &&
		strings.TrimSpace(properties.DeleteRetentionPolicy.AllowPermanentDelete) != "" &&
		effectiveVersion < "2020-02-10" {
		return fmt.Errorf("DeleteRetentionPolicy AllowPermanentDelete is supported for Blob service version 2020-02-10 and later")
	}
	return nil
}

func blobServicePropertiesAPIVersion(apiVersion string) string {
	if apiVersion = strings.TrimSpace(apiVersion); apiVersion != "" {
		return apiVersion
	}
	return dataPlaneAPIVersion
}

func blobServicePropertiesUsesLegacyMetrics(apiVersion string) bool {
	apiVersion = strings.TrimSpace(apiVersion)
	return apiVersion != "" && apiVersion <= "2012-02-12"
}

func legacyBlobServiceProperties(body []byte) ([]byte, error) {
	elements, err := blobServicePropertyRootElements(body)
	if err != nil {
		return nil, err
	}
	metrics := elements["Metrics"]
	if metrics == "" {
		metrics = elements["HourMetrics"]
	}

	var out strings.Builder
	out.WriteString("<StorageServiceProperties>")
	if logging := elements["Logging"]; logging != "" {
		out.WriteString(logging)
	}
	if metrics != "" {
		out.WriteString("<Metrics>")
		out.WriteString(queueServiceRootInnerXML(metrics))
		out.WriteString("</Metrics>")
	}
	if defaultVersion := elements["DefaultServiceVersion"]; defaultVersion != "" {
		out.WriteString(defaultVersion)
	}
	out.WriteString("</StorageServiceProperties>")
	return []byte(out.String()), nil
}

func blobServicePropertiesForVersion(body []byte, apiVersion string) ([]byte, error) {
	if blobServicePropertiesUsesLegacyMetrics(apiVersion) {
		return legacyBlobServiceProperties(body)
	}
	elements, err := blobServicePropertyRootElements(body)
	if err != nil {
		return nil, err
	}
	version := blobServicePropertiesAPIVersion(apiVersion)
	if version < "2017-07-29" {
		delete(elements, "DeleteRetentionPolicy")
	}
	if version < "2018-03-28" {
		delete(elements, "StaticWebsite")
	} else if version < "2019-12-12" {
		staticWebsite := elements["StaticWebsite"]
		if staticWebsite != "" {
			projected, err := blobStaticWebsiteWithoutDefaultIndexDocument(staticWebsite)
			if err != nil {
				return nil, err
			}
			elements["StaticWebsite"] = projected
		}
	}
	return buildBlobServicePropertiesFromElements(elements), nil
}

type blobServiceStaticWebsiteBefore20191212 struct {
	XMLName              xml.Name `xml:"StaticWebsite"`
	Enabled              string   `xml:"Enabled,omitempty"`
	IndexDocument        string   `xml:"IndexDocument,omitempty"`
	ErrorDocument404Path string   `xml:"ErrorDocument404Path,omitempty"`
}

func blobStaticWebsiteWithoutDefaultIndexDocument(element string) (string, error) {
	var properties blobServiceStaticWebsiteBefore20191212
	if err := xml.Unmarshal([]byte(element), &properties); err != nil {
		return "", err
	}
	projected, err := xml.Marshal(properties)
	if err != nil {
		return "", err
	}
	return string(projected), nil
}

func modernBlobServicePropertiesFromLegacy(body []byte) ([]byte, error) {
	elements, err := blobServicePropertyRootElements(body)
	if err != nil {
		return nil, err
	}
	logging := elements["Logging"]
	metrics := elements["Metrics"]
	if logging == "" || metrics == "" {
		return nil, fmt.Errorf("legacy Blob service properties require Logging and Metrics root elements")
	}

	var out strings.Builder
	out.WriteString("<StorageServiceProperties>")
	out.WriteString(logging)
	out.WriteString("<HourMetrics>")
	out.WriteString(queueServiceRootInnerXML(metrics))
	out.WriteString("</HourMetrics>")
	out.WriteString(defaultBlobMinuteMetrics())
	out.WriteString("<Cors></Cors>")
	if defaultVersion := elements["DefaultServiceVersion"]; defaultVersion != "" {
		out.WriteString(defaultVersion)
	}
	out.WriteString("</StorageServiceProperties>")
	return []byte(out.String()), nil
}

func defaultBlobMinuteMetrics() string {
	elements, err := blobServicePropertyRootElements(defaultBlobServiceProperties())
	if err != nil {
		return `<MinuteMetrics><Version>1.0</Version><Enabled>false</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></MinuteMetrics>`
	}
	if minuteMetrics := elements["MinuteMetrics"]; minuteMetrics != "" {
		return minuteMetrics
	}
	return `<MinuteMetrics><Version>1.0</Version><Enabled>false</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></MinuteMetrics>`
}

func validateBlobServiceRootProperties(properties blobServicePropertiesDocument) error {
	if properties.Logging != nil {
		if err := validateQueueServiceLoggingProperties(properties.Logging); err != nil {
			return err
		}
	}
	if properties.Metrics != nil {
		if err := validateQueueServiceMetricsProperties("Metrics", properties.Metrics); err != nil {
			return err
		}
	}
	if properties.HourMetrics != nil {
		if err := validateQueueServiceMetricsProperties("HourMetrics", properties.HourMetrics); err != nil {
			return err
		}
	}
	if properties.MinuteMetrics != nil {
		if err := validateQueueServiceMetricsProperties("MinuteMetrics", properties.MinuteMetrics); err != nil {
			return err
		}
	}
	if properties.DeleteRetentionPolicy != nil {
		if err := validateBlobServiceDeleteRetentionPolicy(properties.DeleteRetentionPolicy); err != nil {
			return err
		}
	}
	if properties.StaticWebsite != nil {
		if err := validateBlobServiceStaticWebsite(properties.StaticWebsite); err != nil {
			return err
		}
	}
	return nil
}

func validateBlobServiceDeleteRetentionPolicy(policy *blobServiceDeleteRetention) error {
	enabled, ok := queueServiceBool(policy.Enabled)
	if strings.TrimSpace(policy.Enabled) == "" {
		return fmt.Errorf("DeleteRetentionPolicy settings require Enabled")
	}
	if !ok {
		return fmt.Errorf("DeleteRetentionPolicy Enabled must be true or false")
	}
	if strings.TrimSpace(policy.AllowPermanentDelete) != "" {
		if _, ok := queueServiceBool(policy.AllowPermanentDelete); !ok {
			return fmt.Errorf("DeleteRetentionPolicy AllowPermanentDelete must be true or false")
		}
	}
	daysText := strings.TrimSpace(policy.Days)
	if !enabled && daysText == "" {
		return nil
	}
	if enabled && daysText == "" {
		return fmt.Errorf("DeleteRetentionPolicy Days is required when retention is enabled")
	}
	days, err := strconv.Atoi(daysText)
	if err != nil || days < 1 || days > 365 {
		return fmt.Errorf("DeleteRetentionPolicy Days must be between 1 and 365")
	}
	return nil
}

func validateBlobServiceStaticWebsite(properties *blobServiceStaticWebsite) error {
	if strings.TrimSpace(properties.Enabled) == "" {
		return fmt.Errorf("StaticWebsite settings require Enabled")
	}
	if _, ok := queueServiceBool(properties.Enabled); !ok {
		return fmt.Errorf("StaticWebsite Enabled must be true or false")
	}
	if strings.TrimSpace(properties.IndexDocument) != "" && strings.TrimSpace(properties.DefaultIndexDocumentPath) != "" {
		return fmt.Errorf("StaticWebsite IndexDocument and DefaultIndexDocumentPath are mutually exclusive")
	}
	return nil
}

func validateBlobServiceCORSProperties(cors *queueServicePropertiesCors) error {
	if len(cors.Rules) > 5 {
		return fmt.Errorf("a blob service can contain at most five CORS rules")
	}
	if queueServiceCORSSettingsSize(cors.Rules) > maxBlobCORSBytes {
		return fmt.Errorf("CORS settings exceed documented size limit")
	}
	for _, rule := range cors.Rules {
		if strings.TrimSpace(rule.AllowedOrigins) == "" ||
			strings.TrimSpace(rule.AllowedMethods) == "" ||
			strings.TrimSpace(rule.MaxAgeInSeconds) == "" ||
			strings.TrimSpace(rule.ExposedHeaders) == "" ||
			strings.TrimSpace(rule.AllowedHeaders) == "" {
			return fmt.Errorf("all CORS rule elements are required")
		}
		if !queueServiceCORSMaxAgeValid(rule.MaxAgeInSeconds) {
			return fmt.Errorf("CORS MaxAgeInSeconds must be a non-negative integer")
		}
		if !queueServiceCORSListWithinLimits(rule.AllowedOrigins, 64, 256) {
			return fmt.Errorf("CORS AllowedOrigins exceeds documented limits")
		}
		if !queueServiceCORSHeaderListWithinLimits(rule.AllowedHeaders) {
			return fmt.Errorf("CORS AllowedHeaders exceeds documented limits")
		}
		if !queueServiceCORSHeaderListWithinLimits(rule.ExposedHeaders) {
			return fmt.Errorf("CORS ExposedHeaders exceeds documented limits")
		}
		for _, method := range strings.Split(rule.AllowedMethods, ",") {
			if !blobServiceCORSMethodAllowed(strings.TrimSpace(method)) {
				return fmt.Errorf("CORS AllowedMethods contains an unsupported method")
			}
		}
	}
	return nil
}

func blobServiceCORSMethodAllowed(method string) bool {
	switch strings.ToUpper(method) {
	case "DELETE", "GET", "HEAD", "MERGE", "PATCH", "POST", "OPTIONS", "PUT":
		return true
	default:
		return false
	}
}

func mergeBlobServiceProperties(current, update []byte) ([]byte, error) {
	currentElements, err := blobServicePropertyRootElements(current)
	if err != nil {
		return nil, err
	}
	updateElements, err := blobServicePropertyRootElements(update)
	if err != nil {
		return nil, err
	}
	for name, element := range updateElements {
		currentElements[name] = element
	}

	return buildBlobServicePropertiesFromElements(currentElements), nil
}

func buildBlobServicePropertiesFromElements(elements map[string]string) []byte {
	var out strings.Builder
	out.WriteString("<StorageServiceProperties>")
	for _, name := range blobServicePropertyRootOrder {
		if element := elements[name]; element != "" {
			out.WriteString(element)
		}
	}
	out.WriteString("</StorageServiceProperties>")
	return []byte(out.String())
}

func blobServicePropertyRootElements(body []byte) (map[string]string, error) {
	var properties blobServicePropertiesRawDocument
	if err := xml.Unmarshal(body, &properties); err != nil {
		return nil, err
	}
	if properties.XMLName.Local != "StorageServiceProperties" {
		return nil, fmt.Errorf("expected StorageServiceProperties root element")
	}

	elements := make(map[string]string)
	add := func(name string, element *queueServicePropertiesRawElement) {
		if element != nil {
			elements[name] = "<" + name + ">" + element.InnerXML + "</" + name + ">"
		}
	}
	add("Logging", properties.Logging)
	add("Metrics", properties.Metrics)
	add("HourMetrics", properties.HourMetrics)
	add("MinuteMetrics", properties.MinuteMetrics)
	add("Cors", properties.Cors)
	add("DefaultServiceVersion", properties.DefaultServiceVersion)
	add("DeleteRetentionPolicy", properties.DeleteRetentionPolicy)
	add("StaticWebsite", properties.StaticWebsite)
	return elements, nil
}

func defaultBlobServiceProperties() []byte {
	return []byte(`<StorageServiceProperties><Logging><Version>1.0</Version><Delete>false</Delete><Read>false</Read><Write>false</Write><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></Logging><HourMetrics><Version>1.0</Version><Enabled>false</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></HourMetrics><MinuteMetrics><Version>1.0</Version><Enabled>false</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></MinuteMetrics><Cors></Cors><DefaultServiceVersion>` + dataPlaneAPIVersion + `</DefaultServiceVersion><DeleteRetentionPolicy><Enabled>false</Enabled></DeleteRetentionPolicy><StaticWebsite><Enabled>false</Enabled></StaticWebsite></StorageServiceProperties>`)
}

func blobBaseHeadersForRequest(header http.Header) map[string]string {
	headers := map[string]string{
		"x-ms-request-id": nextBlobRequestID(),
		"x-ms-version":    dataPlaneAPIVersion,
	}
	if requested := strings.TrimSpace(header.Get("x-ms-version")); requested != "" {
		headers["x-ms-version"] = requested
	}
	if clientRequestID := queueClientRequestID(header.Get("x-ms-client-request-id")); clientRequestID != "" {
		headers["x-ms-client-request-id"] = clientRequestID
	}
	return headers
}

func applyBlobResponseHeaders(header http.Header, resp *service.Response) {
	if resp.Headers == nil {
		resp.Headers = make(map[string]string)
	}
	if resp.Headers["x-ms-request-id"] == "" {
		resp.Headers["x-ms-request-id"] = nextBlobRequestID()
	}
	if requested := strings.TrimSpace(header.Get("x-ms-version")); requested != "" {
		resp.Headers["x-ms-version"] = requested
	} else if resp.Headers["x-ms-version"] == "" {
		resp.Headers["x-ms-version"] = dataPlaneAPIVersion
	}
	if clientRequestID := queueClientRequestID(header.Get("x-ms-client-request-id")); clientRequestID != "" {
		resp.Headers["x-ms-client-request-id"] = clientRequestID
	}
}

func nextBlobRequestID() string {
	return fmt.Sprintf("cloudmock-blob-%016x", blobRequestIDCounter.Add(1))
}

func (s *StorageService) getContainerMetadata(account, containerName string, header http.Header) (*service.Response, error) {
	s.mu.RLock()
	container, ok := s.containers[strings.ToLower(account)][strings.ToLower(containerName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if !blobConditionsMatch(header, container.ETag) {
		return blobConditionNotMetResponse()
	}

	headers := storageHeaders(container.ETag, container.LastModified)
	for key, value := range container.Metadata {
		headers["x-ms-meta-"+key] = value
	}
	return emptyResponse(http.StatusOK, headers)
}

func (s *StorageService) listContainers(account string, query url.Values) (*service.Response, error) {
	prefix := query.Get("prefix")
	marker := query.Get("marker")
	maxResults, invalid := parseListMaxResults(query.Get("maxresults"))
	if invalid {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The maxresults query parameter must be greater than 0.")
	}
	includeMetadata := blobListIncludesMetadata(query.Get("include"))

	s.mu.RLock()
	containers := s.containers[strings.ToLower(account)]
	items := make([]containerListItem, 0, len(containers))
	for _, container := range containers {
		if prefix != "" && !strings.HasPrefix(container.Name, prefix) {
			continue
		}
		item := containerListItem{
			Name: container.Name,
			Properties: containerListProperties{
				LastModified: container.LastModified.UTC().Format(http.TimeFormat),
				ETag:         container.ETag,
			},
		}
		if includeMetadata && len(container.Metadata) > 0 {
			metadata := blobListMetadata(container.Metadata)
			item.Metadata = &metadata
		}
		items = append(items, item)
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	start := 0
	if marker != "" {
		for start < len(items) && items[start].Name < marker {
			start++
		}
	}
	end := len(items)
	nextMarker := ""
	if maxResults > 0 && start+maxResults < len(items) {
		end = start + maxResults
		nextMarker = items[end].Name
	}

	resp, err := xmlResponse(http.StatusOK, containerListResponse{
		ServiceEndpoint: "https://" + account + ".blob.core.windows.net/",
		Prefix:          prefix,
		Marker:          marker,
		MaxResults:      maxResults,
		Containers:      items[start:end],
		NextMarker:      nextMarker,
	})
	if err != nil {
		return nil, err
	}
	resp.Headers = map[string]string{"x-ms-version": dataPlaneAPIVersion}
	return resp, nil
}

func (s *StorageService) findBlobsByTags(account string, query url.Values) (*service.Response, error) {
	whereRaw := strings.TrimSpace(query.Get("where"))
	if whereRaw == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The where query parameter is required.")
	}
	where, err := parseBlobTagWhere(whereRaw)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The where query parameter is invalid.")
	}
	maxResults, invalid := parseListMaxResults(query.Get("maxresults"))
	if invalid {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The maxresults query parameter must be greater than 0.")
	}
	if maxResults == 0 || maxResults > 5000 {
		maxResults = 5000
	}
	marker := query.Get("marker")

	s.mu.RLock()
	containers := s.containers[strings.ToLower(account)]
	items := make([]findBlobsByTagsItem, 0)
	for _, container := range containers {
		if where.ContainerName != "" && container.Name != where.ContainerName {
			continue
		}
		for _, blob := range container.Blobs {
			if !where.matches(blob.Tags) {
				continue
			}
			items = append(items, findBlobsByTagsItem{
				Name:          blob.Name,
				ContainerName: container.Name,
				Tags:          blobTagsPayload{TagSet: blobTagSetFromMap(blob.Tags)},
			})
		}
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		if items[i].ContainerName == items[j].ContainerName {
			return items[i].Name < items[j].Name
		}
		return items[i].ContainerName < items[j].ContainerName
	})

	start := 0
	if marker != "" {
		for start < len(items) && findBlobTagMarker(items[start]) < marker {
			start++
		}
	}
	end := len(items)
	nextMarker := ""
	if start+maxResults < len(items) {
		end = start + maxResults
		nextMarker = findBlobTagMarker(items[end])
	}

	resp, err := xmlResponse(http.StatusOK, findBlobsByTagsResponse{
		ServiceEndpoint: "https://" + account + ".blob.core.windows.net/",
		Where:           whereRaw,
		Blobs:           items[start:end],
		NextMarker:      nextMarker,
	})
	if err != nil {
		return nil, err
	}
	resp.Headers = map[string]string{
		"Content-Length": strconv.Itoa(len(resp.RawBody)),
		"x-ms-version":   dataPlaneAPIVersion,
	}
	return resp, nil
}

func findBlobTagMarker(item findBlobsByTagsItem) string {
	return item.ContainerName + "/" + item.Name
}

func (s *StorageService) putBlob(account, containerName, blobName string, ctx *service.RequestContext) (*service.Response, error) {
	blobType := strings.TrimSpace(ctx.RawRequest.Header.Get("x-ms-blob-type"))
	if blobType == "" {
		blobType = "BlockBlob"
	}
	switch blobType {
	case "BlockBlob":
	case "AppendBlob":
		if len(ctx.Body) != 0 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The Content-Length header must be 0 for AppendBlob Put Blob.")
		}
	case "PageBlob":
		if len(ctx.Body) != 0 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The Content-Length header must be 0 for PageBlob Put Blob.")
		}
	default:
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-type header value is invalid.")
	}
	content := append([]byte(nil), ctx.Body...)
	sequenceNumber := ""
	if blobType == "PageBlob" {
		lengthRaw := strings.TrimSpace(ctx.RawRequest.Header.Get("x-ms-blob-content-length"))
		if lengthRaw == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-blob-content-length header is required for PageBlob.")
		}
		length, err := strconv.Atoi(lengthRaw)
		if err != nil || length < 0 || length%512 != 0 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-content-length header must be a nonnegative multiple of 512.")
		}
		content = make([]byte, length)
		sequenceNumber = strings.TrimSpace(ctx.RawRequest.Header.Get("x-ms-blob-sequence-number"))
		if sequenceNumber == "" {
			sequenceNumber = "0"
		}
		if _, err := strconv.ParseInt(sequenceNumber, 10, 64); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-sequence-number header value is invalid.")
		}
	}

	now := time.Now().UTC()
	etag := s.nextToken("\"blob") + "\""

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if container.Blobs == nil {
		container.Blobs = make(map[string]blobObject)
	}
	leaseID := ""
	leaseState := ""
	if existing, ok := container.Blobs[blobName]; ok {
		if !blobLeaseAllowsWrite(ctx.RawRequest.Header, existing) {
			return blobLeaseWriteFailure(ctx.RawRequest.Header, existing)
		}
		leaseID = existing.LeaseID
		leaseState = existing.LeaseState
	}
	if container.StagedBlocks != nil {
		delete(container.StagedBlocks, blobName)
	}
	container.Blobs[blobName] = blobObject{
		Name:               blobName,
		BlobType:           blobType,
		Content:            content,
		ContentType:        blobContentTypeFromHeaders(ctx.RawRequest.Header),
		CacheControl:       blobHeaderValue(ctx.RawRequest.Header, "x-ms-blob-cache-control", "Cache-Control"),
		ContentEncoding:    blobHeaderValue(ctx.RawRequest.Header, "x-ms-blob-content-encoding", "Content-Encoding"),
		ContentLanguage:    blobHeaderValue(ctx.RawRequest.Header, "x-ms-blob-content-language", "Content-Language"),
		ContentDisposition: blobHeaderValue(ctx.RawRequest.Header, "x-ms-blob-content-disposition", "Content-Disposition"),
		ContentMD5:         blobHeaderValue(ctx.RawRequest.Header, "x-ms-blob-content-md5", "Content-MD5"),
		Metadata:           metadataFromHeaders(ctx.RawRequest.Header),
		ETag:               etag,
		LastModified:       now,
		AppendBlockCount:   0,
		SequenceNumber:     sequenceNumber,
		LeaseID:            leaseID,
		LeaseState:         leaseState,
	}
	container.ETag = etag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container

	return emptyResponse(http.StatusCreated, storageHeaders(etag, now))
}

func (s *StorageService) putBlock(account, containerName, blobName, blockID string, body []byte, header http.Header) (*service.Response, error) {
	if blockID == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The blockid query parameter is required.")
	}
	now := time.Now().UTC()
	etag := s.nextToken("\"block") + "\""

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if existing, ok := container.Blobs[blobName]; ok && !blobLeaseAllowsWrite(header, existing) {
		return blobLeaseWriteFailure(header, existing)
	}
	if container.StagedBlocks == nil {
		container.StagedBlocks = make(map[string]map[string]blobBlock)
	}
	if container.StagedBlocks[blobName] == nil {
		container.StagedBlocks[blobName] = make(map[string]blobBlock)
	}
	container.StagedBlocks[blobName][blockID] = blobBlock{
		ID:      blockID,
		Content: append([]byte(nil), body...),
	}
	s.containers[accountKey][containerKey] = container

	return emptyResponse(http.StatusCreated, storageHeaders(etag, now))
}

func (s *StorageService) appendBlock(account, containerName, blobName string, body []byte, header http.Header) (*service.Response, error) {
	now := time.Now().UTC()
	etag := s.nextToken("\"blob") + "\""

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	blob, ok := container.Blobs[blobName]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if blobTypeOrDefault(blob.BlobType) != "AppendBlob" {
		return azurearm.ErrorResponse(http.StatusConflict, "BlobOperationNotSupported", "Append Block is supported only for append blobs.")
	}
	if !blobLeaseAllowsWrite(header, blob) {
		return blobLeaseWriteFailure(header, blob)
	}
	if !blobConditionsMatch(header, blob.ETag) {
		return blobConditionNotMetResponse()
	}
	appendOffset := len(blob.Content)
	if rawAppendPos := strings.TrimSpace(header.Get("x-ms-blob-condition-appendpos")); rawAppendPos != "" {
		expected, err := strconv.Atoi(rawAppendPos)
		if err != nil || expected < 0 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-condition-appendpos header value is invalid.")
		}
		if expected != appendOffset {
			return azurearm.ErrorResponse(http.StatusPreconditionFailed, "AppendPositionConditionNotMet", "The append position condition specified was not met.")
		}
	}
	if rawMaxSize := strings.TrimSpace(header.Get("x-ms-blob-condition-maxsize")); rawMaxSize != "" {
		maxSize, err := strconv.Atoi(rawMaxSize)
		if err != nil || maxSize < 0 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-condition-maxsize header value is invalid.")
		}
		if appendOffset+len(body) > maxSize {
			return azurearm.ErrorResponse(http.StatusPreconditionFailed, "MaxBlobSizeConditionNotMet", "The max blob size condition specified was not met.")
		}
	}
	if len(body) > 100*1024*1024 {
		return azurearm.ErrorResponse(http.StatusRequestEntityTooLarge, "RequestBodyTooLarge", "The append block request body exceeds the maximum supported size.")
	}
	if blob.AppendBlockCount >= 50000 {
		return azurearm.ErrorResponse(http.StatusConflict, "BlockCountExceedsLimit", "The committed block count cannot exceed 50000 blocks.")
	}

	blob.Content = append(blob.Content, body...)
	blob.AppendBlockCount++
	blob.ETag = etag
	blob.LastModified = now
	container.Blobs[blobName] = blob
	container.ETag = etag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container

	headers := storageHeaders(etag, now)
	headers["x-ms-blob-append-offset"] = strconv.Itoa(appendOffset)
	headers["x-ms-blob-committed-block-count"] = strconv.Itoa(blob.AppendBlockCount)
	return emptyResponse(http.StatusCreated, headers)
}

func (s *StorageService) putPage(account, containerName, blobName string, body []byte, header http.Header) (*service.Response, error) {
	pageRange, ok, err := pageBlobRangeFromHeader(header)
	if err != nil || !ok {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-range or Range header must specify a 512-byte aligned page range.")
	}
	write := strings.ToLower(strings.TrimSpace(header.Get("x-ms-page-write")))
	if write != "update" && write != "clear" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-page-write header must be update or clear.")
	}
	if write == "update" {
		if len(body) != pageRange.end-pageRange.start+1 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The request body length must match the page range.")
		}
		if len(body) > 4*1024*1024 {
			return azurearm.ErrorResponse(http.StatusRequestEntityTooLarge, "RequestBodyTooLarge", "The page update range exceeds the maximum supported size.")
		}
	}
	if write == "clear" && len(body) != 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The request body must be empty when x-ms-page-write is clear.")
	}

	now := time.Now().UTC()
	etag := s.nextToken("\"blob") + "\""

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	blob, ok := container.Blobs[blobName]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if blobTypeOrDefault(blob.BlobType) != "PageBlob" {
		return azurearm.ErrorResponse(http.StatusConflict, "BlobOperationNotSupported", "Put Page is supported only for page blobs.")
	}
	if pageRange.end >= len(blob.Content) {
		return azurearm.ErrorResponse(http.StatusRequestedRangeNotSatisfiable, "InvalidRange", "The page range specified is invalid for the current size of the resource.")
	}
	if !blobLeaseAllowsWrite(header, blob) {
		return blobLeaseWriteFailure(header, blob)
	}
	if !blobConditionsMatch(header, blob.ETag) {
		return blobConditionNotMetResponse()
	}
	if !pageBlobSequenceConditionsMatch(header, blob) {
		return azurearm.ErrorResponse(http.StatusPreconditionFailed, "SequenceNumberConditionNotMet", "The sequence number condition specified was not met.")
	}

	switch write {
	case "update":
		copy(blob.Content[pageRange.start:pageRange.end+1], body)
		blob.PageRanges = addBlobPageRange(blob.PageRanges, fileRange{Start: pageRange.start, End: pageRange.end})
	case "clear":
		for i := pageRange.start; i <= pageRange.end; i++ {
			blob.Content[i] = 0
		}
		blob.PageRanges = clearBlobPageRange(blob.PageRanges, fileRange{Start: pageRange.start, End: pageRange.end})
	}
	blob.ETag = etag
	blob.LastModified = now
	container.Blobs[blobName] = blob
	container.ETag = etag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container

	headers := storageHeaders(etag, now)
	headers["x-ms-blob-sequence-number"] = pageBlobSequenceNumber(blob)
	return emptyResponse(http.StatusCreated, headers)
}

func (s *StorageService) getPageRanges(account, containerName, blobName string, query url.Values, header http.Header) (*service.Response, error) {
	blob, snapshot, ok, resp, err := s.blobForRead(account, containerName, blobName, query.Get("snapshot"))
	if resp != nil || err != nil {
		return resp, err
	}
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if blobTypeOrDefault(blob.BlobType) != "PageBlob" {
		return azurearm.ErrorResponse(http.StatusConflict, "BlobOperationNotSupported", "Get Page Ranges is supported only for page blobs.")
	}
	if conditionResp, ok, err := blobReadConditionsResponse(header, blob); !ok {
		return conditionResp, err
	}
	ranges := blob.PageRanges
	if pageRange, hasRange, err := pageBlobOptionalRangeFromHeader(header, len(blob.Content)); err != nil {
		return azurearm.ErrorResponse(http.StatusRequestedRangeNotSatisfiable, "InvalidRange", "The range specified is invalid for the current size of the resource.")
	} else if hasRange {
		ranges = clipBlobPageRanges(ranges, fileRange{Start: pageRange.start, End: pageRange.end})
	}
	resp, err = xmlResponse(http.StatusOK, blobPageListResponse{PageRanges: blobPageRangeListItems(ranges)})
	if err != nil {
		return nil, err
	}
	headers := storageHeaders(blob.ETag, blob.LastModified)
	headers["Content-Length"] = strconv.Itoa(len(resp.RawBody))
	headers["x-ms-blob-content-length"] = strconv.Itoa(len(blob.Content))
	headers["x-ms-blob-sequence-number"] = pageBlobSequenceNumber(blob)
	if snapshot != "" {
		headers["x-ms-snapshot"] = snapshot
	}
	resp.Headers = headers
	return resp, nil
}

func (s *StorageService) putBlockFromURL(account, containerName, blobName, blockID string, body []byte, header http.Header) (*service.Response, error) {
	if blockID == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The blockid query parameter is required.")
	}
	if len(body) != 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The Content-Length header must be 0 for Put Block From URL.")
	}
	sourceURL := strings.TrimSpace(header.Get("x-ms-copy-source"))
	if sourceURL == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-copy-source header is required.")
	}
	source, ok := parseFileCopySource(account, sourceURL)
	if !ok {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-copy-source header must reference a file or blob.")
	}
	now := time.Now().UTC()
	etag := s.nextToken("\"block") + "\""

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if existing, ok := container.Blobs[blobName]; ok && !blobLeaseAllowsWrite(header, existing) {
		return blobLeaseWriteFailure(header, existing)
	}
	sourceBlob, sourceOK := s.blobCopySource(source)
	if !sourceOK {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified copy source does not exist.")
	}
	content := sourceBlob.Content
	if sourceRange := strings.TrimSpace(header.Get("x-ms-source-range")); sourceRange != "" {
		blobRange, ok, err := requestedByteRange(sourceRange, len(content))
		if err != nil || !ok {
			return azurearm.ErrorResponse(http.StatusRequestedRangeNotSatisfiable, "InvalidRange", "The source range specified is invalid for the current size of the resource.")
		}
		content = content[blobRange.start : blobRange.end+1]
	}
	if container.StagedBlocks == nil {
		container.StagedBlocks = make(map[string]map[string]blobBlock)
	}
	if container.StagedBlocks[blobName] == nil {
		container.StagedBlocks[blobName] = make(map[string]blobBlock)
	}
	container.StagedBlocks[blobName][blockID] = blobBlock{
		ID:      blockID,
		Content: append([]byte(nil), content...),
	}
	s.containers[accountKey][containerKey] = container

	return emptyResponse(http.StatusCreated, storageHeaders(etag, now))
}

func (s *StorageService) putBlockList(account, containerName, blobName string, ctx *service.RequestContext) (*service.Response, error) {
	blockRefs, err := parseBlockList(ctx.Body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", "The specified XML is not syntactically valid.")
	}
	now := time.Now().UTC()
	etag := s.nextToken("\"blob") + "\""

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}

	leaseID := ""
	leaseState := ""
	existingBlob := blobObject{}
	if existing, ok := container.Blobs[blobName]; ok {
		if !blobLeaseAllowsWrite(ctx.RawRequest.Header, existing) {
			return blobLeaseWriteFailure(ctx.RawRequest.Header, existing)
		}
		leaseID = existing.LeaseID
		leaseState = existing.LeaseState
		existingBlob = existing
	}
	stagedBlocks := container.StagedBlocks[blobName]
	content := make([]byte, 0)
	committedBlocks := make([]blobBlock, 0, len(blockRefs))
	for _, blockRef := range blockRefs {
		block, ok := resolveBlockListReference(blockRef, stagedBlocks, existingBlob.CommittedBlocks)
		if !ok {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidBlockList", "The specified block list is invalid.")
		}
		content = append(content, block.Content...)
		committedBlocks = append(committedBlocks, blobBlock{
			ID:      block.ID,
			Content: append([]byte(nil), block.Content...),
		})
	}
	if container.Blobs == nil {
		container.Blobs = make(map[string]blobObject)
	}
	container.Blobs[blobName] = blobObject{
		Name:               blobName,
		BlobType:           "BlockBlob",
		Content:            content,
		ContentType:        blobContentTypeFromHeaders(ctx.RawRequest.Header),
		CacheControl:       blobHeaderValue(ctx.RawRequest.Header, "x-ms-blob-cache-control", "Cache-Control"),
		ContentEncoding:    blobHeaderValue(ctx.RawRequest.Header, "x-ms-blob-content-encoding", "Content-Encoding"),
		ContentLanguage:    blobHeaderValue(ctx.RawRequest.Header, "x-ms-blob-content-language", "Content-Language"),
		ContentDisposition: blobHeaderValue(ctx.RawRequest.Header, "x-ms-blob-content-disposition", "Content-Disposition"),
		ContentMD5:         blobHeaderValue(ctx.RawRequest.Header, "x-ms-blob-content-md5", "Content-MD5"),
		Metadata:           metadataFromHeaders(ctx.RawRequest.Header),
		ETag:               etag,
		LastModified:       now,
		CommittedBlocks:    committedBlocks,
		LeaseID:            leaseID,
		LeaseState:         leaseState,
	}
	if container.StagedBlocks != nil {
		delete(container.StagedBlocks, blobName)
	}
	container.ETag = etag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container

	return emptyResponse(http.StatusCreated, storageHeaders(etag, now))
}

func (s *StorageService) getBlockList(account, containerName, blobName string, query url.Values, header http.Header) (*service.Response, error) {
	blockListType := strings.ToLower(strings.TrimSpace(query.Get("blocklisttype")))
	if blockListType == "" {
		blockListType = "committed"
	}
	if blockListType != "committed" && blockListType != "uncommitted" && blockListType != "all" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The blocklisttype query parameter is invalid.")
	}

	s.mu.RLock()
	container, containerOK := s.containers[strings.ToLower(account)][strings.ToLower(containerName)]
	blob, blobOK := container.Blobs[blobName]
	staged := container.StagedBlocks[blobName]
	s.mu.RUnlock()
	if !containerOK {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	if !blobOK && len(staged) == 0 {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if blobOK && !blobConditionsMatch(header, blob.ETag) {
		return blobConditionNotMetResponse()
	}

	response := blockListResponse{}
	if blockListType == "committed" || blockListType == "all" {
		committed := blocksToBlockList(blob.CommittedBlocks)
		response.CommittedBlocks = &committed
	}
	if blockListType == "uncommitted" || blockListType == "all" {
		uncommitted := stagedBlocksToBlockList(staged)
		response.UncommittedBlocks = &uncommitted
	}

	resp, err := xmlResponse(http.StatusOK, response)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		"x-ms-version": dataPlaneAPIVersion,
	}
	if blobOK {
		for key, value := range storageHeaders(blob.ETag, blob.LastModified) {
			headers[key] = value
		}
		headers["x-ms-blob-content-length"] = strconv.Itoa(len(blob.Content))
	}
	resp.Headers = headers
	return resp, nil
}

type blockListReference struct {
	Type string
	ID   string
}

func parseBlockList(body []byte) ([]blockListReference, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	blockRefs := make([]blockListReference, 0)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return blockRefs, nil
			}
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "Committed", "Uncommitted", "Latest":
			var blockID string
			if err := decoder.DecodeElement(&blockID, &start); err != nil {
				return nil, err
			}
			blockRefs = append(blockRefs, blockListReference{
				Type: start.Name.Local,
				ID:   strings.TrimSpace(blockID),
			})
		}
	}
}

func resolveBlockListReference(ref blockListReference, stagedBlocks map[string]blobBlock, committedBlocks []blobBlock) (blobBlock, bool) {
	switch ref.Type {
	case "Committed":
		return findCommittedBlock(ref.ID, committedBlocks)
	case "Uncommitted":
		block, ok := stagedBlocks[ref.ID]
		return block, ok
	case "Latest":
		if block, ok := stagedBlocks[ref.ID]; ok {
			return block, true
		}
		return findCommittedBlock(ref.ID, committedBlocks)
	default:
		return blobBlock{}, false
	}
}

func findCommittedBlock(blockID string, committedBlocks []blobBlock) (blobBlock, bool) {
	for _, block := range committedBlocks {
		if block.ID == blockID {
			return block, true
		}
	}
	return blobBlock{}, false
}

func (s *StorageService) getBlob(account, containerName, blobName string, query url.Values, header http.Header) (*service.Response, error) {
	blob, snapshot, ok, resp, err := s.blobForRead(account, containerName, blobName, query.Get("snapshot"))
	if resp != nil || err != nil {
		return resp, err
	}
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if !blobLeaseAllowsRead(header, blob) {
		return blobLeaseReadFailure()
	}
	if conditionResp, ok, err := blobReadConditionsResponse(header, blob); !ok {
		return conditionResp, err
	}

	headers := blobPropertyHeaders(blob)
	if snapshot != "" {
		headers["x-ms-snapshot"] = snapshot
	}
	body := append([]byte(nil), blob.Content...)
	status := http.StatusOK
	if blobRange, ok, err := requestedBlobRange(header, len(blob.Content)); err != nil {
		return azurearm.ErrorResponse(http.StatusRequestedRangeNotSatisfiable, "InvalidRange", "The range specified is invalid for the current size of the resource.")
	} else if ok {
		if resp, err := validateBlobRangeHashHeaders(header, blobRange, true); resp != nil || err != nil {
			return resp, err
		}
		status = http.StatusPartialContent
		body = append([]byte(nil), blob.Content[blobRange.start:blobRange.end+1]...)
		headers["Content-Range"] = "bytes " + strconv.Itoa(blobRange.start) + "-" + strconv.Itoa(blobRange.end) + "/" + strconv.Itoa(len(blob.Content))
		headers["Content-Length"] = strconv.Itoa(len(body))
		applyBlobRangeHashHeaders(headers, header, body)
		if blob.ContentMD5 != "" && !blobRangeContentMD5Requested(header) {
			delete(headers, "Content-MD5")
			headers["x-ms-blob-content-md5"] = blob.ContentMD5
		}
	} else if resp, err := validateBlobRangeHashHeaders(header, byteRange{}, false); resp != nil || err != nil {
		return resp, err
	}
	return &service.Response{
		StatusCode:     status,
		Headers:        headers,
		RawBody:        body,
		RawContentType: blob.ContentType,
	}, nil
}

func validateBlobRangeHashHeaders(header http.Header, blobRange byteRange, hasRange bool) (*service.Response, error) {
	if blobRangeContentMD5HeaderPresent(header) && blobRangeContentCRC64HeaderPresent(header) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "Only one range hash header may be specified.")
	}
	if !blobRangeContentMD5Requested(header) && !blobRangeContentCRC64Requested(header) {
		return nil, nil
	}
	if !hasRange {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "Range hash headers require a Range or x-ms-range header.")
	}
	if blobRange.end-blobRange.start+1 > 4*1024*1024 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "Range hash headers are supported only for ranges up to 4 MiB.")
	}
	return nil, nil
}

func applyBlobRangeHashHeaders(headers map[string]string, header http.Header, body []byte) {
	if blobRangeContentMD5Requested(header) {
		sum := md5.Sum(body)
		headers["Content-MD5"] = base64.StdEncoding.EncodeToString(sum[:])
	}
	if blobRangeContentCRC64Requested(header) {
		sum := crc64.Checksum(body, crc64.MakeTable(crc64.ECMA))
		var encoded [8]byte
		binary.BigEndian.PutUint64(encoded[:], sum)
		headers["x-ms-content-crc64"] = base64.StdEncoding.EncodeToString(encoded[:])
	}
}

func blobRangeContentMD5HeaderPresent(header http.Header) bool {
	_, ok := header[http.CanonicalHeaderKey("x-ms-range-get-content-md5")]
	return ok
}

func blobRangeContentCRC64HeaderPresent(header http.Header) bool {
	_, ok := header[http.CanonicalHeaderKey("x-ms-range-get-content-crc64")]
	return ok
}

func blobRangeContentMD5Requested(header http.Header) bool {
	return strings.EqualFold(strings.TrimSpace(header.Get("x-ms-range-get-content-md5")), "true")
}

func blobRangeContentCRC64Requested(header http.Header) bool {
	return strings.EqualFold(strings.TrimSpace(header.Get("x-ms-range-get-content-crc64")), "true")
}

func (s *StorageService) getBlobProperties(account, containerName, blobName string, query url.Values, header http.Header) (*service.Response, error) {
	blob, snapshot, ok, resp, err := s.blobForRead(account, containerName, blobName, query.Get("snapshot"))
	if resp != nil || err != nil {
		return resp, err
	}
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if !blobLeaseAllowsRead(header, blob) {
		return blobLeaseReadFailure()
	}
	if conditionResp, ok, err := blobReadConditionsResponse(header, blob); !ok {
		return conditionResp, err
	}
	headers := blobPropertyHeaders(blob)
	if snapshot != "" {
		headers["x-ms-snapshot"] = snapshot
	}
	return emptyResponse(http.StatusOK, headers)
}

func (s *StorageService) blobForRead(account, containerName, blobName, snapshot string) (blobObject, string, bool, *service.Response, error) {
	s.mu.RLock()
	container, ok := s.containers[strings.ToLower(account)][strings.ToLower(containerName)]
	if !ok {
		s.mu.RUnlock()
		resp, err := azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
		return blobObject{}, "", false, resp, err
	}
	if snapshot != "" {
		if snapshots := container.Snapshots[blobName]; snapshots != nil {
			if blob, ok := snapshots[snapshot]; ok {
				s.mu.RUnlock()
				return blob, snapshot, true, nil, nil
			}
		}
		s.mu.RUnlock()
		return blobObject{}, snapshot, false, nil, nil
	}
	blob, ok := container.Blobs[blobName]
	s.mu.RUnlock()
	return blob, "", ok, nil, nil
}

func (s *StorageService) snapshotBlob(account, containerName, blobName string, header http.Header) (*service.Response, error) {
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	blob, ok := container.Blobs[blobName]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if !blobConditionsMatch(header, blob.ETag) {
		return blobConditionNotMetResponse()
	}
	if leaseID := strings.TrimSpace(header.Get("x-ms-lease-id")); leaseID != "" && (blobLeaseState(blob) != "leased" || leaseID != blob.LeaseID) {
		return azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the blob.")
	}

	snapshotID := snapshotTime(now)
	if container.Snapshots == nil {
		container.Snapshots = make(map[string]map[string]blobObject)
	}
	if container.Snapshots[blobName] == nil {
		container.Snapshots[blobName] = make(map[string]blobObject)
	}
	for container.Snapshots[blobName][snapshotID].ETag != "" {
		now = now.Add(time.Nanosecond)
		snapshotID = snapshotTime(now)
	}

	snapshot := cloneBlobObject(blob)
	if metadata := metadataFromHeaders(header); len(metadata) > 0 {
		snapshot.Metadata = metadata
		snapshot.ETag = s.nextToken("\"snapshot") + "\""
		snapshot.LastModified = now
	}
	snapshot.LeaseID = ""
	snapshot.LeaseState = "available"
	container.Snapshots[blobName][snapshotID] = snapshot
	s.containers[accountKey][containerKey] = container

	headers := storageHeaders(snapshot.ETag, snapshot.LastModified)
	headers["x-ms-snapshot"] = snapshotID
	return emptyResponse(http.StatusCreated, headers)
}

func (s *StorageService) leaseBlob(account, containerName, blobName string, header http.Header) (*service.Response, error) {
	action := strings.ToLower(strings.TrimSpace(header.Get("x-ms-lease-action")))
	if action == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-lease-action header is required.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	blob, ok := container.Blobs[blobName]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}

	switch action {
	case "acquire":
		if strings.TrimSpace(header.Get("x-ms-lease-duration")) == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-lease-duration header is required.")
		}
		leaseID := strings.TrimSpace(header.Get("x-ms-proposed-lease-id"))
		if leaseID == "" {
			leaseID = s.nextToken("lease-")
		}
		if blob.LeaseID != "" && blob.LeaseID != leaseID {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseAlreadyPresent", "There is already a lease present.")
		}
		blob.LeaseID = leaseID
		blob.LeaseState = "leased"
		container.Blobs[blobName] = blob
		s.containers[accountKey][containerKey] = container
		headers := storageHeaders(blob.ETag, blob.LastModified)
		headers["x-ms-lease-id"] = leaseID
		return emptyResponse(http.StatusCreated, headers)
	case "renew":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		if leaseID == "" || leaseID != blob.LeaseID || blobLeaseState(blob) != "leased" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the blob.")
		}
		headers := storageHeaders(blob.ETag, blob.LastModified)
		headers["x-ms-lease-id"] = leaseID
		return emptyResponse(http.StatusOK, headers)
	case "change":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		proposed := strings.TrimSpace(header.Get("x-ms-proposed-lease-id"))
		if proposed == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-proposed-lease-id header is required.")
		}
		if leaseID == "" || leaseID != blob.LeaseID || blobLeaseState(blob) != "leased" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the blob.")
		}
		blob.LeaseID = proposed
		blob.LeaseState = "leased"
		container.Blobs[blobName] = blob
		s.containers[accountKey][containerKey] = container
		headers := storageHeaders(blob.ETag, blob.LastModified)
		headers["x-ms-lease-id"] = proposed
		return emptyResponse(http.StatusOK, headers)
	case "release":
		leaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
		if leaseID == "" || leaseID != blob.LeaseID {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithLeaseOperation", "The lease ID specified did not match the lease ID for the blob.")
		}
		blob.LeaseID = ""
		blob.LeaseState = "available"
		container.Blobs[blobName] = blob
		s.containers[accountKey][containerKey] = container
		return emptyResponse(http.StatusOK, storageHeaders(blob.ETag, blob.LastModified))
	case "break":
		if blobLeaseState(blob) != "leased" && blobLeaseState(blob) != "breaking" && blobLeaseState(blob) != "broken" {
			return azurearm.ErrorResponse(http.StatusConflict, "LeaseNotPresentWithLeaseOperation", "There is currently no lease on the blob.")
		}
		blob.LeaseID = ""
		blob.LeaseState = "broken"
		container.Blobs[blobName] = blob
		s.containers[accountKey][containerKey] = container
		headers := storageHeaders(blob.ETag, blob.LastModified)
		headers["x-ms-lease-time"] = "0"
		return emptyResponse(http.StatusAccepted, headers)
	default:
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-lease-action header value is invalid.")
	}
}

func blocksToBlockList(blocks []blobBlock) blockListBlocks {
	items := make([]blockListBlock, 0, len(blocks))
	for _, block := range blocks {
		items = append(items, blockListBlock{
			Name: block.ID,
			Size: len(block.Content),
		})
	}
	return blockListBlocks{Blocks: items}
}

func stagedBlocksToBlockList(blocks map[string]blobBlock) blockListBlocks {
	keys := make([]string, 0, len(blocks))
	for key := range blocks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]blockListBlock, 0, len(keys))
	for _, key := range keys {
		block := blocks[key]
		items = append(items, blockListBlock{
			Name: block.ID,
			Size: len(block.Content),
		})
	}
	return blockListBlocks{Blocks: items}
}

func (s *StorageService) setBlobProperties(account, containerName, blobName string, header http.Header) (*service.Response, error) {
	now := time.Now().UTC()
	etag := s.nextToken("\"blob") + "\""
	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)

	s.mu.Lock()
	defer s.mu.Unlock()

	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	blob, ok := container.Blobs[blobName]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if !blobLeaseAllowsWrite(header, blob) {
		return blobLeaseWriteFailure(header, blob)
	}
	if !blobConditionsMatch(header, blob.ETag) {
		return blobConditionNotMetResponse()
	}

	var resp *service.Response
	var err error
	blob, resp, err = applyPageBlobSequenceNumberAction(blob, header)
	if resp != nil || err != nil {
		return resp, err
	}
	blob, resp, err = applyPageBlobContentLength(blob, header)
	if resp != nil || err != nil {
		return resp, err
	}
	if blobHTTPPropertyHeadersPresent(header) {
		blob.ContentType = header.Get("x-ms-blob-content-type")
		blob.CacheControl = header.Get("x-ms-blob-cache-control")
		blob.ContentEncoding = header.Get("x-ms-blob-content-encoding")
		blob.ContentLanguage = header.Get("x-ms-blob-content-language")
		blob.ContentDisposition = header.Get("x-ms-blob-content-disposition")
		blob.ContentMD5 = header.Get("x-ms-blob-content-md5")
	}
	blob.ETag = etag
	blob.LastModified = now
	container.Blobs[blobName] = blob
	container.ETag = etag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container

	headers := storageHeaders(etag, now)
	if blobTypeOrDefault(blob.BlobType) == "PageBlob" {
		headers["x-ms-blob-sequence-number"] = pageBlobSequenceNumber(blob)
	}
	return emptyResponse(http.StatusOK, headers)
}

func blobHTTPPropertyHeadersPresent(header http.Header) bool {
	for _, key := range []string{
		"x-ms-blob-cache-control",
		"x-ms-blob-content-type",
		"x-ms-blob-content-md5",
		"x-ms-blob-content-encoding",
		"x-ms-blob-content-language",
		"x-ms-blob-content-disposition",
	} {
		if _, ok := header[http.CanonicalHeaderKey(key)]; ok {
			return true
		}
	}
	return false
}

func applyPageBlobSequenceNumberAction(blob blobObject, header http.Header) (blobObject, *service.Response, error) {
	errorResponse := func(status int, code, message string) (blobObject, *service.Response, error) {
		resp, err := azurearm.ErrorResponse(status, code, message)
		return blob, resp, err
	}
	action := strings.ToLower(strings.TrimSpace(header.Get("x-ms-sequence-number-action")))
	rawSequence := strings.TrimSpace(header.Get("x-ms-blob-sequence-number"))
	if action == "" && rawSequence == "" {
		return blob, nil, nil
	}
	if blobTypeOrDefault(blob.BlobType) != "PageBlob" {
		return errorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-sequence-number-action and x-ms-blob-sequence-number headers apply only to page blobs.")
	}
	if action == "" {
		return errorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-sequence-number-action header is required when x-ms-blob-sequence-number is set.")
	}

	current, err := strconv.ParseInt(pageBlobSequenceNumber(blob), 10, 64)
	if err != nil || current < 0 {
		current = 0
	}
	switch action {
	case "increment":
		if rawSequence != "" {
			return errorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-sequence-number header must not be specified when x-ms-sequence-number-action is increment.")
		}
		blob.SequenceNumber = strconv.FormatInt(current+1, 10)
		return blob, nil, nil
	case "update", "max":
		if rawSequence == "" {
			return errorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-blob-sequence-number header is required when x-ms-sequence-number-action is max or update.")
		}
	default:
		return errorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-sequence-number-action header value is invalid.")
	}

	requested, err := strconv.ParseInt(rawSequence, 10, 64)
	if err != nil || requested < 0 {
		return errorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-sequence-number header value is invalid.")
	}
	if action == "max" && current > requested {
		requested = current
	}
	blob.SequenceNumber = strconv.FormatInt(requested, 10)
	return blob, nil, nil
}

func applyPageBlobContentLength(blob blobObject, header http.Header) (blobObject, *service.Response, error) {
	if _, ok := header[http.CanonicalHeaderKey("x-ms-blob-content-length")]; !ok {
		return blob, nil, nil
	}
	if blobTypeOrDefault(blob.BlobType) != "PageBlob" {
		resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-content-length header applies only to page blobs.")
		return blob, resp, err
	}
	lengthRaw := strings.TrimSpace(header.Get("x-ms-blob-content-length"))
	length, err := strconv.Atoi(lengthRaw)
	if err != nil || length < 0 || length%512 != 0 {
		resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-content-length header must be a nonnegative multiple of 512.")
		return blob, resp, err
	}
	if length < len(blob.Content) {
		blob.Content = append([]byte(nil), blob.Content[:length]...)
		if length == 0 {
			blob.PageRanges = nil
		} else {
			blob.PageRanges = clipBlobPageRanges(blob.PageRanges, fileRange{Start: 0, End: length - 1})
		}
		return blob, nil, nil
	}
	if length > len(blob.Content) {
		content := make([]byte, length)
		copy(content, blob.Content)
		blob.Content = content
	}
	return blob, nil, nil
}

func (s *StorageService) setBlobTier(account, containerName, blobName string, query url.Values, header http.Header) (*service.Response, error) {
	now := time.Now().UTC()
	tier := strings.TrimSpace(header.Get("x-ms-access-tier"))
	if tier == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-access-tier header is required.")
	}
	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)

	s.mu.Lock()
	defer s.mu.Unlock()

	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	snapshot := strings.TrimSpace(query.Get("snapshot"))
	if snapshot != "" {
		if blobServicePropertiesAPIVersion(header.Get("x-ms-version")) < "2019-12-12" {
			resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "FeatureVersionMismatch", "Set Blob Tier on snapshots requires Blob service version 2019-12-12 or later.")
			if resp != nil {
				if resp.Headers == nil {
					resp.Headers = make(map[string]string)
				}
				resp.Headers["x-ms-error-code"] = "FeatureVersionMismatch"
			}
			return resp, err
		}
		snapshotBlob, ok := container.Snapshots[blobName][snapshot]
		if !ok {
			return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
		}
		if !blobConditionsMatch(header, snapshotBlob.ETag) {
			return blobConditionNotMetResponse()
		}
		var status int
		var resp *service.Response
		var err error
		snapshotBlob, status, resp, err = applyBlobTierTransition(snapshotBlob, tier, header, now)
		if resp != nil || err != nil {
			return resp, err
		}
		container.Snapshots[blobName][snapshot] = snapshotBlob
		s.containers[accountKey][containerKey] = container

		headers := storageHeaders(snapshotBlob.ETag, snapshotBlob.LastModified)
		headers["x-ms-snapshot"] = snapshot
		return emptyResponse(status, headers)
	}

	blob, ok := container.Blobs[blobName]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if !blobLeaseAllowsWrite(header, blob) {
		return blobLeaseWriteFailure(header, blob)
	}
	if !blobConditionsMatch(header, blob.ETag) {
		return blobConditionNotMetResponse()
	}

	var status int
	var resp *service.Response
	var err error
	blob, status, resp, err = applyBlobTierTransition(blob, tier, header, now)
	if resp != nil || err != nil {
		return resp, err
	}
	container.Blobs[blobName] = blob
	s.containers[accountKey][containerKey] = container

	return emptyResponse(status, storageHeaders(blob.ETag, blob.LastModified))
}

func applyBlobTierTransition(blob blobObject, tier string, header http.Header, now time.Time) (blobObject, int, *service.Response, error) {
	tier = canonicalBlobAccessTier(tier)
	if !validBlobAccessTierForType(tier, blobTypeOrDefault(blob.BlobType)) {
		resp, err := blobInvalidHeaderValueResponse("The x-ms-access-tier header value is invalid.")
		return blob, 0, resp, err
	}
	if resp, err := blobAccessTierVersionMismatch(tier, header.Get("x-ms-version")); resp != nil || err != nil {
		return blob, 0, resp, err
	}
	priority, priorityHeaderOK := blobRehydratePriority(header.Get("x-ms-rehydrate-priority"))
	if !priorityHeaderOK {
		resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-rehydrate-priority header value is invalid.")
		return blob, 0, resp, err
	}

	if blob.ArchiveStatus != "" {
		pendingTier := blobArchiveStatusTarget(blob.ArchiveStatus)
		if pendingTier != "" && strings.EqualFold(tier, pendingTier) {
			if priority == "High" &&
				!strings.EqualFold(blob.RehydratePriority, "High") &&
				blobServicePropertiesAPIVersion(header.Get("x-ms-version")) >= "2020-06-12" {
				blob.RehydratePriority = "High"
			}
			return blob, http.StatusAccepted, nil, nil
		}
		resp, err := azurearm.ErrorResponse(http.StatusConflict, "BlobBeingRehydrated", "The blob is currently being rehydrated and cannot be retargeted.")
		return blob, 0, resp, err
	}

	if blobTierIsArchive(blob.AccessTier) && blobTierIsOnline(tier) {
		blob.ArchiveStatus = blobArchiveStatusForTier(tier)
		blob.RehydratePriority = priority
		blob.AccessTierChanged = now
		return blob, http.StatusAccepted, nil, nil
	}

	blob.AccessTier = tier
	blob.AccessTierChanged = now
	blob.ArchiveStatus = ""
	blob.RehydratePriority = ""
	return blob, http.StatusOK, nil, nil
}

func validBlobAccessTierForType(tier, blobType string) bool {
	switch blobType {
	case "BlockBlob":
		return blobTierIsArchive(tier) || blobTierIsOnline(tier)
	case "PageBlob":
		switch strings.ToUpper(strings.TrimSpace(tier)) {
		case "P4", "P6", "P10", "P15", "P20", "P30", "P40", "P50", "P60":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func blobAccessTierVersionMismatch(tier, apiVersion string) (*service.Response, error) {
	effectiveVersion := blobServicePropertiesAPIVersion(apiVersion)
	switch {
	case strings.EqualFold(tier, "Cold") && effectiveVersion < "2021-12-02":
		return blobFeatureVersionMismatchResponse("Cold access tier requires Blob service version 2021-12-02 or later.")
	case strings.EqualFold(tier, "Smart") && effectiveVersion < "2026-02-06":
		return blobFeatureVersionMismatchResponse("Smart access tier requires Blob service version 2026-02-06 or later.")
	default:
		return nil, nil
	}
}

func blobFeatureVersionMismatchResponse(message string) (*service.Response, error) {
	resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "FeatureVersionMismatch", message)
	if resp != nil {
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers["x-ms-error-code"] = "FeatureVersionMismatch"
	}
	return resp, err
}

func canonicalBlobAccessTier(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "hot":
		return "Hot"
	case "cool":
		return "Cool"
	case "cold":
		return "Cold"
	case "smart":
		return "Smart"
	case "archive":
		return "Archive"
	default:
		return strings.TrimSpace(tier)
	}
}

func blobRehydratePriority(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return "Standard", true
	case "standard":
		return "Standard", true
	case "high":
		return "High", true
	default:
		return "", false
	}
}

func blobTierIsArchive(tier string) bool {
	return strings.EqualFold(tier, "Archive")
}

func blobTierIsOnline(tier string) bool {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "hot", "cool", "cold", "smart":
		return true
	default:
		return false
	}
}

func blobArchiveStatusForTier(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "hot":
		return "rehydrate-pending-to-hot"
	case "cool":
		return "rehydrate-pending-to-cool"
	case "cold":
		return "rehydrate-pending-to-cold"
	case "smart":
		return "rehydrate-pending-to-smart"
	default:
		return ""
	}
}

func blobArchiveStatusTarget(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "rehydrate-pending-to-hot":
		return "Hot"
	case "rehydrate-pending-to-cool":
		return "Cool"
	case "rehydrate-pending-to-cold":
		return "Cold"
	case "rehydrate-pending-to-smart":
		return "Smart"
	default:
		return ""
	}
}

func (s *StorageService) setBlobTags(account, containerName, blobName string, query url.Values, body []byte, header http.Header) (*service.Response, error) {
	tags, err := parseBlobTags(body)
	if err != nil {
		return blobInvalidTagResponse(err.Error())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	snapshot := strings.TrimSpace(query.Get("snapshot"))
	if snapshot != "" {
		if blobServicePropertiesAPIVersion(header.Get("x-ms-version")) < "2020-08-04" {
			resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "FeatureVersionMismatch", "Set Blob Tags on snapshots requires Blob service version 2020-08-04 or later.")
			if resp != nil {
				if resp.Headers == nil {
					resp.Headers = make(map[string]string)
				}
				resp.Headers["x-ms-error-code"] = "FeatureVersionMismatch"
			}
			return resp, err
		}
		snapshotBlob, ok := container.Snapshots[blobName][snapshot]
		if !ok {
			return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
		}
		if !blobLeaseAllowsWrite(header, snapshotBlob) {
			return blobLeaseWriteFailure(header, snapshotBlob)
		}
		snapshotBlob.Tags = tags
		container.Snapshots[blobName][snapshot] = snapshotBlob
		s.containers[accountKey][containerKey] = container
		return emptyResponse(http.StatusNoContent, map[string]string{"x-ms-version": dataPlaneAPIVersion})
	}
	blob, ok := container.Blobs[blobName]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if !blobLeaseAllowsWrite(header, blob) {
		return blobLeaseWriteFailure(header, blob)
	}
	blob.Tags = tags
	container.Blobs[blobName] = blob
	s.containers[accountKey][containerKey] = container

	return emptyResponse(http.StatusNoContent, map[string]string{"x-ms-version": dataPlaneAPIVersion})
}

func (s *StorageService) getBlobTags(account, containerName, blobName string, query url.Values) (*service.Response, error) {
	blob, _, ok, resp, err := s.blobForRead(account, containerName, blobName, query.Get("snapshot"))
	if resp != nil || err != nil {
		return resp, err
	}
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}

	resp, err = xmlResponse(http.StatusOK, blobTagsDocumentFromMap(blob.Tags))
	if err != nil {
		return nil, err
	}
	resp.Headers = map[string]string{
		"Content-Length": strconv.Itoa(len(resp.RawBody)),
		"x-ms-version":   dataPlaneAPIVersion,
	}
	return resp, nil
}

func parseBlobTags(body []byte) (map[string]string, error) {
	var doc blobTagsDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	if doc.XMLName.Local != "Tags" {
		return nil, fmt.Errorf("expected Tags root element")
	}
	if len(doc.TagSet.Tags) > 10 {
		return nil, fmt.Errorf("a blob tag set can contain at most 10 tags")
	}
	tags := make(map[string]string, len(doc.TagSet.Tags))
	for _, tag := range doc.TagSet.Tags {
		if err := validateBlobTag(tag); err != nil {
			return nil, err
		}
		tags[tag.Key] = tag.Value
	}
	return tags, nil
}

func validateBlobTag(tag blobTag) error {
	if len(tag.Key) < 1 || len(tag.Key) > 128 {
		return fmt.Errorf("blob tag keys must be from 1 to 128 characters")
	}
	if len(tag.Value) > 256 {
		return fmt.Errorf("blob tag values must be from 0 to 256 characters")
	}
	if !validBlobTagText(tag.Key) {
		return fmt.Errorf("blob tag keys contain unsupported characters")
	}
	if !validBlobTagText(tag.Value) {
		return fmt.Errorf("blob tag values contain unsupported characters")
	}
	return nil
}

func validBlobTagText(value string) bool {
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == ' ' || r == '+' || r == '-' || r == '.' || r == '/' || r == ':' || r == '=' || r == '_':
		default:
			return false
		}
	}
	return true
}

func blobInvalidTagResponse(message string) (*service.Response, error) {
	if strings.TrimSpace(message) == "" {
		message = "The specified XML is not syntactically valid."
	}
	resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", message)
	if resp != nil {
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers["x-ms-error-code"] = "InvalidXmlDocument"
	}
	return resp, err
}

func blobTagsDocumentFromMap(tags map[string]string) blobTagsDocument {
	return blobTagsDocument{TagSet: blobTagSetFromMap(tags)}
}

func blobTagSetFromMap(tags map[string]string) blobTagSet {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := blobTagSet{Tags: make([]blobTag, 0, len(keys))}
	for _, key := range keys {
		out.Tags = append(out.Tags, blobTag{
			Key:   key,
			Value: tags[key],
		})
	}
	return out
}

func blobPropertyHeaders(blob blobObject) map[string]string {
	headers := storageHeaders(blob.ETag, blob.LastModified)
	for key, value := range blob.Metadata {
		headers["x-ms-meta-"+key] = value
	}
	headers["Accept-Ranges"] = "bytes"
	headers["Content-Length"] = strconv.Itoa(len(blob.Content))
	headers["Content-Type"] = blob.ContentType
	if blob.CacheControl != "" {
		headers["Cache-Control"] = blob.CacheControl
	}
	if blob.ContentEncoding != "" {
		headers["Content-Encoding"] = blob.ContentEncoding
	}
	if blob.ContentLanguage != "" {
		headers["Content-Language"] = blob.ContentLanguage
	}
	if blob.ContentDisposition != "" {
		headers["Content-Disposition"] = blob.ContentDisposition
	}
	if blob.ContentMD5 != "" {
		headers["Content-MD5"] = blob.ContentMD5
	}
	if len(blob.Tags) > 0 {
		headers["x-ms-tag-count"] = strconv.Itoa(len(blob.Tags))
	}
	if blob.CopyID != "" {
		headers["x-ms-copy-id"] = blob.CopyID
		headers["x-ms-copy-status"] = blob.CopyStatus
		headers["x-ms-copy-source"] = blob.CopySource
		headers["x-ms-copy-progress"] = blob.CopyProgress
		if !blob.CopyCompletionTime.IsZero() {
			headers["x-ms-copy-completion-time"] = blob.CopyCompletionTime.UTC().Format(http.TimeFormat)
		}
	}
	if blob.AccessTier != "" {
		headers["x-ms-access-tier"] = blob.AccessTier
	}
	if !blob.AccessTierChanged.IsZero() {
		headers["x-ms-access-tier-change-time"] = blob.AccessTierChanged.UTC().Format(http.TimeFormat)
	}
	if blob.ArchiveStatus != "" {
		headers["x-ms-archive-status"] = blob.ArchiveStatus
	}
	if blob.RehydratePriority != "" {
		headers["x-ms-rehydrate-priority"] = blob.RehydratePriority
	}
	headers["x-ms-blob-type"] = blobTypeOrDefault(blob.BlobType)
	if blobTypeOrDefault(blob.BlobType) == "AppendBlob" {
		headers["x-ms-blob-committed-block-count"] = strconv.Itoa(blob.AppendBlockCount)
	}
	if blobTypeOrDefault(blob.BlobType) == "PageBlob" {
		headers["x-ms-blob-sequence-number"] = pageBlobSequenceNumber(blob)
	}
	switch blobLeaseState(blob) {
	case "leased", "breaking":
		headers["x-ms-lease-status"] = "locked"
		headers["x-ms-lease-state"] = blobLeaseState(blob)
	case "broken":
		headers["x-ms-lease-status"] = "unlocked"
		headers["x-ms-lease-state"] = "broken"
	default:
		headers["x-ms-lease-status"] = "unlocked"
		headers["x-ms-lease-state"] = "available"
	}
	return headers
}

func blobTypeOrDefault(blobType string) string {
	if blobType == "" {
		return "BlockBlob"
	}
	return blobType
}

func pageBlobSequenceNumber(blob blobObject) string {
	if blob.SequenceNumber == "" {
		return "0"
	}
	return blob.SequenceNumber
}

func pageBlobSequenceConditionsMatch(header http.Header, blob blobObject) bool {
	current, err := strconv.ParseInt(pageBlobSequenceNumber(blob), 10, 64)
	if err != nil {
		current = 0
	}
	for _, condition := range []struct {
		header string
		match  func(int64) bool
	}{
		{header: "x-ms-if-sequence-number-eq", match: func(value int64) bool { return current == value }},
		{header: "x-ms-if-sequence-number-le", match: func(value int64) bool { return current <= value }},
		{header: "x-ms-if-sequence-number-lt", match: func(value int64) bool { return current < value }},
	} {
		raw := strings.TrimSpace(header.Get(condition.header))
		if raw == "" {
			continue
		}
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 0 || !condition.match(value) {
			return false
		}
	}
	return true
}

func pageBlobRangeFromHeader(header http.Header) (byteRange, bool, error) {
	raw := strings.TrimSpace(header.Get("x-ms-range"))
	if raw == "" {
		raw = strings.TrimSpace(header.Get("Range"))
	}
	if raw == "" {
		return byteRange{}, false, nil
	}
	start, end, ok := parseFileRangeHeader(raw)
	if !ok || start < 0 || end < start || start%512 != 0 || (end+1)%512 != 0 {
		return byteRange{}, false, fmt.Errorf("invalid page range")
	}
	return byteRange{start: start, end: end}, true, nil
}

func pageBlobOptionalRangeFromHeader(header http.Header, contentLength int) (byteRange, bool, error) {
	raw := strings.TrimSpace(header.Get("x-ms-range"))
	if raw == "" {
		raw = strings.TrimSpace(header.Get("Range"))
	}
	if raw == "" {
		return byteRange{}, false, nil
	}
	start, end, ok := parseFileRangeHeader(raw)
	if !ok || start < 0 || end < start || contentLength == 0 || start >= contentLength {
		return byteRange{}, false, fmt.Errorf("invalid page range")
	}
	if end >= contentLength {
		end = contentLength - 1
	}
	return byteRange{start: start, end: end}, true, nil
}

func addBlobPageRange(ranges []fileRange, added fileRange) []fileRange {
	out := append([]fileRange(nil), ranges...)
	out = append(out, added)
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	merged := make([]fileRange, 0, len(out))
	for _, candidate := range out {
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

func clearBlobPageRange(ranges []fileRange, cleared fileRange) []fileRange {
	out := make([]fileRange, 0, len(ranges))
	for _, existing := range ranges {
		if existing.End < cleared.Start || existing.Start > cleared.End {
			out = append(out, existing)
			continue
		}
		if existing.Start < cleared.Start {
			out = append(out, fileRange{Start: existing.Start, End: cleared.Start - 1})
		}
		if existing.End > cleared.End {
			out = append(out, fileRange{Start: cleared.End + 1, End: existing.End})
		}
	}
	return out
}

func clipBlobPageRanges(ranges []fileRange, clip fileRange) []fileRange {
	out := make([]fileRange, 0, len(ranges))
	for _, existing := range ranges {
		start := existing.Start
		if start < clip.Start {
			start = clip.Start
		}
		end := existing.End
		if end > clip.End {
			end = clip.End
		}
		if start <= end {
			out = append(out, fileRange{Start: start, End: end})
		}
	}
	return out
}

func blobPageRangeListItems(ranges []fileRange) []blobPageRangeListItem {
	items := make([]blobPageRangeListItem, 0, len(ranges))
	for _, pageRange := range ranges {
		items = append(items, blobPageRangeListItem{
			Start: pageRange.Start,
			End:   pageRange.End,
		})
	}
	return items
}

func snapshotTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.0000000Z")
}

func cloneBlobObject(blob blobObject) blobObject {
	clone := blob
	clone.Content = append([]byte(nil), blob.Content...)
	clone.Metadata = cloneStringMap(blob.Metadata)
	clone.Tags = cloneStringMap(blob.Tags)
	clone.CommittedBlocks = cloneBlobBlocks(blob.CommittedBlocks)
	clone.PageRanges = append([]fileRange(nil), blob.PageRanges...)
	return clone
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneBlobBlocks(in []blobBlock) []blobBlock {
	if len(in) == 0 {
		return nil
	}
	out := make([]blobBlock, len(in))
	for i, block := range in {
		out[i] = blobBlock{
			ID:      block.ID,
			Content: append([]byte(nil), block.Content...),
		}
	}
	return out
}

func blobContentTypeFromHeaders(header http.Header) string {
	if contentType := header.Get("x-ms-blob-content-type"); contentType != "" {
		return contentType
	}
	if contentType := header.Get("Content-Type"); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}

func blobHeaderValue(header http.Header, azureHeader, standardHeader string) string {
	if value := header.Get(azureHeader); value != "" {
		return value
	}
	return header.Get(standardHeader)
}

func (s *StorageService) copyBlob(account, containerName, blobName string, header http.Header) (*service.Response, error) {
	sourceURL := strings.TrimSpace(header.Get("x-ms-copy-source"))
	if sourceURL == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-copy-source header is required.")
	}
	source, ok := parseFileCopySource(account, sourceURL)
	if !ok {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-copy-source header must reference a file or blob.")
	}

	now := time.Now().UTC()
	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)

	s.mu.Lock()
	defer s.mu.Unlock()

	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}

	var leaseID, leaseState string
	if destination, exists := container.Blobs[blobName]; exists {
		if !blobLeaseAllowsWrite(header, destination) {
			return blobLeaseWriteFailure(header, destination)
		}
		if !blobConditionsMatch(header, destination.ETag) {
			return blobConditionNotMetResponse()
		}
		leaseID = destination.LeaseID
		leaseState = destination.LeaseState
	}

	sourceBlob, sourceOK := s.blobCopySource(source)
	if !sourceOK {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified copy source does not exist.")
	}
	if metadataHeadersPresent(header) {
		sourceBlob.Metadata = metadataFromHeaders(header)
	}

	copyID := s.nextToken("copy-")
	etag := s.nextToken("\"blob") + "\""
	destination := blobObject{
		Name:               blobName,
		BlobType:           blobTypeOrDefault(sourceBlob.BlobType),
		Content:            append([]byte(nil), sourceBlob.Content...),
		ContentType:        sourceBlob.ContentType,
		CacheControl:       sourceBlob.CacheControl,
		ContentEncoding:    sourceBlob.ContentEncoding,
		ContentLanguage:    sourceBlob.ContentLanguage,
		ContentDisposition: sourceBlob.ContentDisposition,
		ContentMD5:         sourceBlob.ContentMD5,
		Metadata:           cloneStringMap(sourceBlob.Metadata),
		ETag:               etag,
		LastModified:       now,
		CommittedBlocks:    cloneBlobBlocks(sourceBlob.CommittedBlocks),
		AppendBlockCount:   sourceBlob.AppendBlockCount,
		PageRanges:         append([]fileRange(nil), sourceBlob.PageRanges...),
		SequenceNumber:     sourceBlob.SequenceNumber,
		LeaseID:            leaseID,
		LeaseState:         leaseState,
		CopyID:             copyID,
		CopyStatus:         "success",
		CopySource:         sourceURL,
		CopyProgress:       strconv.Itoa(len(sourceBlob.Content)) + "/" + strconv.Itoa(len(sourceBlob.Content)),
		CopyCompletionTime: now,
	}
	if container.Blobs == nil {
		container.Blobs = make(map[string]blobObject)
	}
	if container.StagedBlocks != nil {
		delete(container.StagedBlocks, blobName)
	}
	container.Blobs[blobName] = destination
	container.ETag = etag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container

	headers := storageHeaders(destination.ETag, destination.LastModified)
	headers["x-ms-copy-id"] = copyID
	headers["x-ms-copy-status"] = "success"
	return emptyResponse(http.StatusAccepted, headers)
}

func (s *StorageService) abortCopyBlob(account, containerName, blobName string, query url.Values, header http.Header) (*service.Response, error) {
	copyAction := strings.TrimSpace(header.Get("x-ms-copy-action"))
	if copyAction == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-copy-action header is required.")
	}
	if !strings.EqualFold(copyAction, "abort") {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-copy-action header value is invalid.")
	}
	copyID := strings.TrimSpace(query.Get("copyid"))
	if copyID == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredQueryParameter", "The copyid query parameter is required.")
	}

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)

	s.mu.Lock()
	defer s.mu.Unlock()

	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	blob, ok := container.Blobs[blobName]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if !blobLeaseAllowsWrite(header, blob) {
		return blobLeaseWriteFailure(header, blob)
	}
	if !strings.EqualFold(blob.CopyStatus, "pending") {
		return azurearm.ErrorResponse(http.StatusConflict, "NoPendingCopyOperation", "There is currently no pending copy operation.")
	}
	if blob.CopyID != copyID {
		return azurearm.ErrorResponse(http.StatusConflict, "CopyIdMismatch", "The specified copy ID did not match the copy ID for the pending copy operation.")
	}

	now := time.Now().UTC()
	blob.Content = nil
	blob.ETag = s.nextToken("\"blob") + "\""
	blob.LastModified = now
	blob.CopyStatus = "aborted"
	blob.CopyProgress = "0/0"
	blob.CopyCompletionTime = now
	container.Blobs[blobName] = blob
	container.ETag = blob.ETag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container

	return emptyResponse(http.StatusNoContent, blobBaseHeadersForRequest(header))
}

func (s *StorageService) putBlobFromURL(account, containerName, blobName string, ctx *service.RequestContext) (*service.Response, error) {
	header := ctx.RawRequest.Header
	if len(ctx.Body) != 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The Content-Length header must be 0 for Put Blob From URL.")
	}
	if !strings.EqualFold(strings.TrimSpace(header.Get("x-ms-blob-type")), "BlockBlob") {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-blob-type header must be BlockBlob.")
	}
	sourceURL := strings.TrimSpace(header.Get("x-ms-copy-source"))
	if sourceURL == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRequiredHeader", "The x-ms-copy-source header is required.")
	}
	source, ok := parseFileCopySource(account, sourceURL)
	if !ok {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", "The x-ms-copy-source header must reference a file or blob.")
	}

	now := time.Now().UTC()
	etag := s.nextToken("\"blob") + "\""
	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)

	s.mu.Lock()
	defer s.mu.Unlock()

	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	var leaseID, leaseState string
	if existing, exists := container.Blobs[blobName]; exists {
		if !blobLeaseAllowsWrite(header, existing) {
			return blobLeaseWriteFailure(header, existing)
		}
		if !blobConditionsMatch(header, existing.ETag) {
			return blobConditionNotMetResponse()
		}
		leaseID = existing.LeaseID
		leaseState = existing.LeaseState
	}

	sourceBlob, sourceOK := s.blobCopySource(source)
	if !sourceOK {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified copy source does not exist.")
	}

	contentType := blobHeaderValue(header, "x-ms-blob-content-type", "Content-Type")
	if contentType == "" {
		contentType = sourceBlob.ContentType
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	destination := blobObject{
		Name:               blobName,
		BlobType:           "BlockBlob",
		Content:            append([]byte(nil), sourceBlob.Content...),
		ContentType:        contentType,
		CacheControl:       blobHeaderWithFallback(header, "x-ms-blob-cache-control", "Cache-Control", sourceBlob.CacheControl),
		ContentEncoding:    blobHeaderWithFallback(header, "x-ms-blob-content-encoding", "Content-Encoding", sourceBlob.ContentEncoding),
		ContentLanguage:    blobHeaderWithFallback(header, "x-ms-blob-content-language", "Content-Language", sourceBlob.ContentLanguage),
		ContentDisposition: blobHeaderWithFallback(header, "x-ms-blob-content-disposition", "Content-Disposition", sourceBlob.ContentDisposition),
		ContentMD5:         blobHeaderWithFallback(header, "x-ms-blob-content-md5", "Content-MD5", sourceBlob.ContentMD5),
		Metadata:           metadataFromHeaders(header),
		ETag:               etag,
		LastModified:       now,
		LeaseID:            leaseID,
		LeaseState:         leaseState,
	}
	if container.Blobs == nil {
		container.Blobs = make(map[string]blobObject)
	}
	if container.StagedBlocks != nil {
		delete(container.StagedBlocks, blobName)
	}
	container.Blobs[blobName] = destination
	container.ETag = etag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container

	return emptyResponse(http.StatusCreated, storageHeaders(etag, now))
}

func blobHeaderWithFallback(header http.Header, azureHeader, standardHeader, fallback string) string {
	if value := blobHeaderValue(header, azureHeader, standardHeader); value != "" {
		return value
	}
	return fallback
}

func (s *StorageService) blobCopySource(source fileCopySource) (blobObject, bool) {
	accountKey := strings.ToLower(source.Account)
	containerKey := strings.ToLower(source.Container)
	switch source.Service {
	case "blob":
		container, ok := s.containers[accountKey][containerKey]
		if !ok {
			return blobObject{}, false
		}
		if source.Snapshot != "" {
			blob, ok := container.Snapshots[source.Path][source.Snapshot]
			return cloneBlobObject(blob), ok
		}
		blob, ok := container.Blobs[source.Path]
		return cloneBlobObject(blob), ok
	case "file":
		share, ok := s.fileShares[accountKey][containerKey]
		if !ok {
			return blobObject{}, false
		}
		file, ok := share.Files[source.Path]
		if !ok {
			return blobObject{}, false
		}
		return blobObject{
			Content:            append([]byte(nil), file.Content...),
			ContentType:        file.ContentType,
			CacheControl:       file.CacheControl,
			ContentEncoding:    file.ContentEncoding,
			ContentLanguage:    file.ContentLanguage,
			ContentDisposition: file.ContentDisposition,
			ContentMD5:         file.ContentMD5,
			Metadata:           cloneStringMap(file.Metadata),
		}, true
	default:
		return blobObject{}, false
	}
}

func (s *StorageService) setBlobMetadata(account, containerName, blobName string, header http.Header) (*service.Response, error) {
	now := time.Now().UTC()
	etag := s.nextToken("\"blob") + "\""
	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)

	s.mu.Lock()
	defer s.mu.Unlock()

	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	blob, ok := container.Blobs[blobName]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if !blobLeaseAllowsWrite(header, blob) {
		return blobLeaseWriteFailure(header, blob)
	}
	if !blobConditionsMatch(header, blob.ETag) {
		return blobConditionNotMetResponse()
	}
	blob.Metadata = metadataFromHeaders(header)
	blob.ETag = etag
	blob.LastModified = now
	container.Blobs[blobName] = blob
	container.ETag = etag
	container.LastModified = now
	s.containers[accountKey][containerKey] = container

	return emptyResponse(http.StatusOK, storageHeaders(etag, now))
}

func (s *StorageService) getBlobMetadata(account, containerName, blobName string, query url.Values, header http.Header) (*service.Response, error) {
	blob, snapshot, ok, resp, err := s.blobForRead(account, containerName, blobName, query.Get("snapshot"))
	if resp != nil || err != nil {
		return resp, err
	}
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if !blobLeaseAllowsRead(header, blob) {
		return blobLeaseReadFailure()
	}
	if conditionResp, ok, err := blobReadConditionsResponse(header, blob); !ok {
		return conditionResp, err
	}
	headers := storageHeaders(blob.ETag, blob.LastModified)
	for key, value := range blob.Metadata {
		headers["x-ms-meta-"+key] = value
	}
	if snapshot != "" {
		headers["x-ms-snapshot"] = snapshot
	}
	return emptyResponse(http.StatusOK, headers)
}

type byteRange struct {
	start int
	end   int
}

func requestedBlobRange(header http.Header, contentLength int) (byteRange, bool, error) {
	raw := strings.TrimSpace(header.Get("x-ms-range"))
	if raw == "" {
		raw = strings.TrimSpace(header.Get("Range"))
	}
	return requestedByteRange(raw, contentLength)
}

func requestedByteRange(raw string, contentLength int) (byteRange, bool, error) {
	if raw == "" {
		return byteRange{}, false, nil
	}
	if !strings.HasPrefix(strings.ToLower(raw), "bytes=") {
		return byteRange{}, false, fmt.Errorf("unsupported range unit")
	}
	spec := strings.TrimSpace(raw[len("bytes="):])
	before, after, ok := strings.Cut(spec, "-")
	if !ok || before == "" {
		return byteRange{}, false, fmt.Errorf("invalid range")
	}
	start, err := strconv.Atoi(before)
	if err != nil || start < 0 {
		return byteRange{}, false, fmt.Errorf("invalid range start")
	}
	end := contentLength - 1
	if after != "" {
		end, err = strconv.Atoi(after)
		if err != nil {
			return byteRange{}, false, fmt.Errorf("invalid range end")
		}
	}
	if contentLength == 0 || start >= contentLength || end < start {
		return byteRange{}, false, fmt.Errorf("range outside content")
	}
	if end >= contentLength {
		end = contentLength - 1
	}
	return byteRange{start: start, end: end}, true, nil
}

func (s *StorageService) listBlobs(account, containerName, prefix, delimiter, marker string, maxResults int, includeMetadata bool) (*service.Response, error) {
	s.mu.RLock()
	container, ok := s.containers[strings.ToLower(account)][strings.ToLower(containerName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}

	items := make([]blobListEntry, 0, len(container.Blobs))
	prefixes := make(map[string]bool)
	for _, blob := range container.Blobs {
		if prefix != "" && !strings.HasPrefix(blob.Name, prefix) {
			continue
		}
		if delimiter != "" {
			suffix := strings.TrimPrefix(blob.Name, prefix)
			if index := strings.Index(suffix, delimiter); index >= 0 {
				prefixName := prefix + suffix[:index+len(delimiter)]
				if !prefixes[prefixName] {
					items = append(items, blobListEntry{Prefix: &blobListPrefix{Name: prefixName}})
					prefixes[prefixName] = true
				}
				continue
			}
		}
		item := &blobListItem{
			Name: blob.Name,
			Properties: blobListProperties{
				LastModified:  blob.LastModified.UTC().Format(http.TimeFormat),
				ETag:          blob.ETag,
				ContentLength: len(blob.Content),
				ContentType:   blob.ContentType,
			},
		}
		if includeMetadata && len(blob.Metadata) > 0 {
			metadata := blobListMetadata(blob.Metadata)
			item.Metadata = &metadata
		}
		items = append(items, blobListEntry{Blob: item})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name() < items[j].Name() })

	start := 0
	if marker != "" {
		for start < len(items) && items[start].Name() < marker {
			start++
		}
	}
	end := len(items)
	nextMarker := ""
	if maxResults > 0 && start+maxResults < len(items) {
		end = start + maxResults
		nextMarker = items[end].Name()
	}

	return xmlResponse(http.StatusOK, blobListResponse{
		ServiceEndpoint: "https://" + account + ".blob.core.windows.net/",
		ContainerName:   containerName,
		Prefix:          prefix,
		Marker:          marker,
		MaxResults:      maxResults,
		Delimiter:       delimiter,
		Blobs:           blobListEntries{Entries: items[start:end]},
		NextMarker:      nextMarker,
	})
}

func (s *StorageService) deleteBlob(account, containerName, blobName string, query url.Values, header http.Header) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	containerKey := strings.ToLower(containerName)
	container, ok := s.containers[accountKey][containerKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ContainerNotFound", "The specified container does not exist.")
	}
	snapshot := strings.TrimSpace(query.Get("snapshot"))
	deleteSnapshots := strings.ToLower(strings.TrimSpace(header.Get("x-ms-delete-snapshots")))
	if snapshot != "" {
		if deleteSnapshots != "" {
			return blobInvalidHeaderValueResponse("The x-ms-delete-snapshots header is not valid for snapshot deletes.")
		}
		snapshotBlob, ok := container.Snapshots[blobName][snapshot]
		if !ok {
			return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
		}
		if !blobConditionsMatch(header, snapshotBlob.ETag) {
			return blobConditionNotMetResponse()
		}
		delete(container.Snapshots[blobName], snapshot)
		if len(container.Snapshots[blobName]) == 0 {
			delete(container.Snapshots, blobName)
		}
		s.containers[accountKey][containerKey] = container
		return emptyResponse(http.StatusAccepted, storageHeaders(s.nextToken("\"delete")+"\"", time.Now().UTC()))
	}

	blob := container.Blobs[blobName]
	if blob.Name == "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "BlobNotFound", "The specified blob does not exist.")
	}
	if !blobLeaseAllowsWrite(header, blob) {
		return blobLeaseWriteFailure(header, blob)
	}
	if !blobConditionsMatch(header, blob.ETag) {
		return blobConditionNotMetResponse()
	}
	hasSnapshots := len(container.Snapshots[blobName]) > 0
	switch deleteSnapshots {
	case "":
		if hasSnapshots {
			return blobSnapshotsPresentResponse()
		}
		delete(container.Blobs, blobName)
	case "only":
		if hasSnapshots {
			delete(container.Snapshots, blobName)
		}
	case "include":
		delete(container.Blobs, blobName)
		if hasSnapshots {
			delete(container.Snapshots, blobName)
		}
	default:
		return blobInvalidHeaderValueResponse("The x-ms-delete-snapshots header value is invalid.")
	}
	s.containers[accountKey][containerKey] = container
	return emptyResponse(http.StatusAccepted, storageHeaders(s.nextToken("\"delete")+"\"", time.Now().UTC()))
}

func blobSnapshotsPresentResponse() (*service.Response, error) {
	resp, err := azurearm.ErrorResponse(http.StatusConflict, "SnapshotsPresent", "This operation is not permitted because the blob has snapshots.")
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = "SnapshotsPresent"
	}
	return resp, err
}

func blobInvalidHeaderValueResponse(message string) (*service.Response, error) {
	resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeaderValue", message)
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = "InvalidHeaderValue"
	}
	return resp, err
}

func blobConditionsMatch(header http.Header, etag string) bool {
	if match := strings.TrimSpace(header.Get("If-Match")); match != "" && match != "*" && match != etag {
		return false
	}
	if noneMatch := strings.TrimSpace(header.Get("If-None-Match")); noneMatch != "" && (noneMatch == "*" || noneMatch == etag) {
		return false
	}
	return true
}

func blobReadConditionsResponse(header http.Header, blob blobObject) (*service.Response, bool, error) {
	if match := strings.TrimSpace(header.Get("If-Match")); match != "" && match != "*" && match != blob.ETag {
		resp, err := blobConditionNotMetResponse()
		return resp, false, err
	}

	lastModified := blob.LastModified.UTC().Truncate(time.Second)
	if unmodifiedSince := strings.TrimSpace(header.Get("If-Unmodified-Since")); unmodifiedSince != "" {
		when, err := http.ParseTime(unmodifiedSince)
		if err != nil {
			resp, err := blobInvalidHeaderValueResponse("The If-Unmodified-Since header is invalid.")
			return resp, false, err
		}
		if lastModified.After(when.UTC()) {
			resp, err := blobConditionNotMetResponse()
			return resp, false, err
		}
	}

	noneMatchPasses := true
	noneMatchPresent := false
	if noneMatch := strings.TrimSpace(header.Get("If-None-Match")); noneMatch != "" {
		noneMatchPresent = true
		noneMatchPasses = noneMatch != "*" && noneMatch != blob.ETag
	}

	modifiedSincePasses := true
	modifiedSincePresent := false
	if modifiedSince := strings.TrimSpace(header.Get("If-Modified-Since")); modifiedSince != "" {
		modifiedSincePresent = true
		when, err := http.ParseTime(modifiedSince)
		if err != nil {
			resp, err := blobInvalidHeaderValueResponse("The If-Modified-Since header is invalid.")
			return resp, false, err
		}
		modifiedSincePasses = lastModified.After(when.UTC())
	}

	if noneMatchPresent || modifiedSincePresent {
		combinedPasses := false
		if noneMatchPresent {
			combinedPasses = combinedPasses || noneMatchPasses
		}
		if modifiedSincePresent {
			combinedPasses = combinedPasses || modifiedSincePasses
		}
		if !combinedPasses {
			resp, err := emptyResponse(http.StatusNotModified, blobPropertyHeaders(blob))
			return resp, false, err
		}
	}
	return nil, true, nil
}

func blobConditionNotMetResponse() (*service.Response, error) {
	resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "ConditionNotMet", "The condition specified using HTTP conditional header(s) is not met.")
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = "ConditionNotMet"
	}
	return resp, err
}

func blobLeaseAllowsWrite(header http.Header, blob blobObject) bool {
	requestLeaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
	switch {
	case blobLeaseState(blob) == "leased" || blobLeaseState(blob) == "breaking":
		return requestLeaseID == blob.LeaseID
	case requestLeaseID != "":
		return false
	default:
		return true
	}
}

func blobLeaseAllowsRead(header http.Header, blob blobObject) bool {
	requestLeaseID := strings.TrimSpace(header.Get("x-ms-lease-id"))
	if requestLeaseID == "" {
		return true
	}
	return blobLeaseState(blob) == "leased" && requestLeaseID == blob.LeaseID
}

func blobLeaseReadFailure() (*service.Response, error) {
	resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseIdMismatchWithBlobOperation", "The lease ID specified did not match the lease ID for the blob.")
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = "LeaseIdMismatchWithBlobOperation"
	}
	return resp, err
}

func blobLeaseWriteFailure(header http.Header, blob blobObject) (*service.Response, error) {
	if (blobLeaseState(blob) == "leased" || blobLeaseState(blob) == "breaking") && strings.TrimSpace(header.Get("x-ms-lease-id")) == "" {
		resp, err := azurearm.ErrorResponse(http.StatusPreconditionFailed, "LeaseIdMissing", "There is currently a lease on the blob and no lease ID was specified in the request.")
		if resp != nil {
			resp.Headers = storageHeaders("", time.Now().UTC())
			resp.Headers["x-ms-error-code"] = "LeaseIdMissing"
		}
		return resp, err
	}
	resp, err := azurearm.ErrorResponse(http.StatusConflict, "LeaseIdMismatchWithBlobOperation", "The lease ID specified did not match the lease ID for the blob.")
	if resp != nil {
		resp.Headers = storageHeaders("", time.Now().UTC())
		resp.Headers["x-ms-error-code"] = "LeaseIdMismatchWithBlobOperation"
	}
	return resp, err
}

func blobLeaseState(blob blobObject) string {
	if blob.LeaseState != "" {
		return blob.LeaseState
	}
	if blob.LeaseID != "" {
		return "leased"
	}
	return "available"
}

type blockListResponse struct {
	XMLName           xml.Name         `xml:"BlockList"`
	CommittedBlocks   *blockListBlocks `xml:"CommittedBlocks,omitempty"`
	UncommittedBlocks *blockListBlocks `xml:"UncommittedBlocks,omitempty"`
}

type blockListBlocks struct {
	Blocks []blockListBlock `xml:"Block"`
}

type blockListBlock struct {
	Name string `xml:"Name"`
	Size int    `xml:"Size"`
}

type blobPageListResponse struct {
	XMLName    xml.Name                `xml:"PageList"`
	PageRanges []blobPageRangeListItem `xml:"PageRange"`
}

type blobPageRangeListItem struct {
	Start int `xml:"Start"`
	End   int `xml:"End"`
}

type findBlobsByTagsResponse struct {
	XMLName         xml.Name              `xml:"EnumerationResults"`
	ServiceEndpoint string                `xml:"ServiceEndpoint,attr"`
	Where           string                `xml:"Where"`
	Blobs           []findBlobsByTagsItem `xml:"Blobs>Blob"`
	NextMarker      string                `xml:"NextMarker"`
}

type findBlobsByTagsItem struct {
	Name          string          `xml:"Name"`
	ContainerName string          `xml:"ContainerName"`
	Tags          blobTagsPayload `xml:"Tags"`
}

type blobContainerSignedIdentifier struct {
	ID           string                    `xml:"Id"`
	AccessPolicy blobContainerAccessPolicy `xml:"AccessPolicy"`
}

type blobContainerAccessPolicy struct {
	Start      string `xml:"Start,omitempty"`
	Expiry     string `xml:"Expiry,omitempty"`
	Permission string `xml:"Permission,omitempty"`
}

type blobContainerACLResponse struct {
	XMLName           xml.Name                        `xml:"SignedIdentifiers"`
	SignedIdentifiers []blobContainerSignedIdentifier `xml:"SignedIdentifier"`
}

type blobTagsPayload struct {
	TagSet blobTagSet `xml:"TagSet"`
}

type blobTagsDocument struct {
	XMLName xml.Name   `xml:"Tags"`
	TagSet  blobTagSet `xml:"TagSet"`
}

type blobTagSet struct {
	Tags []blobTag `xml:"Tag"`
}

type blobTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type blobListResponse struct {
	XMLName         xml.Name        `xml:"EnumerationResults"`
	ServiceEndpoint string          `xml:"ServiceEndpoint,attr,omitempty"`
	ContainerName   string          `xml:"ContainerName,attr,omitempty"`
	Prefix          string          `xml:"Prefix,omitempty"`
	Marker          string          `xml:"Marker,omitempty"`
	MaxResults      int             `xml:"MaxResults,omitempty"`
	Delimiter       string          `xml:"Delimiter,omitempty"`
	Blobs           blobListEntries `xml:"Blobs"`
	NextMarker      string          `xml:"NextMarker,omitempty"`
}

type containerListResponse struct {
	XMLName         xml.Name            `xml:"EnumerationResults"`
	ServiceEndpoint string              `xml:"ServiceEndpoint,attr"`
	Prefix          string              `xml:"Prefix,omitempty"`
	Marker          string              `xml:"Marker,omitempty"`
	MaxResults      int                 `xml:"MaxResults,omitempty"`
	Containers      []containerListItem `xml:"Containers>Container"`
	NextMarker      string              `xml:"NextMarker,omitempty"`
}

type containerListItem struct {
	Name       string                  `xml:"Name"`
	Properties containerListProperties `xml:"Properties"`
	Metadata   *blobListMetadata       `xml:"Metadata,omitempty"`
}

type containerListProperties struct {
	LastModified string `xml:"Last-Modified"`
	ETag         string `xml:"Etag"`
}

type blobListEntries struct {
	Entries []blobListEntry
}

func (entries blobListEntries) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "Blobs"
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, entry := range entries.Entries {
		switch {
		case entry.Blob != nil:
			if err := e.Encode(entry.Blob); err != nil {
				return err
			}
		case entry.Prefix != nil:
			if err := e.Encode(entry.Prefix); err != nil {
				return err
			}
		}
	}
	return e.EncodeToken(start.End())
}

type blobListEntry struct {
	Blob   *blobListItem
	Prefix *blobListPrefix
}

func (entry blobListEntry) Name() string {
	if entry.Prefix != nil {
		return entry.Prefix.Name
	}
	if entry.Blob != nil {
		return entry.Blob.Name
	}
	return ""
}

type blobListItem struct {
	XMLName    xml.Name           `xml:"Blob"`
	Name       string             `xml:"Name"`
	Properties blobListProperties `xml:"Properties"`
	Metadata   *blobListMetadata  `xml:"Metadata,omitempty"`
}

type blobListPrefix struct {
	XMLName xml.Name `xml:"BlobPrefix"`
	Name    string   `xml:"Name"`
}

type blobListProperties struct {
	LastModified  string `xml:"Last-Modified"`
	ETag          string `xml:"Etag"`
	ContentLength int    `xml:"Content-Length"`
	ContentType   string `xml:"Content-Type"`
}

type blobListMetadata map[string]string

func (metadata blobListMetadata) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "Metadata"
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := e.EncodeElement(metadata[key], xml.StartElement{Name: xml.Name{Local: key}}); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
}

func blobListIncludesMetadata(raw string) bool {
	for _, value := range strings.Split(raw, ",") {
		if strings.EqualFold(strings.TrimSpace(value), "metadata") {
			return true
		}
	}
	return false
}

type blobTagWhere struct {
	ContainerName string
	Conditions    []blobTagCondition
}

type blobTagCondition struct {
	Key   string
	Op    string
	Value string
}

func (where blobTagWhere) matches(tags map[string]string) bool {
	for _, condition := range where.Conditions {
		value, ok := tags[condition.Key]
		if !ok {
			return false
		}
		compare := strings.Compare(value, condition.Value)
		switch condition.Op {
		case "=":
			if compare != 0 {
				return false
			}
		case ">":
			if compare <= 0 {
				return false
			}
		case ">=":
			if compare < 0 {
				return false
			}
		case "<":
			if compare >= 0 {
				return false
			}
		case "<=":
			if compare > 0 {
				return false
			}
		default:
			return false
		}
	}
	return len(where.Conditions) > 0
}

func parseBlobTagWhere(raw string) (blobTagWhere, error) {
	clauses := splitBlobTagWhereClauses(raw)
	if len(clauses) == 0 {
		return blobTagWhere{}, fmt.Errorf("empty where expression")
	}

	var where blobTagWhere
	for _, clause := range clauses {
		left, op, right, ok := splitBlobTagCondition(clause)
		if !ok {
			return blobTagWhere{}, fmt.Errorf("invalid where clause")
		}
		value, ok := parseBlobTagStringLiteral(right)
		if !ok {
			return blobTagWhere{}, fmt.Errorf("invalid tag value")
		}
		if strings.EqualFold(left, "@container") {
			if op != "=" {
				return blobTagWhere{}, fmt.Errorf("invalid container operator")
			}
			where.ContainerName = value
			continue
		}
		key, ok := parseBlobTagKey(left)
		if !ok {
			return blobTagWhere{}, fmt.Errorf("invalid tag key")
		}
		where.Conditions = append(where.Conditions, blobTagCondition{
			Key:   key,
			Op:    op,
			Value: value,
		})
	}
	if len(where.Conditions) == 0 {
		return blobTagWhere{}, fmt.Errorf("missing tag condition")
	}
	return where, nil
}

func splitBlobTagWhereClauses(raw string) []string {
	clauses := make([]string, 0)
	start := 0
	inSingleQuote := false
	inDoubleQuote := false
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		}
		if inSingleQuote || inDoubleQuote || i+3 > len(raw) || !strings.EqualFold(raw[i:i+3], "AND") {
			continue
		}
		beforeOK := i == 0 || isBlobTagWhereSpace(raw[i-1])
		afterOK := i+3 == len(raw) || isBlobTagWhereSpace(raw[i+3])
		if !beforeOK || !afterOK {
			continue
		}
		if clause := strings.TrimSpace(raw[start:i]); clause != "" {
			clauses = append(clauses, clause)
		}
		start = i + 3
		i += 2
	}
	if clause := strings.TrimSpace(raw[start:]); clause != "" {
		clauses = append(clauses, clause)
	}
	return clauses
}

func splitBlobTagCondition(raw string) (string, string, string, bool) {
	for _, op := range []string{">=", "<=", "=", ">", "<"} {
		if index := strings.Index(raw, op); index >= 0 {
			left := strings.TrimSpace(raw[:index])
			right := strings.TrimSpace(raw[index+len(op):])
			return left, op, right, left != "" && right != ""
		}
	}
	return "", "", "", false
}

func parseBlobTagStringLiteral(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '\'' || raw[len(raw)-1] != '\'' {
		return "", false
	}
	return raw[1 : len(raw)-1], true
}

func parseBlobTagKey(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if strings.HasPrefix(raw, "\"") || strings.HasSuffix(raw, "\"") {
		if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
			return "", false
		}
		raw = raw[1 : len(raw)-1]
	}
	return raw, raw != ""
}

func isBlobTagWhereSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func parseListMaxResults(raw string) (int, bool) {
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, true
	}
	return value, false
}
