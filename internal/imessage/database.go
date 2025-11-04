package imessage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	_ "github.com/mattn/go-sqlite3"
	"github.com/saravenpi/chime/internal/contacts"
	"github.com/saravenpi/chime/internal/models"
)

func GetDBPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Library", "Messages", "chat.db")
}

func GetContactsDBPath() string {
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, "Library", "Application Support", "AddressBook")

	possibleVersions := []string{
		"AddressBook-v22.abcddb",
		"AddressBook-v23.abcddb",
		"AddressBook-v24.abcddb",
	}

	for _, version := range possibleVersions {
		path := filepath.Join(baseDir, version)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return filepath.Join(baseDir, "AddressBook-v22.abcddb")
}

func OpenDatabase() (*sql.DB, error) {
	dbPath := GetDBPath()
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

func OpenDatabaseReadWrite() (*sql.DB, error) {
	dbPath := GetDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

func normalizePhoneNumber(phone string) string {
	reg := regexp.MustCompile(`[^0-9+]`)
	normalized := reg.ReplaceAllString(phone, "")
	if strings.HasPrefix(normalized, "1") && len(normalized) == 11 {
		normalized = normalized[1:]
	}
	if strings.HasPrefix(normalized, "+1") && len(normalized) == 12 {
		normalized = normalized[2:]
	}
	return normalized
}

// GetContactName retrieves a contact name from ~/.chime/contacts/.
// Returns empty string if not found.
func GetContactName(identifier string) string {
	if identifier == "" {
		return ""
	}

	return contacts.FindContactByIdentifier(identifier)
}

func extractTextFromAttributedBody(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	nsStringMarker := []byte("NSString")
	parts := [][]byte{}

	remaining := data
	for {
		idx := indexOf(remaining, nsStringMarker)
		if idx == -1 {
			break
		}

		start := idx + len(nsStringMarker)
		if start+5 >= len(remaining) {
			break
		}

		start += 5

		if start >= len(remaining) {
			break
		}

		lengthByte := remaining[start]
		var textLength int
		var textStart int

		if lengthByte == 0x81 {
			if start+3 > len(remaining) {
				break
			}
			textLength = int(remaining[start+1]) | (int(remaining[start+2]) << 8)
			textStart = start + 3
		} else {
			textLength = int(lengthByte)
			textStart = start + 1
		}

		if textStart+textLength > len(remaining) {
			break
		}

		textBytes := remaining[textStart : textStart+textLength]
		if utf8.Valid(textBytes) {
			text := string(textBytes)
			text = strings.TrimSpace(text)
			if len(text) > 0 && !strings.Contains(text, "\x00") {
				parts = append(parts, textBytes)
			}
		}

		remaining = remaining[textStart+textLength:]
	}

	if len(parts) == 0 {
		return ""
	}

	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		text := string(part)
		text = strings.TrimSpace(text)
		if len(text) > 0 {
			texts = append(texts, text)
		}
	}

	return strings.Join(texts, " ")
}

func indexOf(data []byte, pattern []byte) int {
	for i := 0; i <= len(data)-len(pattern); i++ {
		match := true
		for j := 0; j < len(pattern); j++ {
			if data[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func GetChats() ([]models.Chat, error) {
	db, err := OpenDatabase()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			c.ROWID,
			COALESCE(c.chat_identifier, ''),
			COALESCE(c.display_name, ''),
			COALESCE(m.text, ''),
			m.attributedBody,
			COALESCE(m.date, 0),
			COALESCE(unread.count, 0)
		FROM chat c
		LEFT JOIN (
			SELECT cmj.chat_id, cmj.message_id, m.text, m.attributedBody, m.date
			FROM chat_message_join cmj
			JOIN message m ON cmj.message_id = m.ROWID
			WHERE cmj.message_id IN (
				SELECT MAX(message_id)
				FROM chat_message_join
				GROUP BY chat_id
			)
		) m ON c.ROWID = m.chat_id
		LEFT JOIN (
			SELECT cmj.chat_id, COUNT(*) as count
			FROM chat_message_join cmj
			JOIN message msg ON cmj.message_id = msg.ROWID
			WHERE msg.is_read = 0 AND msg.is_from_me = 0
			GROUP BY cmj.chat_id
		) unread ON c.ROWID = unread.chat_id
		ORDER BY m.date DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats: %w", err)
	}
	defer rows.Close()

	var chats []models.Chat
	var chatIDs []int64

	for rows.Next() {
		var chat models.Chat
		var dateNano int64
		var attributedBody []byte
		err := rows.Scan(&chat.ROWID, &chat.ChatID, &chat.DisplayName, &chat.LastMessage, &attributedBody, &dateNano, &chat.UnreadCount)
		if err != nil {
			continue
		}

		if chat.LastMessage == "" && len(attributedBody) > 0 {
			chat.LastMessage = extractTextFromAttributedBody(attributedBody)
		}

		chat.IsGroup = strings.HasPrefix(chat.ChatID, "chat")
		chat.HasUnread = chat.UnreadCount > 0

		if dateNano > 0 {
			chat.LastTime = time.Unix(0, dateNano+978307200000000000)
		}

		chats = append(chats, chat)
		chatIDs = append(chatIDs, chat.ROWID)
	}

	participantsMap, err := GetAllChatParticipants(db, chatIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}

	for i := range chats {
		chat := &chats[i]

		if chat.IsGroup {
			if participants, ok := participantsMap[chat.ROWID]; ok {
				chat.Participants = participants
				if chat.DisplayName == "" && len(participants) > 0 {
					participantNames := make([]string, 0, len(participants))
					for _, p := range participants {
						name := GetContactName(p)
						if name != "" {
							participantNames = append(participantNames, name)
						} else {
							participantNames = append(participantNames, p)
						}
					}
					chat.DisplayName = strings.Join(participantNames, ", ")
				}
			}
		} else {
			chat.Participants = []string{chat.ChatID}
		}

		if chat.DisplayName == "" {
			contactName := GetContactName(chat.ChatID)
			if contactName != "" {
				chat.DisplayName = contactName
			} else {
				chat.DisplayName = chat.ChatID
			}
		}
	}

	return chats, nil
}

// GetAllChatParticipants fetches participants for multiple chats in a single query.
// Returns a map of chatID -> []participants for efficient lookup.
func GetAllChatParticipants(db *sql.DB, chatIDs []int64) (map[int64][]string, error) {
	if len(chatIDs) == 0 {
		return make(map[int64][]string), nil
	}

	query := `
		SELECT chj.chat_id, h.id
		FROM chat_handle_join chj
		JOIN handle h ON chj.handle_id = h.ROWID
		ORDER BY chj.chat_id, h.id
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	participantsMap := make(map[int64][]string)
	for rows.Next() {
		var chatID int64
		var participant string
		if err := rows.Scan(&chatID, &participant); err != nil {
			continue
		}
		participantsMap[chatID] = append(participantsMap[chatID], participant)
	}

	return participantsMap, nil
}

func GetChatParticipants(db *sql.DB, chatID int64) ([]string, error) {
	query := `
		SELECT DISTINCT h.id
		FROM chat_handle_join chj
		JOIN handle h ON chj.handle_id = h.ROWID
		WHERE chj.chat_id = ?
		ORDER BY h.id
	`

	rows, err := db.Query(query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	var participants []string
	for rows.Next() {
		var participant string
		if err := rows.Scan(&participant); err != nil {
			continue
		}
		participants = append(participants, participant)
	}

	return participants, nil
}

func GetMessages(chatID int64) ([]models.Message, error) {
	db, err := OpenDatabase()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			m.ROWID,
			m.guid,
			COALESCE(m.text, ''),
			m.attributedBody,
			COALESCE(h.id, ''),
			m.is_from_me,
			m.date,
			COALESCE(a.filename, '')
		FROM message m
		JOIN chat_message_join cmj ON m.ROWID = cmj.message_id
		LEFT JOIN handle h ON m.handle_id = h.ROWID
		LEFT JOIN message_attachment_join maj ON m.ROWID = maj.message_id
		LEFT JOIN attachment a ON maj.attachment_id = a.ROWID
		WHERE cmj.chat_id = ?
		ORDER BY m.date ASC
	`

	rows, err := db.Query(query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		var dateNano int64
		var attributedBody []byte
		err := rows.Scan(&msg.ROWID, &msg.GUID, &msg.Text, &attributedBody, &msg.Handle, &msg.IsFromMe, &dateNano, &msg.AttachmentPath)
		if err != nil {
			continue
		}

		if msg.Text == "" && len(attributedBody) > 0 {
			msg.Text = extractTextFromAttributedBody(attributedBody)
		}

		if !msg.IsFromMe && msg.Handle != "" {
			contactName := GetContactName(msg.Handle)
			if contactName != "" {
				msg.Handle = contactName
			}
		}

		if dateNano > 0 {
			msg.Date = time.Unix(0, dateNano+978307200000000000)
		}
		msg.ChatID = chatID

		messages = append(messages, msg)
	}

	return messages, nil
}

func MarkChatAsRead(chatID int64) error {
	db, err := OpenDatabaseReadWrite()
	if err != nil {
		return err
	}
	defer db.Close()

	query := `
		UPDATE message
		SET is_read = 1
		WHERE ROWID IN (
			SELECT message_id
			FROM chat_message_join
			WHERE chat_id = ?
		)
		AND is_from_me = 0
		AND is_read = 0
	`

	_, err = db.Exec(query, chatID)
	if err != nil {
		return fmt.Errorf("failed to mark messages as read: %w", err)
	}

	return nil
}
