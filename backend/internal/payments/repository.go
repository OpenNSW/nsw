package payments

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// PaymentRepository defines the interface for managing PaymentTransactions.
type PaymentRepository interface {
	Create(ctx context.Context, tx *PaymentTransaction) error
	GetByReferenceNumber(ctx context.Context, referenceNumber string) (*PaymentTransaction, error)
	GetByTaskID(ctx context.Context, taskID string) (*PaymentTransaction, error)
	Update(ctx context.Context, tx *PaymentTransaction) error
	UpdateStatus(ctx context.Context, referenceNumber string, status PaymentStatus) error
	WithTx(tx *gorm.DB) PaymentRepository
}

type paymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository creates a new instance of PaymentRepository.
func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

// WithTx enables transaction propagation.
func (r *paymentRepository) WithTx(tx *gorm.DB) PaymentRepository {
	return NewPaymentRepository(tx)
}

// Create inserts a new PaymentTransaction into the database.
func (r *paymentRepository) Create(ctx context.Context, ptx *PaymentTransaction) error {
	return r.db.WithContext(ctx).Create(ptx).Error
}

// GetByReferenceNumber retrieves a PaymentTransaction by its reference number.
func (r *paymentRepository) GetByReferenceNumber(ctx context.Context, referenceNumber string) (*PaymentTransaction, error) {
	var ptx PaymentTransaction
	if err := r.db.WithContext(ctx).Where("reference_number = ?", referenceNumber).First(&ptx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil when not found to easily check existence
		}
		return nil, err
	}
	return &ptx, nil
}

// GetByTaskID retrieves a PaymentTransaction by its associated TaskID.
func (r *paymentRepository) GetByTaskID(ctx context.Context, taskID string) (*PaymentTransaction, error) {
	var ptx PaymentTransaction
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&ptx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ptx, nil
}

// Update saves changes to an existing PaymentTransaction.
func (r *paymentRepository) Update(ctx context.Context, ptx *PaymentTransaction) error {
	return r.db.WithContext(ctx).Save(ptx).Error
}

// UpdateStatus updates only the status field of a PaymentTransaction.
func (r *paymentRepository) UpdateStatus(ctx context.Context, referenceNumber string, status PaymentStatus) error {
	return r.db.WithContext(ctx).Model(&PaymentTransaction{}).Where("reference_number = ?", referenceNumber).Updates(map[string]interface{}{"status": status}).Error
}
