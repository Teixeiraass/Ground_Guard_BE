package service

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

const legalDocumentTypeTerms = "TERMS"

type ContentService interface {
	GetFaq(ctx context.Context, faqUUID uuid.UUID) (*db.Faq, error)
	ListFaq(ctx context.Context, limit, offset int32) ([]db.Faq, error)
	GetHelpContent(ctx context.Context, helpContentUUID uuid.UUID) (*db.HelpContent, error)
	ListHelpContent(ctx context.Context, category string, limit, offset int32) ([]db.HelpContent, error)
	GetTutorial(ctx context.Context, tutorialUUID uuid.UUID) (*db.Tutorial, error)
	ListTutorial(ctx context.Context, limit, offset int32) ([]db.Tutorial, error)
	GetLegalDocument(ctx context.Context, legalDocumentUUID uuid.UUID) (*db.LegalDocument, error)
	ListLegalDocument(ctx context.Context, limit, offset int32) ([]db.LegalDocument, error)
	GetTermsAcceptanceStatus(ctx context.Context, userID int64) (needsAcceptance bool, document *db.LegalDocument, err error)
	AcceptLegalDocument(ctx context.Context, userID int64, documentUUID uuid.UUID) error
}

type contentService struct {
	store db.Store
}

func NewContentService(store db.Store) ContentService {
	return &contentService{store: store}
}

func (s *contentService) GetFaq(ctx context.Context, faqUUID uuid.UUID) (*db.Faq, error) {
	faq, err := s.store.GetFaq(ctx, faqUUID)
	if err != nil {
		return nil, err
	}
	return &faq, nil
}

func (s *contentService) ListFaq(ctx context.Context, limit, offset int32) ([]db.Faq, error) {
	arg := db.ListFaqsParams{
		Limit:  limit,
		Offset: offset,
	}
	return s.store.ListFaqs(ctx, arg)
}

func (s *contentService) GetHelpContent(ctx context.Context, helpContentUUID uuid.UUID) (*db.HelpContent, error) {
	helpContent, err := s.store.GetHelpContent(ctx, helpContentUUID)
	if err != nil {
		return nil, err
	}
	return &helpContent, nil
}

func (s *contentService) ListHelpContent(ctx context.Context, category string, limit, offset int32) ([]db.HelpContent, error) {
	if category != "" {
		return s.store.ListHelpContentsByCategory(ctx, db.ListHelpContentsByCategoryParams{
			Category: category,
			Limit:    limit,
			Offset:   offset,
		})
	}

	arg := db.ListHelpContentsParams{
		Limit:  limit,
		Offset: offset,
	}
	return s.store.ListHelpContents(ctx, arg)
}

func (s *contentService) GetTutorial(ctx context.Context, tutorialUUID uuid.UUID) (*db.Tutorial, error) {
	tutorial, err := s.store.GetTutorial(ctx, tutorialUUID)
	if err != nil {
		return nil, err
	}
	return &tutorial, nil
}

func (s *contentService) ListTutorial(ctx context.Context, limit, offset int32) ([]db.Tutorial, error) {
	arg := db.ListTutorialsParams{
		Limit:  limit,
		Offset: offset,
	}
	return s.store.ListTutorials(ctx, arg)
}

func (s *contentService) GetLegalDocument(ctx context.Context, legalDocumentUUID uuid.UUID) (*db.LegalDocument, error) {
	legalDocument, err := s.store.GetLegalDocument(ctx, legalDocumentUUID)
	if err != nil {
		return nil, err
	}
	return &legalDocument, nil
}

func (s *contentService) ListLegalDocument(ctx context.Context, limit, offset int32) ([]db.LegalDocument, error) {
	arg := db.ListLegalDocumentsParams{
		Limit:  limit,
		Offset: offset,
	}
	return s.store.ListLegalDocuments(ctx, arg)
}

// GetTermsAcceptanceStatus checks whether userID has accepted the current
// active TERMS document. If no active TERMS document exists at all, there's
// nothing to accept (needsAcceptance is false, document is nil).
func (s *contentService) GetTermsAcceptanceStatus(ctx context.Context, userID int64) (bool, *db.LegalDocument, error) {
	document, err := s.store.GetLatestActiveLegalDocumentByType(ctx, legalDocumentTypeTerms)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil, nil
		}
		return false, nil, err
	}

	_, err = s.store.GetUserAcceptedTermByUserAndDocument(ctx, db.GetUserAcceptedTermByUserAndDocumentParams{
		UserID:          userID,
		LegalDocumentID: document.ID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return true, &document, nil
		}
		return false, nil, err
	}

	return false, &document, nil
}

// AcceptLegalDocument records that userID accepted documentUUID. Idempotent:
// accepting the same document twice is a no-op, not an error.
func (s *contentService) AcceptLegalDocument(ctx context.Context, userID int64, documentUUID uuid.UUID) error {
	document, err := s.store.GetLegalDocument(ctx, documentUUID)
	if err != nil {
		return err
	}

	_, err = s.store.GetUserAcceptedTermByUserAndDocument(ctx, db.GetUserAcceptedTermByUserAndDocumentParams{
		UserID:          userID,
		LegalDocumentID: document.ID,
	})
	if err == nil {
		return nil // já aceitou esse documento exato
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err = s.store.CreateUserAcceptedTerm(ctx, db.CreateUserAcceptedTermParams{
		UserID:          userID,
		LegalDocumentID: document.ID,
	})
	return err
}
