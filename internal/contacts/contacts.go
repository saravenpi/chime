package contacts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Contact struct {
	Name         string   `yaml:"name"`
	PhoneNumbers []string `yaml:"phone_numbers,omitempty"`
	Emails       []string `yaml:"emails,omitempty"`
}

var (
	contactCache      []Contact
	contactLookupMap  map[string]string
	contactCacheMutex sync.RWMutex
	contactCacheTime  time.Time
	cacheDuration     = 30 * time.Second
)

// GetContactsDir returns the path to the contacts directory (~/.chime/contacts).
func GetContactsDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".chime", "contacts")
}

// ensureContactsDir creates the contacts directory if it doesn't exist.
func ensureContactsDir() error {
	dir := GetContactsDir()
	return os.MkdirAll(dir, 0755)
}

// sanitizeFilename converts a contact name to a safe filename.
func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	return name
}

// getContactFilePath returns the full path to a contact's YAML file.
func getContactFilePath(name string) string {
	filename := sanitizeFilename(name) + ".yml"
	return filepath.Join(GetContactsDir(), filename)
}

// SaveContact saves a contact to a YAML file in ~/.chime/contacts/.
func SaveContact(contact Contact) error {
	if contact.Name == "" {
		return fmt.Errorf("contact name cannot be empty")
	}

	if err := ensureContactsDir(); err != nil {
		return fmt.Errorf("failed to create contacts directory: %w", err)
	}

	filePath := getContactFilePath(contact.Name)

	data, err := yaml.Marshal(&contact)
	if err != nil {
		return fmt.Errorf("failed to marshal contact: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write contact file: %w", err)
	}

	return nil
}

// LoadContact loads a contact from a YAML file by name.
func LoadContact(name string) (*Contact, error) {
	filePath := getContactFilePath(name)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("contact not found: %s", name)
		}
		return nil, fmt.Errorf("failed to read contact file: %w", err)
	}

	var contact Contact
	if err := yaml.Unmarshal(data, &contact); err != nil {
		return nil, fmt.Errorf("failed to parse contact file: %w", err)
	}

	return &contact, nil
}

// DeleteContact deletes a contact's YAML file.
func DeleteContact(name string) error {
	filePath := getContactFilePath(name)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("contact not found: %s", name)
		}
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	return nil
}

// ListContacts returns all contacts from the contacts directory.
// Results are cached for 30 seconds to improve performance.
func ListContacts() ([]Contact, error) {
	contactCacheMutex.RLock()
	if time.Since(contactCacheTime) < cacheDuration && contactCache != nil {
		defer contactCacheMutex.RUnlock()
		return contactCache, nil
	}
	contactCacheMutex.RUnlock()

	contactCacheMutex.Lock()
	defer contactCacheMutex.Unlock()

	if time.Since(contactCacheTime) < cacheDuration && contactCache != nil {
		return contactCache, nil
	}

	dir := GetContactsDir()

	if err := ensureContactsDir(); err != nil {
		return nil, fmt.Errorf("failed to create contacts directory: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read contacts directory: %w", err)
	}

	var contacts []Contact
	lookupMap := make(map[string]string)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var contact Contact
		if err := yaml.Unmarshal(data, &contact); err != nil {
			continue
		}

		contacts = append(contacts, contact)

		for _, phone := range contact.PhoneNumbers {
			normalized := normalizeIdentifier(phone)
			lookupMap[normalized] = contact.Name
		}

		for _, email := range contact.Emails {
			normalized := strings.ToLower(strings.TrimSpace(email))
			lookupMap[normalized] = contact.Name
		}
	}

	contactCache = contacts
	contactLookupMap = lookupMap
	contactCacheTime = time.Now()

	return contacts, nil
}

// InvalidateCache forces the contact cache to be refreshed on next access.
func InvalidateCache() {
	contactCacheMutex.Lock()
	defer contactCacheMutex.Unlock()
	contactCacheTime = time.Time{}
}

// FindContactByIdentifier searches for a contact by phone number or email.
// Returns the contact name if found, empty string otherwise.
// Uses an optimized lookup map for O(1) performance.
func FindContactByIdentifier(identifier string) string {
	if identifier == "" {
		return ""
	}

	_, err := ListContacts()
	if err != nil {
		return ""
	}

	contactCacheMutex.RLock()
	defer contactCacheMutex.RUnlock()

	identifier = strings.TrimSpace(identifier)
	normalizedIdentifier := normalizeIdentifier(identifier)

	if name, ok := contactLookupMap[normalizedIdentifier]; ok {
		return name
	}

	return ""
}

// normalizeIdentifier removes non-numeric characters from phone numbers for comparison.
func normalizeIdentifier(s string) string {
	if strings.Contains(s, "@") {
		return strings.ToLower(s)
	}

	result := ""
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '+' {
			result += string(r)
		}
	}

	if strings.HasPrefix(result, "1") && len(result) == 11 {
		result = result[1:]
	}
	if strings.HasPrefix(result, "+1") && len(result) == 12 {
		result = result[2:]
	}

	return result
}
