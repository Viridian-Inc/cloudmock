package codecommit

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newCommitID() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", sha1.Sum(b))
}

// Pull request status constants.
const (
	PRStatusOpen   = "OPEN"
	PRStatusClosed = "CLOSED"
	PRStatusMerged = "MERGED"
)

// Repository represents a CodeCommit repository.
type Repository struct {
	ID            string
	Name          string
	ARN           string
	CloneURLHTTP  string
	CloneURLSSH   string
	Description   string
	DefaultBranch string
	CreatedAt     time.Time
	LastModified  time.Time
	Tags          map[string]string
}

// Branch represents a branch within a repository.
type Branch struct {
	Name     string
	CommitID string
}

// Commit represents a single commit.
type Commit struct {
	CommitID       string
	TreeID         string
	ParentIDs      []string
	Author         UserInfo
	Committer      UserInfo
	Message        string
	AdditionalData string
}

// UserInfo represents commit author/committer information.
type UserInfo struct {
	Name  string
	Email string
	Date  time.Time
}

// PullRequest represents a pull request.
type PullRequest struct {
	ID                string
	Title             string
	Description       string
	Status            string
	AuthorARN         string
	CreationDate      time.Time
	LastActivityDate  time.Time
	RepositoryName    string
	SourceReference   string
	DestinationReference string
	MergeMetadata     *MergeMetadata
}

// MergeMetadata holds merge state for a pull request.
type MergeMetadata struct {
	IsMerged  bool
	MergedBy  string
	MergeCommitID string
}

// Difference represents a file difference between two commits.
type Difference struct {
	BeforeBlob *BlobInfo
	AfterBlob  *BlobInfo
	ChangeType string
}

// BlobInfo describes a file blob in a diff.
type BlobInfo struct {
	BlobID string
	Path   string
	Mode   string
}

// Comment represents a comment on a pull request.
type Comment struct {
	CommentID       string
	Content         string
	PullRequestID   string
	RepositoryName  string
	BeforeCommitID  string
	AfterCommitID   string
	FilePath        string
	AuthorARN       string
	CreationDate    time.Time
	LastModified    time.Time
	Deleted         bool
}

// RepositoryTrigger represents a trigger on a repository.
type RepositoryTrigger struct {
	Name            string
	DestinationARN  string
	CustomData      string
	Branches        []string
	Events          []string
}

// Store is the in-memory store for all CodeCommit resources.
type Store struct {
	mu           sync.RWMutex
	accountID    string
	region       string
	repositories map[string]*Repository
	branches     map[string]map[string]*Branch   // repoName -> branchName -> Branch
	commits      map[string]map[string]*Commit   // repoName -> commitID -> Commit
	pullRequests map[string]*PullRequest         // prID -> PullRequest
	comments     map[string][]*Comment           // prID -> []Comment
	triggers     map[string][]RepositoryTrigger  // repoName -> []Trigger
	prCounter    int
	commentCounter int
	tags         map[string]map[string]string
}

// NewStore creates an empty CodeCommit store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:    accountID,
		region:       region,
		repositories: make(map[string]*Repository),
		branches:     make(map[string]map[string]*Branch),
		commits:      make(map[string]map[string]*Commit),
		pullRequests: make(map[string]*PullRequest),
		comments:     make(map[string][]*Comment),
		triggers:     make(map[string][]RepositoryTrigger),
		tags:         make(map[string]map[string]string),
	}
}

// ---- ARN builders ----

func (s *Store) repositoryARN(name string) string {
	return fmt.Sprintf("arn:aws:codecommit:%s:%s:%s", s.region, s.accountID, name)
}

func (s *Store) cloneURLHTTP(name string) string {
	return fmt.Sprintf("https://git-codecommit.%s.amazonaws.com/v1/repos/%s", s.region, name)
}

func (s *Store) cloneURLSSH(name string) string {
	return fmt.Sprintf("ssh://git-codecommit.%s.amazonaws.com/v1/repos/%s", s.region, name)
}

// ---- Repository operations ----

func (s *Store) CreateRepository(name, description string, tags map[string]string) (*Repository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, service.ErrValidation("Repository name is required.")
	}
	if _, exists := s.repositories[name]; exists {
		return nil, service.NewAWSError("RepositoryNameExistsException",
			fmt.Sprintf("Repository already exists: %s", name), http.StatusConflict)
	}

	if tags == nil {
		tags = make(map[string]string)
	}
	now := time.Now().UTC()
	repo := &Repository{
		ID:           newUUID(),
		Name:         name,
		ARN:          s.repositoryARN(name),
		CloneURLHTTP: s.cloneURLHTTP(name),
		CloneURLSSH:  s.cloneURLSSH(name),
		Description:  description,
		CreatedAt:    now,
		LastModified: now,
		Tags:         tags,
	}
	s.repositories[name] = repo

	// Initialize with a default branch and initial commit
	commitID := newCommitID()
	s.commits[name] = map[string]*Commit{
		commitID: {
			CommitID:  commitID,
			TreeID:    newCommitID(),
			Author:    UserInfo{Name: "System", Email: "system@cloudmock", Date: now},
			Committer: UserInfo{Name: "System", Email: "system@cloudmock", Date: now},
			Message:   "Initial commit",
		},
	}
	s.branches[name] = map[string]*Branch{
		"main": {Name: "main", CommitID: commitID},
	}
	repo.DefaultBranch = "main"

	return repo, nil
}

func (s *Store) GetRepository(name string) (*Repository, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.repositories[name]
	if !ok {
		return nil, service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", name), http.StatusNotFound)
	}
	return repo, nil
}

func (s *Store) ListRepositories() []*Repository {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Repository, 0, len(s.repositories))
	for _, r := range s.repositories {
		result = append(result, r)
	}
	return result
}

func (s *Store) DeleteRepository(name string) (*Repository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.repositories[name]
	if !ok {
		return nil, service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", name), http.StatusNotFound)
	}
	delete(s.repositories, name)
	delete(s.branches, name)
	delete(s.commits, name)
	return repo, nil
}

func (s *Store) UpdateRepositoryName(oldName, newName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.repositories[oldName]
	if !ok {
		return service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", oldName), http.StatusNotFound)
	}
	if _, exists := s.repositories[newName]; exists {
		return service.NewAWSError("RepositoryNameExistsException",
			fmt.Sprintf("Repository already exists: %s", newName), http.StatusConflict)
	}

	delete(s.repositories, oldName)
	repo.Name = newName
	repo.ARN = s.repositoryARN(newName)
	repo.CloneURLHTTP = s.cloneURLHTTP(newName)
	repo.CloneURLSSH = s.cloneURLSSH(newName)
	repo.LastModified = time.Now().UTC()
	s.repositories[newName] = repo

	// Move branches and commits
	if b, ok := s.branches[oldName]; ok {
		s.branches[newName] = b
		delete(s.branches, oldName)
	}
	if c, ok := s.commits[oldName]; ok {
		s.commits[newName] = c
		delete(s.commits, oldName)
	}
	return nil
}

func (s *Store) UpdateRepositoryDescription(name, description string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.repositories[name]
	if !ok {
		return service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", name), http.StatusNotFound)
	}
	repo.Description = description
	repo.LastModified = time.Now().UTC()
	return nil
}

// ---- Branch operations ----

func (s *Store) CreateBranch(repoName, branchName, commitID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.repositories[repoName]; !ok {
		return service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", repoName), http.StatusNotFound)
	}

	branches := s.branches[repoName]
	if branches == nil {
		branches = make(map[string]*Branch)
		s.branches[repoName] = branches
	}

	if _, exists := branches[branchName]; exists {
		return service.NewAWSError("BranchNameExistsException",
			fmt.Sprintf("Branch already exists: %s", branchName), http.StatusConflict)
	}

	// Verify commit exists
	commits := s.commits[repoName]
	if commits == nil || commits[commitID] == nil {
		return service.NewAWSError("CommitDoesNotExistException",
			fmt.Sprintf("Commit does not exist: %s", commitID), http.StatusNotFound)
	}

	branches[branchName] = &Branch{Name: branchName, CommitID: commitID}
	return nil
}

func (s *Store) GetBranch(repoName, branchName string) (*Branch, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.repositories[repoName]; !ok {
		return nil, service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", repoName), http.StatusNotFound)
	}

	branches := s.branches[repoName]
	if branches == nil {
		return nil, service.NewAWSError("BranchDoesNotExistException",
			fmt.Sprintf("Branch does not exist: %s", branchName), http.StatusNotFound)
	}

	branch, ok := branches[branchName]
	if !ok {
		return nil, service.NewAWSError("BranchDoesNotExistException",
			fmt.Sprintf("Branch does not exist: %s", branchName), http.StatusNotFound)
	}
	return branch, nil
}

func (s *Store) ListBranches(repoName string) ([]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.repositories[repoName]; !ok {
		return nil, service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", repoName), http.StatusNotFound)
	}

	branches := s.branches[repoName]
	names := make([]string, 0, len(branches))
	for name := range branches {
		names = append(names, name)
	}
	return names, nil
}

func (s *Store) DeleteBranch(repoName, branchName string) (*Branch, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.repositories[repoName]; !ok {
		return nil, service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", repoName), http.StatusNotFound)
	}

	branches := s.branches[repoName]
	if branches == nil {
		return nil, service.NewAWSError("BranchDoesNotExistException",
			fmt.Sprintf("Branch does not exist: %s", branchName), http.StatusNotFound)
	}

	branch, ok := branches[branchName]
	if !ok {
		return nil, service.NewAWSError("BranchDoesNotExistException",
			fmt.Sprintf("Branch does not exist: %s", branchName), http.StatusNotFound)
	}

	repo := s.repositories[repoName]
	if repo.DefaultBranch == branchName {
		return nil, service.NewAWSError("DefaultBranchCannotBeDeletedException",
			"Cannot delete the default branch.", http.StatusBadRequest)
	}

	delete(branches, branchName)
	return branch, nil
}

// ---- Commit operations ----

func (s *Store) GetCommit(repoName, commitID string) (*Commit, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.repositories[repoName]; !ok {
		return nil, service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", repoName), http.StatusNotFound)
	}

	commits := s.commits[repoName]
	if commits == nil {
		return nil, service.NewAWSError("CommitDoesNotExistException",
			fmt.Sprintf("Commit does not exist: %s", commitID), http.StatusNotFound)
	}

	commit, ok := commits[commitID]
	if !ok {
		return nil, service.NewAWSError("CommitDoesNotExistException",
			fmt.Sprintf("Commit does not exist: %s", commitID), http.StatusNotFound)
	}
	return commit, nil
}

func (s *Store) GetDifferences(repoName, beforeCommitSpec, afterCommitSpec string) ([]Difference, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.repositories[repoName]; !ok {
		return nil, service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", repoName), http.StatusNotFound)
	}

	// Return synthetic differences
	return []Difference{
		{
			ChangeType: "M",
			BeforeBlob: &BlobInfo{BlobID: newCommitID(), Path: "README.md", Mode: "100644"},
			AfterBlob:  &BlobInfo{BlobID: newCommitID(), Path: "README.md", Mode: "100644"},
		},
	}, nil
}

// ---- Pull Request operations ----

func (s *Store) CreatePullRequest(repoName, title, description, sourceRef, destRef string) (*PullRequest, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.repositories[repoName]; !ok {
		return nil, service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", repoName), http.StatusNotFound)
	}

	if title == "" {
		return nil, service.ErrValidation("Pull request title is required.")
	}

	s.prCounter++
	now := time.Now().UTC()
	pr := &PullRequest{
		ID:                   fmt.Sprintf("%d", s.prCounter),
		Title:                title,
		Description:          description,
		Status:               PRStatusOpen,
		AuthorARN:            fmt.Sprintf("arn:aws:iam::%s:root", s.accountID),
		CreationDate:         now,
		LastActivityDate:     now,
		RepositoryName:       repoName,
		SourceReference:      sourceRef,
		DestinationReference: destRef,
	}
	s.pullRequests[pr.ID] = pr
	return pr, nil
}

func (s *Store) GetPullRequest(prID string) (*PullRequest, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pr, ok := s.pullRequests[prID]
	if !ok {
		return nil, service.NewAWSError("PullRequestDoesNotExistException",
			fmt.Sprintf("Pull request does not exist: %s", prID), http.StatusNotFound)
	}
	return pr, nil
}

func (s *Store) ListPullRequests(repoName, status string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ids []string
	for id, pr := range s.pullRequests {
		if repoName != "" && pr.RepositoryName != repoName {
			continue
		}
		if status != "" && pr.Status != status {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func (s *Store) UpdatePullRequestStatus(prID, status string) (*PullRequest, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pr, ok := s.pullRequests[prID]
	if !ok {
		return nil, service.NewAWSError("PullRequestDoesNotExistException",
			fmt.Sprintf("Pull request does not exist: %s", prID), http.StatusNotFound)
	}

	pr.Status = status
	pr.LastActivityDate = time.Now().UTC()
	return pr, nil
}

func (s *Store) UpdatePullRequestTitle(prID, title string) (*PullRequest, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pr, ok := s.pullRequests[prID]
	if !ok {
		return nil, service.NewAWSError("PullRequestDoesNotExistException",
			fmt.Sprintf("Pull request does not exist: %s", prID), http.StatusNotFound)
	}
	if title == "" {
		return nil, service.ErrValidation("Pull request title is required.")
	}
	pr.Title = title
	pr.LastActivityDate = time.Now().UTC()
	return pr, nil
}

func (s *Store) MergePullRequestByFastForward(prID, repoName string) (*PullRequest, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pr, ok := s.pullRequests[prID]
	if !ok {
		return nil, service.NewAWSError("PullRequestDoesNotExistException",
			fmt.Sprintf("Pull request does not exist: %s", prID), http.StatusNotFound)
	}
	if pr.Status != PRStatusOpen {
		return nil, service.NewAWSError("PullRequestStatusRequiredException",
			"Pull request must be open to merge.", http.StatusBadRequest)
	}

	mergeCommitID := newCommitID()
	pr.Status = PRStatusMerged
	pr.MergeMetadata = &MergeMetadata{
		IsMerged:      true,
		MergedBy:      fmt.Sprintf("arn:aws:iam::%s:root", s.accountID),
		MergeCommitID: mergeCommitID,
	}
	pr.LastActivityDate = time.Now().UTC()

	if s.commits[repoName] == nil {
		s.commits[repoName] = make(map[string]*Commit)
	}
	now := time.Now().UTC()
	s.commits[repoName][mergeCommitID] = &Commit{
		CommitID:  mergeCommitID,
		TreeID:    newCommitID(),
		Author:    UserInfo{Name: "System", Email: "system@cloudmock", Date: now},
		Committer: UserInfo{Name: "System", Email: "system@cloudmock", Date: now},
		Message:   fmt.Sprintf("Fast-forward merge PR #%s: %s", prID, pr.Title),
	}

	branches := s.branches[repoName]
	if branches != nil {
		if destBranch, ok := branches[pr.DestinationReference]; ok {
			destBranch.CommitID = mergeCommitID
		}
	}

	return pr, nil
}

func (s *Store) PostCommentForPullRequest(prID, repoName, content, beforeCommitID, afterCommitID, filePath string) (*Comment, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pullRequests[prID]; !ok {
		return nil, service.NewAWSError("PullRequestDoesNotExistException",
			fmt.Sprintf("Pull request does not exist: %s", prID), http.StatusNotFound)
	}
	if content == "" {
		return nil, service.ErrValidation("Comment content is required.")
	}

	s.commentCounter++
	now := time.Now().UTC()
	comment := &Comment{
		CommentID:      fmt.Sprintf("%d", s.commentCounter),
		Content:        content,
		PullRequestID:  prID,
		RepositoryName: repoName,
		BeforeCommitID: beforeCommitID,
		AfterCommitID:  afterCommitID,
		FilePath:       filePath,
		AuthorARN:      fmt.Sprintf("arn:aws:iam::%s:root", s.accountID),
		CreationDate:   now,
		LastModified:   now,
	}
	s.comments[prID] = append(s.comments[prID], comment)
	return comment, nil
}

func (s *Store) GetCommentsForPullRequest(prID string) ([]*Comment, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.pullRequests[prID]; !ok {
		return nil, service.NewAWSError("PullRequestDoesNotExistException",
			fmt.Sprintf("Pull request does not exist: %s", prID), http.StatusNotFound)
	}
	return s.comments[prID], nil
}

func (s *Store) PutRepositoryTriggers(repoName string, triggers []RepositoryTrigger) (string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.repositories[repoName]; !ok {
		return "", service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", repoName), http.StatusNotFound)
	}
	s.triggers[repoName] = triggers
	// Return a configuration ID (hash-like)
	return newCommitID()[:8], nil
}

func (s *Store) GetRepositoryTriggers(repoName string) ([]RepositoryTrigger, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.repositories[repoName]; !ok {
		return nil, service.NewAWSError("RepositoryDoesNotExistException",
			fmt.Sprintf("Repository does not exist: %s", repoName), http.StatusNotFound)
	}
	return s.triggers[repoName], nil
}

func (s *Store) MergePullRequestBySquash(prID, repoName string) (*PullRequest, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pr, ok := s.pullRequests[prID]
	if !ok {
		return nil, service.NewAWSError("PullRequestDoesNotExistException",
			fmt.Sprintf("Pull request does not exist: %s", prID), http.StatusNotFound)
	}

	if pr.Status != PRStatusOpen {
		return nil, service.NewAWSError("PullRequestStatusRequiredException",
			"Pull request must be open to merge.", http.StatusBadRequest)
	}

	mergeCommitID := newCommitID()
	pr.Status = PRStatusClosed
	pr.MergeMetadata = &MergeMetadata{
		IsMerged:      true,
		MergedBy:      fmt.Sprintf("arn:aws:iam::%s:root", s.accountID),
		MergeCommitID: mergeCommitID,
	}
	pr.LastActivityDate = time.Now().UTC()

	// Create merge commit in repo
	if s.commits[repoName] == nil {
		s.commits[repoName] = make(map[string]*Commit)
	}
	now := time.Now().UTC()
	s.commits[repoName][mergeCommitID] = &Commit{
		CommitID:  mergeCommitID,
		TreeID:    newCommitID(),
		Author:    UserInfo{Name: "System", Email: "system@cloudmock", Date: now},
		Committer: UserInfo{Name: "System", Email: "system@cloudmock", Date: now},
		Message:   fmt.Sprintf("Squash merge PR #%s: %s", prID, pr.Title),
	}

	// Update destination branch
	branches := s.branches[repoName]
	if branches != nil {
		if destBranch, ok := branches[pr.DestinationReference]; ok {
			destBranch.CommitID = mergeCommitID
		}
	}

	return pr, nil
}

// ---- Tag operations ----

func (s *Store) TagResource(arn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
	return nil
}

func (s *Store) UntagResource(arn string, keys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.tags[arn]
	if m == nil {
		return nil
	}
	for _, k := range keys {
		delete(m, k)
	}
	return nil
}

func (s *Store) ListTagsForResource(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m := s.tags[arn]
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
