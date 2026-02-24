package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/connectors/github"
	"github.com/google/uuid"
)

// timeFromNullTime converts sql.NullTime to time.Time
func timeFromNullTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Time{}
}

type githubRepositoryRepository struct {
	queries *Queries
}

func NewGitHubRepositoryRepository(db *sql.DB) github.GitHubRepositoryRepository {
	return &githubRepositoryRepository{queries: New(db)}
}

func (r *githubRepositoryRepository) Store(ctx context.Context, repo github.GitHubRepository) error {
	err := r.queries.UpsertGitHubRepository(ctx, UpsertGitHubRepositoryParams{
		ID:                    repo.ID,
		IntegrationID:         repo.IntegrationID,
		GithubRepositoryID:    repo.GitHubRepositoryID,
		RepositoryName:        repo.RepositoryName,
		RepositoryFullName:    repo.RepositoryFullName,
		RepositoryUrl:         repo.RepositoryURL,
		IsPrivate:             repo.IsPrivate,
		DefaultBranch:         nullString(repo.DefaultBranch),
		PermissionAdmin:       repo.PermissionAdmin,
		PermissionPush:        repo.PermissionPush,
		PermissionPull:        repo.PermissionPull,
		RepositoryDescription: nullString(repo.RepositoryDescription),
		RepositoryLanguage:    nullString(repo.RepositoryLanguage),
		CreatedAt:             repo.CreatedAt,
		UpdatedAt:             repo.UpdatedAt,
		LastSyncedAt:          repo.LastSyncedAt,
		GithubCreatedAt:       nullTime(repo.GitHubCreatedAt),
		GithubUpdatedAt:       nullTime(repo.GitHubUpdatedAt),
		GithubPushedAt:        nullTime(repo.GitHubPushedAt),
	})

	if err != nil {
		return fmt.Errorf("failed to upsert github repository: %w", err)
	}

	return nil
}

func (r *githubRepositoryRepository) ListByIntegrationID(ctx context.Context, integrationID uuid.UUID) ([]github.GitHubRepository, error) {
	dbRepos, err := r.queries.FindGitHubRepositoriesByIntegrationID(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list github repositories: %w", err)
	}

	var repositories []github.GitHubRepository
	for _, dbRepo := range dbRepos {
		repo := github.GitHubRepository{
			ID:                    dbRepo.ID,
			IntegrationID:         dbRepo.IntegrationID,
			GitHubRepositoryID:    dbRepo.GithubRepositoryID,
			RepositoryName:        dbRepo.RepositoryName,
			RepositoryFullName:    dbRepo.RepositoryFullName,
			RepositoryURL:         dbRepo.RepositoryUrl,
			IsPrivate:             dbRepo.IsPrivate,
			DefaultBranch:         dbRepo.DefaultBranch.String,
			PermissionAdmin:       dbRepo.PermissionAdmin,
			PermissionPush:        dbRepo.PermissionPush,
			PermissionPull:        dbRepo.PermissionPull,
			RepositoryDescription: dbRepo.RepositoryDescription.String,
			RepositoryLanguage:    dbRepo.RepositoryLanguage.String,
			CreatedAt:             dbRepo.CreatedAt,
			UpdatedAt:             dbRepo.UpdatedAt,
			LastSyncedAt:          dbRepo.LastSyncedAt,
			GitHubCreatedAt:       timeFromNullTime(dbRepo.GithubCreatedAt),
			GitHubUpdatedAt:       timeFromNullTime(dbRepo.GithubUpdatedAt),
			GitHubPushedAt:        timeFromNullTime(dbRepo.GithubPushedAt),
		}
		repositories = append(repositories, repo)
	}

	return repositories, nil
}

func (r *githubRepositoryRepository) GetByGitHubID(ctx context.Context, integrationID uuid.UUID, repositoryID int64) (github.GitHubRepository, error) {
	dbRepo, err := r.queries.FindGitHubRepositoryByGitHubID(ctx, FindGitHubRepositoryByGitHubIDParams{
		IntegrationID:      integrationID,
		GithubRepositoryID: repositoryID,
	})

	if err != nil {
		if err == sql.ErrNoRows {
			return github.GitHubRepository{}, nil
		}
		return github.GitHubRepository{}, fmt.Errorf("failed to get github repository: %w", err)
	}

	repo := github.GitHubRepository{
		ID:                    dbRepo.ID,
		IntegrationID:         dbRepo.IntegrationID,
		GitHubRepositoryID:    dbRepo.GithubRepositoryID,
		RepositoryName:        dbRepo.RepositoryName,
		RepositoryFullName:    dbRepo.RepositoryFullName,
		RepositoryURL:         dbRepo.RepositoryUrl,
		IsPrivate:             dbRepo.IsPrivate,
		DefaultBranch:         dbRepo.DefaultBranch.String,
		PermissionAdmin:       dbRepo.PermissionAdmin,
		PermissionPush:        dbRepo.PermissionPush,
		PermissionPull:        dbRepo.PermissionPull,
		RepositoryDescription: dbRepo.RepositoryDescription.String,
		RepositoryLanguage:    dbRepo.RepositoryLanguage.String,
		CreatedAt:             dbRepo.CreatedAt,
		UpdatedAt:             dbRepo.UpdatedAt,
		LastSyncedAt:          dbRepo.LastSyncedAt,
		GitHubCreatedAt:       timeFromNullTime(dbRepo.GithubCreatedAt),
		GitHubUpdatedAt:       timeFromNullTime(dbRepo.GithubUpdatedAt),
		GitHubPushedAt:        timeFromNullTime(dbRepo.GithubPushedAt),
	}

	return repo, nil
}

func (r *githubRepositoryRepository) DeleteByGitHubID(ctx context.Context, integrationID uuid.UUID, repositoryID int64) error {
	err := r.queries.DeleteGitHubRepositoryByGitHubID(ctx, DeleteGitHubRepositoryByGitHubIDParams{
		IntegrationID:      integrationID,
		GithubRepositoryID: repositoryID,
	})

	if err != nil {
		return fmt.Errorf("failed to delete github repository: %w", err)
	}

	return nil
}

func (r *githubRepositoryRepository) UpdatePermissions(ctx context.Context, integrationID uuid.UUID, repositoryID int64, permissions github.RepositoryPermissions) error {
	err := r.queries.UpdateGitHubRepositoryPermissions(ctx, UpdateGitHubRepositoryPermissionsParams{
		PermissionAdmin:    permissions.Admin,
		PermissionPush:     permissions.Push,
		PermissionPull:     permissions.Pull,
		IntegrationID:      integrationID,
		GithubRepositoryID: repositoryID,
	})

	if err != nil {
		return fmt.Errorf("failed to update repository permissions: %w", err)
	}

	return nil
}

func (r *githubRepositoryRepository) BulkDelete(ctx context.Context, integrationID uuid.UUID, repositoryIDs []int64) error {
	if len(repositoryIDs) == 0 {
		return nil
	}

	err := r.queries.BulkDeleteGitHubRepositories(ctx, BulkDeleteGitHubRepositoriesParams{
		IntegrationID: integrationID,
		Column2:       repositoryIDs,
	})

	if err != nil {
		return fmt.Errorf("failed to bulk delete github repositories: %w", err)
	}

	return nil
}

func (r *githubRepositoryRepository) UpdateLastSyncTime(ctx context.Context, integrationID uuid.UUID, syncTime time.Time) error {
	err := r.queries.UpdateGitHubRepositoryLastSyncTime(ctx, UpdateGitHubRepositoryLastSyncTimeParams{
		LastSyncedAt:  syncTime,
		IntegrationID: integrationID,
	})

	if err != nil {
		return fmt.Errorf("failed to update last sync time: %w", err)
	}

	return nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: t, Valid: true}
}
