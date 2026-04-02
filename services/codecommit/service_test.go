package codecommit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/codecommit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.CodeCommitService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func createRepo(t *testing.T, s *svc.CodeCommitService, name string) map[string]any {
	t.Helper()
	ctx := jsonCtx("CreateRepository", map[string]any{
		"repositoryName":        name,
		"repositoryDescription": "test repo",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return respBody(t, resp)
}

// --- Repository Tests ---

func TestCreateRepository(t *testing.T) {
	s := newService()
	body := createRepo(t, s, "my-repo")
	metadata := body["repositoryMetadata"].(map[string]any)
	assert.Equal(t, "my-repo", metadata["repositoryName"])
	assert.NotEmpty(t, metadata["repositoryId"])
	assert.Contains(t, metadata["Arn"], "codecommit")
	assert.Contains(t, metadata["cloneUrlHttp"], "my-repo")
	assert.Contains(t, metadata["cloneUrlSsh"], "my-repo")
	assert.Equal(t, "main", metadata["defaultBranch"])
}

func TestCloneURLFormat(t *testing.T) {
	s := newService()
	body := createRepo(t, s, "clone-url-repo")
	metadata := body["repositoryMetadata"].(map[string]any)
	httpURL := metadata["cloneUrlHttp"].(string)
	sshURL := metadata["cloneUrlSsh"].(string)

	// Validate realistic clone URL format
	assert.Equal(t, "https://git-codecommit.us-east-1.amazonaws.com/v1/repos/clone-url-repo", httpURL)
	assert.Equal(t, "ssh://git-codecommit.us-east-1.amazonaws.com/v1/repos/clone-url-repo", sshURL)
}

func TestCreateRepositoryDuplicate(t *testing.T) {
	s := newService()
	createRepo(t, s, "dup-repo")
	ctx := jsonCtx("CreateRepository", map[string]any{"repositoryName": "dup-repo"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RepositoryNameExistsException")
}

func TestCreateRepositoryMissingName(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateRepository", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestGetRepository(t *testing.T) {
	s := newService()
	createRepo(t, s, "get-repo")

	ctx := jsonCtx("GetRepository", map[string]any{"repositoryName": "get-repo"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	metadata := body["repositoryMetadata"].(map[string]any)
	assert.Equal(t, "get-repo", metadata["repositoryName"])
}

func TestGetRepositoryNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("GetRepository", map[string]any{"repositoryName": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RepositoryDoesNotExistException")
}

func TestListRepositories(t *testing.T) {
	s := newService()
	createRepo(t, s, "repo-1")
	createRepo(t, s, "repo-2")

	resp, err := s.HandleRequest(jsonCtx("ListRepositories", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	repos := body["repositories"].([]any)
	assert.Len(t, repos, 2)
}

func TestDeleteRepository(t *testing.T) {
	s := newService()
	createRepo(t, s, "del-repo")

	ctx := jsonCtx("DeleteRepository", map[string]any{"repositoryName": "del-repo"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["repositoryId"])

	// Verify gone
	resp2, _ := s.HandleRequest(jsonCtx("ListRepositories", map[string]any{}))
	body2 := respBody(t, resp2)
	assert.Len(t, body2["repositories"].([]any), 0)
}

func TestDeleteRepositoryNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("DeleteRepository", map[string]any{"repositoryName": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RepositoryDoesNotExistException")
}

func TestUpdateRepositoryName(t *testing.T) {
	s := newService()
	createRepo(t, s, "old-name")

	ctx := jsonCtx("UpdateRepositoryName", map[string]any{
		"oldName": "old-name",
		"newName": "new-name",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify new name works
	resp2, err2 := s.HandleRequest(jsonCtx("GetRepository", map[string]any{"repositoryName": "new-name"}))
	require.NoError(t, err2)
	body2 := respBody(t, resp2)
	assert.Equal(t, "new-name", body2["repositoryMetadata"].(map[string]any)["repositoryName"])
}

func TestUpdateRepositoryDescription(t *testing.T) {
	s := newService()
	createRepo(t, s, "desc-repo")

	ctx := jsonCtx("UpdateRepositoryDescription", map[string]any{
		"repositoryName":        "desc-repo",
		"repositoryDescription": "updated description",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify updated
	resp2, _ := s.HandleRequest(jsonCtx("GetRepository", map[string]any{"repositoryName": "desc-repo"}))
	body := respBody(t, resp2)
	assert.Equal(t, "updated description", body["repositoryMetadata"].(map[string]any)["repositoryDescription"])
}

// --- Branch Tests ---

func TestCreateBranch(t *testing.T) {
	s := newService()
	body := createRepo(t, s, "branch-repo")
	// Get the main branch commit ID
	resp, _ := s.HandleRequest(jsonCtx("GetBranch", map[string]any{
		"repositoryName": "branch-repo",
		"branchName":     "main",
	}))
	brBody := respBody(t, resp)
	commitID := brBody["branch"].(map[string]any)["commitId"].(string)

	ctx := jsonCtx("CreateBranch", map[string]any{
		"repositoryName": "branch-repo",
		"branchName":     "feature-1",
		"commitId":       commitID,
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	_ = body
}

func TestCreateBranchDuplicate(t *testing.T) {
	s := newService()
	createRepo(t, s, "dupbr-repo")

	resp, _ := s.HandleRequest(jsonCtx("GetBranch", map[string]any{
		"repositoryName": "dupbr-repo",
		"branchName":     "main",
	}))
	brBody := respBody(t, resp)
	commitID := brBody["branch"].(map[string]any)["commitId"].(string)

	// Try to create main again
	ctx := jsonCtx("CreateBranch", map[string]any{
		"repositoryName": "dupbr-repo",
		"branchName":     "main",
		"commitId":       commitID,
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BranchNameExistsException")
}

func TestGetBranch(t *testing.T) {
	s := newService()
	createRepo(t, s, "getbr-repo")

	ctx := jsonCtx("GetBranch", map[string]any{
		"repositoryName": "getbr-repo",
		"branchName":     "main",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	branch := body["branch"].(map[string]any)
	assert.Equal(t, "main", branch["branchName"])
	assert.NotEmpty(t, branch["commitId"])
}

func TestGetBranchNotFound(t *testing.T) {
	s := newService()
	createRepo(t, s, "nobr-repo")

	ctx := jsonCtx("GetBranch", map[string]any{
		"repositoryName": "nobr-repo",
		"branchName":     "nonexistent",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BranchDoesNotExistException")
}

func TestListBranches(t *testing.T) {
	s := newService()
	createRepo(t, s, "listbr-repo")

	// Get commit ID and create another branch
	resp, _ := s.HandleRequest(jsonCtx("GetBranch", map[string]any{
		"repositoryName": "listbr-repo",
		"branchName":     "main",
	}))
	commitID := respBody(t, resp)["branch"].(map[string]any)["commitId"].(string)
	s.HandleRequest(jsonCtx("CreateBranch", map[string]any{
		"repositoryName": "listbr-repo",
		"branchName":     "develop",
		"commitId":       commitID,
	}))

	ctx := jsonCtx("ListBranches", map[string]any{"repositoryName": "listbr-repo"})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	branches := body["branches"].([]any)
	assert.Len(t, branches, 2)
}

func TestDeleteBranch(t *testing.T) {
	s := newService()
	createRepo(t, s, "delbr-repo")

	resp, _ := s.HandleRequest(jsonCtx("GetBranch", map[string]any{
		"repositoryName": "delbr-repo",
		"branchName":     "main",
	}))
	commitID := respBody(t, resp)["branch"].(map[string]any)["commitId"].(string)

	// Create a branch to delete
	s.HandleRequest(jsonCtx("CreateBranch", map[string]any{
		"repositoryName": "delbr-repo",
		"branchName":     "feature",
		"commitId":       commitID,
	}))

	ctx := jsonCtx("DeleteBranch", map[string]any{
		"repositoryName": "delbr-repo",
		"branchName":     "feature",
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	deleted := body["deletedBranch"].(map[string]any)
	assert.Equal(t, "feature", deleted["branchName"])
}

func TestDeleteDefaultBranch(t *testing.T) {
	s := newService()
	createRepo(t, s, "defbr-repo")

	ctx := jsonCtx("DeleteBranch", map[string]any{
		"repositoryName": "defbr-repo",
		"branchName":     "main",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DefaultBranchCannotBeDeletedException")
}

// --- Pull Request Tests ---

func TestCreatePullRequest(t *testing.T) {
	s := newService()
	createRepo(t, s, "pr-repo")

	// Create a feature branch
	resp, _ := s.HandleRequest(jsonCtx("GetBranch", map[string]any{
		"repositoryName": "pr-repo",
		"branchName":     "main",
	}))
	commitID := respBody(t, resp)["branch"].(map[string]any)["commitId"].(string)
	s.HandleRequest(jsonCtx("CreateBranch", map[string]any{
		"repositoryName": "pr-repo",
		"branchName":     "feature",
		"commitId":       commitID,
	}))

	ctx := jsonCtx("CreatePullRequest", map[string]any{
		"title":       "My PR",
		"description": "Fix things",
		"targets": []any{
			map[string]any{
				"repositoryName":       "pr-repo",
				"sourceReference":      "feature",
				"destinationReference": "main",
			},
		},
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	pr := body["pullRequest"].(map[string]any)
	assert.Equal(t, "My PR", pr["title"])
	assert.Equal(t, "OPEN", pr["pullRequestStatus"])
	assert.NotEmpty(t, pr["pullRequestId"])
}

func TestGetPullRequest(t *testing.T) {
	s := newService()
	createRepo(t, s, "getpr-repo")

	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title": "PR",
		"targets": []any{
			map[string]any{
				"repositoryName":       "getpr-repo",
				"sourceReference":      "feature",
				"destinationReference": "main",
			},
		},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	ctx := jsonCtx("GetPullRequest", map[string]any{"pullRequestId": prID})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	assert.Equal(t, prID, body["pullRequest"].(map[string]any)["pullRequestId"])
}

func TestGetPullRequestNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("GetPullRequest", map[string]any{"pullRequestId": "999"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PullRequestDoesNotExistException")
}

func TestListPullRequests(t *testing.T) {
	s := newService()
	createRepo(t, s, "listpr-repo")

	s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "PR 1",
		"targets": []any{map[string]any{"repositoryName": "listpr-repo", "sourceReference": "f1", "destinationReference": "main"}},
	}))
	s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "PR 2",
		"targets": []any{map[string]any{"repositoryName": "listpr-repo", "sourceReference": "f2", "destinationReference": "main"}},
	}))

	ctx := jsonCtx("ListPullRequests", map[string]any{"repositoryName": "listpr-repo"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	ids := body["pullRequestIds"].([]any)
	assert.Len(t, ids, 2)
}

func TestUpdatePullRequestStatus(t *testing.T) {
	s := newService()
	createRepo(t, s, "updpr-repo")

	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "PR",
		"targets": []any{map[string]any{"repositoryName": "updpr-repo", "sourceReference": "f1", "destinationReference": "main"}},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	ctx := jsonCtx("UpdatePullRequestStatus", map[string]any{
		"pullRequestId":     prID,
		"pullRequestStatus": "CLOSED",
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	assert.Equal(t, "CLOSED", body["pullRequest"].(map[string]any)["pullRequestStatus"])
}

func TestMergePullRequestBySquash(t *testing.T) {
	s := newService()
	createRepo(t, s, "merge-repo")

	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "Merge PR",
		"targets": []any{map[string]any{"repositoryName": "merge-repo", "sourceReference": "feature", "destinationReference": "main"}},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	ctx := jsonCtx("MergePullRequestBySquash", map[string]any{
		"pullRequestId":  prID,
		"repositoryName": "merge-repo",
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	pr := body["pullRequest"].(map[string]any)
	assert.Equal(t, "CLOSED", pr["pullRequestStatus"])
	targets := pr["pullRequestTargets"].([]any)
	mergeMetadata := targets[0].(map[string]any)["mergeMetadata"].(map[string]any)
	assert.Equal(t, true, mergeMetadata["isMerged"])
	assert.NotEmpty(t, mergeMetadata["mergeCommitId"])
}

func TestMergeClosedPullRequest(t *testing.T) {
	s := newService()
	createRepo(t, s, "mergeclosed-repo")

	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "PR",
		"targets": []any{map[string]any{"repositoryName": "mergeclosed-repo", "sourceReference": "f1", "destinationReference": "main"}},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	// Close it
	s.HandleRequest(jsonCtx("UpdatePullRequestStatus", map[string]any{
		"pullRequestId":     prID,
		"pullRequestStatus": "CLOSED",
	}))

	// Try to merge
	ctx := jsonCtx("MergePullRequestBySquash", map[string]any{
		"pullRequestId":  prID,
		"repositoryName": "mergeclosed-repo",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PullRequestStatusRequiredException")
}

// --- Commit & Diff Tests ---

func TestGetCommit(t *testing.T) {
	s := newService()
	createRepo(t, s, "commit-repo")

	// Get the initial commit
	resp, _ := s.HandleRequest(jsonCtx("GetBranch", map[string]any{
		"repositoryName": "commit-repo",
		"branchName":     "main",
	}))
	commitID := respBody(t, resp)["branch"].(map[string]any)["commitId"].(string)

	ctx := jsonCtx("GetCommit", map[string]any{
		"repositoryName": "commit-repo",
		"commitId":       commitID,
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	commit := body["commit"].(map[string]any)
	assert.Equal(t, commitID, commit["commitId"])
	assert.Equal(t, "Initial commit", commit["message"])
}

func TestGetCommitNotFound(t *testing.T) {
	s := newService()
	createRepo(t, s, "nocommit-repo")

	ctx := jsonCtx("GetCommit", map[string]any{
		"repositoryName": "nocommit-repo",
		"commitId":       "nonexistent",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CommitDoesNotExistException")
}

func TestGetDifferences(t *testing.T) {
	s := newService()
	createRepo(t, s, "diff-repo")

	ctx := jsonCtx("GetDifferences", map[string]any{
		"repositoryName":        "diff-repo",
		"beforeCommitSpecifier": "abc",
		"afterCommitSpecifier":  "def",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	diffs := body["differences"].([]any)
	assert.Len(t, diffs, 1)
	diff := diffs[0].(map[string]any)
	assert.Equal(t, "M", diff["changeType"])
}

// --- UpdatePullRequestTitle ---

func TestUpdatePullRequestTitle(t *testing.T) {
	s := newService()
	createRepo(t, s, "title-repo")

	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "Old Title",
		"targets": []any{map[string]any{"repositoryName": "title-repo", "sourceReference": "feature", "destinationReference": "main"}},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	ctx := jsonCtx("UpdatePullRequestTitle", map[string]any{
		"pullRequestId": prID,
		"title":         "New Title",
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	assert.Equal(t, "New Title", body["pullRequest"].(map[string]any)["title"])
}

func TestUpdatePullRequestTitleMissingTitle(t *testing.T) {
	s := newService()
	createRepo(t, s, "notitle-repo")
	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "PR",
		"targets": []any{map[string]any{"repositoryName": "notitle-repo", "sourceReference": "f", "destinationReference": "main"}},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	ctx := jsonCtx("UpdatePullRequestTitle", map[string]any{
		"pullRequestId": prID,
		"title":         "",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

// --- MergePullRequestByFastForward ---

func TestMergePullRequestByFastForward(t *testing.T) {
	s := newService()
	createRepo(t, s, "ffmerge-repo")

	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "FF PR",
		"targets": []any{map[string]any{"repositoryName": "ffmerge-repo", "sourceReference": "feature", "destinationReference": "main"}},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	ctx := jsonCtx("MergePullRequestByFastForward", map[string]any{
		"pullRequestId":  prID,
		"repositoryName": "ffmerge-repo",
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	pr := body["pullRequest"].(map[string]any)
	assert.Equal(t, "MERGED", pr["pullRequestStatus"])
	targets := pr["pullRequestTargets"].([]any)
	mergeMetadata := targets[0].(map[string]any)["mergeMetadata"].(map[string]any)
	assert.Equal(t, true, mergeMetadata["isMerged"])
}

func TestMergePullRequestByFastForwardAlreadyMerged(t *testing.T) {
	s := newService()
	createRepo(t, s, "ffmerge2-repo")

	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "FF PR",
		"targets": []any{map[string]any{"repositoryName": "ffmerge2-repo", "sourceReference": "feature", "destinationReference": "main"}},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	// Merge it
	s.HandleRequest(jsonCtx("MergePullRequestByFastForward", map[string]any{
		"pullRequestId":  prID,
		"repositoryName": "ffmerge2-repo",
	}))

	// Try to merge again
	_, err := s.HandleRequest(jsonCtx("MergePullRequestByFastForward", map[string]any{
		"pullRequestId":  prID,
		"repositoryName": "ffmerge2-repo",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PullRequestStatusRequiredException")
}

// --- PostCommentForPullRequest ---

func TestPostCommentForPullRequest(t *testing.T) {
	s := newService()
	createRepo(t, s, "comment-repo")

	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "Commented PR",
		"targets": []any{map[string]any{"repositoryName": "comment-repo", "sourceReference": "feature", "destinationReference": "main"}},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	ctx := jsonCtx("PostCommentForPullRequest", map[string]any{
		"pullRequestId":  prID,
		"repositoryName": "comment-repo",
		"content":        "LGTM!",
		"beforeCommitId": "abc123",
		"afterCommitId":  "def456",
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	comment := body["comment"].(map[string]any)
	assert.Equal(t, "LGTM!", comment["content"])
	assert.NotEmpty(t, comment["commentId"])
	assert.Equal(t, prID, comment["pullRequestId"])
}

func TestPostCommentForPullRequestMissingContent(t *testing.T) {
	s := newService()
	createRepo(t, s, "nocomment-repo")
	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "PR",
		"targets": []any{map[string]any{"repositoryName": "nocomment-repo", "sourceReference": "f", "destinationReference": "main"}},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	_, err := s.HandleRequest(jsonCtx("PostCommentForPullRequest", map[string]any{
		"pullRequestId":  prID,
		"repositoryName": "nocomment-repo",
		"content":        "",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

// --- GetCommentsForPullRequest ---

func TestGetCommentsForPullRequest(t *testing.T) {
	s := newService()
	createRepo(t, s, "getcomments-repo")

	resp, _ := s.HandleRequest(jsonCtx("CreatePullRequest", map[string]any{
		"title":   "PR with Comments",
		"targets": []any{map[string]any{"repositoryName": "getcomments-repo", "sourceReference": "feature", "destinationReference": "main"}},
	}))
	prID := respBody(t, resp)["pullRequest"].(map[string]any)["pullRequestId"].(string)

	s.HandleRequest(jsonCtx("PostCommentForPullRequest", map[string]any{
		"pullRequestId": prID, "repositoryName": "getcomments-repo", "content": "Comment 1",
	}))
	s.HandleRequest(jsonCtx("PostCommentForPullRequest", map[string]any{
		"pullRequestId": prID, "repositoryName": "getcomments-repo", "content": "Comment 2",
	}))

	ctx := jsonCtx("GetCommentsForPullRequest", map[string]any{"pullRequestId": prID})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp2)
	comments := body["commentsForPullRequestData"].([]any)
	assert.Len(t, comments, 2)
}

func TestGetCommentsForPullRequestNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("GetCommentsForPullRequest", map[string]any{"pullRequestId": "999"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PullRequestDoesNotExistException")
}

// --- PutRepositoryTriggers / GetRepositoryTriggers ---

func TestPutAndGetRepositoryTriggers(t *testing.T) {
	s := newService()
	createRepo(t, s, "trigger-repo")

	ctx := jsonCtx("PutRepositoryTriggers", map[string]any{
		"repositoryName": "trigger-repo",
		"triggers": []any{
			map[string]any{
				"name":           "my-trigger",
				"destinationArn": "arn:aws:sns:us-east-1:123456789012:my-topic",
				"branches":       []any{"main"},
				"events":         []any{"all"},
			},
		},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["configurationId"])

	ctx2 := jsonCtx("GetRepositoryTriggers", map[string]any{"repositoryName": "trigger-repo"})
	resp2, err2 := s.HandleRequest(ctx2)
	require.NoError(t, err2)
	body2 := respBody(t, resp2)
	triggers := body2["triggers"].([]any)
	assert.Len(t, triggers, 1)
	trigger := triggers[0].(map[string]any)
	assert.Equal(t, "my-trigger", trigger["name"])
}

func TestGetRepositoryTriggersEmpty(t *testing.T) {
	s := newService()
	createRepo(t, s, "emptytrigger-repo")

	ctx := jsonCtx("GetRepositoryTriggers", map[string]any{"repositoryName": "emptytrigger-repo"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Empty(t, body["triggers"])
}

func TestPutRepositoryTriggersRepoNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("PutRepositoryTriggers", map[string]any{
		"repositoryName": "nope",
		"triggers":       []any{},
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RepositoryDoesNotExistException")
}

// --- TagResource / UntagResource / ListTagsForResource ---

func TestCodeCommitTagging(t *testing.T) {
	s := newService()
	body := createRepo(t, s, "tagged-repo")
	arn := body["repositoryMetadata"].(map[string]any)["Arn"].(string)

	ctx := jsonCtx("TagResource", map[string]any{
		"resourceArn": arn,
		"tags":        map[string]any{"env": "prod", "team": "backend"},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	ctx2 := jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn})
	resp2, err2 := s.HandleRequest(ctx2)
	require.NoError(t, err2)
	body2 := respBody(t, resp2)
	tags := body2["tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
	assert.Equal(t, "backend", tags["team"])

	ctx3 := jsonCtx("UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []any{"env"},
	})
	resp3, err3 := s.HandleRequest(ctx3)
	require.NoError(t, err3)
	assert.Equal(t, 200, resp3.StatusCode)

	resp4, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	body4 := respBody(t, resp4)
	tags4 := body4["tags"].(map[string]any)
	assert.Len(t, tags4, 1)
	assert.Equal(t, "backend", tags4["team"])
}

func TestTagResourceMissingArn(t *testing.T) {
	s := newService()
	ctx := jsonCtx("TagResource", map[string]any{
		"tags": map[string]any{"key": "val"},
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

// --- Invalid Action ---

func TestInvalidAction(t *testing.T) {
	s := newService()
	ctx := jsonCtx("BogusAction", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

// --- Service Metadata ---

func TestServiceName(t *testing.T) {
	s := newService()
	assert.Equal(t, "codecommit", s.Name())
}

func TestHealthCheck(t *testing.T) {
	s := newService()
	assert.NoError(t, s.HealthCheck())
}
