package email

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/audetv/urms/internal/core/domain"
)

// AddressNormalizer нормализует email адреса
type AddressNormalizer struct {
	domainAliases map[string]string
}

// NewAddressNormalizer создает новый нормализатор
func NewAddressNormalizer() *AddressNormalizer {
	return &AddressNormalizer{
		domainAliases: map[string]string{
			"gmail.com":      "gmail.com",
			"googlemail.com": "gmail.com",
			"yahoo.com":      "yahoo.com",
			"ymail.com":      "yahoo.com",
		},
	}
}

// NormalizeEmailAddress нормализует один email адрес
func (n *AddressNormalizer) NormalizeEmailAddress(address string) (string, error) {
	if address == "" {
		return "", fmt.Errorf("empty email address")
	}

	// Парсим адрес
	parsed, err := mail.ParseAddress(address)
	if err != nil {
		// Пробуем извлечь адрес с помощью regex
		return n.extractWithRegex(address)
	}

	// Нормализуем домен
	normalized := n.normalizeDomain(parsed.Address)

	// Проверяем валидность
	if err := n.validateEmail(normalized); err != nil {
		return "", fmt.Errorf("invalid email address %s: %w", normalized, err)
	}

	return normalized, nil
}

// ConvertToDomainAddresses конвертирует строки в domain.EmailAddress
func (n *AddressNormalizer) ConvertToDomainAddresses(addresses []string) []domain.EmailAddress {
	result := make([]domain.EmailAddress, 0, len(addresses))

	for _, addr := range addresses {
		normalized, err := n.NormalizeEmailAddress(addr)
		if err != nil {
			continue // Пропускаем невалидные
		}
		result = append(result, domain.EmailAddress(normalized))
	}

	return result
}

// normalizeDomain нормализует доменную часть адреса
func (n *AddressNormalizer) normalizeDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	localPart := parts[0]
	domainPart := strings.ToLower(parts[1])

	// Применяем алиасы доменов
	if alias, exists := n.domainAliases[domainPart]; exists {
		domainPart = alias
	}

	// Специальная обработка для Gmail
	if domainPart == "gmail.com" {
		localPart = n.normalizeGmailLocalPart(localPart)
	}

	return localPart + "@" + domainPart
}

// normalizeGmailLocalPart нормализует локальную часть для Gmail
func (n *AddressNormalizer) normalizeGmailLocalPart(localPart string) string {
	// Убираем точки
	localPart = strings.ReplaceAll(localPart, ".", "")

	// Убираем всё после +
	if plusIndex := strings.Index(localPart, "+"); plusIndex != -1 {
		localPart = localPart[:plusIndex]
	}

	return strings.ToLower(localPart)
}

// extractWithRegex извлекает email адрес с помощью regex
func (n *AddressNormalizer) extractWithRegex(text string) (string, error) {
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	matches := emailRegex.FindStringSubmatch(text)

	if len(matches) == 0 {
		return "", fmt.Errorf("no email address found in: %s", text)
	}

	return n.normalizeDomain(strings.ToLower(matches[0])), nil
}

// validateEmail проверяет валидность email адреса
func (n *AddressNormalizer) validateEmail(email string) error {
	if len(email) < 3 || len(email) > 254 {
		return fmt.Errorf("email length invalid")
	}

	if !utf8.ValidString(email) {
		return fmt.Errorf("email contains invalid characters")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("email must contain exactly one @")
	}

	localPart := parts[0]
	domainPart := parts[1]

	if len(localPart) == 0 || len(localPart) > 64 {
		return fmt.Errorf("local part length invalid")
	}

	if len(domainPart) == 0 || len(domainPart) > 253 {
		return fmt.Errorf("domain part length invalid")
	}

	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !domainRegex.MatchString(domainPart) {
		return fmt.Errorf("invalid domain format")
	}

	return nil
}
