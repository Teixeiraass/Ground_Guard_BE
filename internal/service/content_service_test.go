package service_test

import (
	"context"
	"database/sql"
	"testing"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetTermsAcceptanceStatusNoActiveDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetLatestActiveLegalDocumentByType(gomock.Any(), "TERMS").Return(db.LegalDocument{}, sql.ErrNoRows)

	contentService := service.NewContentService(store)
	needsAcceptance, document, err := contentService.GetTermsAcceptanceStatus(context.Background(), 1)
	require.NoError(t, err)
	require.False(t, needsAcceptance)
	require.Nil(t, document)
}

func TestGetTermsAcceptanceStatusNotYetAccepted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	userID := int64(1)
	legalDocument := db.LegalDocument{ID: 10, Type: "TERMS", Version: "1.0"}

	store.EXPECT().GetLatestActiveLegalDocumentByType(gomock.Any(), "TERMS").Return(legalDocument, nil)
	store.EXPECT().
		GetUserAcceptedTermByUserAndDocument(gomock.Any(), db.GetUserAcceptedTermByUserAndDocumentParams{
			UserID:          userID,
			LegalDocumentID: legalDocument.ID,
		}).
		Return(db.UserAcceptedTerm{}, sql.ErrNoRows)

	contentService := service.NewContentService(store)
	needsAcceptance, document, err := contentService.GetTermsAcceptanceStatus(context.Background(), userID)
	require.NoError(t, err)
	require.True(t, needsAcceptance)
	require.NotNil(t, document)
	require.Equal(t, legalDocument.ID, document.ID)
}

func TestGetTermsAcceptanceStatusAlreadyAccepted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	userID := int64(1)
	legalDocument := db.LegalDocument{ID: 10, Type: "TERMS", Version: "1.0"}

	store.EXPECT().GetLatestActiveLegalDocumentByType(gomock.Any(), "TERMS").Return(legalDocument, nil)
	store.EXPECT().
		GetUserAcceptedTermByUserAndDocument(gomock.Any(), db.GetUserAcceptedTermByUserAndDocumentParams{
			UserID:          userID,
			LegalDocumentID: legalDocument.ID,
		}).
		Return(db.UserAcceptedTerm{UserID: userID, LegalDocumentID: legalDocument.ID}, nil)

	contentService := service.NewContentService(store)
	needsAcceptance, _, err := contentService.GetTermsAcceptanceStatus(context.Background(), userID)
	require.NoError(t, err)
	require.False(t, needsAcceptance)
}

func TestAcceptLegalDocumentRecordsAcceptance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	userID := int64(1)
	documentUUID := uuid.New()
	legalDocument := db.LegalDocument{ID: 10, Uuid: documentUUID, Type: "TERMS", Version: "1.0"}

	store.EXPECT().GetLegalDocument(gomock.Any(), documentUUID).Return(legalDocument, nil)
	store.EXPECT().
		GetUserAcceptedTermByUserAndDocument(gomock.Any(), db.GetUserAcceptedTermByUserAndDocumentParams{
			UserID:          userID,
			LegalDocumentID: legalDocument.ID,
		}).
		Return(db.UserAcceptedTerm{}, sql.ErrNoRows)
	store.EXPECT().
		CreateUserAcceptedTerm(gomock.Any(), db.CreateUserAcceptedTermParams{
			UserID:          userID,
			LegalDocumentID: legalDocument.ID,
		}).
		Return(db.UserAcceptedTerm{UserID: userID, LegalDocumentID: legalDocument.ID}, nil)

	contentService := service.NewContentService(store)
	err := contentService.AcceptLegalDocument(context.Background(), userID, documentUUID)
	require.NoError(t, err)
}

func TestAcceptLegalDocumentIsIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	userID := int64(1)
	documentUUID := uuid.New()
	legalDocument := db.LegalDocument{ID: 10, Uuid: documentUUID, Type: "TERMS", Version: "1.0"}

	store.EXPECT().GetLegalDocument(gomock.Any(), documentUUID).Return(legalDocument, nil)
	store.EXPECT().
		GetUserAcceptedTermByUserAndDocument(gomock.Any(), db.GetUserAcceptedTermByUserAndDocumentParams{
			UserID:          userID,
			LegalDocumentID: legalDocument.ID,
		}).
		Return(db.UserAcceptedTerm{UserID: userID, LegalDocumentID: legalDocument.ID}, nil)
	// CreateUserAcceptedTerm não deve ser chamado de novo (já aceitou esse documento).

	contentService := service.NewContentService(store)
	err := contentService.AcceptLegalDocument(context.Background(), userID, documentUUID)
	require.NoError(t, err)
}
