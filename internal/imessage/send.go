package imessage

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/saravenpi/chime/internal/models"
)

// escapeAppleScript escapes special characters in strings for safe use in AppleScript.
// It handles backslashes, quotes, and control characters (newlines, tabs, etc).
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

var contactCache = make(map[string]string)
var contactCacheMutex sync.RWMutex
var contactCacheInitialized bool

// LoadContactForIdentifier asynchronously looks up a contact name for the given identifier
// (phone number or email). Results are cached and delivered via callback when found.
func LoadContactForIdentifier(identifier string, callback func(string, string)) {
	go func() {
		contactCacheMutex.RLock()
		if name, ok := contactCache[identifier]; ok {
			contactCacheMutex.RUnlock()
			callback(identifier, name)
			return
		}
		contactCacheMutex.RUnlock()

		cleanedIdentifier := strings.Map(func(r rune) rune {
			if (r >= '0' && r <= '9') || r == '+' {
				return r
			}
			return -1
		}, identifier)

		script := fmt.Sprintf(`
			tell application "Contacts"
				set targetNumber to "%s"
				try
					repeat with aPerson in people
						try
							repeat with aPhone in phones of aPerson
								set phoneValue to value of aPhone
								set cleanPhone to do shell script "echo " & quoted form of phoneValue & " | tr -cd '0-9+'"
								if cleanPhone contains targetNumber or targetNumber contains cleanPhone then
									return name of aPerson
								end if
							end repeat
						end try
						try
							repeat with anEmail in emails of aPerson
								set emailValue to value of anEmail
								if emailValue is equal to "%s" then
									return name of aPerson
								end if
							end repeat
						end try
					end repeat
				end try
				return ""
			end tell
		`, escapeAppleScript(cleanedIdentifier), escapeAppleScript(identifier))

		cmd := exec.Command("osascript", "-e", script)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return
		}

		name := strings.TrimSpace(string(output))
		if name != "" && name != "missing value" {
			contactCacheMutex.Lock()
			contactCache[identifier] = name
			contactCache[cleanedIdentifier] = name
			contactCacheMutex.Unlock()
			callback(identifier, name)
		}
	}()
}

// GetContactNameViaAppleScript retrieves a contact name from the cache.
// Returns empty string if not found. Does not trigger a lookup.
func GetContactNameViaAppleScript(identifier string) string {
	if identifier == "" {
		return ""
	}

	contactCacheMutex.RLock()
	defer contactCacheMutex.RUnlock()

	if name, ok := contactCache[identifier]; ok {
		return name
	}

	cleanedIdentifier := strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || r == '+' {
			return r
		}
		return -1
	}, identifier)

	if cleanedIdentifier != identifier {
		if name, ok := contactCache[cleanedIdentifier]; ok {
			return name
		}
	}

	for cachedID, name := range contactCache {
		cachedCleaned := strings.Map(func(r rune) rune {
			if (r >= '0' && r <= '9') || r == '+' {
				return r
			}
			return -1
		}, cachedID)

		if strings.Contains(cachedCleaned, cleanedIdentifier) || strings.Contains(cleanedIdentifier, cachedCleaned) {
			if len(cleanedIdentifier) >= 8 && len(cachedCleaned) >= 8 {
				return name
			}
		}
	}

	return ""
}

// SendMessageToChat sends a message to the specified chat using AppleScript.
// For group chats, tries multiple strategies: chat ID, display name, and participant list.
func SendMessageToChat(chat models.Chat, message string) error {
	if chat.IsGroup {
		var lastErr error

		if chat.ChatID != "" {
			err := sendGroupMessageByChatID(chat.ChatID, message)
			if err == nil {
				return nil
			}
			lastErr = err
		}

		if chat.DisplayName != "" {
			err := sendGroupMessageByName(chat.DisplayName, message)
			if err == nil {
				return nil
			}
			lastErr = err
		}

		if len(chat.Participants) > 0 {
			err := sendGroupMessage(chat.Participants, message)
			if err == nil {
				return nil
			}
			lastErr = err
		}

		if lastErr != nil {
			return fmt.Errorf("all group message sending strategies failed: %w", lastErr)
		}
		return fmt.Errorf("no valid strategy available for group chat")
	}
	return sendIndividualMessage(chat.ChatID, message)
}

// SendMessage sends a message to a single recipient via AppleScript.
func SendMessage(recipient, message string) error {
	return sendIndividualMessage(recipient, message)
}

// sendIndividualMessage sends a message to an individual contact.
// Tries iMessage first, then falls back to SMS if iMessage fails.
func sendIndividualMessage(recipient, message string) error {
	escapedMessage := escapeAppleScript(message)
	escapedRecipient := escapeAppleScript(recipient)

	script := fmt.Sprintf(`
		tell application "Messages"
			try
				if not (exists service 1) then
					return "ERROR: Messages is not signed in to any service"
				end if
			on error
				return "ERROR: Messages is not running or not accessible"
			end try

			set targetService to missing value
			set targetBuddy to missing value
			set messageWasSent to false

			try
				set targetService to 1st service whose service type = iMessage
				try
					set targetBuddy to buddy "%s" of targetService
					send "%s" to targetBuddy
					set messageWasSent to true
				end try
			end try

			if not messageWasSent then
				try
					repeat with svc in services
						if service type of svc is SMS then
							try
								set targetBuddy to buddy "%s" of svc
								send "%s" to targetBuddy
								set messageWasSent to true
								exit repeat
							end try
						end if
					end repeat
				end try
			end if

			if not messageWasSent then
				return "ERROR: Could not send message via iMessage or SMS. Make sure the recipient is valid and SMS forwarding is enabled if messaging Android users"
			end if

			return "SUCCESS"
		end tell
	`, escapedRecipient, escapedMessage, escapedRecipient, escapedMessage)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		if strings.HasPrefix(outputStr, "ERROR:") {
			return fmt.Errorf("%s", strings.TrimPrefix(outputStr, "ERROR: "))
		}
		if strings.HasPrefix(outputStr, "WARNING:") {
			return nil
		}
		if strings.Contains(outputStr, "syntax error") {
			return fmt.Errorf("AppleScript syntax error at %s.\n\nRecipient: %q\nMessage: %q\n\nScript:\n%s\n\nError: %w", outputStr, recipient, message, script, err)
		}
		return fmt.Errorf("AppleScript execution failed: %w (output: %s)", err, outputStr)
	}

	if strings.HasPrefix(outputStr, "ERROR:") {
		return fmt.Errorf("%s", strings.TrimPrefix(outputStr, "ERROR: "))
	}

	if strings.HasPrefix(outputStr, "WARNING:") {
		return nil
	}

	if !strings.Contains(outputStr, "SUCCESS") {
		return fmt.Errorf("unexpected response from Messages.app: %s", outputStr)
	}

	return nil
}

// sendGroupMessageByName sends a message to a group chat by looking it up by display name.
func sendGroupMessageByName(chatName, message string) error {
	escapedMessage := escapeAppleScript(message)
	escapedChatName := escapeAppleScript(chatName)

	script := fmt.Sprintf(`
		tell application "Messages"
			try
				if not (exists service 1) then
					return "ERROR: Messages is not signed in to any service"
				end if
			on error
				return "ERROR: Messages is not running or not accessible"
			end try

			set targetService to missing value
			try
				set targetService to 1st service whose service type = iMessage
			on error
				return "ERROR: iMessage service is not available. Make sure you're signed in to iMessage"
			end try

			set targetChat to missing value
			try
				set targetChat to 1st chat of targetService whose name is "%s"
			on error errMsg
				return "ERROR: Could not find group chat named '%s' - " & errMsg
			end try

			try
				send "%s" to targetChat
			on error errMsg
				return "ERROR: Failed to send message - " & errMsg
			end try

			return "SUCCESS"
		end tell
	`, escapedChatName, escapedChatName, escapedMessage)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		if strings.HasPrefix(outputStr, "ERROR:") {
			return fmt.Errorf("%s", strings.TrimPrefix(outputStr, "ERROR: "))
		}
		if strings.HasPrefix(outputStr, "WARNING:") {
			return nil
		}
		return fmt.Errorf("AppleScript execution failed: %w (output: %s)", err, outputStr)
	}

	if strings.HasPrefix(outputStr, "ERROR:") {
		return fmt.Errorf("%s", strings.TrimPrefix(outputStr, "ERROR: "))
	}

	if strings.HasPrefix(outputStr, "WARNING:") {
		return nil
	}

	if !strings.Contains(outputStr, "SUCCESS") {
		return fmt.Errorf("unexpected response from Messages.app: %s", outputStr)
	}

	return nil
}

// sendGroupMessageByChatID sends a message to a group chat using the chat ID from the database.
// Converts database format (chat123) to AppleScript format (any;+;chat123).
func sendGroupMessageByChatID(chatID, message string) error {
	if strings.HasPrefix(chatID, "chat") && !strings.Contains(chatID, ";") {
		chatID = "any;+;" + chatID
	}

	escapedMessage := escapeAppleScript(message)
	escapedChatID := escapeAppleScript(chatID)

	script := fmt.Sprintf(`
		tell application "Messages"
			try
				if not (exists service 1) then
					return "ERROR: Messages is not signed in to any service"
				end if
			on error
				return "ERROR: Messages is not running or not accessible"
			end try

			set myid to "%s"
			set mymessage to "%s"

			try
				send mymessage to text chat id myid
			on error errMsg
				return "ERROR: Failed to send message - " & errMsg
			end try

			return "SUCCESS"
		end tell
	`, escapedChatID, escapedMessage)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		if strings.HasPrefix(outputStr, "ERROR:") {
			return fmt.Errorf("%s", strings.TrimPrefix(outputStr, "ERROR: "))
		}
		if strings.HasPrefix(outputStr, "WARNING:") {
			return nil
		}
		return fmt.Errorf("AppleScript execution failed: %w (output: %s)", err, outputStr)
	}

	if strings.HasPrefix(outputStr, "ERROR:") {
		return fmt.Errorf("%s", strings.TrimPrefix(outputStr, "ERROR: "))
	}

	if strings.HasPrefix(outputStr, "WARNING:") {
		return nil
	}

	if !strings.Contains(outputStr, "SUCCESS") {
		return fmt.Errorf("unexpected response from Messages.app: %s", outputStr)
	}

	return nil
}

// sendGroupMessage sends a message to a group chat by creating a new chat with the participant list.
func sendGroupMessage(participants []string, message string) error {
	if len(participants) == 0 {
		return fmt.Errorf("no participants in group chat")
	}

	escapedMessage := escapeAppleScript(message)

	escapedParticipants := make([]string, len(participants))
	for i, p := range participants {
		escapedParticipants[i] = escapeAppleScript(p)
	}

	participantsList := strings.Join(escapedParticipants, "\", \"")

	script := fmt.Sprintf(`
		tell application "Messages"
			try
				if not (exists service 1) then
					return "ERROR: Messages is not signed in to any service"
				end if
			on error
				return "ERROR: Messages is not running or not accessible"
			end try

			set targetService to missing value
			try
				set targetService to 1st service whose service type = iMessage
			on error
				return "ERROR: iMessage service is not available. Make sure you're signed in to iMessage"
			end try

			set participantList to {"%s"}

			try
				set thisChat to make new text chat with properties {participants:participantList}
				send "%s" to thisChat
			on error errMsg
				return "ERROR: Failed to send message - " & errMsg
			end try

			return "SUCCESS"
		end tell
	`, participantsList, escapedMessage)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		if strings.HasPrefix(outputStr, "ERROR:") {
			return fmt.Errorf("%s", strings.TrimPrefix(outputStr, "ERROR: "))
		}
		if strings.HasPrefix(outputStr, "WARNING:") {
			return nil
		}
		return fmt.Errorf("AppleScript execution failed: %w (output: %s)", err, outputStr)
	}

	if strings.HasPrefix(outputStr, "ERROR:") {
		return fmt.Errorf("%s", strings.TrimPrefix(outputStr, "ERROR: "))
	}

	if strings.HasPrefix(outputStr, "WARNING:") {
		return nil
	}

	if !strings.Contains(outputStr, "SUCCESS") {
		return fmt.Errorf("unexpected response from Messages.app: %s", outputStr)
	}

	return nil
}
