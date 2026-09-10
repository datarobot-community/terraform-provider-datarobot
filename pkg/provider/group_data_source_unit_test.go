package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
)

func directoryResponse(totalCount int, entities ...client.DirectoryEntity) *client.ListDirectoryEntitiesResponse {
	return &client.ListDirectoryEntitiesResponse{
		Count:      len(entities),
		TotalCount: totalCount,
		Data:       entities,
	}
}

func TestResolveGroupByNameReturnsTheSingleMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	mockService.EXPECT().
		ListDirectoryEntities(gomock.Any(), &client.ListDirectoryEntitiesRequest{
			EntityType: "group",
			Name:       "FinanceTeam",
		}).
		Return(directoryResponse(1, client.DirectoryEntity{
			ID:                 "6a7f9181e27b37fa0f6b9b04",
			Name:               "FinanceTeam",
			EntityType:         "group",
			ProvisioningSource: "scim",
		}), nil)

	group, err := resolveGroupByName(context.Background(), mockService, "FinanceTeam")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if group.ID != "6a7f9181e27b37fa0f6b9b04" {
		t.Errorf("expected the matched group's ID, got %q", group.ID)
	}
}

// A name that matches nothing must fail rather than resolve to nothing, and the
// message should point at casing, which is the usual cause.
func TestResolveGroupByNameRejectsNoMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	mockService.EXPECT().
		ListDirectoryEntities(gomock.Any(), gomock.Any()).
		Return(directoryResponse(0), nil)

	_, err := resolveGroupByName(context.Background(), mockService, "financeteam")
	if err == nil {
		t.Fatal("expected an error when no group matches")
	}
	if !strings.Contains(err.Error(), "case sensitively") {
		t.Errorf("expected the error to mention case sensitivity, got %q", err.Error())
	}
}

// The endpoint returns every match rather than erroring on an ambiguous name,
// so the caller has to reject it. Silently taking the first match is exactly
// the failure this guards against.
func TestResolveGroupByNameRejectsAmbiguousName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	mockService.EXPECT().
		ListDirectoryEntities(gomock.Any(), gomock.Any()).
		Return(directoryResponse(2,
			client.DirectoryEntity{ID: "aaa", Name: "FinanceTeam"},
			client.DirectoryEntity{ID: "bbb", Name: "FinanceTeam"},
		), nil)

	_, err := resolveGroupByName(context.Background(), mockService, "FinanceTeam")
	if err == nil {
		t.Fatal("expected an error when the name is ambiguous")
	}
	if !strings.Contains(err.Error(), "found 2") {
		t.Errorf("expected the error to report the number of matches, got %q", err.Error())
	}
}

func TestResolveGroupByNameRejectsInconsistentResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	mockService.EXPECT().
		ListDirectoryEntities(gomock.Any(), gomock.Any()).
		Return(directoryResponse(1), nil)

	if _, err := resolveGroupByName(context.Background(), mockService, "FinanceTeam"); err == nil {
		t.Fatal("expected an error when the count and the payload disagree")
	}
}

func TestResolveGroupByNamePropagatesAPIErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	mockService.EXPECT().
		ListDirectoryEntities(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("boom"))

	if _, err := resolveGroupByName(context.Background(), mockService, "FinanceTeam"); err == nil {
		t.Fatal("expected the API error to propagate")
	}
}

func TestSharedRoleID(t *testing.T) {
	if got := sharedRoleID("dep123", "grp456"); got != "dep123:grp456" {
		t.Errorf("expected dep123:grp456, got %q", got)
	}
}
