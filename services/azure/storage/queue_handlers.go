package storage

import (
	"bytes"
	"encoding/xml"
	"fmt"
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

const (
	maxQueueMessageBytes       = 64 * 1024
	legacyMaxQueueMessageBytes = 8 * 1024
	maxQueueVisibility         = 7 * 24 * 60 * 60
	defaultQueueTTL            = 7 * 24 * 60 * 60
	maxQueueCORSBytes          = 2 * 1024
	defaultQueueListMax        = 5000
)

var queueRequestIDCounter atomic.Uint64

func (s *StorageService) handleQueue(ctx *service.RequestContext, account string) (resp *service.Response, err error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-queue") {
		parts = parts[1:]
	}
	if account == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The queue request URI is invalid.")
	}
	if ctx.RawRequest.Method == http.MethodOptions {
		if ctx.RawRequest.Header.Get("Origin") == "" || ctx.RawRequest.Header.Get("Access-Control-Request-Method") == "" {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidHeader", "A CORS preflight request requires Origin and Access-Control-Request-Method headers.")
		}
		if resp, ok := s.queueCORSPreflightResponse(account, ctx.RawRequest.Header); ok {
			return resp, nil
		}
		return azurearm.ErrorResponse(http.StatusForbidden, "CorsNotAllowed", "No CORS rule matches the preflight request.")
	}
	defer func() {
		if resp != nil {
			applyQueueResponseHeaders(ctx.RawRequest.Header, resp)
			s.applyQueueCORSActualHeaders(account, ctx.RawRequest, resp)
		}
	}()
	if len(parts) == 0 {
		if isQueueServicePropertiesRequest(ctx.RawRequest.URL.Query()) {
			switch ctx.RawRequest.Method {
			case http.MethodGet:
				return s.getQueueServiceProperties(account, ctx.RawRequest.Header.Get("x-ms-version"))
			case http.MethodPut:
				return s.setQueueServiceProperties(account, ctx.Body, ctx.RawRequest.Header.Get("x-ms-version"))
			}
		}
		if ctx.RawRequest.Method == http.MethodGet && strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "list") {
			return s.listQueues(account, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header.Get("x-ms-version"))
		}
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The queue request URI is invalid.")
	}

	queueName := parts[0]
	if len(parts) == 1 {
		if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "acl") {
			switch ctx.RawRequest.Method {
			case http.MethodPut:
				return s.setQueueACL(account, queueName, ctx.Body)
			case http.MethodGet:
				return s.getQueueACL(account, queueName, false)
			case http.MethodHead:
				return s.getQueueACL(account, queueName, true)
			}
		}
		if strings.EqualFold(ctx.RawRequest.URL.Query().Get("comp"), "metadata") {
			switch ctx.RawRequest.Method {
			case http.MethodPut:
				return s.setQueueMetadata(account, queueName, ctx.RawRequest.Header)
			case http.MethodGet:
				return s.getQueueMetadata(account, queueName)
			case http.MethodHead:
				return s.getQueueMetadata(account, queueName)
			}
		}
		if ctx.RawRequest.Method == http.MethodPut {
			return s.createQueue(account, queueName, ctx.RawRequest.Header)
		}
		if ctx.RawRequest.Method == http.MethodDelete {
			return s.deleteQueue(account, queueName)
		}
	}
	if len(parts) == 2 && strings.EqualFold(parts[1], "messages") {
		switch ctx.RawRequest.Method {
		case http.MethodPost:
			return s.putMessage(account, queueName, ctx)
		case http.MethodGet:
			if strings.EqualFold(ctx.RawRequest.URL.Query().Get("peekonly"), "true") {
				return s.peekMessages(account, queueName, ctx.RawRequest.URL.Query())
			}
			return s.getMessages(account, queueName, ctx.RawRequest.URL.Query(), ctx.RawRequest.Header.Get("x-ms-version"))
		case http.MethodDelete:
			return s.clearMessages(account, queueName)
		}
	}
	if len(parts) == 3 && strings.EqualFold(parts[1], "messages") {
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.updateMessage(account, queueName, parts[2], ctx)
		case http.MethodDelete:
			return s.deleteMessage(account, queueName, parts[2], ctx.RawRequest.URL.Query().Get("popreceipt"))
		}
	}

	return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue route was not found.")
}

func isQueueServicePropertiesRequest(query url.Values) bool {
	return strings.EqualFold(query.Get("restype"), "service") && strings.EqualFold(query.Get("comp"), "properties")
}

func (s *StorageService) getQueueServiceProperties(account, apiVersion string) (*service.Response, error) {
	accountKey := strings.ToLower(account)

	s.mu.RLock()
	stored := append([]byte(nil), s.queueProps[accountKey]...)
	s.mu.RUnlock()

	if len(stored) == 0 {
		stored = defaultQueueServiceProperties()
	}
	if queueServicePropertiesUsesLegacyMetrics(apiVersion) {
		var err error
		stored, err = legacyQueueServiceProperties(stored)
		if err != nil {
			return nil, err
		}
	}
	return &service.Response{
		StatusCode:     http.StatusOK,
		Headers:        queueBaseHeadersForVersion(apiVersion),
		RawBody:        stored,
		RawContentType: "application/xml",
	}, nil
}

func (s *StorageService) setQueueServiceProperties(account string, body []byte, apiVersion string) (*service.Response, error) {
	properties, err := parseQueueServiceProperties(body, apiVersion)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", err.Error())
	}

	accountKey := strings.ToLower(account)
	s.mu.Lock()
	current := append([]byte(nil), s.queueProps[accountKey]...)
	if len(current) == 0 {
		current = defaultQueueServiceProperties()
	}
	merged, err := mergeQueueServiceProperties(current, properties)
	if err != nil {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", err.Error())
	}
	s.queueProps[accountKey] = merged
	s.mu.Unlock()

	return emptyResponse(http.StatusAccepted, queueBaseHeadersForVersion(apiVersion))
}

func (s *StorageService) queueCORSPreflightResponse(account string, header http.Header) (*service.Response, bool) {
	origin := header.Get("Origin")
	method := header.Get("Access-Control-Request-Method")
	if origin == "" || method == "" {
		return nil, false
	}

	accountKey := strings.ToLower(account)
	s.mu.RLock()
	stored := append([]byte(nil), s.queueProps[accountKey]...)
	s.mu.RUnlock()
	if len(stored) == 0 {
		return nil, false
	}
	var properties queueServicePropertiesDocument
	if err := xml.Unmarshal(stored, &properties); err != nil || properties.Cors == nil {
		return nil, false
	}

	requestedHeaders := queueCORSHeaderValues(header.Get("Access-Control-Request-Headers"))
	for _, rule := range properties.Cors.Rules {
		if !queueCORSOriginMatches(rule.AllowedOrigins, origin) ||
			!queueCORSMethodMatches(rule.AllowedMethods, method) ||
			!queueCORSHeadersMatch(rule.AllowedHeaders, requestedHeaders) {
			continue
		}
		headers := queueBaseHeaders()
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

func (s *StorageService) applyQueueCORSActualHeaders(account string, req *http.Request, resp *service.Response) {
	origin := req.Header.Get("Origin")
	if origin == "" {
		return
	}
	rule, wildcardOrigin, ok := s.queueCORSActualRule(account, origin, req.Method, req.Header)
	if !ok {
		if (req.Method == http.MethodGet || req.Method == http.MethodHead) && s.queueCORSRulesEnabled(account) {
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

func (s *StorageService) queueCORSActualRule(account, origin, method string, header http.Header) (queueServicePropertiesCorsRule, bool, bool) {
	accountKey := strings.ToLower(account)
	s.mu.RLock()
	stored := append([]byte(nil), s.queueProps[accountKey]...)
	s.mu.RUnlock()
	if len(stored) == 0 {
		return queueServicePropertiesCorsRule{}, false, false
	}
	var properties queueServicePropertiesDocument
	if err := xml.Unmarshal(stored, &properties); err != nil || properties.Cors == nil {
		return queueServicePropertiesCorsRule{}, false, false
	}
	requestedHeaders := queueCORSActualRequestHeaders(header)
	for _, rule := range properties.Cors.Rules {
		wildcardOrigin, originMatched := queueCORSOriginMatchType(rule.AllowedOrigins, origin)
		if originMatched && queueCORSMethodMatches(rule.AllowedMethods, method) && queueCORSHeadersMatch(rule.AllowedHeaders, requestedHeaders) {
			return rule, wildcardOrigin, true
		}
	}
	return queueServicePropertiesCorsRule{}, false, false
}

func queueCORSActualRequestHeaders(header http.Header) []string {
	names := make([]string, 0, len(header))
	for name := range header {
		if queueCORSActualRequestHeaderIgnored(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func queueCORSActualRequestHeaderIgnored(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "accept", "accept-language", "authorization", "content-language", "origin", "x-ms-version":
		return true
	default:
		return false
	}
}

func (s *StorageService) queueCORSRulesEnabled(account string) bool {
	accountKey := strings.ToLower(account)
	s.mu.RLock()
	stored := append([]byte(nil), s.queueProps[accountKey]...)
	s.mu.RUnlock()
	if len(stored) == 0 {
		return false
	}
	var properties queueServicePropertiesDocument
	if err := xml.Unmarshal(stored, &properties); err != nil || properties.Cors == nil {
		return false
	}
	return len(properties.Cors.Rules) > 0
}

func (s *StorageService) listQueues(account string, query url.Values, apiVersion string) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	prefix := query.Get("prefix")
	marker := strings.ToLower(query.Get("marker"))
	requestedMaxResults, invalidMaxResults := parseListMaxResults(query.Get("maxresults"))
	if invalidMaxResults {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The maxresults query parameter must be greater than 0.")
	}
	effectiveMaxResults := requestedMaxResults
	if query.Get("maxresults") == "" {
		effectiveMaxResults = defaultQueueListMax
	}
	includeMetadata := blobListIncludesMetadata(query.Get("include"))
	legacyShape := queueListQueuesUsesLegacyShape(apiVersion)

	s.mu.RLock()
	queues := s.queues[accountKey]
	names := make([]string, 0, len(queues))
	for key, q := range queues {
		if prefix != "" && !strings.HasPrefix(q.Name, prefix) {
			continue
		}
		if marker != "" && key < marker {
			continue
		}
		names = append(names, key)
	}
	sort.Strings(names)

	nextMarker := ""
	if effectiveMaxResults > 0 && len(names) > effectiveMaxResults {
		nextMarker = names[effectiveMaxResults]
		names = names[:effectiveMaxResults]
	}

	items := make([]queueListItem, 0, len(names))
	for _, key := range names {
		q := queues[key]
		item := queueListItem{Name: q.Name}
		if legacyShape {
			item.URL = "https://" + account + ".queue.core.windows.net/" + q.Name
		}
		if includeMetadata && len(q.Metadata) > 0 {
			metadata := queueListMetadata(q.Metadata)
			item.Metadata = &metadata
		}
		items = append(items, item)
	}
	s.mu.RUnlock()

	result := queueListResponse{
		Prefix:     prefix,
		Marker:     query.Get("marker"),
		MaxResults: requestedMaxResults,
		Queues:     items,
		NextMarker: nextMarker,
	}
	if legacyShape {
		result.AccountName = account
	} else {
		result.ServiceEndpoint = "https://" + account + ".queue.core.windows.net/"
	}
	resp, err := xmlResponse(http.StatusOK, result)
	if err != nil {
		return nil, err
	}
	resp.Headers = queueBaseHeaders()
	return resp, nil
}

func (s *StorageService) createQueue(account, name string, header http.Header) (*service.Response, error) {
	if !validQueueName(name) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "OutOfRangeInput", "The specified queue name is invalid.")
	}
	metadata, err := queueMetadataFromHeaders(header)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidMetadata", err.Error())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	queueKey := strings.ToLower(name)
	if s.queues[strings.ToLower(account)] == nil {
		s.queues[accountKey] = make(map[string]queue)
	}
	if existing, exists := s.queues[accountKey][queueKey]; exists {
		if !queueMetadataEqual(existing.Metadata, metadata) {
			return azurearm.ErrorResponse(http.StatusConflict, "QueueAlreadyExists", "The specified queue already exists with different metadata.")
		}
		return emptyResponse(http.StatusNoContent, queueBaseHeaders())
	}
	s.queues[accountKey][queueKey] = queue{
		Name:           name,
		Metadata:       metadata,
		CreatedVersion: queueCreationVersion(header.Get("x-ms-version")),
	}

	return emptyResponse(http.StatusCreated, queueBaseHeaders())
}

func (s *StorageService) deleteQueue(account, queueName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	queueKey := strings.ToLower(queueName)
	if _, ok := s.queues[accountKey][queueKey]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}
	delete(s.queues[accountKey], queueKey)
	return emptyResponse(http.StatusNoContent, queueBaseHeaders())
}

func (s *StorageService) setQueueMetadata(account, queueName string, header http.Header) (*service.Response, error) {
	metadata, err := queueMetadataFromHeaders(header)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidMetadata", err.Error())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	q, ok := s.queues[strings.ToLower(account)][strings.ToLower(queueName)]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}
	q.Metadata = metadata
	s.queues[strings.ToLower(account)][strings.ToLower(queueName)] = q
	return emptyResponse(http.StatusNoContent, queueBaseHeaders())
}

func (s *StorageService) getQueueMetadata(account, queueName string) (*service.Response, error) {
	s.mu.Lock()
	q, ok := s.queues[strings.ToLower(account)][strings.ToLower(queueName)]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}
	q.Messages = pruneExpiredQueueMessages(q.Messages, time.Now().UTC())
	s.queues[strings.ToLower(account)][strings.ToLower(queueName)] = q
	s.mu.Unlock()
	return emptyResponse(http.StatusOK, queueMetadataHeaders(q))
}

func (s *StorageService) setQueueACL(account, queueName string, body []byte) (*service.Response, error) {
	policies, err := parseQueueACL(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", err.Error())
	}
	if len(policies) > 5 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", "A queue can contain at most five stored access policies.")
	}
	for _, policy := range policies {
		if len(policy.ID) > 64 {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", "A stored access policy identifier cannot exceed 64 characters.")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	queueKey := strings.ToLower(queueName)
	q, ok := s.queues[accountKey][queueKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}
	q.AccessPolicies = cloneQueueAccessPolicies(policies)
	s.queues[accountKey][queueKey] = q
	return emptyResponse(http.StatusNoContent, queueBaseHeaders())
}

func (s *StorageService) getQueueACL(account, queueName string, headOnly bool) (*service.Response, error) {
	s.mu.RLock()
	q, ok := s.queues[strings.ToLower(account)][strings.ToLower(queueName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}

	if headOnly {
		resp, err := emptyResponse(http.StatusOK, queueBaseHeaders())
		if err != nil {
			return nil, err
		}
		resp.RawContentType = "application/xml"
		return resp, nil
	}
	resp, err := xmlResponse(http.StatusOK, queueACLResponse{SignedIdentifiers: cloneQueueAccessPolicies(q.AccessPolicies)})
	if err != nil {
		return nil, err
	}
	resp.Headers = queueBaseHeaders()
	return resp, nil
}

func (s *StorageService) putMessage(account, queueName string, ctx *service.RequestContext) (*service.Response, error) {
	messageText, resp, err := parseQueueMessageText(ctx.Body)
	if resp != nil || err != nil {
		return resp, err
	}
	apiVersion := ctx.RawRequest.Header.Get("x-ms-version")
	if len([]byte(messageText)) > queueMaxMessageBytes(apiVersion) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MessageTooLarge", "The message exceeds the maximum allowed size.")
	}

	now := time.Now().UTC()
	visibilityTimeout, resp, err := queuePutVisibilityTimeout(ctx.RawRequest.URL.Query(), apiVersion)
	if resp != nil || err != nil {
		return resp, err
	}
	ttl, resp, err := queueMessageTTL(ctx.RawRequest.URL.Query(), apiVersion)
	if resp != nil || err != nil {
		return resp, err
	}
	if ttl > 0 && visibilityTimeout >= ttl {
		return azurearm.ErrorResponse(http.StatusBadRequest, "OutOfRangeInput", "The visibilitytimeout query parameter must be smaller than messagettl.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	queueKey := strings.ToLower(queueName)
	q, ok := s.queues[accountKey][queueKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}
	q.Messages = pruneExpiredQueueMessages(q.Messages, now)

	msg := queueMessage{
		ID:              s.nextQueueMessageID(),
		Text:            messageText,
		PopReceipt:      s.nextToken("receipt"),
		InsertionTime:   now,
		ExpirationTime:  queueExpirationTime(now, ttl),
		TimeNextVisible: now.Add(time.Duration(visibilityTimeout) * time.Second),
	}
	q.Messages = append(q.Messages, msg)
	s.queues[accountKey][queueKey] = q

	if !queuePutMessageReturnsBody(ctx.RawRequest.Header.Get("x-ms-version")) {
		return emptyResponse(http.StatusCreated, queueBaseHeaders())
	}
	return xmlResponse(http.StatusCreated, queuePutMessagesResponse{Messages: []queuePutMessageXML{putMessageXML(msg)}})
}

func (s *StorageService) getMessages(account, queueName string, query url.Values, apiVersion string) (*service.Response, error) {
	now := time.Now().UTC()
	limit, invalidLimit := queueMessageLimit(query)
	if invalidLimit {
		return queueOutOfRangeQueryParameterResponse("numofmessages", query.Get("numofmessages"), "1", "32")
	}
	visibilityTimeout, resp, err := queueGetVisibilityTimeout(query, apiVersion)
	if resp != nil || err != nil {
		return resp, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	queueKey := strings.ToLower(queueName)
	q, ok := s.queues[accountKey][queueKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}
	q.Messages = pruneExpiredQueueMessages(q.Messages, now)

	out := make([]queueMessageXML, 0, limit)
	for i := range q.Messages {
		if len(out) == limit {
			break
		}
		if q.Messages[i].ExpirationTime.Before(now) || q.Messages[i].TimeNextVisible.After(now) {
			continue
		}
		q.Messages[i].PopReceipt = s.nextToken("receipt")
		q.Messages[i].DequeueCount++
		q.Messages[i].TimeNextVisible = now.Add(time.Duration(visibilityTimeout) * time.Second)
		out = append(out, messageXML(q.Messages[i], queueCreatedWithDequeueCount(q.CreatedVersion)))
	}
	s.queues[accountKey][queueKey] = q

	return xmlResponse(http.StatusOK, queueMessagesResponse{Messages: out})
}

func (s *StorageService) peekMessages(account, queueName string, query url.Values) (*service.Response, error) {
	now := time.Now().UTC()
	limit, invalidLimit := queueMessageLimit(query)
	if invalidLimit {
		return queueOutOfRangeQueryParameterResponse("numofmessages", query.Get("numofmessages"), "1", "32")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	queueKey := strings.ToLower(queueName)
	q, ok := s.queues[accountKey][queueKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}
	q.Messages = pruneExpiredQueueMessages(q.Messages, now)
	s.queues[accountKey][queueKey] = q

	out := make([]queuePeekMessageXML, 0, limit)
	for _, msg := range q.Messages {
		if len(out) == limit {
			break
		}
		if msg.ExpirationTime.Before(now) || msg.TimeNextVisible.After(now) {
			continue
		}
		out = append(out, peekMessageXML(msg, queueCreatedWithDequeueCount(q.CreatedVersion)))
	}

	return xmlResponse(http.StatusOK, queuePeekMessagesResponse{Messages: out})
}

func (s *StorageService) clearMessages(account, queueName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	queueKey := strings.ToLower(queueName)
	q, ok := s.queues[accountKey][queueKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}
	q.Messages = nil
	s.queues[accountKey][queueKey] = q
	return emptyResponse(http.StatusNoContent, queueBaseHeaders())
}

func (s *StorageService) updateMessage(account, queueName, messageID string, ctx *service.RequestContext) (*service.Response, error) {
	if queueAPIVersionBefore(ctx.RawRequest.Header.Get("x-ms-version"), "2011-08-18") {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "Update Message requires x-ms-version 2011-08-18 or later.")
	}
	popReceipt := strings.TrimSpace(ctx.RawRequest.URL.Query().Get("popreceipt"))
	if popReceipt == "" {
		return queueMissingRequiredQueryParameterResponse("popreceipt")
	}
	if ctx.RawRequest.URL.Query().Get("visibilitytimeout") == "" {
		return queueMissingRequiredQueryParameterResponse("visibilitytimeout")
	}
	visibilityTimeout, _, err := parseQueueInt(ctx.RawRequest.URL.Query(), "visibilitytimeout")
	if err != nil {
		return queueInvalidQueryParameterResponse("visibilitytimeout", ctx.RawRequest.URL.Query().Get("visibilitytimeout"), "invalid integer")
	}
	if visibilityTimeout < 0 || visibilityTimeout > 7*24*60*60 {
		return queueOutOfRangeQueryParameterResponse("visibilitytimeout", ctx.RawRequest.URL.Query().Get("visibilitytimeout"), "0", strconv.Itoa(maxQueueVisibility))
	}
	messageText, resp, err := parseQueueMessageText(ctx.Body)
	if resp != nil || err != nil {
		return resp, err
	}
	if len([]byte(messageText)) > maxQueueMessageBytes {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MessageTooLarge", "The message exceeds the maximum allowed size.")
	}

	now := time.Now().UTC()
	nextVisible := now.Add(time.Duration(visibilityTimeout) * time.Second)

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	queueKey := strings.ToLower(queueName)
	q, ok := s.queues[accountKey][queueKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}
	q.Messages = pruneExpiredQueueMessages(q.Messages, now)
	s.queues[accountKey][queueKey] = q
	for i, msg := range q.Messages {
		if msg.ID != messageID {
			continue
		}
		if msg.ExpirationTime.Before(now) {
			return azurearm.ErrorResponse(http.StatusNotFound, "MessageNotFound", "The specified message does not exist.")
		}
		if msg.PopReceipt != popReceipt {
			return queuePopReceiptMismatchResponse()
		}
		if nextVisible.After(msg.ExpirationTime) {
			return azurearm.ErrorResponse(http.StatusBadRequest, "OutOfRangeInput", "The visibilitytimeout query parameter must be smaller than the remaining message time-to-live.")
		}
		q.Messages[i].Text = messageText
		q.Messages[i].PopReceipt = s.nextToken("receipt")
		q.Messages[i].TimeNextVisible = nextVisible
		s.queues[accountKey][queueKey] = q

		headers := queueBaseHeaders()
		headers["x-ms-popreceipt"] = q.Messages[i].PopReceipt
		headers["x-ms-time-next-visible"] = nextVisible.Format(http.TimeFormat)
		return emptyResponse(http.StatusNoContent, headers)
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "MessageNotFound", "The specified message does not exist.")
}

func (s *StorageService) deleteMessage(account, queueName, messageID, popReceipt string) (*service.Response, error) {
	if strings.TrimSpace(popReceipt) == "" {
		return queueMissingRequiredQueryParameterResponse("popreceipt")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accountKey := strings.ToLower(account)
	queueKey := strings.ToLower(queueName)
	q, ok := s.queues[accountKey][queueKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "QueueNotFound", "The specified queue does not exist.")
	}
	q.Messages = pruneExpiredQueueMessages(q.Messages, time.Now().UTC())
	s.queues[accountKey][queueKey] = q
	for i, msg := range q.Messages {
		if msg.ID != messageID {
			continue
		}
		if msg.ExpirationTime.Before(time.Now().UTC()) {
			return azurearm.ErrorResponse(http.StatusNotFound, "MessageNotFound", "The specified message does not exist.")
		}
		if msg.PopReceipt != popReceipt {
			return azurearm.ErrorResponse(http.StatusNotFound, "MessageNotFound", "The specified message does not exist.")
		}
		q.Messages = append(q.Messages[:i], q.Messages[i+1:]...)
		s.queues[accountKey][queueKey] = q
		return emptyResponse(http.StatusNoContent, map[string]string{"x-ms-version": dataPlaneAPIVersion})
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "MessageNotFound", "The specified message does not exist.")
}

func parseSeconds(query url.Values, key string, fallback int) int {
	raw := query.Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func queuePutVisibilityTimeout(query url.Values, apiVersion string) (int, *service.Response, error) {
	visibilityTimeout, ok, err := parseQueueInt(query, "visibilitytimeout")
	if err != nil {
		resp, respErr := queueInvalidQueryParameterResponse("visibilitytimeout", query.Get("visibilitytimeout"), "invalid integer")
		return 0, resp, respErr
	}
	if !ok {
		return 0, nil, nil
	}
	if queueAPIVersionBefore(apiVersion, "2011-08-18") {
		resp, respErr := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidQueryParameterValue", "The visibilitytimeout query parameter requires x-ms-version 2011-08-18 or later.")
		return 0, resp, respErr
	}
	if visibilityTimeout < 0 || visibilityTimeout > maxQueueVisibility {
		resp, respErr := queueOutOfRangeQueryParameterResponse("visibilitytimeout", query.Get("visibilitytimeout"), "0", strconv.Itoa(maxQueueVisibility))
		return 0, resp, respErr
	}
	return visibilityTimeout, nil, nil
}

func queueGetVisibilityTimeout(query url.Values, apiVersion string) (int, *service.Response, error) {
	visibilityTimeout, ok, err := parseQueueInt(query, "visibilitytimeout")
	if err != nil {
		resp, respErr := queueInvalidQueryParameterResponse("visibilitytimeout", query.Get("visibilitytimeout"), "invalid integer")
		return 0, resp, respErr
	}
	if !ok {
		return 30, nil, nil
	}
	maxVisibility := queueGetVisibilityTimeoutMax(apiVersion)
	if visibilityTimeout < 1 || visibilityTimeout > maxVisibility {
		resp, respErr := queueOutOfRangeQueryParameterResponse("visibilitytimeout", query.Get("visibilitytimeout"), "1", strconv.Itoa(maxVisibility))
		return 0, resp, respErr
	}
	return visibilityTimeout, nil, nil
}

func queueGetVisibilityTimeoutMax(apiVersion string) int {
	if apiVersion = strings.TrimSpace(apiVersion); apiVersion != "" && apiVersion < "2011-08-18" {
		return 2 * 60 * 60
	}
	return maxQueueVisibility
}

func queueMessageTTL(query url.Values, apiVersion string) (int, *service.Response, error) {
	ttl, ok, err := parseQueueInt(query, "messagettl")
	if err != nil {
		resp, respErr := queueInvalidQueryParameterResponse("messagettl", query.Get("messagettl"), "invalid integer")
		return 0, resp, respErr
	}
	if !ok {
		return defaultQueueTTL, nil, nil
	}
	if queueAPIVersionBefore(apiVersion, "2017-07-29") && (ttl == -1 || ttl > defaultQueueTTL) {
		resp, respErr := azurearm.ErrorResponse(http.StatusBadRequest, "OutOfRangeInput", "The messagettl query parameter must be between 1 and 604800 seconds for Queue Storage versions before 2017-07-29.")
		return 0, resp, respErr
	}
	if ttl == -1 {
		return -1, nil, nil
	}
	if ttl <= 0 {
		resp, respErr := azurearm.ErrorResponse(http.StatusBadRequest, "OutOfRangeInput", "The messagettl query parameter must be positive or -1.")
		return 0, resp, respErr
	}
	return ttl, nil, nil
}

func parseQueueMessageText(body []byte) (string, *service.Response, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	var text strings.Builder
	depth := 0
	messageTextDepth := -1
	seenMessageText := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return queueMessageTextError("The request XML was invalid.")
		}
		switch typed := token.(type) {
		case xml.StartElement:
			switch {
			case depth == 0:
				if typed.Name.Local != "QueueMessage" {
					return queueMessageTextError("The request XML was invalid.")
				}
			case depth == 1 && typed.Name.Local == "MessageText" && !seenMessageText:
				seenMessageText = true
				messageTextDepth = depth + 1
			case messageTextDepth > 0:
				return queueMessageTextError("The message content must be XML-safe text. Encode markup before enqueueing it.")
			default:
				return queueMessageTextError("The request XML was invalid.")
			}
			depth++
		case xml.EndElement:
			if depth == 0 {
				return queueMessageTextError("The request XML was invalid.")
			}
			if depth == messageTextDepth {
				messageTextDepth = -1
			}
			depth--
		case xml.CharData:
			if messageTextDepth > 0 {
				text.Write([]byte(typed))
			} else if strings.TrimSpace(string(typed)) != "" {
				return queueMessageTextError("The request XML was invalid.")
			}
		}
	}
	if depth != 0 || !seenMessageText {
		return queueMessageTextError("The request XML was invalid.")
	}
	return text.String(), nil, nil
}

func queueMessageTextError(message string) (string, *service.Response, error) {
	resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidXmlDocument", message)
	return "", resp, err
}

func (s *StorageService) nextQueueMessageID() string {
	s.nextID++
	return fmt.Sprintf("00000000-0000-4000-8000-%012x", s.nextID)
}

func queueAPIVersionBefore(apiVersion, boundary string) bool {
	apiVersion = strings.TrimSpace(apiVersion)
	return apiVersion != "" && apiVersion < boundary
}

func queueMaxMessageBytes(apiVersion string) int {
	if queueAPIVersionBefore(apiVersion, "2011-08-18") {
		return legacyMaxQueueMessageBytes
	}
	return maxQueueMessageBytes
}

func queuePopReceiptMismatchResponse() (*service.Response, error) {
	return azurearm.ErrorResponse(http.StatusBadRequest, "PopReceiptMismatch", "The specified pop receipt did not match the pop receipt for a dequeued message.")
}

func parseQueueInt(query url.Values, key string) (int, bool, error) {
	raw := query.Get(key)
	if raw == "" {
		return 0, false, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, true, err
	}
	return value, true, nil
}

func queueExpirationTime(now time.Time, ttl int) time.Time {
	if ttl == -1 {
		return time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}
	return now.Add(time.Duration(ttl) * time.Second)
}

func pruneExpiredQueueMessages(messages []queueMessage, now time.Time) []queueMessage {
	write := 0
	for _, msg := range messages {
		if !msg.ExpirationTime.After(now) {
			continue
		}
		messages[write] = msg
		write++
	}
	return messages[:write]
}

func parseQueueACL(body []byte) ([]queueSignedIdentifier, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, nil
	}
	var acl queueACLResponse
	if err := xml.Unmarshal(body, &acl); err != nil {
		return nil, err
	}
	return cloneQueueAccessPolicies(acl.SignedIdentifiers), nil
}

func cloneQueueAccessPolicies(in []queueSignedIdentifier) []queueSignedIdentifier {
	if len(in) == 0 {
		return nil
	}
	out := make([]queueSignedIdentifier, len(in))
	copy(out, in)
	return out
}

func parseQueueServiceProperties(body []byte, apiVersion string) ([]byte, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, &xml.SyntaxError{Msg: "missing StorageServiceProperties"}
	}
	var properties queueServicePropertiesDocument
	if err := xml.Unmarshal([]byte(trimmed), &properties); err != nil {
		return nil, err
	}
	if properties.XMLName.Local != "StorageServiceProperties" {
		return nil, fmt.Errorf("expected StorageServiceProperties root element")
	}
	if err := validateQueueServicePropertiesVersion(properties, apiVersion); err != nil {
		return nil, err
	}
	if err := validateQueueServiceAnalyticsProperties(properties); err != nil {
		return nil, err
	}
	if properties.Cors != nil {
		if len(properties.Cors.Rules) > 5 {
			return nil, fmt.Errorf("a queue service can contain at most five CORS rules")
		}
		if queueServiceCORSSettingsSize(properties.Cors.Rules) > maxQueueCORSBytes {
			return nil, fmt.Errorf("CORS settings exceed documented size limit")
		}
		for _, rule := range properties.Cors.Rules {
			if strings.TrimSpace(rule.AllowedOrigins) == "" ||
				strings.TrimSpace(rule.AllowedMethods) == "" ||
				strings.TrimSpace(rule.MaxAgeInSeconds) == "" ||
				strings.TrimSpace(rule.ExposedHeaders) == "" ||
				strings.TrimSpace(rule.AllowedHeaders) == "" {
				return nil, fmt.Errorf("all CORS rule elements are required")
			}
			if !queueServiceCORSMaxAgeValid(rule.MaxAgeInSeconds) {
				return nil, fmt.Errorf("CORS MaxAgeInSeconds must be a non-negative integer")
			}
			if !queueServiceCORSListWithinLimits(rule.AllowedOrigins, 64, 256) {
				return nil, fmt.Errorf("CORS AllowedOrigins exceeds documented limits")
			}
			if !queueServiceCORSHeaderListWithinLimits(rule.AllowedHeaders) {
				return nil, fmt.Errorf("CORS AllowedHeaders exceeds documented limits")
			}
			if !queueServiceCORSHeaderListWithinLimits(rule.ExposedHeaders) {
				return nil, fmt.Errorf("CORS ExposedHeaders exceeds documented limits")
			}
			for _, method := range strings.Split(rule.AllowedMethods, ",") {
				if !queueServiceCORSMethodAllowed(strings.TrimSpace(method)) {
					return nil, fmt.Errorf("CORS AllowedMethods contains an unsupported method")
				}
			}
		}
	}
	if queueServicePropertiesUsesLegacyMetrics(apiVersion) {
		return modernQueueServicePropertiesFromLegacy([]byte(trimmed))
	}
	return []byte(trimmed), nil
}

func validateQueueServicePropertiesVersion(properties queueServicePropertiesDocument, apiVersion string) error {
	if !queueServicePropertiesUsesLegacyMetrics(apiVersion) {
		if properties.Metrics != nil {
			return fmt.Errorf("modern Queue service properties do not support the legacy Metrics root element")
		}
		return nil
	}
	if properties.Logging == nil || properties.Metrics == nil {
		return fmt.Errorf("legacy Queue service properties require Logging and Metrics root elements")
	}
	if properties.HourMetrics != nil || properties.MinuteMetrics != nil || properties.Cors != nil {
		return fmt.Errorf("legacy Queue service properties do not support HourMetrics, MinuteMetrics, or Cors root elements")
	}
	return nil
}

func validateQueueServiceAnalyticsProperties(properties queueServicePropertiesDocument) error {
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
	return nil
}

func validateQueueServiceLoggingProperties(logging *queueServicePropertiesLogging) error {
	if strings.TrimSpace(logging.Version) == "" ||
		strings.TrimSpace(logging.Delete) == "" ||
		strings.TrimSpace(logging.Read) == "" ||
		strings.TrimSpace(logging.Write) == "" {
		return fmt.Errorf("Logging settings require Version, Delete, Read, and Write elements")
	}
	for _, value := range []string{logging.Delete, logging.Read, logging.Write} {
		if _, ok := queueServiceBool(value); !ok {
			return fmt.Errorf("Logging boolean settings must be true or false")
		}
	}
	return validateQueueServiceRetentionPolicy("Logging", logging.RetentionPolicy)
}

func validateQueueServiceMetricsProperties(name string, metrics *queueServicePropertiesMetrics) error {
	if strings.TrimSpace(metrics.Version) == "" || strings.TrimSpace(metrics.Enabled) == "" {
		return fmt.Errorf("%s settings require Version and Enabled elements", name)
	}
	enabled, ok := queueServiceBool(metrics.Enabled)
	if !ok {
		return fmt.Errorf("%s Enabled must be true or false", name)
	}
	if strings.TrimSpace(metrics.IncludeAPIs) != "" {
		if _, ok := queueServiceBool(metrics.IncludeAPIs); !ok {
			return fmt.Errorf("%s IncludeAPIs must be true or false", name)
		}
	}
	if enabled && strings.TrimSpace(metrics.IncludeAPIs) == "" {
		return fmt.Errorf("%s settings require IncludeAPIs when metrics are enabled", name)
	}
	return validateQueueServiceRetentionPolicy(name, metrics.RetentionPolicy)
}

func validateQueueServiceRetentionPolicy(name string, policy *queueServicePropertiesRetentionPolicy) error {
	if policy == nil || strings.TrimSpace(policy.Enabled) == "" {
		return fmt.Errorf("%s settings require RetentionPolicy/Enabled", name)
	}
	enabled, ok := queueServiceBool(policy.Enabled)
	if !ok {
		return fmt.Errorf("%s RetentionPolicy/Enabled must be true or false", name)
	}
	daysText := strings.TrimSpace(policy.Days)
	if !enabled && daysText == "" {
		return nil
	}
	days, err := strconv.Atoi(daysText)
	if enabled && daysText == "" {
		return fmt.Errorf("%s RetentionPolicy/Days is required when retention is enabled", name)
	}
	if err != nil || days < 1 || days > 365 {
		return fmt.Errorf("%s RetentionPolicy/Days must be between 1 and 365", name)
	}
	return nil
}

func queueServiceBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func queueServicePropertiesUsesLegacyMetrics(apiVersion string) bool {
	apiVersion = strings.TrimSpace(apiVersion)
	return apiVersion != "" && apiVersion <= "2012-02-12"
}

func queuePutMessageReturnsBody(apiVersion string) bool {
	apiVersion = strings.TrimSpace(apiVersion)
	return apiVersion == "" || apiVersion >= "2016-05-31"
}

func queueListQueuesUsesLegacyShape(apiVersion string) bool {
	apiVersion = strings.TrimSpace(apiVersion)
	return apiVersion != "" && apiVersion < "2013-08-15"
}

func queueCreationVersion(apiVersion string) string {
	if apiVersion = strings.TrimSpace(apiVersion); apiVersion != "" {
		return apiVersion
	}
	return dataPlaneAPIVersion
}

func queueCreatedWithDequeueCount(apiVersion string) bool {
	return queueCreationVersion(apiVersion) >= "2009-09-19"
}

func legacyQueueServiceProperties(body []byte) ([]byte, error) {
	elements, err := queueServicePropertyRootElements(body)
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
	out.WriteString("</StorageServiceProperties>")
	return []byte(out.String()), nil
}

func modernQueueServicePropertiesFromLegacy(body []byte) ([]byte, error) {
	elements, err := queueServicePropertyRootElements(body)
	if err != nil {
		return nil, err
	}
	logging := elements["Logging"]
	metrics := elements["Metrics"]
	if logging == "" || metrics == "" {
		return nil, fmt.Errorf("legacy Queue service properties require Logging and Metrics root elements")
	}

	var out strings.Builder
	out.WriteString("<StorageServiceProperties>")
	out.WriteString(logging)
	out.WriteString("<HourMetrics>")
	out.WriteString(queueServiceRootInnerXML(metrics))
	out.WriteString("</HourMetrics>")
	out.WriteString(defaultQueueMinuteMetrics())
	out.WriteString("<Cors></Cors>")
	out.WriteString("</StorageServiceProperties>")
	return []byte(out.String()), nil
}

func defaultQueueMinuteMetrics() string {
	elements, err := queueServicePropertyRootElements(defaultQueueServiceProperties())
	if err != nil {
		return `<MinuteMetrics><Version>1.0</Version><Enabled>false</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></MinuteMetrics>`
	}
	if minuteMetrics := elements["MinuteMetrics"]; minuteMetrics != "" {
		return minuteMetrics
	}
	return `<MinuteMetrics><Version>1.0</Version><Enabled>false</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></MinuteMetrics>`
}

func queueServiceRootInnerXML(element string) string {
	start := strings.Index(element, ">")
	end := strings.LastIndex(element, "</")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return element[start+1 : end]
}

func queueServiceCORSMaxAgeValid(value string) bool {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	return err == nil && seconds >= 0
}

func mergeQueueServiceProperties(current, update []byte) ([]byte, error) {
	currentElements, err := queueServicePropertyRootElements(current)
	if err != nil {
		return nil, err
	}
	updateElements, err := queueServicePropertyRootElements(update)
	if err != nil {
		return nil, err
	}
	for name, element := range updateElements {
		currentElements[name] = element
	}

	var out strings.Builder
	out.WriteString("<StorageServiceProperties>")
	for _, name := range queueServicePropertyRootOrder {
		if element, ok := currentElements[name]; ok {
			out.WriteString(element)
		}
	}
	out.WriteString("</StorageServiceProperties>")
	return []byte(out.String()), nil
}

func queueServicePropertyRootElements(body []byte) (map[string]string, error) {
	var properties queueServicePropertiesRawDocument
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
	return elements, nil
}

func queueServiceCORSListWithinLimits(value string, maxEntries, maxLength int) bool {
	entries := strings.Split(value, ",")
	if len(entries) > maxEntries {
		return false
	}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" || len(entry) > maxLength {
			return false
		}
	}
	return true
}

func queueServiceCORSSettingsSize(rules []queueServicePropertiesCorsRule) int {
	size := 0
	for _, rule := range rules {
		size += len(strings.TrimSpace(rule.AllowedOrigins))
		size += len(strings.TrimSpace(rule.AllowedMethods))
		size += len(strings.TrimSpace(rule.MaxAgeInSeconds))
		size += len(strings.TrimSpace(rule.ExposedHeaders))
		size += len(strings.TrimSpace(rule.AllowedHeaders))
	}
	return size
}

func queueCORSOriginMatches(allowedOrigins, origin string) bool {
	_, matched := queueCORSOriginMatchType(allowedOrigins, origin)
	return matched
}

func queueCORSOriginMatchType(allowedOrigins, origin string) (bool, bool) {
	for _, allowed := range strings.Split(allowedOrigins, ",") {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" {
			return true, true
		}
		if allowed == origin {
			return false, true
		}
		if queueCORSWildcardOriginMatches(allowed, origin) {
			return false, true
		}
	}
	return false, false
}

func queueCORSWildcardOriginMatches(allowed, origin string) bool {
	if !strings.Contains(allowed, "*") || allowed == "*" {
		return false
	}
	prefix, suffix, ok := strings.Cut(allowed, "*")
	if !ok || strings.Contains(suffix, "*") {
		return false
	}
	if !strings.HasPrefix(origin, prefix) || !strings.HasSuffix(origin, suffix) {
		return false
	}
	return len(strings.TrimSuffix(strings.TrimPrefix(origin, prefix), suffix)) > 0
}

func queueCORSMethodMatches(allowedMethods, method string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	for _, allowed := range strings.Split(allowedMethods, ",") {
		if strings.ToUpper(strings.TrimSpace(allowed)) == method {
			return true
		}
	}
	return false
}

func queueCORSHeadersMatch(allowedHeaders string, requestedHeaders []string) bool {
	if len(requestedHeaders) == 0 {
		return true
	}
	allowed := queueCORSHeaderValues(allowedHeaders)
	for _, requested := range requestedHeaders {
		if !queueCORSHeaderAllowed(allowed, requested) {
			return false
		}
	}
	return true
}

func queueCORSHeaderAllowed(allowed []string, requested string) bool {
	requested = strings.ToLower(strings.TrimSpace(requested))
	for _, candidate := range allowed {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == "*" || candidate == requested {
			return true
		}
		if strings.HasSuffix(candidate, "*") && strings.HasPrefix(requested, strings.TrimSuffix(candidate, "*")) {
			return true
		}
	}
	return false
}

func queueCORSHeaderValues(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func queueServiceCORSHeaderListWithinLimits(value string) bool {
	entries := strings.Split(value, ",")
	literals := 0
	prefixed := 0
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" || len(entry) > 256 {
			return false
		}
		if strings.HasSuffix(entry, "*") {
			prefixed++
			continue
		}
		literals++
	}
	return literals <= 64 && prefixed <= 2
}

func queueServiceCORSMethodAllowed(method string) bool {
	switch strings.ToUpper(method) {
	case "DELETE", "GET", "HEAD", "MERGE", "POST", "OPTIONS", "PUT":
		return true
	default:
		return false
	}
}

func defaultQueueServiceProperties() []byte {
	return []byte(`<StorageServiceProperties><Logging><Version>1.0</Version><Delete>false</Delete><Read>false</Read><Write>false</Write><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></Logging><HourMetrics><Version>1.0</Version><Enabled>false</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></HourMetrics><MinuteMetrics><Version>1.0</Version><Enabled>false</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></MinuteMetrics><Cors></Cors></StorageServiceProperties>`)
}

var queueServicePropertyRootOrder = []string{"Logging", "Metrics", "HourMetrics", "MinuteMetrics", "Cors"}

type queueServicePropertiesRawDocument struct {
	XMLName       xml.Name                          `xml:"StorageServiceProperties"`
	Logging       *queueServicePropertiesRawElement `xml:"Logging"`
	Metrics       *queueServicePropertiesRawElement `xml:"Metrics"`
	HourMetrics   *queueServicePropertiesRawElement `xml:"HourMetrics"`
	MinuteMetrics *queueServicePropertiesRawElement `xml:"MinuteMetrics"`
	Cors          *queueServicePropertiesRawElement `xml:"Cors"`
}

type queueServicePropertiesRawElement struct {
	InnerXML string `xml:",innerxml"`
}

type queueServicePropertiesDocument struct {
	XMLName       xml.Name                       `xml:"StorageServiceProperties"`
	Logging       *queueServicePropertiesLogging `xml:"Logging"`
	Metrics       *queueServicePropertiesMetrics `xml:"Metrics"`
	HourMetrics   *queueServicePropertiesMetrics `xml:"HourMetrics"`
	MinuteMetrics *queueServicePropertiesMetrics `xml:"MinuteMetrics"`
	Cors          *queueServicePropertiesCors    `xml:"Cors"`
}

type queueServicePropertiesLogging struct {
	Version         string                                 `xml:"Version"`
	Delete          string                                 `xml:"Delete"`
	Read            string                                 `xml:"Read"`
	Write           string                                 `xml:"Write"`
	RetentionPolicy *queueServicePropertiesRetentionPolicy `xml:"RetentionPolicy"`
}

type queueServicePropertiesMetrics struct {
	Version         string                                 `xml:"Version"`
	Enabled         string                                 `xml:"Enabled"`
	IncludeAPIs     string                                 `xml:"IncludeAPIs"`
	RetentionPolicy *queueServicePropertiesRetentionPolicy `xml:"RetentionPolicy"`
}

type queueServicePropertiesRetentionPolicy struct {
	Enabled string `xml:"Enabled"`
	Days    string `xml:"Days"`
}

type queueServicePropertiesCors struct {
	Rules []queueServicePropertiesCorsRule `xml:"CorsRule"`
}

type queueServicePropertiesCorsRule struct {
	AllowedOrigins  string `xml:"AllowedOrigins"`
	AllowedMethods  string `xml:"AllowedMethods"`
	MaxAgeInSeconds string `xml:"MaxAgeInSeconds"`
	ExposedHeaders  string `xml:"ExposedHeaders"`
	AllowedHeaders  string `xml:"AllowedHeaders"`
}

func queueMessageLimit(query url.Values) (int, bool) {
	raw := query.Get("numofmessages")
	if raw == "" {
		return 1, false
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 32 {
		return 0, true
	}
	return value, false
}

func queueOutOfRangeQueryParameterResponse(parameter, value, minimum, maximum string) (*service.Response, error) {
	return queueStorageErrorResponse(http.StatusBadRequest, queueStorageErrorXML{
		Code:                "OutOfRangeQueryParameterValue",
		Message:             "One of the query parameters specified in the request URI is outside the permissible range.",
		QueryParameterName:  parameter,
		QueryParameterValue: value,
		MinimumAllowed:      minimum,
		MaximumAllowed:      maximum,
	})
}

func queueMissingRequiredQueryParameterResponse(parameter string) (*service.Response, error) {
	return queueStorageErrorResponse(http.StatusBadRequest, queueStorageErrorXML{
		Code:               "MissingRequiredQueryParameter",
		Message:            "A required query parameter was not specified for this request.",
		QueryParameterName: parameter,
	})
}

func queueInvalidQueryParameterResponse(parameter, value, reason string) (*service.Response, error) {
	return queueStorageErrorResponse(http.StatusBadRequest, queueStorageErrorXML{
		Code:                "InvalidQueryParameterValue",
		Message:             "Value for one of the query parameters specified in the request URI is invalid.",
		QueryParameterName:  parameter,
		QueryParameterValue: value,
		Reason:              reason,
	})
}

func queueStorageErrorResponse(statusCode int, storageErr queueStorageErrorXML) (*service.Response, error) {
	body, err := xml.Marshal(storageErr)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     statusCode,
		Headers:        map[string]string{"x-ms-error-code": storageErr.Code},
		RawBody:        append([]byte(xml.Header), body...),
		RawContentType: "application/xml",
	}, nil
}

type queueStorageErrorXML struct {
	XMLName             xml.Name `xml:"Error"`
	Code                string   `xml:"Code"`
	Message             string   `xml:"Message"`
	QueryParameterName  string   `xml:"QueryParameterName,omitempty"`
	QueryParameterValue string   `xml:"QueryParameterValue,omitempty"`
	MinimumAllowed      string   `xml:"MinimumAllowed,omitempty"`
	MaximumAllowed      string   `xml:"MaximumAllowed,omitempty"`
	Reason              string   `xml:"Reason,omitempty"`
}

type queueMessagesResponse struct {
	XMLName  xml.Name          `xml:"QueueMessagesList"`
	Messages []queueMessageXML `xml:"QueueMessage"`
}

type queuePutMessagesResponse struct {
	XMLName  xml.Name             `xml:"QueueMessagesList"`
	Messages []queuePutMessageXML `xml:"QueueMessage"`
}

type queuePeekMessagesResponse struct {
	XMLName  xml.Name              `xml:"QueueMessagesList"`
	Messages []queuePeekMessageXML `xml:"QueueMessage"`
}

type queueListResponse struct {
	XMLName         xml.Name        `xml:"EnumerationResults"`
	ServiceEndpoint string          `xml:"ServiceEndpoint,attr,omitempty"`
	AccountName     string          `xml:"AccountName,attr,omitempty"`
	Prefix          string          `xml:"Prefix,omitempty"`
	Marker          string          `xml:"Marker,omitempty"`
	MaxResults      int             `xml:"MaxResults,omitempty"`
	Queues          []queueListItem `xml:"Queues>Queue"`
	NextMarker      string          `xml:"NextMarker"`
}

type queueListItem struct {
	Name     string             `xml:"Name"`
	URL      string             `xml:"Url,omitempty"`
	Metadata *queueListMetadata `xml:"Metadata,omitempty"`
}

type queueListMetadata map[string]string

func (metadata queueListMetadata) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
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
		name := key
		value := metadata[key]
		if !validQueueMetadataName(key) {
			name = "x-ms-invalid-name"
			value = key
		}
		if err := e.EncodeElement(value, xml.StartElement{Name: xml.Name{Local: name}}); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
}

type queueSignedIdentifier struct {
	ID           string            `xml:"Id"`
	AccessPolicy queueAccessPolicy `xml:"AccessPolicy"`
}

type queueAccessPolicy struct {
	Start      string `xml:"Start,omitempty"`
	Expiry     string `xml:"Expiry,omitempty"`
	Permission string `xml:"Permission,omitempty"`
}

type queueACLResponse struct {
	XMLName           xml.Name                `xml:"SignedIdentifiers"`
	SignedIdentifiers []queueSignedIdentifier `xml:"SignedIdentifier"`
}

type queueMessageXML struct {
	MessageID       string `xml:"MessageId"`
	InsertionTime   string `xml:"InsertionTime"`
	ExpirationTime  string `xml:"ExpirationTime"`
	PopReceipt      string `xml:"PopReceipt"`
	TimeNextVisible string `xml:"TimeNextVisible"`
	DequeueCount    *int   `xml:"DequeueCount,omitempty"`
	MessageText     string `xml:"MessageText"`
}

type queuePutMessageXML struct {
	MessageID       string `xml:"MessageId"`
	InsertionTime   string `xml:"InsertionTime"`
	ExpirationTime  string `xml:"ExpirationTime"`
	PopReceipt      string `xml:"PopReceipt"`
	TimeNextVisible string `xml:"TimeNextVisible"`
}

type queuePeekMessageXML struct {
	MessageID      string `xml:"MessageId"`
	InsertionTime  string `xml:"InsertionTime"`
	ExpirationTime string `xml:"ExpirationTime"`
	DequeueCount   *int   `xml:"DequeueCount,omitempty"`
	MessageText    string `xml:"MessageText"`
}

func messageXML(msg queueMessage, includeDequeueCount bool) queueMessageXML {
	out := queueMessageXML{
		MessageID:       msg.ID,
		InsertionTime:   msg.InsertionTime.Format(http.TimeFormat),
		ExpirationTime:  msg.ExpirationTime.Format(http.TimeFormat),
		PopReceipt:      msg.PopReceipt,
		TimeNextVisible: msg.TimeNextVisible.Format(http.TimeFormat),
		MessageText:     msg.Text,
	}
	if includeDequeueCount {
		out.DequeueCount = &msg.DequeueCount
	}
	return out
}

func putMessageXML(msg queueMessage) queuePutMessageXML {
	return queuePutMessageXML{
		MessageID:       msg.ID,
		InsertionTime:   msg.InsertionTime.Format(http.TimeFormat),
		ExpirationTime:  msg.ExpirationTime.Format(http.TimeFormat),
		PopReceipt:      msg.PopReceipt,
		TimeNextVisible: msg.TimeNextVisible.Format(http.TimeFormat),
	}
}

func peekMessageXML(msg queueMessage, includeDequeueCount bool) queuePeekMessageXML {
	out := queuePeekMessageXML{
		MessageID:      msg.ID,
		InsertionTime:  msg.InsertionTime.Format(http.TimeFormat),
		ExpirationTime: msg.ExpirationTime.Format(http.TimeFormat),
		MessageText:    msg.Text,
	}
	if includeDequeueCount {
		out.DequeueCount = &msg.DequeueCount
	}
	return out
}

func queueBaseHeaders() map[string]string {
	return map[string]string{"x-ms-version": dataPlaneAPIVersion}
}

func queueBaseHeadersForVersion(apiVersion string) map[string]string {
	headers := queueBaseHeaders()
	if requested := strings.TrimSpace(apiVersion); requested != "" {
		headers["x-ms-version"] = requested
	}
	return headers
}

func applyQueueResponseHeaders(header http.Header, resp *service.Response) {
	requested := strings.TrimSpace(header.Get("x-ms-version"))
	if resp.Headers == nil {
		resp.Headers = make(map[string]string)
	}
	resp.Headers["x-ms-request-id"] = nextQueueRequestID()
	if requested != "" {
		if queueVersionReturnsVersionHeader(requested) {
			resp.Headers["x-ms-version"] = requested
		} else {
			delete(resp.Headers, "x-ms-version")
		}
	}
	if clientRequestID := queueClientRequestID(header.Get("x-ms-client-request-id")); clientRequestID != "" {
		resp.Headers["x-ms-client-request-id"] = clientRequestID
	}
}

func nextQueueRequestID() string {
	return fmt.Sprintf("cloudmock-queue-%016x", queueRequestIDCounter.Add(1))
}

func queueVersionReturnsVersionHeader(apiVersion string) bool {
	return strings.TrimSpace(apiVersion) >= "2009-09-19"
}

func queueClientRequestID(value string) string {
	if value == "" || len(value) > 1024 {
		return ""
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] > 0x7e {
			return ""
		}
	}
	return value
}

func queueMetadataHeaders(q queue) map[string]string {
	headers := queueBaseHeaders()
	headers["x-ms-approximate-messages-count"] = strconv.Itoa(len(q.Messages))
	for key, value := range q.Metadata {
		headers["x-ms-meta-"+key] = value
	}
	return headers
}

func queueMetadataEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if right[key] != leftValue {
			return false
		}
	}
	return true
}

func validQueueName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	if !queueNameAlphanumeric(name[0]) || !queueNameAlphanumeric(name[len(name)-1]) {
		return false
	}
	previousDash := false
	for i := 0; i < len(name); i++ {
		ch := name[i]
		switch {
		case ch >= 'a' && ch <= 'z':
			previousDash = false
		case ch >= '0' && ch <= '9':
			previousDash = false
		case ch == '-':
			if previousDash {
				return false
			}
			previousDash = true
		default:
			return false
		}
	}
	return true
}

func queueMetadataFromHeaders(header http.Header) (map[string]string, error) {
	metadata := make(map[string]string)
	seen := make(map[string]struct{})
	enforceIdentifierNames := queueMetadataNamesRequireCSharpIdentifiers(header.Get("x-ms-version"))

	for key, values := range header {
		lowerKey := strings.ToLower(key)
		if !strings.HasPrefix(lowerKey, "x-ms-meta-") || len(values) == 0 {
			continue
		}

		name := strings.TrimPrefix(lowerKey, "x-ms-meta-")
		if _, ok := seen[name]; ok || len(values) > 1 {
			return nil, fmt.Errorf("duplicate metadata name %q", name)
		}
		seen[name] = struct{}{}

		if enforceIdentifierNames && !validQueueMetadataName(name) {
			return nil, fmt.Errorf("invalid metadata name %q", name)
		}
		metadata[name] = values[0]
	}

	if len(metadata) == 0 {
		return nil, nil
	}
	return metadata, nil
}

func queueMetadataNamesRequireCSharpIdentifiers(apiVersion string) bool {
	apiVersion = strings.TrimSpace(apiVersion)
	return apiVersion == "" || apiVersion >= "2009-09-19"
}

func validQueueMetadataName(name string) bool {
	if name == "" {
		return false
	}
	if !isQueueMetadataFirstIdentifierByte(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !isQueueMetadataIdentifierByte(name[i]) {
			return false
		}
	}
	return true
}

func isQueueMetadataFirstIdentifierByte(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isQueueMetadataIdentifierByte(ch byte) bool {
	return isQueueMetadataFirstIdentifierByte(ch) || (ch >= '0' && ch <= '9')
}

func queueNameAlphanumeric(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}
