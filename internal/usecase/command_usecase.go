package usecase

import (
	"context"
	"time"

	"github.com/akordium-id/dukungps/internal/domain/command"
	"github.com/akordium-id/dukungps/internal/repository"
)

type CommandUseCase interface {
	ListCommands(ctx context.Context) ([]command.Command, error)
	GetCommand(ctx context.Context, id int64) (*command.Command, error)
	GetCommandsByDevice(ctx context.Context, deviceID int64) ([]command.Command, error)
	CreateCommand(ctx context.Context, c *command.Command) error
	UpdateCommand(ctx context.Context, c *command.Command) error
	DeleteCommand(ctx context.Context, id int64) error
	DispatchCommand(ctx context.Context, c *command.Command) error
}

type CommandService struct {
	commandRepo repository.CommandRepository
}

func NewCommandService(commandRepo repository.CommandRepository) *CommandService {
	return &CommandService{commandRepo: commandRepo}
}

func (s *CommandService) ListCommands(ctx context.Context) ([]command.Command, error) {
	return s.commandRepo.FindAll(ctx)
}

func (s *CommandService) GetCommand(ctx context.Context, id int64) (*command.Command, error) {
	return s.commandRepo.FindByID(ctx, id)
}

func (s *CommandService) GetCommandsByDevice(ctx context.Context, deviceID int64) ([]command.Command, error) {
	return s.commandRepo.FindByDeviceID(ctx, deviceID)
}

func (s *CommandService) CreateCommand(ctx context.Context, c *command.Command) error {
	return s.commandRepo.Create(ctx, c)
}

func (s *CommandService) UpdateCommand(ctx context.Context, c *command.Command) error {
	return s.commandRepo.Update(ctx, c)
}

func (s *CommandService) DeleteCommand(ctx context.Context, id int64) error {
	return s.commandRepo.Delete(ctx, id)
}

func (s *CommandService) DispatchCommand(ctx context.Context, c *command.Command) error {
	if err := s.commandRepo.Create(ctx, c); err != nil {
		return err
	}
	return s.commandRepo.MarkSent(ctx, c.ID, time.Now())
}

var _ CommandUseCase = (*CommandService)(nil)
