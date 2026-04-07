package codecommit

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidInputException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStrSlice(m map[string]any, key string) []string {
	arr, ok := m[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// ---- Repository handlers ----

func handleCreateRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "repositoryName")
	description := getStr(req, "repositoryDescription")

	var tags map[string]string
	if tagMap, ok := req["tags"].(map[string]any); ok {
		tags = make(map[string]string)
		for k, v := range tagMap {
			if sv, ok := v.(string); ok {
				tags[k] = sv
			}
		}
	}

	repo, awsErr := store.CreateRepository(name, description, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"repositoryMetadata": repoToMap(repo)})
}

func handleGetRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "repositoryName")
	repo, awsErr := store.GetRepository(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"repositoryMetadata": repoToMap(repo)})
}

func handleListRepositories(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	repos := store.ListRepositories()

	result := make([]map[string]any, len(repos))
	for i, r := range repos {
		result[i] = map[string]any{
			"repositoryId":   r.ID,
			"repositoryName": r.Name,
		}
	}
	return jsonOK(map[string]any{"repositories": result})
}

func handleDeleteRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "repositoryName")
	repo, awsErr := store.DeleteRepository(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"repositoryId": repo.ID})
}

func handleUpdateRepositoryName(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	oldName := getStr(req, "oldName")
	newName := getStr(req, "newName")

	if awsErr := store.UpdateRepositoryName(oldName, newName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleUpdateRepositoryDescription(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "repositoryName")
	description := getStr(req, "repositoryDescription")

	if awsErr := store.UpdateRepositoryDescription(name, description); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

// ---- Branch handlers ----

func handleCreateBranch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	repoName := getStr(req, "repositoryName")
	branchName := getStr(req, "branchName")
	commitID := getStr(req, "commitId")

	if awsErr := store.CreateBranch(repoName, branchName, commitID); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleGetBranch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	repoName := getStr(req, "repositoryName")
	branchName := getStr(req, "branchName")

	branch, awsErr := store.GetBranch(repoName, branchName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"branch": map[string]any{
			"branchName": branch.Name,
			"commitId":   branch.CommitID,
		},
	})
}

func handleListBranches(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	repoName := getStr(req, "repositoryName")
	names, awsErr := store.ListBranches(repoName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if names == nil {
		names = []string{}
	}
	return jsonOK(map[string]any{"branches": names})
}

func handleDeleteBranch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	repoName := getStr(req, "repositoryName")
	branchName := getStr(req, "branchName")

	branch, awsErr := store.DeleteBranch(repoName, branchName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"deletedBranch": map[string]any{
			"branchName": branch.Name,
			"commitId":   branch.CommitID,
		},
	})
}

// ---- Pull Request handlers ----

func handleCreatePullRequest(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	title := getStr(req, "title")
	description := getStr(req, "description")

	var repoName, sourceRef, destRef string
	if targets, ok := req["targets"].([]any); ok && len(targets) > 0 {
		if t, ok := targets[0].(map[string]any); ok {
			repoName = getStr(t, "repositoryName")
			sourceRef = getStr(t, "sourceReference")
			destRef = getStr(t, "destinationReference")
		}
	}

	pr, awsErr := store.CreatePullRequest(repoName, title, description, sourceRef, destRef)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"pullRequest": prToMap(pr)})
}

func handleGetPullRequest(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	prID := getStr(req, "pullRequestId")
	pr, awsErr := store.GetPullRequest(prID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"pullRequest": prToMap(pr)})
}

func handleListPullRequests(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	repoName := getStr(req, "repositoryName")
	status := getStr(req, "pullRequestStatus")

	ids := store.ListPullRequests(repoName, status)
	if ids == nil {
		ids = []string{}
	}
	return jsonOK(map[string]any{"pullRequestIds": ids})
}

func handleUpdatePullRequestStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	prID := getStr(req, "pullRequestId")
	status := getStr(req, "pullRequestStatus")

	pr, awsErr := store.UpdatePullRequestStatus(prID, status)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"pullRequest": prToMap(pr)})
}

func handleMergePullRequestBySquash(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	prID := getStr(req, "pullRequestId")
	repoName := getStr(req, "repositoryName")

	pr, awsErr := store.MergePullRequestBySquash(prID, repoName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"pullRequest": prToMap(pr)})
}

func handleUpdatePullRequestTitle(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	prID := getStr(req, "pullRequestId")
	title := getStr(req, "title")

	pr, awsErr := store.UpdatePullRequestTitle(prID, title)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"pullRequest": prToMap(pr)})
}

func handleMergePullRequestByFastForward(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	prID := getStr(req, "pullRequestId")
	repoName := getStr(req, "repositoryName")

	pr, awsErr := store.MergePullRequestByFastForward(prID, repoName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"pullRequest": prToMap(pr)})
}

func handlePostCommentForPullRequest(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	prID := getStr(req, "pullRequestId")
	repoName := getStr(req, "repositoryName")
	content := getStr(req, "content")
	beforeCommitID := getStr(req, "beforeCommitId")
	afterCommitID := getStr(req, "afterCommitId")
	filePath := getStr(req, "filePath")

	comment, awsErr := store.PostCommentForPullRequest(prID, repoName, content, beforeCommitID, afterCommitID, filePath)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"comment": commentToMap(comment)})
}

func handleGetCommentsForPullRequest(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	prID := getStr(req, "pullRequestId")
	comments, awsErr := store.GetCommentsForPullRequest(prID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	result := make([]map[string]any, len(comments))
	for i, c := range comments {
		result[i] = commentToMap(c)
	}
	return jsonOK(map[string]any{"commentsForPullRequestData": result})
}

func handlePutRepositoryTriggers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	repoName := getStr(req, "repositoryName")
	var triggers []RepositoryTrigger
	if tList, ok := req["triggers"].([]any); ok {
		for _, t := range tList {
			if tm, ok := t.(map[string]any); ok {
				trigger := RepositoryTrigger{
					Name:           getStr(tm, "name"),
					DestinationARN: getStr(tm, "destinationArn"),
					CustomData:     getStr(tm, "customData"),
				}
				if branches, ok := tm["branches"].([]any); ok {
					for _, b := range branches {
						if bs, ok := b.(string); ok {
							trigger.Branches = append(trigger.Branches, bs)
						}
					}
				}
				if events, ok := tm["events"].([]any); ok {
					for _, e := range events {
						if es, ok := e.(string); ok {
							trigger.Events = append(trigger.Events, es)
						}
					}
				}
				triggers = append(triggers, trigger)
			}
		}
	}

	configID, awsErr := store.PutRepositoryTriggers(repoName, triggers)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"configurationId": configID})
}

func handleGetRepositoryTriggers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	repoName := getStr(req, "repositoryName")
	triggers, awsErr := store.GetRepositoryTriggers(repoName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	result := make([]map[string]any, len(triggers))
	for i, t := range triggers {
		result[i] = map[string]any{
			"name":           t.Name,
			"destinationArn": t.DestinationARN,
			"customData":     t.CustomData,
			"branches":       t.Branches,
			"events":         t.Events,
		}
	}
	return jsonOK(map[string]any{"triggers": result})
}

// ---- Tag handlers ----

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}

	var tags map[string]string
	if tagMap, ok := req["tags"].(map[string]any); ok {
		tags = make(map[string]string)
		for k, v := range tagMap {
			if sv, ok := v.(string); ok {
				tags[k] = sv
			}
		}
	}

	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "resourceArn")
	keys := getStrSlice(req, "tagKeys")

	if awsErr := store.UntagResource(arn, keys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "resourceArn")
	tags := store.ListTagsForResource(arn)
	return jsonOK(map[string]any{"tags": tags})
}

// ---- Commit & Diff handlers ----

func handleGetCommit(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	repoName := getStr(req, "repositoryName")
	commitID := getStr(req, "commitId")

	commit, awsErr := store.GetCommit(repoName, commitID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"commit": map[string]any{
			"commitId":       commit.CommitID,
			"treeId":         commit.TreeID,
			"parents":        commit.ParentIDs,
			"message":        commit.Message,
			"additionalData": commit.AdditionalData,
			"author": map[string]any{
				"name":  commit.Author.Name,
				"email": commit.Author.Email,
				"date":  float64(commit.Author.Date.Unix()),
			},
			"committer": map[string]any{
				"name":  commit.Committer.Name,
				"email": commit.Committer.Email,
				"date":  float64(commit.Committer.Date.Unix()),
			},
		},
	})
}

func handleGetDifferences(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	repoName := getStr(req, "repositoryName")
	beforeSpec := getStr(req, "beforeCommitSpecifier")
	afterSpec := getStr(req, "afterCommitSpecifier")

	diffs, awsErr := store.GetDifferences(repoName, beforeSpec, afterSpec)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	result := make([]map[string]any, len(diffs))
	for i, d := range diffs {
		dm := map[string]any{"changeType": d.ChangeType}
		if d.BeforeBlob != nil {
			dm["beforeBlob"] = map[string]any{
				"blobId": d.BeforeBlob.BlobID,
				"path":   d.BeforeBlob.Path,
				"mode":   d.BeforeBlob.Mode,
			}
		}
		if d.AfterBlob != nil {
			dm["afterBlob"] = map[string]any{
				"blobId": d.AfterBlob.BlobID,
				"path":   d.AfterBlob.Path,
				"mode":   d.AfterBlob.Mode,
			}
		}
		result[i] = dm
	}
	return jsonOK(map[string]any{"differences": result})
}

// ---- conversion helpers ----

func repoToMap(r *Repository) map[string]any {
	return map[string]any{
		"repositoryId":          r.ID,
		"repositoryName":        r.Name,
		"Arn":                   r.ARN,
		"cloneUrlHttp":          r.CloneURLHTTP,
		"cloneUrlSsh":           r.CloneURLSSH,
		"repositoryDescription": r.Description,
		"defaultBranch":         r.DefaultBranch,
		"creationDate":          float64(r.CreatedAt.Unix()),
		"lastModifiedDate":      float64(r.LastModified.Unix()),
	}
}

func commentToMap(c *Comment) map[string]any {
	return map[string]any{
		"commentId":      c.CommentID,
		"content":        c.Content,
		"authorArn":      c.AuthorARN,
		"pullRequestId":  c.PullRequestID,
		"repositoryName": c.RepositoryName,
		"beforeCommitId": c.BeforeCommitID,
		"afterCommitId":  c.AfterCommitID,
		"filePath":       c.FilePath,
		"creationDate":   float64(c.CreationDate.Unix()),
		"lastModifiedDate": float64(c.LastModified.Unix()),
		"deleted":        c.Deleted,
	}
}

func prToMap(pr *PullRequest) map[string]any {
	m := map[string]any{
		"pullRequestId":    pr.ID,
		"title":            pr.Title,
		"description":      pr.Description,
		"pullRequestStatus": pr.Status,
		"authorArn":        pr.AuthorARN,
		"creationDate":     float64(pr.CreationDate.Unix()),
		"lastActivityDate": float64(pr.LastActivityDate.Unix()),
		"pullRequestTargets": []map[string]any{
			{
				"repositoryName":       pr.RepositoryName,
				"sourceReference":      pr.SourceReference,
				"destinationReference": pr.DestinationReference,
			},
		},
	}
	if pr.MergeMetadata != nil {
		m["pullRequestTargets"].([]map[string]any)[0]["mergeMetadata"] = map[string]any{
			"isMerged":      pr.MergeMetadata.IsMerged,
			"mergedBy":      pr.MergeMetadata.MergedBy,
			"mergeCommitId": pr.MergeMetadata.MergeCommitID,
		}
	}
	return m
}
