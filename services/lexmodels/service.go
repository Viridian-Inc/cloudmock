package lexmodels

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS lex service.
type Service struct {
	store *Store
}

// New returns a new lexmodels Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "lex" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateBotVersion", Method: http.MethodPost, IAMAction: "lex:CreateBotVersion"},
		{Name: "CreateIntentVersion", Method: http.MethodPost, IAMAction: "lex:CreateIntentVersion"},
		{Name: "CreateSlotTypeVersion", Method: http.MethodPost, IAMAction: "lex:CreateSlotTypeVersion"},
		{Name: "DeleteBot", Method: http.MethodDelete, IAMAction: "lex:DeleteBot"},
		{Name: "DeleteBotAlias", Method: http.MethodDelete, IAMAction: "lex:DeleteBotAlias"},
		{Name: "DeleteBotChannelAssociation", Method: http.MethodDelete, IAMAction: "lex:DeleteBotChannelAssociation"},
		{Name: "DeleteBotVersion", Method: http.MethodDelete, IAMAction: "lex:DeleteBotVersion"},
		{Name: "DeleteIntent", Method: http.MethodDelete, IAMAction: "lex:DeleteIntent"},
		{Name: "DeleteIntentVersion", Method: http.MethodDelete, IAMAction: "lex:DeleteIntentVersion"},
		{Name: "DeleteSlotType", Method: http.MethodDelete, IAMAction: "lex:DeleteSlotType"},
		{Name: "DeleteSlotTypeVersion", Method: http.MethodDelete, IAMAction: "lex:DeleteSlotTypeVersion"},
		{Name: "DeleteUtterances", Method: http.MethodDelete, IAMAction: "lex:DeleteUtterances"},
		{Name: "GetBot", Method: http.MethodGet, IAMAction: "lex:GetBot"},
		{Name: "GetBotAlias", Method: http.MethodGet, IAMAction: "lex:GetBotAlias"},
		{Name: "GetBotAliases", Method: http.MethodGet, IAMAction: "lex:GetBotAliases"},
		{Name: "GetBotChannelAssociation", Method: http.MethodGet, IAMAction: "lex:GetBotChannelAssociation"},
		{Name: "GetBotChannelAssociations", Method: http.MethodGet, IAMAction: "lex:GetBotChannelAssociations"},
		{Name: "GetBotVersions", Method: http.MethodGet, IAMAction: "lex:GetBotVersions"},
		{Name: "GetBots", Method: http.MethodGet, IAMAction: "lex:GetBots"},
		{Name: "GetBuiltinIntent", Method: http.MethodGet, IAMAction: "lex:GetBuiltinIntent"},
		{Name: "GetBuiltinIntents", Method: http.MethodGet, IAMAction: "lex:GetBuiltinIntents"},
		{Name: "GetBuiltinSlotTypes", Method: http.MethodGet, IAMAction: "lex:GetBuiltinSlotTypes"},
		{Name: "GetExport", Method: http.MethodGet, IAMAction: "lex:GetExport"},
		{Name: "GetImport", Method: http.MethodGet, IAMAction: "lex:GetImport"},
		{Name: "GetIntent", Method: http.MethodGet, IAMAction: "lex:GetIntent"},
		{Name: "GetIntentVersions", Method: http.MethodGet, IAMAction: "lex:GetIntentVersions"},
		{Name: "GetIntents", Method: http.MethodGet, IAMAction: "lex:GetIntents"},
		{Name: "GetMigration", Method: http.MethodGet, IAMAction: "lex:GetMigration"},
		{Name: "GetMigrations", Method: http.MethodGet, IAMAction: "lex:GetMigrations"},
		{Name: "GetSlotType", Method: http.MethodGet, IAMAction: "lex:GetSlotType"},
		{Name: "GetSlotTypeVersions", Method: http.MethodGet, IAMAction: "lex:GetSlotTypeVersions"},
		{Name: "GetSlotTypes", Method: http.MethodGet, IAMAction: "lex:GetSlotTypes"},
		{Name: "GetUtterancesView", Method: http.MethodGet, IAMAction: "lex:GetUtterancesView"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "lex:ListTagsForResource"},
		{Name: "PutBot", Method: http.MethodPut, IAMAction: "lex:PutBot"},
		{Name: "PutBotAlias", Method: http.MethodPut, IAMAction: "lex:PutBotAlias"},
		{Name: "PutIntent", Method: http.MethodPut, IAMAction: "lex:PutIntent"},
		{Name: "PutSlotType", Method: http.MethodPut, IAMAction: "lex:PutSlotType"},
		{Name: "StartImport", Method: http.MethodPost, IAMAction: "lex:StartImport"},
		{Name: "StartMigration", Method: http.MethodPost, IAMAction: "lex:StartMigration"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "lex:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "lex:UntagResource"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateBotVersion":
		return handleCreateBotVersion(ctx, s.store)
	case "CreateIntentVersion":
		return handleCreateIntentVersion(ctx, s.store)
	case "CreateSlotTypeVersion":
		return handleCreateSlotTypeVersion(ctx, s.store)
	case "DeleteBot":
		return handleDeleteBot(ctx, s.store)
	case "DeleteBotAlias":
		return handleDeleteBotAlias(ctx, s.store)
	case "DeleteBotChannelAssociation":
		return handleDeleteBotChannelAssociation(ctx, s.store)
	case "DeleteBotVersion":
		return handleDeleteBotVersion(ctx, s.store)
	case "DeleteIntent":
		return handleDeleteIntent(ctx, s.store)
	case "DeleteIntentVersion":
		return handleDeleteIntentVersion(ctx, s.store)
	case "DeleteSlotType":
		return handleDeleteSlotType(ctx, s.store)
	case "DeleteSlotTypeVersion":
		return handleDeleteSlotTypeVersion(ctx, s.store)
	case "DeleteUtterances":
		return handleDeleteUtterances(ctx, s.store)
	case "GetBot":
		return handleGetBot(ctx, s.store)
	case "GetBotAlias":
		return handleGetBotAlias(ctx, s.store)
	case "GetBotAliases":
		return handleGetBotAliases(ctx, s.store)
	case "GetBotChannelAssociation":
		return handleGetBotChannelAssociation(ctx, s.store)
	case "GetBotChannelAssociations":
		return handleGetBotChannelAssociations(ctx, s.store)
	case "GetBotVersions":
		return handleGetBotVersions(ctx, s.store)
	case "GetBots":
		return handleGetBots(ctx, s.store)
	case "GetBuiltinIntent":
		return handleGetBuiltinIntent(ctx, s.store)
	case "GetBuiltinIntents":
		return handleGetBuiltinIntents(ctx, s.store)
	case "GetBuiltinSlotTypes":
		return handleGetBuiltinSlotTypes(ctx, s.store)
	case "GetExport":
		return handleGetExport(ctx, s.store)
	case "GetImport":
		return handleGetImport(ctx, s.store)
	case "GetIntent":
		return handleGetIntent(ctx, s.store)
	case "GetIntentVersions":
		return handleGetIntentVersions(ctx, s.store)
	case "GetIntents":
		return handleGetIntents(ctx, s.store)
	case "GetMigration":
		return handleGetMigration(ctx, s.store)
	case "GetMigrations":
		return handleGetMigrations(ctx, s.store)
	case "GetSlotType":
		return handleGetSlotType(ctx, s.store)
	case "GetSlotTypeVersions":
		return handleGetSlotTypeVersions(ctx, s.store)
	case "GetSlotTypes":
		return handleGetSlotTypes(ctx, s.store)
	case "GetUtterancesView":
		return handleGetUtterancesView(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "PutBot":
		return handlePutBot(ctx, s.store)
	case "PutBotAlias":
		return handlePutBotAlias(ctx, s.store)
	case "PutIntent":
		return handlePutIntent(ctx, s.store)
	case "PutSlotType":
		return handlePutSlotType(ctx, s.store)
	case "StartImport":
		return handleStartImport(ctx, s.store)
	case "StartMigration":
		return handleStartMigration(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
