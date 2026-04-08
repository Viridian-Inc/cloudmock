package lexmodels

import (
	gojson "github.com/goccy/go-json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type BotAliasMetadata struct {
	BotName *string `json:"botName,omitempty"`
	BotVersion *string `json:"botVersion,omitempty"`
	Checksum *string `json:"checksum,omitempty"`
	ConversationLogs *ConversationLogsResponse `json:"conversationLogs,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
}

type BotChannelAssociation struct {
	BotAlias *string `json:"botAlias,omitempty"`
	BotConfiguration map[string]string `json:"botConfiguration,omitempty"`
	BotName *string `json:"botName,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	FailureReason *string `json:"failureReason,omitempty"`
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
	Type *string `json:"type,omitempty"`
}

type BotMetadata struct {
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
	Version *string `json:"version,omitempty"`
}

type BuiltinIntentMetadata struct {
	Signature *string `json:"signature,omitempty"`
	SupportedLocales []string `json:"supportedLocales,omitempty"`
}

type BuiltinIntentSlot struct {
	Name *string `json:"name,omitempty"`
}

type BuiltinSlotTypeMetadata struct {
	Signature *string `json:"signature,omitempty"`
	SupportedLocales []string `json:"supportedLocales,omitempty"`
}

type CodeHook struct {
	MessageVersion string `json:"messageVersion,omitempty"`
	Uri string `json:"uri,omitempty"`
}

type ConversationLogsRequest struct {
	IamRoleArn string `json:"iamRoleArn,omitempty"`
	LogSettings []LogSettingsRequest `json:"logSettings,omitempty"`
}

type ConversationLogsResponse struct {
	IamRoleArn *string `json:"iamRoleArn,omitempty"`
	LogSettings []LogSettingsResponse `json:"logSettings,omitempty"`
}

type CreateBotVersionRequest struct {
	Checksum *string `json:"checksum,omitempty"`
	Name string `json:"name,omitempty"`
}

type CreateBotVersionResponse struct {
	AbortStatement *Statement `json:"abortStatement,omitempty"`
	Checksum *string `json:"checksum,omitempty"`
	ChildDirected bool `json:"childDirected,omitempty"`
	ClarificationPrompt *Prompt `json:"clarificationPrompt,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	DetectSentiment bool `json:"detectSentiment,omitempty"`
	EnableModelImprovements bool `json:"enableModelImprovements,omitempty"`
	FailureReason *string `json:"failureReason,omitempty"`
	IdleSessionTTLInSeconds int `json:"idleSessionTTLInSeconds,omitempty"`
	Intents []Intent `json:"intents,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Locale *string `json:"locale,omitempty"`
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
	Version *string `json:"version,omitempty"`
	VoiceId *string `json:"voiceId,omitempty"`
}

type CreateIntentVersionRequest struct {
	Checksum *string `json:"checksum,omitempty"`
	Name string `json:"name,omitempty"`
}

type CreateIntentVersionResponse struct {
	Checksum *string `json:"checksum,omitempty"`
	ConclusionStatement *Statement `json:"conclusionStatement,omitempty"`
	ConfirmationPrompt *Prompt `json:"confirmationPrompt,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	DialogCodeHook *CodeHook `json:"dialogCodeHook,omitempty"`
	FollowUpPrompt *FollowUpPrompt `json:"followUpPrompt,omitempty"`
	FulfillmentActivity *FulfillmentActivity `json:"fulfillmentActivity,omitempty"`
	InputContexts []InputContext `json:"inputContexts,omitempty"`
	KendraConfiguration *KendraConfiguration `json:"kendraConfiguration,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
	OutputContexts []OutputContext `json:"outputContexts,omitempty"`
	ParentIntentSignature *string `json:"parentIntentSignature,omitempty"`
	RejectionStatement *Statement `json:"rejectionStatement,omitempty"`
	SampleUtterances []string `json:"sampleUtterances,omitempty"`
	Slots []Slot `json:"slots,omitempty"`
	Version *string `json:"version,omitempty"`
}

type CreateSlotTypeVersionRequest struct {
	Checksum *string `json:"checksum,omitempty"`
	Name string `json:"name,omitempty"`
}

type CreateSlotTypeVersionResponse struct {
	Checksum *string `json:"checksum,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	EnumerationValues []EnumerationValue `json:"enumerationValues,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
	ParentSlotTypeSignature *string `json:"parentSlotTypeSignature,omitempty"`
	SlotTypeConfigurations []SlotTypeConfiguration `json:"slotTypeConfigurations,omitempty"`
	ValueSelectionStrategy *string `json:"valueSelectionStrategy,omitempty"`
	Version *string `json:"version,omitempty"`
}

type DeleteBotAliasRequest struct {
	BotName string `json:"botName,omitempty"`
	Name string `json:"name,omitempty"`
}

type DeleteBotChannelAssociationRequest struct {
	BotAlias string `json:"aliasName,omitempty"`
	BotName string `json:"botName,omitempty"`
	Name string `json:"name,omitempty"`
}

type DeleteBotRequest struct {
	Name string `json:"name,omitempty"`
}

type DeleteBotVersionRequest struct {
	Name string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type DeleteIntentRequest struct {
	Name string `json:"name,omitempty"`
}

type DeleteIntentVersionRequest struct {
	Name string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type DeleteSlotTypeRequest struct {
	Name string `json:"name,omitempty"`
}

type DeleteSlotTypeVersionRequest struct {
	Name string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type DeleteUtterancesRequest struct {
	BotName string `json:"botName,omitempty"`
	UserId string `json:"userId,omitempty"`
}

type EnumerationValue struct {
	Synonyms []string `json:"synonyms,omitempty"`
	Value string `json:"value,omitempty"`
}

type FollowUpPrompt struct {
	Prompt Prompt `json:"prompt,omitempty"`
	RejectionStatement Statement `json:"rejectionStatement,omitempty"`
}

type FulfillmentActivity struct {
	CodeHook *CodeHook `json:"codeHook,omitempty"`
	Type string `json:"type,omitempty"`
}

type GetBotAliasRequest struct {
	BotName string `json:"botName,omitempty"`
	Name string `json:"name,omitempty"`
}

type GetBotAliasResponse struct {
	BotName *string `json:"botName,omitempty"`
	BotVersion *string `json:"botVersion,omitempty"`
	Checksum *string `json:"checksum,omitempty"`
	ConversationLogs *ConversationLogsResponse `json:"conversationLogs,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
}

type GetBotAliasesRequest struct {
	BotName string `json:"botName,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NameContains *string `json:"nameContains,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetBotAliasesResponse struct {
	BotAliases []BotAliasMetadata `json:"BotAliases,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetBotChannelAssociationRequest struct {
	BotAlias string `json:"aliasName,omitempty"`
	BotName string `json:"botName,omitempty"`
	Name string `json:"name,omitempty"`
}

type GetBotChannelAssociationResponse struct {
	BotAlias *string `json:"botAlias,omitempty"`
	BotConfiguration map[string]string `json:"botConfiguration,omitempty"`
	BotName *string `json:"botName,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	FailureReason *string `json:"failureReason,omitempty"`
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
	Type *string `json:"type,omitempty"`
}

type GetBotChannelAssociationsRequest struct {
	BotAlias string `json:"aliasName,omitempty"`
	BotName string `json:"botName,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NameContains *string `json:"nameContains,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetBotChannelAssociationsResponse struct {
	BotChannelAssociations []BotChannelAssociation `json:"botChannelAssociations,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetBotRequest struct {
	Name string `json:"name,omitempty"`
	VersionOrAlias string `json:"versionoralias,omitempty"`
}

type GetBotResponse struct {
	AbortStatement *Statement `json:"abortStatement,omitempty"`
	Checksum *string `json:"checksum,omitempty"`
	ChildDirected bool `json:"childDirected,omitempty"`
	ClarificationPrompt *Prompt `json:"clarificationPrompt,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	DetectSentiment bool `json:"detectSentiment,omitempty"`
	EnableModelImprovements bool `json:"enableModelImprovements,omitempty"`
	FailureReason *string `json:"failureReason,omitempty"`
	IdleSessionTTLInSeconds int `json:"idleSessionTTLInSeconds,omitempty"`
	Intents []Intent `json:"intents,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Locale *string `json:"locale,omitempty"`
	Name *string `json:"name,omitempty"`
	NluIntentConfidenceThreshold float64 `json:"nluIntentConfidenceThreshold,omitempty"`
	Status *string `json:"status,omitempty"`
	Version *string `json:"version,omitempty"`
	VoiceId *string `json:"voiceId,omitempty"`
}

type GetBotVersionsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	Name string `json:"name,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetBotVersionsResponse struct {
	Bots []BotMetadata `json:"bots,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetBotsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NameContains *string `json:"nameContains,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetBotsResponse struct {
	Bots []BotMetadata `json:"bots,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetBuiltinIntentRequest struct {
	Signature string `json:"signature,omitempty"`
}

type GetBuiltinIntentResponse struct {
	Signature *string `json:"signature,omitempty"`
	Slots []BuiltinIntentSlot `json:"slots,omitempty"`
	SupportedLocales []string `json:"supportedLocales,omitempty"`
}

type GetBuiltinIntentsRequest struct {
	Locale *string `json:"locale,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	SignatureContains *string `json:"signatureContains,omitempty"`
}

type GetBuiltinIntentsResponse struct {
	Intents []BuiltinIntentMetadata `json:"intents,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetBuiltinSlotTypesRequest struct {
	Locale *string `json:"locale,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	SignatureContains *string `json:"signatureContains,omitempty"`
}

type GetBuiltinSlotTypesResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	SlotTypes []BuiltinSlotTypeMetadata `json:"slotTypes,omitempty"`
}

type GetExportRequest struct {
	ExportType string `json:"exportType,omitempty"`
	Name string `json:"name,omitempty"`
	ResourceType string `json:"resourceType,omitempty"`
	Version string `json:"version,omitempty"`
}

type GetExportResponse struct {
	ExportStatus *string `json:"exportStatus,omitempty"`
	ExportType *string `json:"exportType,omitempty"`
	FailureReason *string `json:"failureReason,omitempty"`
	Name *string `json:"name,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
	Url *string `json:"url,omitempty"`
	Version *string `json:"version,omitempty"`
}

type GetImportRequest struct {
	ImportId string `json:"importId,omitempty"`
}

type GetImportResponse struct {
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	FailureReason []string `json:"failureReason,omitempty"`
	ImportId *string `json:"importId,omitempty"`
	ImportStatus *string `json:"importStatus,omitempty"`
	MergeStrategy *string `json:"mergeStrategy,omitempty"`
	Name *string `json:"name,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
}

type GetIntentRequest struct {
	Name string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type GetIntentResponse struct {
	Checksum *string `json:"checksum,omitempty"`
	ConclusionStatement *Statement `json:"conclusionStatement,omitempty"`
	ConfirmationPrompt *Prompt `json:"confirmationPrompt,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	DialogCodeHook *CodeHook `json:"dialogCodeHook,omitempty"`
	FollowUpPrompt *FollowUpPrompt `json:"followUpPrompt,omitempty"`
	FulfillmentActivity *FulfillmentActivity `json:"fulfillmentActivity,omitempty"`
	InputContexts []InputContext `json:"inputContexts,omitempty"`
	KendraConfiguration *KendraConfiguration `json:"kendraConfiguration,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
	OutputContexts []OutputContext `json:"outputContexts,omitempty"`
	ParentIntentSignature *string `json:"parentIntentSignature,omitempty"`
	RejectionStatement *Statement `json:"rejectionStatement,omitempty"`
	SampleUtterances []string `json:"sampleUtterances,omitempty"`
	Slots []Slot `json:"slots,omitempty"`
	Version *string `json:"version,omitempty"`
}

type GetIntentVersionsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	Name string `json:"name,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetIntentVersionsResponse struct {
	Intents []IntentMetadata `json:"intents,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetIntentsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NameContains *string `json:"nameContains,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetIntentsResponse struct {
	Intents []IntentMetadata `json:"intents,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetMigrationRequest struct {
	MigrationId string `json:"migrationId,omitempty"`
}

type GetMigrationResponse struct {
	Alerts []MigrationAlert `json:"alerts,omitempty"`
	MigrationId *string `json:"migrationId,omitempty"`
	MigrationStatus *string `json:"migrationStatus,omitempty"`
	MigrationStrategy *string `json:"migrationStrategy,omitempty"`
	MigrationTimestamp *time.Time `json:"migrationTimestamp,omitempty"`
	V1BotLocale *string `json:"v1BotLocale,omitempty"`
	V1BotName *string `json:"v1BotName,omitempty"`
	V1BotVersion *string `json:"v1BotVersion,omitempty"`
	V2BotId *string `json:"v2BotId,omitempty"`
	V2BotRole *string `json:"v2BotRole,omitempty"`
}

type GetMigrationsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	MigrationStatusEquals *string `json:"migrationStatusEquals,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	SortByAttribute *string `json:"sortByAttribute,omitempty"`
	SortByOrder *string `json:"sortByOrder,omitempty"`
	V1BotNameContains *string `json:"v1BotNameContains,omitempty"`
}

type GetMigrationsResponse struct {
	MigrationSummaries []MigrationSummary `json:"migrationSummaries,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetSlotTypeRequest struct {
	Name string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type GetSlotTypeResponse struct {
	Checksum *string `json:"checksum,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	EnumerationValues []EnumerationValue `json:"enumerationValues,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
	ParentSlotTypeSignature *string `json:"parentSlotTypeSignature,omitempty"`
	SlotTypeConfigurations []SlotTypeConfiguration `json:"slotTypeConfigurations,omitempty"`
	ValueSelectionStrategy *string `json:"valueSelectionStrategy,omitempty"`
	Version *string `json:"version,omitempty"`
}

type GetSlotTypeVersionsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	Name string `json:"name,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetSlotTypeVersionsResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	SlotTypes []SlotTypeMetadata `json:"slotTypes,omitempty"`
}

type GetSlotTypesRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NameContains *string `json:"nameContains,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetSlotTypesResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	SlotTypes []SlotTypeMetadata `json:"slotTypes,omitempty"`
}

type GetUtterancesViewRequest struct {
	BotName string `json:"botname,omitempty"`
	BotVersions []string `json:"bot_versions,omitempty"`
	StatusType string `json:"status_type,omitempty"`
}

type GetUtterancesViewResponse struct {
	BotName *string `json:"botName,omitempty"`
	Utterances []UtteranceList `json:"utterances,omitempty"`
}

type InputContext struct {
	Name string `json:"name,omitempty"`
}

type Intent struct {
	IntentName string `json:"intentName,omitempty"`
	IntentVersion string `json:"intentVersion,omitempty"`
}

type IntentMetadata struct {
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
	Version *string `json:"version,omitempty"`
}

type KendraConfiguration struct {
	KendraIndex string `json:"kendraIndex,omitempty"`
	QueryFilterString *string `json:"queryFilterString,omitempty"`
	Role string `json:"role,omitempty"`
}

type ListTagsForResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	Tags []Tag `json:"tags,omitempty"`
}

type LogSettingsRequest struct {
	Destination string `json:"destination,omitempty"`
	KmsKeyArn *string `json:"kmsKeyArn,omitempty"`
	LogType string `json:"logType,omitempty"`
	ResourceArn string `json:"resourceArn,omitempty"`
}

type LogSettingsResponse struct {
	Destination *string `json:"destination,omitempty"`
	KmsKeyArn *string `json:"kmsKeyArn,omitempty"`
	LogType *string `json:"logType,omitempty"`
	ResourceArn *string `json:"resourceArn,omitempty"`
	ResourcePrefix *string `json:"resourcePrefix,omitempty"`
}

type Message struct {
	Content string `json:"content,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	GroupNumber int `json:"groupNumber,omitempty"`
}

type MigrationAlert struct {
	Details []string `json:"details,omitempty"`
	Message *string `json:"message,omitempty"`
	ReferenceURLs []string `json:"referenceURLs,omitempty"`
	Type *string `json:"type,omitempty"`
}

type MigrationSummary struct {
	MigrationId *string `json:"migrationId,omitempty"`
	MigrationStatus *string `json:"migrationStatus,omitempty"`
	MigrationStrategy *string `json:"migrationStrategy,omitempty"`
	MigrationTimestamp *time.Time `json:"migrationTimestamp,omitempty"`
	V1BotLocale *string `json:"v1BotLocale,omitempty"`
	V1BotName *string `json:"v1BotName,omitempty"`
	V1BotVersion *string `json:"v1BotVersion,omitempty"`
	V2BotId *string `json:"v2BotId,omitempty"`
	V2BotRole *string `json:"v2BotRole,omitempty"`
}

type OutputContext struct {
	Name string `json:"name,omitempty"`
	TimeToLiveInSeconds int `json:"timeToLiveInSeconds,omitempty"`
	TurnsToLive int `json:"turnsToLive,omitempty"`
}

type Prompt struct {
	MaxAttempts int `json:"maxAttempts,omitempty"`
	Messages []Message `json:"messages,omitempty"`
	ResponseCard *string `json:"responseCard,omitempty"`
}

type PutBotAliasRequest struct {
	BotName string `json:"botName,omitempty"`
	BotVersion string `json:"botVersion,omitempty"`
	Checksum *string `json:"checksum,omitempty"`
	ConversationLogs *ConversationLogsRequest `json:"conversationLogs,omitempty"`
	Description *string `json:"description,omitempty"`
	Name string `json:"name,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type PutBotAliasResponse struct {
	BotName *string `json:"botName,omitempty"`
	BotVersion *string `json:"botVersion,omitempty"`
	Checksum *string `json:"checksum,omitempty"`
	ConversationLogs *ConversationLogsResponse `json:"conversationLogs,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type PutBotRequest struct {
	AbortStatement *Statement `json:"abortStatement,omitempty"`
	Checksum *string `json:"checksum,omitempty"`
	ChildDirected bool `json:"childDirected,omitempty"`
	ClarificationPrompt *Prompt `json:"clarificationPrompt,omitempty"`
	CreateVersion bool `json:"createVersion,omitempty"`
	Description *string `json:"description,omitempty"`
	DetectSentiment bool `json:"detectSentiment,omitempty"`
	EnableModelImprovements bool `json:"enableModelImprovements,omitempty"`
	IdleSessionTTLInSeconds int `json:"idleSessionTTLInSeconds,omitempty"`
	Intents []Intent `json:"intents,omitempty"`
	Locale string `json:"locale,omitempty"`
	Name string `json:"name,omitempty"`
	NluIntentConfidenceThreshold float64 `json:"nluIntentConfidenceThreshold,omitempty"`
	ProcessBehavior *string `json:"processBehavior,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	VoiceId *string `json:"voiceId,omitempty"`
}

type PutBotResponse struct {
	AbortStatement *Statement `json:"abortStatement,omitempty"`
	Checksum *string `json:"checksum,omitempty"`
	ChildDirected bool `json:"childDirected,omitempty"`
	ClarificationPrompt *Prompt `json:"clarificationPrompt,omitempty"`
	CreateVersion bool `json:"createVersion,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	DetectSentiment bool `json:"detectSentiment,omitempty"`
	EnableModelImprovements bool `json:"enableModelImprovements,omitempty"`
	FailureReason *string `json:"failureReason,omitempty"`
	IdleSessionTTLInSeconds int `json:"idleSessionTTLInSeconds,omitempty"`
	Intents []Intent `json:"intents,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Locale *string `json:"locale,omitempty"`
	Name *string `json:"name,omitempty"`
	NluIntentConfidenceThreshold float64 `json:"nluIntentConfidenceThreshold,omitempty"`
	Status *string `json:"status,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	Version *string `json:"version,omitempty"`
	VoiceId *string `json:"voiceId,omitempty"`
}

type PutIntentRequest struct {
	Checksum *string `json:"checksum,omitempty"`
	ConclusionStatement *Statement `json:"conclusionStatement,omitempty"`
	ConfirmationPrompt *Prompt `json:"confirmationPrompt,omitempty"`
	CreateVersion bool `json:"createVersion,omitempty"`
	Description *string `json:"description,omitempty"`
	DialogCodeHook *CodeHook `json:"dialogCodeHook,omitempty"`
	FollowUpPrompt *FollowUpPrompt `json:"followUpPrompt,omitempty"`
	FulfillmentActivity *FulfillmentActivity `json:"fulfillmentActivity,omitempty"`
	InputContexts []InputContext `json:"inputContexts,omitempty"`
	KendraConfiguration *KendraConfiguration `json:"kendraConfiguration,omitempty"`
	Name string `json:"name,omitempty"`
	OutputContexts []OutputContext `json:"outputContexts,omitempty"`
	ParentIntentSignature *string `json:"parentIntentSignature,omitempty"`
	RejectionStatement *Statement `json:"rejectionStatement,omitempty"`
	SampleUtterances []string `json:"sampleUtterances,omitempty"`
	Slots []Slot `json:"slots,omitempty"`
}

type PutIntentResponse struct {
	Checksum *string `json:"checksum,omitempty"`
	ConclusionStatement *Statement `json:"conclusionStatement,omitempty"`
	ConfirmationPrompt *Prompt `json:"confirmationPrompt,omitempty"`
	CreateVersion bool `json:"createVersion,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	DialogCodeHook *CodeHook `json:"dialogCodeHook,omitempty"`
	FollowUpPrompt *FollowUpPrompt `json:"followUpPrompt,omitempty"`
	FulfillmentActivity *FulfillmentActivity `json:"fulfillmentActivity,omitempty"`
	InputContexts []InputContext `json:"inputContexts,omitempty"`
	KendraConfiguration *KendraConfiguration `json:"kendraConfiguration,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
	OutputContexts []OutputContext `json:"outputContexts,omitempty"`
	ParentIntentSignature *string `json:"parentIntentSignature,omitempty"`
	RejectionStatement *Statement `json:"rejectionStatement,omitempty"`
	SampleUtterances []string `json:"sampleUtterances,omitempty"`
	Slots []Slot `json:"slots,omitempty"`
	Version *string `json:"version,omitempty"`
}

type PutSlotTypeRequest struct {
	Checksum *string `json:"checksum,omitempty"`
	CreateVersion bool `json:"createVersion,omitempty"`
	Description *string `json:"description,omitempty"`
	EnumerationValues []EnumerationValue `json:"enumerationValues,omitempty"`
	Name string `json:"name,omitempty"`
	ParentSlotTypeSignature *string `json:"parentSlotTypeSignature,omitempty"`
	SlotTypeConfigurations []SlotTypeConfiguration `json:"slotTypeConfigurations,omitempty"`
	ValueSelectionStrategy *string `json:"valueSelectionStrategy,omitempty"`
}

type PutSlotTypeResponse struct {
	Checksum *string `json:"checksum,omitempty"`
	CreateVersion bool `json:"createVersion,omitempty"`
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	EnumerationValues []EnumerationValue `json:"enumerationValues,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
	ParentSlotTypeSignature *string `json:"parentSlotTypeSignature,omitempty"`
	SlotTypeConfigurations []SlotTypeConfiguration `json:"slotTypeConfigurations,omitempty"`
	ValueSelectionStrategy *string `json:"valueSelectionStrategy,omitempty"`
	Version *string `json:"version,omitempty"`
}

type Slot struct {
	DefaultValueSpec *SlotDefaultValueSpec `json:"defaultValueSpec,omitempty"`
	Description *string `json:"description,omitempty"`
	Name string `json:"name,omitempty"`
	ObfuscationSetting *string `json:"obfuscationSetting,omitempty"`
	Priority int `json:"priority,omitempty"`
	ResponseCard *string `json:"responseCard,omitempty"`
	SampleUtterances []string `json:"sampleUtterances,omitempty"`
	SlotConstraint string `json:"slotConstraint,omitempty"`
	SlotType *string `json:"slotType,omitempty"`
	SlotTypeVersion *string `json:"slotTypeVersion,omitempty"`
	ValueElicitationPrompt *Prompt `json:"valueElicitationPrompt,omitempty"`
}

type SlotDefaultValue struct {
	DefaultValue string `json:"defaultValue,omitempty"`
}

type SlotDefaultValueSpec struct {
	DefaultValueList []SlotDefaultValue `json:"defaultValueList,omitempty"`
}

type SlotTypeConfiguration struct {
	RegexConfiguration *SlotTypeRegexConfiguration `json:"regexConfiguration,omitempty"`
}

type SlotTypeMetadata struct {
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	Description *string `json:"description,omitempty"`
	LastUpdatedDate *time.Time `json:"lastUpdatedDate,omitempty"`
	Name *string `json:"name,omitempty"`
	Version *string `json:"version,omitempty"`
}

type SlotTypeRegexConfiguration struct {
	Pattern string `json:"pattern,omitempty"`
}

type StartImportRequest struct {
	MergeStrategy string `json:"mergeStrategy,omitempty"`
	Payload []byte `json:"payload,omitempty"`
	ResourceType string `json:"resourceType,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type StartImportResponse struct {
	CreatedDate *time.Time `json:"createdDate,omitempty"`
	ImportId *string `json:"importId,omitempty"`
	ImportStatus *string `json:"importStatus,omitempty"`
	MergeStrategy *string `json:"mergeStrategy,omitempty"`
	Name *string `json:"name,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type StartMigrationRequest struct {
	MigrationStrategy string `json:"migrationStrategy,omitempty"`
	V1BotName string `json:"v1BotName,omitempty"`
	V1BotVersion string `json:"v1BotVersion,omitempty"`
	V2BotName string `json:"v2BotName,omitempty"`
	V2BotRole string `json:"v2BotRole,omitempty"`
}

type StartMigrationResponse struct {
	MigrationId *string `json:"migrationId,omitempty"`
	MigrationStrategy *string `json:"migrationStrategy,omitempty"`
	MigrationTimestamp *time.Time `json:"migrationTimestamp,omitempty"`
	V1BotLocale *string `json:"v1BotLocale,omitempty"`
	V1BotName *string `json:"v1BotName,omitempty"`
	V1BotVersion *string `json:"v1BotVersion,omitempty"`
	V2BotId *string `json:"v2BotId,omitempty"`
	V2BotRole *string `json:"v2BotRole,omitempty"`
}

type Statement struct {
	Messages []Message `json:"messages,omitempty"`
	ResponseCard *string `json:"responseCard,omitempty"`
}

type Tag struct {
	Key string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type TagResourceResponse struct {
}

type UntagResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
	TagKeys []string `json:"tagKeys,omitempty"`
}

type UntagResourceResponse struct {
}

type UtteranceData struct {
	Count int `json:"count,omitempty"`
	DistinctUsers int `json:"distinctUsers,omitempty"`
	FirstUtteredDate *time.Time `json:"firstUtteredDate,omitempty"`
	LastUtteredDate *time.Time `json:"lastUtteredDate,omitempty"`
	UtteranceString *string `json:"utteranceString,omitempty"`
}

type UtteranceList struct {
	BotVersion *string `json:"botVersion,omitempty"`
	Utterances []UtteranceData `json:"utterances,omitempty"`
}



// ── Handler helpers ──────────────────────────────────────────────────────────

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

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleCreateBotVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateBotVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateBotVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateBotVersion"})
}

func handleCreateIntentVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateIntentVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateIntentVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateIntentVersion"})
}

func handleCreateSlotTypeVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateSlotTypeVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateSlotTypeVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateSlotTypeVersion"})
}

func handleDeleteBot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteBotRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteBot business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteBot"})
}

func handleDeleteBotAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteBotAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteBotAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteBotAlias"})
}

func handleDeleteBotChannelAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteBotChannelAssociationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteBotChannelAssociation business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteBotChannelAssociation"})
}

func handleDeleteBotVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteBotVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteBotVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteBotVersion"})
}

func handleDeleteIntent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteIntentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteIntent business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteIntent"})
}

func handleDeleteIntentVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteIntentVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteIntentVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteIntentVersion"})
}

func handleDeleteSlotType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteSlotTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteSlotType business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteSlotType"})
}

func handleDeleteSlotTypeVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteSlotTypeVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteSlotTypeVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteSlotTypeVersion"})
}

func handleDeleteUtterances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteUtterancesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteUtterances business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteUtterances"})
}

func handleGetBot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetBotRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetBot business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetBot"})
}

func handleGetBotAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetBotAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetBotAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetBotAlias"})
}

func handleGetBotAliases(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetBotAliasesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetBotAliases business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetBotAliases"})
}

func handleGetBotChannelAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetBotChannelAssociationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetBotChannelAssociation business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetBotChannelAssociation"})
}

func handleGetBotChannelAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetBotChannelAssociationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetBotChannelAssociations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetBotChannelAssociations"})
}

func handleGetBotVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetBotVersionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetBotVersions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetBotVersions"})
}

func handleGetBots(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetBotsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetBots business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetBots"})
}

func handleGetBuiltinIntent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetBuiltinIntentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetBuiltinIntent business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetBuiltinIntent"})
}

func handleGetBuiltinIntents(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetBuiltinIntentsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetBuiltinIntents business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetBuiltinIntents"})
}

func handleGetBuiltinSlotTypes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetBuiltinSlotTypesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetBuiltinSlotTypes business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetBuiltinSlotTypes"})
}

func handleGetExport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetExportRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetExport business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetExport"})
}

func handleGetImport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetImportRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetImport business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetImport"})
}

func handleGetIntent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetIntentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetIntent business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetIntent"})
}

func handleGetIntentVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetIntentVersionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetIntentVersions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetIntentVersions"})
}

func handleGetIntents(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetIntentsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetIntents business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetIntents"})
}

func handleGetMigration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMigrationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMigration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMigration"})
}

func handleGetMigrations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMigrationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMigrations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMigrations"})
}

func handleGetSlotType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetSlotTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetSlotType business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetSlotType"})
}

func handleGetSlotTypeVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetSlotTypeVersionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetSlotTypeVersions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetSlotTypeVersions"})
}

func handleGetSlotTypes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetSlotTypesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetSlotTypes business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetSlotTypes"})
}

func handleGetUtterancesView(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetUtterancesViewRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetUtterancesView business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetUtterancesView"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handlePutBot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutBotRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutBot business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutBot"})
}

func handlePutBotAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutBotAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutBotAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutBotAlias"})
}

func handlePutIntent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutIntentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutIntent business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutIntent"})
}

func handlePutSlotType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutSlotTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutSlotType business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutSlotType"})
}

func handleStartImport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartImportRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartImport business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartImport"})
}

func handleStartMigration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartMigrationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartMigration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartMigration"})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TagResource"})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UntagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UntagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UntagResource"})
}

