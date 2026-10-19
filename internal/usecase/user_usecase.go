package usecase

import (
	"context"

	"github.com/akordium-id/dukungps/internal/domain/user"
	"github.com/akordium-id/dukungps/internal/repository"
)

type UserUseCase interface {
	ListUsers(ctx context.Context) ([]*user.User, error)
	GetUser(ctx context.Context, id int64) (*user.User, error)
	CreateUser(ctx context.Context, u *user.User) error
	UpdateUser(ctx context.Context, u *user.User) error
	DeleteUser(ctx context.Context, id int64) error
}

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) ListUsers(ctx context.Context) ([]*user.User, error) {
	return s.userRepo.FindAll(ctx)
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*user.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

func (s *UserService) CreateUser(ctx context.Context, u *user.User) error {
	return s.userRepo.Create(ctx, u)
}

func (s *UserService) UpdateUser(ctx context.Context, u *user.User) error {
	return s.userRepo.Update(ctx, u)
}

func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	return s.userRepo.Delete(ctx, id)
}

var _ UserUseCase = (*UserService)(nil)
