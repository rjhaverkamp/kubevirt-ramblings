package kubevirt

import (
	"context"
	"github.com/kubevirt/kvirtos2/internal/models"
)

// VMClientInterface defines the interface for VM operations
type VMClientInterface interface {
	CreateVM(ctx context.Context, req models.CreateVMParams) (*models.InternalVM, error)
	GetVM(ctx context.Context, serverID string) (*models.InternalVM, error)
	ListVMs(ctx context.Context) ([]models.InternalVM, error)
	DeleteVM(ctx context.Context, serverID string) error
	StartVM(ctx context.Context, serverID string) error
	StopVM(ctx context.Context, serverID string) error
}

// Ensure Client implements VMClientInterface
var _ VMClientInterface = (*Client)(nil)