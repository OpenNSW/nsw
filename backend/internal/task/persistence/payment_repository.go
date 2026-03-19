package persistence

import (
	"context"
	"fmt"

	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type paymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository creates a new instance of payment_types.PaymentRepository.
func NewPaymentRepository(db *gorm.DB) payment_types.PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreateTransaction(ctx context.Context, trx *payment_types.PaymentTransactionDB) error {
	return r.db.WithContext(ctx).Create(trx).Error
}

func (r *paymentRepository) GetTransactionByReference(ctx context.Context, ref string, forUpdate bool) (*payment_types.PaymentTransactionDB, error) {
	var trx payment_types.PaymentTransactionDB
	query := r.db.WithContext(ctx).Where("reference_number = ?", ref)
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.First(&trx).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("transaction not found for reference: %s", ref)
		}
		return nil, err
	}
	return &trx, nil
}

func (r *paymentRepository) GetTransactionByExecutionID(ctx context.Context, execID string) (*payment_types.PaymentTransactionDB, error) {
	var trx payment_types.PaymentTransactionDB
	err := r.db.WithContext(ctx).Where("execution_id = ?", execID).First(&trx).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("transaction not found for execution ID: %s", execID)
		}
		return nil, err
	}
	return &trx, nil
}

func (r *paymentRepository) UpdateTransactionStatus(ctx context.Context, ref string, status string) error {
	return r.db.WithContext(ctx).Model(&payment_types.PaymentTransactionDB{}).
		Where("reference_number = ?", ref).
		Update("status", status).Error
}
