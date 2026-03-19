package payment_types

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PaymentTransactionDB maps to the payment_transactions table in the database
type PaymentTransactionDB struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key"`
	TaskID          uuid.UUID `gorm:"type:uuid;not null;index"`
	ExecutionID     string    `gorm:"type:varchar(100);not null;index"`
	ReferenceNumber string    `gorm:"type:varchar(100);not null;unique"`
	ProviderID      string    `gorm:"type:varchar(50);not null;index"`
	Status          string    `gorm:"type:varchar(50);not null;default:'PENDING'"`
	Amount          float64   `gorm:"type:numeric(15,2);not null"`
	Currency        string    `gorm:"type:varchar(10);not null;default:'LKR'"`
	PayerName       string    `gorm:"type:varchar(255)"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (PaymentTransactionDB) TableName() string {
	return "payment_transactions"
}

// PaymentRepository defines the data access layer for payment transactions
type PaymentRepository interface {
	CreateTransaction(ctx context.Context, trx *PaymentTransactionDB) error
	GetTransactionByReference(ctx context.Context, ref string, forUpdate bool) (*PaymentTransactionDB, error)
	GetTransactionByExecutionID(ctx context.Context, execID string) (*PaymentTransactionDB, error)
	UpdateTransactionStatus(ctx context.Context, ref string, status string) error
}
