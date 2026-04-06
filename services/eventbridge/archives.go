package eventbridge

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Archive represents an EventBridge archive.
type Archive struct {
	ArchiveName    string
	ArchiveArn     string
	EventSourceArn string
	Description    string
	EventPattern   string
	RetentionDays  int
	State          string
	CreationTime   time.Time
}

// Replay represents an EventBridge replay.
type Replay struct {
	ReplayName      string
	ReplayArn       string
	EventSourceArn  string
	State           string
	EventStartTime  float64
	EventEndTime    float64
	ReplayStartTime time.Time
	ReplayEndTime   time.Time
}

var (
	archivesMu sync.RWMutex
	archives   = make(map[string]map[string]*Archive) // keyed by accountID:region -> archiveName

	replaysMu sync.RWMutex
	replays    = make(map[string]map[string]*Replay) // keyed by accountID:region -> replayName
)

func archiveStoreKey(store *Store) string {
	return store.accountID + ":" + store.region
}

// ---- CreateArchive ----

type createArchiveRequest struct {
	ArchiveName    string `json:"ArchiveName"`
	EventSourceArn string `json:"EventSourceArn"`
	Description    string `json:"Description"`
	EventPattern   string `json:"EventPattern"`
	RetentionDays  int    `json:"RetentionDays"`
}

func handleCreateArchive(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createArchiveRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ArchiveName == "" {
		return jsonErr(service.ErrValidation("ArchiveName is required."))
	}
	if req.EventSourceArn == "" {
		return jsonErr(service.ErrValidation("EventSourceArn is required."))
	}

	key := archiveStoreKey(store)
	now := time.Now().UTC()
	arn := fmt.Sprintf("arn:aws:events:%s:%s:archive/%s", store.region, store.accountID, req.ArchiveName)

	archivesMu.Lock()
	defer archivesMu.Unlock()

	if archives[key] == nil {
		archives[key] = make(map[string]*Archive)
	}
	if _, exists := archives[key][req.ArchiveName]; exists {
		return jsonErr(service.NewAWSError("ResourceAlreadyExistsException",
			"Archive "+req.ArchiveName+" already exists.", http.StatusConflict))
	}

	archive := &Archive{
		ArchiveName:    req.ArchiveName,
		ArchiveArn:     arn,
		EventSourceArn: req.EventSourceArn,
		Description:    req.Description,
		EventPattern:   req.EventPattern,
		RetentionDays:  req.RetentionDays,
		State:          "ENABLED",
		CreationTime:   now,
	}
	archives[key][req.ArchiveName] = archive

	return jsonOK(map[string]any{
		"ArchiveArn":   arn,
		"ArchiveName":  req.ArchiveName,
		"State":        "ENABLED",
		"CreationTime": now.Unix(),
	})
}

// ---- DescribeArchive ----

type describeArchiveRequest struct {
	ArchiveName string `json:"ArchiveName"`
}

func handleDescribeArchive(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeArchiveRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ArchiveName == "" {
		return jsonErr(service.ErrValidation("ArchiveName is required."))
	}

	key := archiveStoreKey(store)

	archivesMu.RLock()
	defer archivesMu.RUnlock()

	storeArchives := archives[key]
	if storeArchives == nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Archive "+req.ArchiveName+" does not exist.", http.StatusBadRequest))
	}
	archive, ok := storeArchives[req.ArchiveName]
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Archive "+req.ArchiveName+" does not exist.", http.StatusBadRequest))
	}

	return jsonOK(map[string]any{
		"ArchiveArn":     archive.ArchiveArn,
		"ArchiveName":    archive.ArchiveName,
		"EventSourceArn": archive.EventSourceArn,
		"Description":    archive.Description,
		"EventPattern":   archive.EventPattern,
		"RetentionDays":  archive.RetentionDays,
		"State":          archive.State,
		"CreationTime":   archive.CreationTime.Unix(),
	})
}

// ---- ListArchives ----

func handleListArchives(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	key := archiveStoreKey(store)

	archivesMu.RLock()
	defer archivesMu.RUnlock()

	storeArchives := archives[key]
	entries := make([]map[string]any, 0, len(storeArchives))
	for _, a := range storeArchives {
		entries = append(entries, map[string]any{
			"ArchiveArn":     a.ArchiveArn,
			"ArchiveName":    a.ArchiveName,
			"EventSourceArn": a.EventSourceArn,
			"State":          a.State,
			"RetentionDays":  a.RetentionDays,
			"CreationTime":   a.CreationTime.Unix(),
		})
	}

	return jsonOK(map[string]any{
		"Archives": entries,
	})
}

// ---- DeleteArchive ----

type deleteArchiveRequest struct {
	ArchiveName string `json:"ArchiveName"`
}

func handleDeleteArchive(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteArchiveRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ArchiveName == "" {
		return jsonErr(service.ErrValidation("ArchiveName is required."))
	}

	key := archiveStoreKey(store)

	archivesMu.Lock()
	defer archivesMu.Unlock()

	storeArchives := archives[key]
	if storeArchives == nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Archive "+req.ArchiveName+" does not exist.", http.StatusBadRequest))
	}
	if _, ok := storeArchives[req.ArchiveName]; !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Archive "+req.ArchiveName+" does not exist.", http.StatusBadRequest))
	}

	delete(storeArchives, req.ArchiveName)
	return emptyOK()
}

// ---- StartReplay ----

type startReplayRequest struct {
	ReplayName     string  `json:"ReplayName"`
	EventSourceArn string  `json:"EventSourceArn"`
	EventStartTime float64 `json:"EventStartTime"`
	EventEndTime   float64 `json:"EventEndTime"`
}

func handleStartReplay(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startReplayRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ReplayName == "" {
		return jsonErr(service.ErrValidation("ReplayName is required."))
	}
	if req.EventSourceArn == "" {
		return jsonErr(service.ErrValidation("EventSourceArn is required."))
	}

	key := archiveStoreKey(store)
	now := time.Now().UTC()
	arn := fmt.Sprintf("arn:aws:events:%s:%s:replay/%s", store.region, store.accountID, req.ReplayName)

	replaysMu.Lock()
	defer replaysMu.Unlock()

	if replays[key] == nil {
		replays[key] = make(map[string]*Replay)
	}
	if _, exists := replays[key][req.ReplayName]; exists {
		return jsonErr(service.NewAWSError("ResourceAlreadyExistsException",
			"Replay "+req.ReplayName+" already exists.", http.StatusConflict))
	}

	replay := &Replay{
		ReplayName:      req.ReplayName,
		ReplayArn:       arn,
		EventSourceArn:  req.EventSourceArn,
		State:           "COMPLETED",
		EventStartTime:  req.EventStartTime,
		EventEndTime:    req.EventEndTime,
		ReplayStartTime: now,
		ReplayEndTime:   now,
	}
	replays[key][req.ReplayName] = replay

	return jsonOK(map[string]any{
		"ReplayArn":       arn,
		"ReplayName":      req.ReplayName,
		"State":           "COMPLETED",
		"ReplayStartTime": now.Unix(),
		"ReplayEndTime":   now.Unix(),
	})
}

// ---- DescribeReplay ----

type describeReplayRequest struct {
	ReplayName string `json:"ReplayName"`
}

func handleDescribeReplay(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeReplayRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ReplayName == "" {
		return jsonErr(service.ErrValidation("ReplayName is required."))
	}

	key := archiveStoreKey(store)

	replaysMu.RLock()
	defer replaysMu.RUnlock()

	storeReplays := replays[key]
	if storeReplays == nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Replay "+req.ReplayName+" does not exist.", http.StatusBadRequest))
	}
	replay, ok := storeReplays[req.ReplayName]
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Replay "+req.ReplayName+" does not exist.", http.StatusBadRequest))
	}

	return jsonOK(map[string]any{
		"ReplayArn":       replay.ReplayArn,
		"ReplayName":      replay.ReplayName,
		"EventSourceArn":  replay.EventSourceArn,
		"State":           replay.State,
		"EventStartTime":  replay.EventStartTime,
		"EventEndTime":    replay.EventEndTime,
		"ReplayStartTime": replay.ReplayStartTime.Unix(),
		"ReplayEndTime":   replay.ReplayEndTime.Unix(),
	})
}
