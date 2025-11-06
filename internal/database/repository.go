package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Recipient struct {
	ID             int64
	UserID         int64
	Username       string
	IsActive       bool
	AllowSending   bool
	LastSentAt     *time.Time
	DeliveryStatus string
	ErrorMessage   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MessageLog struct {
	ID               int64
	MessageID        int
	MessageType      string
	MessageText      string
	SentAt           time.Time
	TotalRecipients  int
	SuccessfullySent int
}

type Repository struct {
	db *Database
}

func NewRepository(db *Database) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetActiveRecipients(ctx context.Context) ([]*Recipient, error) {
	query := `
        SELECT id, user_id, username, is_active, allow_sending, last_sent_at, 
               delivery_status, error_message, created_at, updated_at
        FROM recipients
        WHERE is_active = true AND allow_sending = true
        ORDER BY user_id
    `

	rows, err := r.db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	var recipients []*Recipient
	for rows.Next() {
		var recipient Recipient
		err := rows.Scan(
			&recipient.ID,
			&recipient.UserID,
			&recipient.Username,
			&recipient.IsActive,
			&recipient.AllowSending,
			&recipient.LastSentAt,
			&recipient.DeliveryStatus,
			&recipient.ErrorMessage,
			&recipient.CreatedAt,
			&recipient.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		recipients = append(recipients, &recipient)
	}

	return recipients, nil
}

func (r *Repository) UpsertRecipient(ctx context.Context, userID int64, username string) error {
	query := `
        INSERT INTO recipients (user_id, username, is_active, allow_sending, delivery_status)
        VALUES ($1, $2, true, true, 'pending')
        ON CONFLICT (user_id)
        DO UPDATE SET
            username = EXCLUDED.username,
            updated_at = CURRENT_TIMESTAMP
    `

	_, err := r.db.pool.Exec(ctx, query, userID, username)
	if err != nil {
		return fmt.Errorf("ошибка добавления/обновления получателя: %w", err)
	}

	return nil
}

func (r *Repository) UpdateDeliveryStatus(ctx context.Context, userID int64, status string, errorMsg *string) error {
	query := `
        UPDATE recipients
        SET delivery_status = $1,
            error_message = $2,
            last_sent_at = CURRENT_TIMESTAMP,
            updated_at = CURRENT_TIMESTAMP
        WHERE user_id = $3
    `

	_, err := r.db.pool.Exec(ctx, query, status, errorMsg, userID)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса доставки: %w", err)
	}

	return nil
}

func (r *Repository) CreateMessageLog(ctx context.Context, messageID int, messageType, messageText string, totalRecipients, successfullySent int) error {
	query := `
        INSERT INTO message_logs (message_id, message_type, message_text, total_recipients, successfully_sent)
        VALUES ($1, $2, $3, $4, $5)
    `

	_, err := r.db.pool.Exec(ctx, query, messageID, messageType, messageText, totalRecipients, successfullySent)
	if err != nil {
		return fmt.Errorf("ошибка создания лога сообщения: %w", err)
	}

	return nil
}

func (r *Repository) GetRecipientByUserID(ctx context.Context, userID int64) (*Recipient, error) {
	query := `
        SELECT id, user_id, username, is_active, allow_sending, last_sent_at, 
               delivery_status, error_message, created_at, updated_at
        FROM recipients
        WHERE user_id = $1
    `

	var recipient Recipient
	err := r.db.pool.QueryRow(ctx, query, userID).Scan(
		&recipient.ID,
		&recipient.UserID,
		&recipient.Username,
		&recipient.IsActive,
		&recipient.AllowSending,
		&recipient.LastSentAt,
		&recipient.DeliveryStatus,
		&recipient.ErrorMessage,
		&recipient.CreatedAt,
		&recipient.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("получатель с user_id %d не найден", userID)
		}
		return nil, fmt.Errorf("ошибка получения получателя: %w", err)
	}

	return &recipient, nil
}

func (r *Repository) DeactivateRecipient(ctx context.Context, userID int64) error {
	query := `
        UPDATE recipients
        SET is_active = false,
            updated_at = CURRENT_TIMESTAMP
        WHERE user_id = $1
    `

	_, err := r.db.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("ошибка деактивации получателя: %w", err)
	}

	return nil
}

func (r *Repository) SetAllowSending(ctx context.Context, userID int64, allowSending bool) error {
	query := `
        UPDATE recipients
        SET allow_sending = $1,
            updated_at = CURRENT_TIMESTAMP
        WHERE user_id = $2
    `

	_, err := r.db.pool.Exec(ctx, query, allowSending, userID)
	if err != nil {
		return fmt.Errorf("ошибка обновления разрешения на отправку: %w", err)
	}

	return nil
}

func (r *Repository) IsEventSent(ctx context.Context, eventHash string) (bool, error) {
	query := `
        SELECT EXISTS(SELECT 1 FROM sent_events WHERE event_hash = $1)
    `

	var exists bool
	err := r.db.pool.QueryRow(ctx, query, eventHash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки отправленного события: %w", err)
	}

	return exists, nil
}

func (r *Repository) MarkEventAsSent(ctx context.Context, eventDate time.Time, eventDescription, eventHash string) error {
	query := `
        INSERT INTO sent_events (event_date, event_description, event_hash, sent_at)
        VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
        ON CONFLICT (event_hash) DO NOTHING
    `

	_, err := r.db.pool.Exec(ctx, query, eventDate, eventDescription, eventHash)
	if err != nil {
		return fmt.Errorf("ошибка сохранения отправленного события: %w", err)
	}

	return nil
}

func GenerateEventHash(eventDate time.Time, eventDescription string) string {
	dateStr := eventDate.Format("2006-01-02")
	data := fmt.Sprintf("%s|%s", dateStr, eventDescription)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
