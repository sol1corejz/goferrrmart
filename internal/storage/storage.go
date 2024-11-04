package storage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/sol1corejz/goferrrmart/cmd/config"
	"github.com/sol1corejz/goferrrmart/internal/logger"
	"github.com/sol1corejz/goferrrmart/internal/models"
	"go.uber.org/zap"
	"time"
)

var (
	DB                     *sql.DB
	ErrConnectionFailed    = errors.New("db connection failed")
	ErrCreatingTableFailed = errors.New("creating table failed")
)

func Init() error {
	if config.DatabaseURI == "" {
		return ErrConnectionFailed
	}

	db, err := sql.Open("pgx", config.DatabaseURI)
	if err != nil {
		logger.Log.Fatal("Error opening database connection", zap.Error(err))
		return ErrConnectionFailed
	}
	DB = db

	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY NOT NULL,
			login VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id),
			order_number VARCHAR(255) UNIQUE NOT NULL,
			status VARCHAR(20) NOT NULL,
			accrual DECIMAL(10, 2) DEFAULT 0.00,
			uploaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS user_balances (
    		id SERIAL PRIMARY KEY NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id),
			current_balance DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			withdrawn_total DECIMAL(10, 2) NOT NULL DEFAULT 0.00
		);`,
		`CREATE TABLE IF NOT EXISTS withdrawals (
			id SERIAL PRIMARY KEY NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id),
			order_number VARCHAR(255) NOT NULL,
			sum DECIMAL(10, 2) NOT NULL,
			processed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, table := range tables {
		if _, err := DB.Exec(table); err != nil {
			logger.Log.Error("Error creating table", zap.Error(err))
			return ErrCreatingTableFailed
		}
	}

	return nil
}

func GetUserByLogin(ctx context.Context, login string) (models.User, error) {

	var existingUser models.User

	err := DB.QueryRowContext(ctx, `
		SELECT * FROM users WHERE login = $1;
	`, login).Scan(&existingUser.ID, &existingUser.Login, &existingUser.PasswordHash, &existingUser.CreatedAt)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return models.User{}, err
		}
	}

	return existingUser, nil
}

func CreateUser(ctx context.Context, userID string, login string, passwordHash string) error {

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	var uID string

	err = tx.QueryRowContext(ctx, `
		INSERT INTO users (id, login, password_hash) VALUES ($1, $2, $3) RETURNING id;
	`, userID, login, passwordHash).Scan(&uID)

	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_balances (user_id) VALUES ($1);
	`, uID)

	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func CreateOrder(ctx context.Context, userID string, orderNumber string) error {

	_, err := DB.ExecContext(ctx, `
        INSERT INTO orders (user_id, order_number, status) VALUES ($1, $2, $3) ON CONFLICT (order_number) DO NOTHING;
    `, userID, orderNumber, models.NEW)

	if err != nil {
		logger.Log.Error("Error creating order: %v", zap.Error(err))
		return err
	}

	return nil
}

func GetUserOrders(ctx context.Context, UUID uuid.UUID) ([]models.Order, error) {

	var orders []models.Order

	rows, err := DB.QueryContext(ctx, `
		SELECT * FROM orders WHERE user_id = $1;
	`, UUID)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return []models.Order{}, err
		}
	}

	defer rows.Close()

	for rows.Next() {
		var order models.Order
		err = rows.Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func GetOrderByNumber(ctx context.Context, orderNumber string) (models.Order, error) {

	var order models.Order

	err := DB.QueryRowContext(ctx, `
		SELECT * FROM orders WHERE order_number = $1;
	`, orderNumber).Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Status, &order.Accrual, &order.UploadedAt)

	if err != nil {
		return models.Order{}, err
	}

	return order, nil
}

func GetUserBalance(ctx context.Context, UUID uuid.UUID) (models.UserBalance, error) {

	var balance models.UserBalance

	err := DB.QueryRowContext(ctx, `
		SELECT * FROM user_balances WHERE user_id = $1;
	`, UUID).Scan(&balance.ID, &balance.UserID, &balance.CurrentBalance, &balance.WithdrawnTotal)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return models.UserBalance{}, err
		}
	}

	return balance, nil
}

func GetUserWithdrawals(ctx context.Context, UUID uuid.UUID) ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal

	rows, err := DB.QueryContext(ctx, `
		SELECT * FROM withdrawals WHERE user_id = $1 ORDER BY processed_at;
	`, UUID)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return []models.Withdrawal{}, err
		}
	}

	defer rows.Close()

	for rows.Next() {
		var withdrawal models.Withdrawal
		err = rows.Scan(&withdrawal.ID, &withdrawal.UserID, &withdrawal.OrderNumber, &withdrawal.Sum, &withdrawal.ProcessedAt)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return withdrawals, nil
}

func CreateWithdrawal(ctx context.Context, userID uuid.UUID, order string, sum float64) error {
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO withdrawals (user_id, order_number, sum, processed_at) 
		VALUES ($1, $2, $3, $4)
	`, userID, order, sum, time.Now().Format(time.RFC3339))
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE user_balances 
		SET current_balance = current_balance - $1, withdrawn_total = withdrawn_total + $1 
		WHERE user_id = $2
	`, sum, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func GetAllUnprocessedOrders(ctx context.Context) ([]models.Order, error) {
	var orders []models.Order

	rows, err := DB.QueryContext(ctx, `
		SELECT * FROM orders WHERE status NOT IN ('INVALID', 'PROCESSED');;
	`)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return []models.Order{}, err
		}
	}

	defer rows.Close()

	for rows.Next() {
		var order models.Order
		err = rows.Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func UpdateOrder(ctx context.Context, orderID int, orderStatus string, orderAccrual float64, userID uuid.UUID) error {
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE orders SET status = $1, accrual = $2 WHERE id = $3`, orderStatus, orderAccrual, orderID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE user_balances SET current_balance = $1 WHERE user_id = $2`, orderAccrual, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
