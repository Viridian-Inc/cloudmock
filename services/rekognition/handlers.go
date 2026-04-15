package rekognition

import (
	"net/http"
	"strings"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── map[string]any decoders ──────────────────────────────────────────────────

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return 0
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		}
	}
	return 0
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getMapList(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, x := range arr {
		if xm, ok := x.(map[string]any); ok {
			out = append(out, xm)
		}
	}
	return out
}

func getStrList(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func getStringMap(m map[string]any, key string) map[string]string {
	out := make(map[string]string)
	mm := getMap(m, key)
	for k, v := range mm {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func rfc3339(t time.Time) string { return t.Format(time.RFC3339) }

// ── Conversion helpers ───────────────────────────────────────────────────────

func collectionToMap(c *StoredCollection) map[string]any {
	return map[string]any{
		"CollectionId":      c.CollectionID,
		"CollectionARN":     c.CollectionArn,
		"FaceModelVersion":  c.FaceModelVersion,
		"CreationTimestamp": rfc3339(c.CreationTime),
	}
}

func faceToMap(f *StoredFace, collectionID string) map[string]any {
	out := map[string]any{
		"FaceId":                 f.FaceID,
		"ImageId":                f.ImageID,
		"BoundingBox":            f.BoundingBox,
		"Confidence":             f.Confidence,
		"IndexFacesModelVersion": f.IndexFacesModelVersion,
	}
	if collectionID != "" {
		out["CollectionId"] = collectionID
	}
	if f.ExternalImageID != "" {
		out["ExternalImageId"] = f.ExternalImageID
	}
	if f.UserID != "" {
		out["UserId"] = f.UserID
	}
	return out
}

func userToMap(u *StoredUser) map[string]any {
	return map[string]any{
		"UserId":     u.UserID,
		"UserStatus": u.UserStatus,
	}
}

func projectToMap(p *StoredProject) map[string]any {
	return map[string]any{
		"ProjectArn":        p.ProjectArn,
		"ProjectName":       p.ProjectName,
		"Status":            p.Status,
		"CreationTimestamp": rfc3339(p.CreationTime),
		"Feature":           p.Feature,
		"AutoUpdate":        p.AutoUpdate,
	}
}

func projectVersionToMap(v *StoredProjectVersion) map[string]any {
	out := map[string]any{
		"ProjectVersionArn": v.ProjectVersionArn,
		"VersionName":       v.VersionName,
		"Status":            v.Status,
		"StatusMessage":     v.StatusMessage,
		"CreationTimestamp": rfc3339(v.CreationTime),
		"MinInferenceUnits": v.MinInferenceUnits,
		"MaxInferenceUnits": v.MaxInferenceUnits,
		"Feature":           v.Feature,
	}
	if v.KmsKeyID != "" {
		out["KmsKeyId"] = v.KmsKeyID
	}
	return out
}

func datasetToMap(d *StoredDataset) map[string]any {
	return map[string]any{
		"DatasetArn":           d.DatasetArn,
		"DatasetType":          d.DatasetType,
		"ProjectArn":           d.ProjectArn,
		"Status":               d.Status,
		"StatusMessage":        d.StatusMessage,
		"CreationTimestamp":    rfc3339(d.CreationTime),
		"LastUpdatedTimestamp": rfc3339(d.LastUpdatedTime),
		"DatasetStats":         d.Stats,
	}
}

func streamProcessorToMap(sp *StoredStreamProcessor) map[string]any {
	out := map[string]any{
		"Name":                  sp.Name,
		"StreamProcessorArn":    sp.StreamProcessorArn,
		"Status":                sp.Status,
		"StatusMessage":         sp.StatusMessage,
		"Input":                 sp.Input,
		"Output":                sp.Output,
		"Settings":              sp.Settings,
		"NotificationChannel":   sp.Notification,
		"RegionsOfInterest":     sp.RegionsOfInterest,
		"DataSharingPreference": sp.DataSharing,
		"RoleArn":               sp.RoleArn,
		"CreationTimestamp":     rfc3339(sp.CreationTime),
		"LastUpdateTimestamp":   rfc3339(sp.LastUpdateTime),
	}
	if sp.KmsKeyID != "" {
		out["KmsKeyId"] = sp.KmsKeyID
	}
	return out
}

func mediaJobToMap(j *StoredMediaAnalysisJob) map[string]any {
	out := map[string]any{
		"JobId":             j.JobID,
		"JobName":           j.JobName,
		"Status":            j.Status,
		"OperationsConfig":  j.OperationsConfig,
		"Input":             j.Input,
		"OutputConfig":      j.Output,
		"Results":           j.Results,
		"CreationTimestamp": rfc3339(j.CreationTime),
		"CompletionTimestamp": rfc3339(j.CompletionTime),
	}
	if j.KmsKeyID != "" {
		out["KmsKeyId"] = j.KmsKeyID
	}
	return out
}

// ── Sample/static response builders ─────────────────────────────────────────

func sampleFaceDetail() map[string]any {
	return map[string]any{
		"BoundingBox": sampleBoundingBox(),
		"Confidence":  99.5,
		"AgeRange":    map[string]any{"Low": 25, "High": 35},
		"Smile":       map[string]any{"Value": true, "Confidence": 95.5},
		"Eyeglasses":  map[string]any{"Value": false, "Confidence": 92.0},
		"Sunglasses":  map[string]any{"Value": false, "Confidence": 99.0},
		"Gender":      map[string]any{"Value": "Female", "Confidence": 99.1},
		"Beard":       map[string]any{"Value": false, "Confidence": 99.0},
		"Mustache":    map[string]any{"Value": false, "Confidence": 99.0},
		"EyesOpen":    map[string]any{"Value": true, "Confidence": 99.0},
		"MouthOpen":   map[string]any{"Value": false, "Confidence": 95.0},
		"Emotions": []map[string]any{
			{"Type": "HAPPY", "Confidence": 95.0},
			{"Type": "CALM", "Confidence": 4.2},
		},
		"Pose":    map[string]any{"Roll": 0.5, "Yaw": -2.1, "Pitch": 1.2},
		"Quality": map[string]any{"Brightness": 85.0, "Sharpness": 90.0},
		"Landmarks": []map[string]any{
			{"Type": "eyeLeft", "X": 0.4, "Y": 0.4},
			{"Type": "eyeRight", "X": 0.6, "Y": 0.4},
			{"Type": "nose", "X": 0.5, "Y": 0.5},
			{"Type": "mouthLeft", "X": 0.42, "Y": 0.65},
			{"Type": "mouthRight", "X": 0.58, "Y": 0.65},
		},
	}
}

func sampleLabel(name string, conf float64) map[string]any {
	return map[string]any{
		"Name":       name,
		"Confidence": conf,
		"Instances":  []map[string]any{},
		"Parents":    []map[string]any{},
		"Categories": []map[string]any{
			{"Name": "Person Description"},
		},
	}
}

func sampleTextDetection(text string, id int) map[string]any {
	return map[string]any{
		"DetectedText": text,
		"Type":         "WORD",
		"Id":           id,
		"Confidence":   99.0,
		"Geometry": map[string]any{
			"BoundingBox": sampleBoundingBox(),
			"Polygon": []map[string]any{
				{"X": 0.25, "Y": 0.25},
				{"X": 0.75, "Y": 0.25},
				{"X": 0.75, "Y": 0.75},
				{"X": 0.25, "Y": 0.75},
			},
		},
	}
}

// ── Collection handlers ──────────────────────────────────────────────────────

func handleCreateCollection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "CollectionId")
	if id == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	c, err := store.CreateCollection(id, getStringMap(req, "Tags"))
	if err != nil {
		return jsonErr(err)
	}
	store.TagResource(c.CollectionArn, getStringMap(req, "Tags"))
	return jsonOK(map[string]any{
		"CollectionArn":    c.CollectionArn,
		"FaceModelVersion": c.FaceModelVersion,
		"StatusCode":       200,
	})
}

func handleDeleteCollection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "CollectionId")
	if id == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	if err := store.DeleteCollection(id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"StatusCode": 200})
}

func handleDescribeCollection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "CollectionId")
	if id == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	c, err := store.GetCollection(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"CollectionARN":     c.CollectionArn,
		"CreationTimestamp": rfc3339(c.CreationTime),
		"FaceCount":         store.CountFaces(id),
		"FaceModelVersion":  c.FaceModelVersion,
		"UserCount":         store.CountUsers(id),
	})
}

func handleListCollections(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	cols := store.ListCollections()
	ids := make([]string, 0, len(cols))
	versions := make([]string, 0, len(cols))
	for _, c := range cols {
		ids = append(ids, c.CollectionID)
		versions = append(versions, c.FaceModelVersion)
	}
	return jsonOK(map[string]any{
		"CollectionIds":     ids,
		"FaceModelVersions": versions,
	})
}

// ── Face handlers ────────────────────────────────────────────────────────────

func handleIndexFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	if collectionID == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	externalID := getStr(req, "ExternalImageId")
	face, err := store.IndexFace(collectionID, externalID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"FaceModelVersion": "6.0",
		"FaceRecords": []map[string]any{
			{
				"Face": faceToMap(face, ""),
				"FaceDetail": sampleFaceDetail(),
			},
		},
		"OrientationCorrection": "ROTATE_0",
		"UnindexedFaces":        []map[string]any{},
	})
}

func handleDetectFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_ = req // image is opaque
	return jsonOK(map[string]any{
		"FaceDetails":           []map[string]any{sampleFaceDetail()},
		"OrientationCorrection": "ROTATE_0",
	})
}

func handleListFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	if collectionID == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	faces, err := store.ListFaces(collectionID, getStr(req, "UserId"))
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(faces))
	for _, f := range faces {
		out = append(out, faceToMap(f, ""))
	}
	return jsonOK(map[string]any{
		"Faces":            out,
		"FaceModelVersion": "6.0",
	})
}

func handleDeleteFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	if collectionID == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	faceIDs := getStrList(req, "FaceIds")
	if len(faceIDs) == 0 {
		return jsonErr(service.ErrValidation("FaceIds is required."))
	}
	deleted, failed, err := store.DeleteFaces(collectionID, faceIDs)
	if err != nil {
		return jsonErr(err)
	}
	unsuccessful := make([]map[string]any, 0, len(failed))
	for _, fid := range failed {
		unsuccessful = append(unsuccessful, map[string]any{
			"FaceId":  fid,
			"Reasons": []string{"FACE_NOT_FOUND"},
		})
	}
	return jsonOK(map[string]any{
		"DeletedFaces":              deleted,
		"UnsuccessfulFaceDeletions": unsuccessful,
	})
}

func handleSearchFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	faceID := getStr(req, "FaceId")
	if collectionID == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	if faceID == "" {
		return jsonErr(service.ErrValidation("FaceId is required."))
	}
	if _, err := store.GetFace(collectionID, faceID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"SearchedFaceId":   faceID,
		"FaceModelVersion": "6.0",
		"FaceMatches":      []map[string]any{},
	})
}

func handleSearchFacesByImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	if collectionID == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	if _, err := store.GetCollection(collectionID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"SearchedFaceBoundingBox": sampleBoundingBox(),
		"SearchedFaceConfidence":  99.5,
		"FaceModelVersion":        "6.0",
		"FaceMatches":             []map[string]any{},
	})
}

func handleCompareFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getMap(req, "SourceImage") == nil {
		return jsonErr(service.ErrValidation("SourceImage is required."))
	}
	if getMap(req, "TargetImage") == nil {
		return jsonErr(service.ErrValidation("TargetImage is required."))
	}
	return jsonOK(map[string]any{
		"FaceMatches": []map[string]any{
			{
				"Similarity": 99.5,
				"Face": map[string]any{
					"BoundingBox": sampleBoundingBox(),
					"Confidence":  99.0,
					"Landmarks":   []map[string]any{},
					"Pose":        map[string]any{"Roll": 0, "Yaw": 0, "Pitch": 0},
					"Quality":     map[string]any{"Brightness": 80.0, "Sharpness": 90.0},
				},
			},
		},
		"SourceImageFace": map[string]any{
			"BoundingBox": sampleBoundingBox(),
			"Confidence":  99.0,
		},
		"UnmatchedFaces":                   []map[string]any{},
		"SourceImageOrientationCorrection": "ROTATE_0",
		"TargetImageOrientationCorrection": "ROTATE_0",
	})
}

func handleAssociateFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	userID := getStr(req, "UserId")
	if collectionID == "" || userID == "" {
		return jsonErr(service.ErrValidation("CollectionId and UserId are required."))
	}
	faceIDs := getStrList(req, "FaceIds")
	if len(faceIDs) == 0 {
		return jsonErr(service.ErrValidation("FaceIds is required."))
	}
	associated, unsuccessful, status, err := store.AssociateFaces(collectionID, userID, faceIDs)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"AssociatedFaces":              associated,
		"UnsuccessfulFaceAssociations": unsuccessful,
		"UserStatus":                   status,
	})
}

func handleDisassociateFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	userID := getStr(req, "UserId")
	if collectionID == "" || userID == "" {
		return jsonErr(service.ErrValidation("CollectionId and UserId are required."))
	}
	faceIDs := getStrList(req, "FaceIds")
	if len(faceIDs) == 0 {
		return jsonErr(service.ErrValidation("FaceIds is required."))
	}
	disassociated, unsuccessful, status, err := store.DisassociateFaces(collectionID, userID, faceIDs)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"DisassociatedFaces":              disassociated,
		"UnsuccessfulFaceDisassociations": unsuccessful,
		"UserStatus":                      status,
	})
}

// ── User handlers ────────────────────────────────────────────────────────────

func handleCreateUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	userID := getStr(req, "UserId")
	if collectionID == "" || userID == "" {
		return jsonErr(service.ErrValidation("CollectionId and UserId are required."))
	}
	if err := store.CreateUser(collectionID, userID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	userID := getStr(req, "UserId")
	if collectionID == "" || userID == "" {
		return jsonErr(service.ErrValidation("CollectionId and UserId are required."))
	}
	if err := store.DeleteUser(collectionID, userID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListUsers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	if collectionID == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	users, err := store.ListUsers(collectionID)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, userToMap(u))
	}
	return jsonOK(map[string]any{"Users": out})
}

func handleSearchUsers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	if collectionID == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	if _, err := store.GetCollection(collectionID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"FaceModelVersion": "6.0",
		"UserMatches":      []map[string]any{},
		"SearchedFace": map[string]any{
			"FaceId": getStr(req, "FaceId"),
		},
	})
}

func handleSearchUsersByImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	collectionID := getStr(req, "CollectionId")
	if collectionID == "" {
		return jsonErr(service.ErrValidation("CollectionId is required."))
	}
	if _, err := store.GetCollection(collectionID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"FaceModelVersion": "6.0",
		"UserMatches":      []map[string]any{},
		"SearchedFace": map[string]any{
			"FaceBoundingBox": sampleBoundingBox(),
			"FaceConfidence":  99.5,
		},
		"UnsearchedFaces": []map[string]any{},
	})
}

// ── Synchronous detection handlers ───────────────────────────────────────────

func handleDetectLabels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_ = req
	return jsonOK(map[string]any{
		"Labels": []map[string]any{
			sampleLabel("Person", 99.5),
			sampleLabel("Human", 99.5),
		},
		"LabelModelVersion":     "3.0",
		"OrientationCorrection": "ROTATE_0",
		"ImageProperties": map[string]any{
			"Quality": map[string]any{"Brightness": 85.0, "Sharpness": 90.0, "Contrast": 80.0},
		},
	})
}

func handleDetectModerationLabels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_ = req
	return jsonOK(map[string]any{
		"ModerationLabels":            []map[string]any{},
		"ModerationModelVersion":      "7.0",
		"HumanLoopActivationOutput":   nil,
		"ProjectVersion":              "",
		"ContentTypes":                []map[string]any{},
	})
}

func handleDetectText(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_ = req
	return jsonOK(map[string]any{
		"TextDetections": []map[string]any{
			sampleTextDetection("Hello", 0),
			sampleTextDetection("World", 1),
		},
		"TextModelVersion": "3.0",
	})
}

func handleDetectCustomLabels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	versionArn := getStr(req, "ProjectVersionArn")
	if versionArn == "" {
		return jsonErr(service.ErrValidation("ProjectVersionArn is required."))
	}
	return jsonOK(map[string]any{
		"CustomLabels": []map[string]any{
			{
				"Name":       "sample-label",
				"Confidence": 95.0,
				"Geometry": map[string]any{
					"BoundingBox": sampleBoundingBox(),
				},
			},
		},
	})
}

func handleDetectProtectiveEquipment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_ = req
	return jsonOK(map[string]any{
		"ProtectiveEquipmentModelVersion": "1.0",
		"Persons": []map[string]any{
			{
				"Id":          0,
				"BoundingBox": sampleBoundingBox(),
				"Confidence":  99.5,
				"BodyParts": []map[string]any{
					{
						"Name":       "FACE",
						"Confidence": 99.0,
						"EquipmentDetections": []map[string]any{
							{
								"Type":             "FACE_COVER",
								"Confidence":       95.0,
								"BoundingBox":      sampleBoundingBox(),
								"CoversBodyPart":   map[string]any{"Value": true, "Confidence": 95.0},
							},
						},
					},
				},
			},
		},
		"Summary": map[string]any{
			"PersonsWithRequiredEquipment":    []int{0},
			"PersonsWithoutRequiredEquipment": []int{},
			"PersonsIndeterminate":            []int{},
		},
	})
}

func handleRecognizeCelebrities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_ = req
	return jsonOK(map[string]any{
		"CelebrityFaces":                   []map[string]any{},
		"UnrecognizedFaces":                []map[string]any{},
		"OrientationCorrection":            "ROTATE_0",
	})
}

// ── Project handlers ─────────────────────────────────────────────────────────

func handleCreateProject(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "ProjectName")
	if name == "" {
		return jsonErr(service.ErrValidation("ProjectName is required."))
	}
	p, err := store.CreateProject(name, getStr(req, "Feature"), getStr(req, "AutoUpdate"), getStringMap(req, "Tags"))
	if err != nil {
		return jsonErr(err)
	}
	store.TagResource(p.ProjectArn, getStringMap(req, "Tags"))
	return jsonOK(map[string]any{"ProjectArn": p.ProjectArn})
}

func handleCreateProjectVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	projectArn := getStr(req, "ProjectArn")
	versionName := getStr(req, "VersionName")
	if projectArn == "" || versionName == "" {
		return jsonErr(service.ErrValidation("ProjectArn and VersionName are required."))
	}
	v, err := store.CreateProjectVersion(projectArn, versionName, getStr(req, "KmsKeyId"), getStringMap(req, "Tags"))
	if err != nil {
		return jsonErr(err)
	}
	store.TagResource(v.ProjectVersionArn, getStringMap(req, "Tags"))
	return jsonOK(map[string]any{"ProjectVersionArn": v.ProjectVersionArn})
}

func handleDeleteProject(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ProjectArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ProjectArn is required."))
	}
	p, err := store.DeleteProject(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Status": p.Status})
}

func handleDeleteProjectVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ProjectVersionArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ProjectVersionArn is required."))
	}
	v, err := store.DeleteProjectVersion(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Status": v.Status})
}

func handleDescribeProjects(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	projects := store.ListProjects(getStrList(req, "Features"), getStrList(req, "ProjectNames"))
	out := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		out = append(out, projectToMap(p))
	}
	return jsonOK(map[string]any{"ProjectDescriptions": out})
}

func handleDescribeProjectVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	projectArn := getStr(req, "ProjectArn")
	if projectArn == "" {
		return jsonErr(service.ErrValidation("ProjectArn is required."))
	}
	versions, err := store.DescribeProjectVersions(projectArn, getStrList(req, "VersionNames"))
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(versions))
	for _, v := range versions {
		out = append(out, projectVersionToMap(v))
	}
	return jsonOK(map[string]any{"ProjectVersionDescriptions": out})
}

func handleStartProjectVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ProjectVersionArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ProjectVersionArn is required."))
	}
	minInf := getInt(req, "MinInferenceUnits")
	maxInf := getInt(req, "MaxInferenceUnits")
	v, err := store.StartProjectVersion(arn, minInf, maxInf)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Status": v.Status})
}

func handleStopProjectVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ProjectVersionArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ProjectVersionArn is required."))
	}
	v, err := store.StopProjectVersion(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Status": v.Status})
}

func handleCopyProjectVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	srcArn := getStr(req, "SourceProjectVersionArn")
	destArn := getStr(req, "DestinationProjectArn")
	versionName := getStr(req, "VersionName")
	if srcArn == "" || destArn == "" || versionName == "" {
		return jsonErr(service.ErrValidation("SourceProjectVersionArn, DestinationProjectArn, and VersionName are required."))
	}
	v, err := store.CopyProjectVersion(srcArn, destArn, versionName, getStr(req, "KmsKeyId"), getStringMap(req, "Tags"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ProjectVersionArn": v.ProjectVersionArn})
}

func handlePutProjectPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	projectArn := getStr(req, "ProjectArn")
	policyName := getStr(req, "PolicyName")
	policyDoc := getStr(req, "PolicyDocument")
	if projectArn == "" || policyName == "" || policyDoc == "" {
		return jsonErr(service.ErrValidation("ProjectArn, PolicyName, and PolicyDocument are required."))
	}
	rev, err := store.PutProjectPolicy(projectArn, policyName, policyDoc, getStr(req, "PolicyRevisionId"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"PolicyRevisionId": rev})
}

func handleDeleteProjectPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	projectArn := getStr(req, "ProjectArn")
	policyName := getStr(req, "PolicyName")
	if projectArn == "" || policyName == "" {
		return jsonErr(service.ErrValidation("ProjectArn and PolicyName are required."))
	}
	if err := store.DeleteProjectPolicy(projectArn, policyName); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListProjectPolicies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	projectArn := getStr(req, "ProjectArn")
	if projectArn == "" {
		return jsonErr(service.ErrValidation("ProjectArn is required."))
	}
	policies, err := store.ListProjectPolicies(projectArn)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(policies))
	for _, p := range policies {
		out = append(out, map[string]any{
			"ProjectArn":             projectArn,
			"PolicyName":             p.PolicyName,
			"PolicyRevisionId":       p.PolicyRevID,
			"PolicyDocument":         p.PolicyDocument,
			"CreationTimestamp":      rfc3339(p.CreationTime),
			"LastUpdatedTimestamp":   rfc3339(p.LastUpdated),
		})
	}
	return jsonOK(map[string]any{"ProjectPolicies": out})
}

// ── Dataset handlers ─────────────────────────────────────────────────────────

func handleCreateDataset(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	projectArn := getStr(req, "ProjectArn")
	datasetType := getStr(req, "DatasetType")
	if projectArn == "" || datasetType == "" {
		return jsonErr(service.ErrValidation("ProjectArn and DatasetType are required."))
	}
	d, err := store.CreateDataset(projectArn, datasetType, getStringMap(req, "Tags"))
	if err != nil {
		return jsonErr(err)
	}
	store.TagResource(d.DatasetArn, getStringMap(req, "Tags"))
	return jsonOK(map[string]any{"DatasetArn": d.DatasetArn})
}

func handleDeleteDataset(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "DatasetArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("DatasetArn is required."))
	}
	if err := store.DeleteDataset(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeDataset(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "DatasetArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("DatasetArn is required."))
	}
	d, err := store.GetDataset(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"DatasetDescription": map[string]any{
			"CreationTimestamp":    rfc3339(d.CreationTime),
			"LastUpdatedTimestamp": rfc3339(d.LastUpdatedTime),
			"Status":               d.Status,
			"StatusMessage":        d.StatusMessage,
			"StatusMessageCode":    "SUCCESS",
			"DatasetStats":         d.Stats,
		},
	})
}

func handleDistributeDatasetEntries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	datasets := getMapList(req, "Datasets")
	arns := make([]string, 0, len(datasets))
	for _, d := range datasets {
		if a := getStr(d, "Arn"); a != "" {
			arns = append(arns, a)
		}
	}
	if len(arns) == 0 {
		return jsonErr(service.ErrValidation("Datasets is required."))
	}
	if err := store.DistributeDatasetEntries(arns); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListDatasetEntries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "DatasetArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("DatasetArn is required."))
	}
	entries, err := store.ListDatasetEntries(arn)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if b, err := gojson.Marshal(e); err == nil {
			out = append(out, string(b))
		}
	}
	return jsonOK(map[string]any{"DatasetEntries": out})
}

func handleListDatasetLabels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "DatasetArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("DatasetArn is required."))
	}
	labels, err := store.ListDatasetLabels(arn)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(labels))
	for _, l := range labels {
		out = append(out, map[string]any{
			"LabelName": l,
			"LabelStats": map[string]any{
				"EntryCount":       0,
				"BoundingBoxCount": 0,
			},
		})
	}
	return jsonOK(map[string]any{"DatasetLabelDescriptions": out})
}

func handleUpdateDatasetEntries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "DatasetArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("DatasetArn is required."))
	}
	entries := getMapList(req, "Changes")
	if entries == nil {
		// Changes is usually a struct {GroundTruth: bytes} — accept either form.
		if changes := getMap(req, "Changes"); changes != nil {
			entries = []map[string]any{changes}
		}
	}
	if err := store.UpdateDatasetEntries(arn, entries); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Stream processor handlers ────────────────────────────────────────────────

func handleCreateStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if getMap(req, "Input") == nil {
		return jsonErr(service.ErrValidation("Input is required."))
	}
	if getMap(req, "Output") == nil {
		return jsonErr(service.ErrValidation("Output is required."))
	}
	if getStr(req, "RoleArn") == "" {
		return jsonErr(service.ErrValidation("RoleArn is required."))
	}
	sp, err := store.CreateStreamProcessor(
		name,
		getMap(req, "Input"),
		getMap(req, "Output"),
		getMap(req, "Settings"),
		getMap(req, "NotificationChannel"),
		getMapList(req, "RegionsOfInterest"),
		getMap(req, "DataSharingPreference"),
		getStr(req, "RoleArn"),
		getStr(req, "KmsKeyId"),
		getStringMap(req, "Tags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	store.TagResource(sp.StreamProcessorArn, getStringMap(req, "Tags"))
	return jsonOK(map[string]any{"StreamProcessorArn": sp.StreamProcessorArn})
}

func handleDeleteStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if err := store.DeleteStreamProcessor(name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	sp, err := store.GetStreamProcessor(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(streamProcessorToMap(sp))
}

func handleListStreamProcessors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	procs := store.ListStreamProcessors()
	out := make([]map[string]any, 0, len(procs))
	for _, sp := range procs {
		out = append(out, map[string]any{
			"Name":   sp.Name,
			"Status": sp.Status,
		})
	}
	return jsonOK(map[string]any{"StreamProcessors": out})
}

func handleStartStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if err := store.StartStreamProcessor(name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"SessionId": generateUUID(),
	})
}

func handleStopStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if err := store.StopStreamProcessor(name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleUpdateStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if err := store.UpdateStreamProcessor(
		name,
		getMap(req, "SettingsForUpdate"),
		getMapList(req, "RegionsOfInterestForUpdate"),
		getMap(req, "DataSharingPreferenceForUpdate"),
		getStrList(req, "ParametersToDelete"),
	); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Async video job handlers ─────────────────────────────────────────────────

func startVideoHandler(ctx *service.RequestContext, store *Store, jobType string) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	video := getMap(req, "Video")
	if video == nil {
		return jsonErr(service.ErrValidation("Video is required."))
	}
	id := store.StartVideoJob(jobType, video, req)
	return jsonOK(map[string]any{"JobId": id})
}

func getVideoHandler(ctx *service.RequestContext, store *Store, jobType, resultsKey string, results any) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "JobId")
	if id == "" {
		return jsonErr(service.ErrValidation("JobId is required."))
	}
	job, err := store.GetVideoJob(id)
	if err != nil {
		return jsonErr(err)
	}
	if job.JobType != "" && job.JobType != jobType {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"JobId is not a "+jobType+" job", http.StatusBadRequest))
	}
	out := map[string]any{
		"JobStatus":     job.Status,
		"StatusMessage": job.StatusMessage,
		"VideoMetadata": map[string]any{
			"Codec":            "h264",
			"DurationMillis":   30000,
			"Format":           "QuickTime / MOV",
			"FrameRate":        30.0,
			"FrameHeight":      720,
			"FrameWidth":       1280,
		},
	}
	if results != nil && resultsKey != "" {
		out[resultsKey] = results
	}
	return jsonOK(out)
}

func handleStartLabelDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startVideoHandler(ctx, store, "label")
}

func handleGetLabelDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return getVideoHandler(ctx, store, "label", "Labels", []map[string]any{
		{
			"Timestamp": 0,
			"Label":     sampleLabel("Person", 99.5),
		},
	})
}

func handleStartTextDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startVideoHandler(ctx, store, "text")
}

func handleGetTextDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return getVideoHandler(ctx, store, "text", "TextDetections", []map[string]any{
		{
			"Timestamp":     0,
			"TextDetection": sampleTextDetection("Sample", 0),
		},
	})
}

func handleStartContentModeration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startVideoHandler(ctx, store, "moderation")
}

func handleGetContentModeration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return getVideoHandler(ctx, store, "moderation", "ModerationLabels", []map[string]any{})
}

func handleStartFaceDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startVideoHandler(ctx, store, "face")
}

func handleGetFaceDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return getVideoHandler(ctx, store, "face", "Faces", []map[string]any{
		{"Timestamp": 0, "Face": sampleFaceDetail()},
	})
}

func handleStartFaceSearch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startVideoHandler(ctx, store, "facesearch")
}

func handleGetFaceSearch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return getVideoHandler(ctx, store, "facesearch", "Persons", []map[string]any{})
}

func handleStartPersonTracking(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startVideoHandler(ctx, store, "person")
}

func handleGetPersonTracking(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return getVideoHandler(ctx, store, "person", "Persons", []map[string]any{
		{
			"Timestamp": 0,
			"Person": map[string]any{
				"Index":       0,
				"BoundingBox": sampleBoundingBox(),
			},
		},
	})
}

func handleStartCelebrityRecognition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startVideoHandler(ctx, store, "celebrity")
}

func handleGetCelebrityRecognition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return getVideoHandler(ctx, store, "celebrity", "Celebrities", []map[string]any{})
}

func handleStartSegmentDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startVideoHandler(ctx, store, "segment")
}

func handleGetSegmentDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return getVideoHandler(ctx, store, "segment", "Segments", []map[string]any{})
}

// ── Media analysis handlers ──────────────────────────────────────────────────

func handleStartMediaAnalysisJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getMap(req, "Input") == nil {
		return jsonErr(service.ErrValidation("Input is required."))
	}
	if getMap(req, "OutputConfig") == nil {
		return jsonErr(service.ErrValidation("OutputConfig is required."))
	}
	if getMap(req, "OperationsConfig") == nil {
		return jsonErr(service.ErrValidation("OperationsConfig is required."))
	}
	job := store.CreateMediaAnalysisJob(
		getStr(req, "JobName"),
		getStr(req, "KmsKeyId"),
		getMap(req, "OperationsConfig"),
		getMap(req, "Input"),
		getMap(req, "OutputConfig"),
		getStringMap(req, "Tags"),
	)
	store.TagResource(job.JobArn, getStringMap(req, "Tags"))
	return jsonOK(map[string]any{"JobId": job.JobID})
}

func handleGetMediaAnalysisJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "JobId")
	if id == "" {
		return jsonErr(service.ErrValidation("JobId is required."))
	}
	job, err := store.GetMediaAnalysisJob(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(mediaJobToMap(job))
}

func handleListMediaAnalysisJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	jobs := store.ListMediaAnalysisJobs()
	out := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, mediaJobToMap(j))
	}
	return jsonOK(map[string]any{"MediaAnalysisJobs": out})
}

// ── Face liveness handlers ───────────────────────────────────────────────────

func handleCreateFaceLivenessSession(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := store.CreateFaceLivenessSession(getStr(req, "KmsKeyId"), getMap(req, "Settings"))
	return jsonOK(map[string]any{"SessionId": id})
}

func handleGetFaceLivenessSessionResults(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "SessionId")
	if id == "" {
		return jsonErr(service.ErrValidation("SessionId is required."))
	}
	sess, err := store.GetFaceLivenessSession(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"SessionId":  sess.SessionID,
		"Status":     sess.Status,
		"Confidence": sess.Confidence,
		"ReferenceImage": map[string]any{
			"Bytes":       []byte{},
			"BoundingBox": sampleBoundingBox(),
		},
		"AuditImages": []map[string]any{},
		"Challenge": map[string]any{
			"Type":    "FaceMovementAndLightChallenge",
			"Version": "1.0.0",
		},
	})
}

// ── Celebrity info ───────────────────────────────────────────────────────────

func handleGetCelebrityInfo(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "Id")
	if id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	return jsonOK(map[string]any{
		"Name": "Sample Celebrity",
		"Urls": []string{"https://www.imdb.com/name/" + id},
		"KnownGender": map[string]any{
			"Type": "Female",
		},
	})
}

// ── Tag handlers ─────────────────────────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	store.TagResource(arn, getStringMap(req, "Tags"))
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	store.UntagResource(arn, getStrList(req, "TagKeys"))
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags := store.ListTags(arn)
	return jsonOK(map[string]any{"Tags": tags})
}

// quiet unused-import warning for strings (helpers may use it later)
var _ = strings.ToLower
