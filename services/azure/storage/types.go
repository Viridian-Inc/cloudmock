package storage

import "time"

const (
	controlPlaneAPIVersion = "2024-01-01"
	dataPlaneAPIVersion    = "2023-11-03"
)

type StorageAccount struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Type       string                   `json:"type"`
	Location   string                   `json:"location"`
	Tags       map[string]string        `json:"tags,omitempty"`
	Kind       string                   `json:"kind"`
	SKU        StorageSKU               `json:"sku"`
	Properties StorageAccountProperties `json:"properties"`
}

type StorageSKU struct {
	Name string `json:"name"`
}

type StorageAccountProperties struct {
	ProvisioningState string            `json:"provisioningState"`
	PrimaryEndpoints  map[string]string `json:"primaryEndpoints"`
}

type StorageAccountListKeysResult struct {
	Keys []StorageAccountKey `json:"keys"`
}

type StorageAccountKey struct {
	KeyName     string `json:"keyName"`
	Permissions string `json:"permissions"`
	Value       string `json:"value"`
}

type blobContainer struct {
	Name           string
	Metadata       map[string]string
	ETag           string
	LastModified   time.Time
	LeaseID        string
	LeaseState     string
	LeaseDuration  string
	PublicAccess   string
	AccessPolicies []blobContainerSignedIdentifier
	Blobs          map[string]blobObject
	StagedBlocks   map[string]map[string]blobBlock
	Snapshots      map[string]map[string]blobObject
}

type blobObject struct {
	Name               string
	BlobType           string
	Content            []byte
	ContentType        string
	CacheControl       string
	ContentEncoding    string
	ContentLanguage    string
	ContentDisposition string
	ContentMD5         string
	Metadata           map[string]string
	Tags               map[string]string
	ETag               string
	LastModified       time.Time
	CommittedBlocks    []blobBlock
	AppendBlockCount   int
	PageRanges         []fileRange
	SequenceNumber     string
	LeaseID            string
	LeaseState         string
	CopyID             string
	CopyStatus         string
	CopySource         string
	CopyProgress       string
	CopyCompletionTime time.Time
	AccessTier         string
	AccessTierChanged  time.Time
	ArchiveStatus      string
	RehydratePriority  string
}

type blobBlock struct {
	ID      string
	Content []byte
}

type queue struct {
	Name           string
	Metadata       map[string]string
	CreatedVersion string
	AccessPolicies []queueSignedIdentifier
	Messages       []queueMessage
}

type queueMessage struct {
	ID              string
	Text            string
	PopReceipt      string
	DequeueCount    int
	InsertionTime   time.Time
	ExpirationTime  time.Time
	TimeNextVisible time.Time
}

type table struct {
	Name     string
	Entities map[string]tableEntity
}

type tableEntity struct {
	Properties   map[string]any
	ETag         string
	LastModified time.Time
}

type fileShare struct {
	Name             string
	Metadata         map[string]string
	ETag             string
	LastModified     time.Time
	Quota            string
	AccessTier       string
	EnabledProtocols string
	RootSquash       string
	SnapshotVDir     string
	LeaseID          string
	LeaseState       string
	AccessPolicies   []fileShareSignedIdentifier
	FilePermissions  map[string]fileSharePermission
	Directories      map[string]fileDirectory
	Files            map[string]fileObject
	Snapshots        map[string]fileShare
}

type fileSharePermission struct {
	Permission string
	Format     string
}

type fileShareSignedIdentifier struct {
	ID           string                `xml:"Id"`
	AccessPolicy fileShareAccessPolicy `xml:"AccessPolicy"`
}

type fileShareAccessPolicy struct {
	Start      string `xml:"Start,omitempty"`
	Expiry     string `xml:"Expiry,omitempty"`
	Permission string `xml:"Permission,omitempty"`
}

type fileDirectory struct {
	Path          string
	Metadata      map[string]string
	Attributes    string
	CreationTime  string
	LastWriteTime string
	ChangeTime    string
	ETag          string
	LastModified  time.Time
}

type fileObject struct {
	Path               string
	Content            []byte
	Ranges             []fileRange
	ContentType        string
	CacheControl       string
	ContentEncoding    string
	ContentLanguage    string
	ContentDisposition string
	ContentMD5         string
	Metadata           map[string]string
	ETag               string
	LastModified       time.Time
	LeaseID            string
	LeaseState         string
	CopyID             string
	CopyStatus         string
	CopySource         string
	CopyProgress       string
	CopyCompletionTime time.Time
}

type fileRange struct {
	Start int
	End   int
}
