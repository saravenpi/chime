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

// getLocalContactName looks up a contact in ~/.chime/contacts/.
func getLocalContactName(identifier string) string {
	return contacts.FindContactByIdentifier(identifier)
}

// GetContactName retrieves a contact name for the given identifier (phone/email).
// First checks local ~/.chime contacts, then falls back to AppleScript and system contacts.
func GetContactName(identifier string) string {
	if identifier == "" {
		return ""
	}

	localName := getLocalContactName(identifier)
	if localName != "" {
		return localName
	}

	name := GetContactNameViaAppleScript(identifier)
	if name != "" {
		return name
	}

	contactsDBPath := GetContactsDBPath()
	if _, err := os.Stat(contactsDBPath); os.IsNotExist(err) {
		return ""
	}

	db, err := sql.Open("sqlite3", contactsDBPath+"?mode=ro")
	if err != nil {
		return ""
	}
	defer db.Close()

	normalizedID := normalizePhoneNumber(identifier)

	phoneQuery := `
		SELECT DISTINCT ZABCDRECORD.ZFIRSTNAME, ZABCDRECORD.ZLASTNAME, ZABCDPHONENUMBER.ZFULLNUMBER
		FROM ZABCDRECORD
		LEFT JOIN ZABCDPHONENUMBER ON ZABCDRECORD.Z_PK = ZABCDPHONENUMBER.ZOWNER
		WHERE ZABCDPHONENUMBER.ZFULLNUMBER IS NOT NULL
	`

	rows, err := db.Query(phoneQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var firstName, lastName sql.NullString
			var phoneNumber string
			if err := rows.Scan(&firstName, &lastName, &phoneNumber); err != nil {
				continue
			}

			normalizedPhone := normalizePhoneNumber(phoneNumber)
			if normalizedPhone == normalizedID || strings.Contains(normalizedPhone, normalizedID) || strings.Contains(normalizedID, normalizedPhone) {
				name := strings.TrimSpace(firstName.String + " " + lastName.String)
				if name != "" {
					return name
				}
			}
		}
	}

	emailQuery := `
		SELECT DISTINCT ZABCDRECORD.ZFIRSTNAME, ZABCDRECORD.ZLASTNAME, ZABCDEMAILADDRESS.ZADDRESS
		FROM ZABCDRECORD
		LEFT JOIN ZABCDEMAILADDRESS ON ZABCDRECORD.Z_PK = ZABCDEMAILADDRESS.ZOWNER
		WHERE ZABCDEMAILADDRESS.ZADDRESS IS NOT NULL
	`

	rows2, err := db.Query(emailQuery)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var firstName, lastName sql.NullString
			var email string
			if err := rows2.Scan(&firstName, &lastName, &email); err != nil {
				continue
			}

			if strings.EqualFold(email, identifier) {
				name := strings.TrimSpace(firstName.String + " " + lastName.String)
				if name != "" {
					return name
				}
			}
		}
	}

	messagingQuery := `
		SELECT DISTINCT ZABCDRECORD.ZFIRSTNAME, ZABCDRECORD.ZLASTNAME, ZABCDMESSAGINGADDRESS.ZADDRESS
		FROM ZABCDRECORD
		LEFT JOIN ZABCDMESSAGINGADDRESS ON ZABCDRECORD.Z_PK = ZABCDMESSAGINGADDRESS.ZOWNER
		WHERE ZABCDMESSAGINGADDRESS.ZADDRESS IS NOT NULL
	`

	rows3, err := db.Query(messagingQuery)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var firstName, lastName sql.NullString
			var address string
			if err := rows3.Scan(&firstName, &lastName, &address); err != nil {
				continue
			}

			if strings.EqualFold(address, identifier) {
				name := strings.TrimSpace(firstName.String + " " + lastName.String)
				if name != "" {
					return name
				}
			}
		}
	}

	return ""
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

func GetChats(onContactLoaded func(int64, string)) ([]models.Chat, error) {
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
			COALESCE(m.date, 0)
		FROM chat c
		LEFT JOIN chat_message_join cmj ON c.ROWID = cmj.chat_id
		LEFT JOIN message m ON cmj.message_id = m.ROWID
		WHERE m.ROWID IN (
			SELECT MAX(message_id)
			FROM chat_message_join
			WHERE chat_id = c.ROWID
		) OR m.ROWID IS NULL
		ORDER BY m.date DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats: %w", err)
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var chat models.Chat
		var dateNano int64
		var attributedBody []byte
		err := rows.Scan(&chat.ROWID, &chat.ChatID, &chat.DisplayName, &chat.LastMessage, &attributedBody, &dateNano)
		if err != nil {
			continue
		}

		if chat.LastMessage == "" && len(attributedBody) > 0 {
			chat.LastMessage = extractTextFromAttributedBody(attributedBody)
		}

		chat.IsGroup = strings.HasPrefix(chat.ChatID, "chat")

		if chat.IsGroup {
			participants, err := GetChatParticipants(db, chat.ROWID)
			if err == nil {
				chat.Participants = participants
				if chat.DisplayName == "" && len(participants) > 0 {
					participantNames := make([]string, 0, len(participants))
					for _, p := range participants {
						name := GetContactName(p)
						if name != "" {
							participantNames = append(participantNames, name)
						} else {
							participantNames = append(participantNames, p)
							if onContactLoaded != nil {
								LoadContactForIdentifier(p, func(identifier, contactName string) {
									onContactLoaded(chat.ROWID, contactName)
								})
							}
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
				if onContactLoaded != nil {
					LoadContactForIdentifier(chat.ChatID, func(identifier, name string) {
						onContactLoaded(chat.ROWID, name)
					})
				}
			}
		}

		if dateNano > 0 {
			chat.LastTime = time.Unix(0, dateNano+978307200000000000)
		}

		chats = append(chats, chat)
	}

	return chats, nil
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
