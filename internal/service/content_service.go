package service

import (
	"context"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type ContentService interface {
	GetFaq(ctx context.Context, faqUUID uuid.UUID) (*db.Faq, error)
	ListFaq(ctx context.Context, limit, offset int32) ([]db.Faq, error)
	GetHelpContent(ctx context.Context, helpContentUUID uuid.UUID) (*db.HelpContent, error)
	ListHelpContent(ctx context.Context, limit, offset int32) ([]db.HelpContent, error)
	GetTutorial(ctx context.Context, tutorialUUID uuid.UUID) (*db.Tutorial, error)
	ListTutorial(ctx context.Context, limit, offset int32) ([]db.Tutorial, error)
	GetLegalDocument(ctx context.Context, legalDocumentUUID uuid.UUID) (*db.LegalDocument, error)
	ListLegalDocument(ctx context.Context, limit, offset int32) ([]db.LegalDocument, error)
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

func (s *contentService) ListHelpContent(ctx context.Context, limit, offset int32) ([]db.HelpContent, error) {
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
