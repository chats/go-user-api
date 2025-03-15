package transaction

import (
	"context"
	"fmt"
)

// Manager defines a generic transaction manager interface
type Manager[T any] interface {
	// ExecuteTx runs a function inside a transaction
	ExecuteTx(ctx context.Context, fn func(repo T) error) error
}

// Executor is a generic interface for executing database transactions
type Executor interface {
	Commit() error
	Rollback() error
}

// GenericManager implements a generic transaction pattern
type GenericManager[T any, E Executor] struct {
	beginTx    func(ctx context.Context) (E, error)
	createRepo func(tx E) T
}

// NewGenericManager creates a new generic transaction manager
func NewGenericManager[T any, E Executor](
	beginTx func(ctx context.Context) (E, error),
	createRepo func(tx E) T,
) *GenericManager[T, E] {
	return &GenericManager[T, E]{
		beginTx:    beginTx,
		createRepo: createRepo,
	}
}

// ExecuteTx implements the Manager interface
func (m *GenericManager[T, E]) ExecuteTx(ctx context.Context, fn func(repo T) error) error {
	tx, err := m.beginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create the repository with this transaction
	repo := m.createRepo(tx)

	// Execute the provided function
	if err := fn(repo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx failed: %v, unable to rollback: %v", err, rbErr)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
