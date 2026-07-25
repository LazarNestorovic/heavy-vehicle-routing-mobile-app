package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type ChatMessage struct {
	ID           int64
	FromDriverID int64
	ToDriverID   int64
	Body         string
	SentAt       time.Time
	ReadAt       *time.Time
}

// ChatConversation summarizes one thread for the chat list screen: who it's
// with, the last message, and how many are unread by the requesting driver.
type ChatConversation struct {
	CounterpartID int64
	LastMessage   string
	LastMessageAt time.Time
	UnreadCount   int
}

type ChatMessageStore struct {
	db *sql.DB
}

func NewChatMessageStore(db *sql.DB) *ChatMessageStore {
	return &ChatMessageStore{db: db}
}

func (s *ChatMessageStore) Create(ctx context.Context, fromDriverID, toDriverID int64, body string) (ChatMessage, error) {
	m := ChatMessage{FromDriverID: fromDriverID, ToDriverID: toDriverID, Body: body}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO chat_messages (from_driver_id, to_driver_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, sent_at`, fromDriverID, toDriverID, body)

	if err := row.Scan(&m.ID, &m.SentAt); err != nil {
		return ChatMessage{}, fmt.Errorf("insert chat message: %w", err)
	}
	return m, nil
}

func (s *ChatMessageStore) ListThread(ctx context.Context, a, b int64) ([]ChatMessage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, from_driver_id, to_driver_id, body, sent_at, read_at
		FROM chat_messages
		WHERE (from_driver_id = $1 AND to_driver_id = $2) OR (from_driver_id = $2 AND to_driver_id = $1)
		ORDER BY sent_at ASC`, a, b)
	if err != nil {
		return nil, fmt.Errorf("list chat thread: %w", err)
	}
	defer rows.Close()

	messages := []ChatMessage{}
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.FromDriverID, &m.ToDriverID, &m.Body, &m.SentAt, &m.ReadAt); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// ListConversations returns one row per counterpart driverID has ever
// exchanged a message with (only existing threads - see internal/httpapi
// GET /api/v1/drivers for the broader "start a new chat" contact list).
func (s *ChatMessageStore) ListConversations(ctx context.Context, driverID int64) ([]ChatConversation, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH convo AS (
			SELECT
				CASE WHEN from_driver_id = $1 THEN to_driver_id ELSE from_driver_id END AS counterpart_id,
				body, sent_at, read_at, to_driver_id
			FROM chat_messages
			WHERE from_driver_id = $1 OR to_driver_id = $1
		),
		last_msg AS (
			SELECT DISTINCT ON (counterpart_id) counterpart_id, body, sent_at
			FROM convo
			ORDER BY counterpart_id, sent_at DESC
		),
		unread AS (
			SELECT counterpart_id, COUNT(*) AS unread_count
			FROM convo
			WHERE to_driver_id = $1 AND read_at IS NULL
			GROUP BY counterpart_id
		)
		SELECT lm.counterpart_id, lm.body, lm.sent_at, COALESCE(u.unread_count, 0)
		FROM last_msg lm
		LEFT JOIN unread u ON u.counterpart_id = lm.counterpart_id
		ORDER BY lm.sent_at DESC`, driverID)
	if err != nil {
		return nil, fmt.Errorf("list chat conversations: %w", err)
	}
	defer rows.Close()

	conversations := []ChatConversation{}
	for rows.Next() {
		var c ChatConversation
		if err := rows.Scan(&c.CounterpartID, &c.LastMessage, &c.LastMessageAt, &c.UnreadCount); err != nil {
			return nil, fmt.Errorf("scan chat conversation: %w", err)
		}
		conversations = append(conversations, c)
	}
	return conversations, rows.Err()
}

// MarkRead marks every message counterpartID sent to driverID as read.
func (s *ChatMessageStore) MarkRead(ctx context.Context, driverID, counterpartID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE chat_messages SET read_at = now()
		WHERE to_driver_id = $1 AND from_driver_id = $2 AND read_at IS NULL`,
		driverID, counterpartID)
	if err != nil {
		return fmt.Errorf("mark chat read: %w", err)
	}
	return nil
}
