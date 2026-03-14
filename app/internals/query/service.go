package query

import (
	"context"

	"featuretrace.io/app/internals/model"
	"featuretrace.io/app/internals/storage/repository"
	"featuretrace.io/app/pkg/logger"
)

// Service encapsulates the business logic for searching and filtering logs.
type Service struct {
	repo repository.LogRepository
	log  *logger.Logger
}

// NewService creates a query service backed by the given repository.
func NewService(repo repository.LogRepository) *Service {
	return &Service{
		repo: repo,
		log:  logger.New("query-service"),
	}
}

// Search returns log records matching the filter criteria.
func (s *Service) Search(ctx context.Context, filter repository.LogFilter) ([]model.LogRecord, error) {
	records, err := s.repo.Query(ctx, filter)
	if err != nil {
		s.log.Error("search failed: %v", err)
		return nil, err
	}

	s.log.Debug("search returned %d records", len(records))
	return records, nil
}

