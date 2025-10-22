// internal/core/domain/email_search_config_test.go
package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailSearchConfig_Validation(t *testing.T) {
	tests := []struct {
		name          string
		defaultDays   int
		extendedDays  int
		maxDays       int
		timeout       time.Duration
		wantErr       bool
		errorContains string
	}{
		{
			name:         "valid configuration",
			defaultDays:  30,
			extendedDays: 90,
			maxDays:      180,
			timeout:      60 * time.Second,
			wantErr:      false,
		},
		{
			name:         "valid configuration with equal days",
			defaultDays:  30,
			extendedDays: 30,
			maxDays:      30,
			timeout:      60 * time.Second,
			wantErr:      false,
		},
		{
			name:          "invalid default days",
			defaultDays:   0,
			extendedDays:  90,
			maxDays:       180,
			timeout:       60 * time.Second,
			wantErr:       true,
			errorContains: "must be positive", // ✅ Сообщение из кода
		},
		{
			name:          "extended less than default",
			defaultDays:   90,
			extendedDays:  30,
			maxDays:       180,
			timeout:       60 * time.Second,
			wantErr:       true,
			errorContains: "extendedDaysBack (30) must be greater than or equal to defaultDaysBack (90)",
		},
		{
			name:          "max less than extended",
			defaultDays:   30,
			extendedDays:  90,
			maxDays:       60,
			timeout:       60 * time.Second,
			wantErr:       true,
			errorContains: "maxDaysBack (60) must be greater than or equal to extendedDaysBack (90)",
		},
		{
			name:          "invalid timeout",
			defaultDays:   30,
			extendedDays:  90,
			maxDays:       180,
			timeout:       0,
			wantErr:       true,
			errorContains: "fetchTimeout must be positive", // ✅ Сообщение из кода
		},
		{
			name:          "max days too large",
			defaultDays:   30,
			extendedDays:  90,
			maxDays:       800,
			timeout:       60 * time.Second,
			wantErr:       true,
			errorContains: "cannot exceed 730 days", // ✅ Сообщение из кода
		},
		{
			name:          "negative days",
			defaultDays:   -1,
			extendedDays:  90,
			maxDays:       180,
			timeout:       60 * time.Second,
			wantErr:       true,
			errorContains: "must be positive", // ✅ Сообщение из кода
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewEmailSearchConfig(
				tt.defaultDays,
				tt.extendedDays,
				tt.maxDays,
				tt.timeout,
				true,
				[]string{"Re:", "Fwd:"},
			)

			if tt.wantErr {
				require.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains,
						"Error message should contain: %s. Got: %s", tt.errorContains, err.Error())
				}
				assert.Nil(t, config, "Config should be nil when error occurs")
			} else {
				require.NoError(t, err, "Expected no error for test case: %s", tt.name)
				require.NotNil(t, config, "Config should not be nil for valid configuration")
				assert.Equal(t, tt.defaultDays, config.DefaultDaysBack())
				assert.Equal(t, tt.extendedDays, config.ExtendedDaysBack())
				assert.Equal(t, tt.maxDays, config.MaxDaysBack())
				assert.Equal(t, tt.timeout, config.FetchTimeout())
			}
		})
	}
}

// internal/core/domain/email_search_config_test.go

func TestNewEmailSearchConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		defaultDays  int
		extendedDays int
		maxDays      int
		shouldWork   bool
	}{
		{
			name:         "minimum valid values",
			defaultDays:  1,
			extendedDays: 1,
			maxDays:      1,
			shouldWork:   true,
		},
		{
			name:         "maximum valid values",
			defaultDays:  730,
			extendedDays: 730,
			maxDays:      730,
			shouldWork:   true,
		},
		{
			name:         "extended equals default",
			defaultDays:  30,
			extendedDays: 30,
			maxDays:      90,
			shouldWork:   true,
		},
		{
			name:         "max equals extended",
			defaultDays:  30,
			extendedDays: 90,
			maxDays:      90,
			shouldWork:   true,
		},
		{
			name:         "all equal",
			defaultDays:  60,
			extendedDays: 60,
			maxDays:      60,
			shouldWork:   true,
		},
		{
			name:         "extended slightly larger than default",
			defaultDays:  30,
			extendedDays: 31,
			maxDays:      90,
			shouldWork:   true,
		},
		{
			name:         "max slightly larger than extended",
			defaultDays:  30,
			extendedDays: 90,
			maxDays:      91,
			shouldWork:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewEmailSearchConfig(
				tt.defaultDays,
				tt.extendedDays,
				tt.maxDays,
				60*time.Second,
				true,
				[]string{"Re:"},
			)

			if tt.shouldWork {
				require.NoError(t, err)
				require.NotNil(t, config)
				assert.Equal(t, tt.defaultDays, config.DefaultDaysBack())
				assert.Equal(t, tt.extendedDays, config.ExtendedDaysBack())
				assert.Equal(t, tt.maxDays, config.MaxDaysBack())
			} else {
				require.Error(t, err)
				assert.Nil(t, config)
			}
		})
	}
}

func TestEmailSearchConfig_IsValidSearchType(t *testing.T) {
	config, err := NewEmailSearchConfig(30, 90, 180, 60*time.Second, true, nil)
	require.NoError(t, err)

	validTypes := []string{"standard", "extended", "maximum"}
	invalidTypes := []string{"", "unknown", "basic", "full"}

	for _, searchType := range validTypes {
		t.Run("valid_"+searchType, func(t *testing.T) {
			assert.True(t, config.IsValidSearchType(searchType),
				"Search type %s should be valid", searchType)
		})
	}

	for _, searchType := range invalidTypes {
		t.Run("invalid_"+searchType, func(t *testing.T) {
			assert.False(t, config.IsValidSearchType(searchType),
				"Search type %s should be invalid", searchType)
		})
	}
}

func TestEmailSearchConfig_SubjectVariants(t *testing.T) {
	config, err := NewEmailSearchConfig(
		30, 90, 180, 60*time.Second, true,
		[]string{"Re:", "Fwd:", "Ответ:"},
	)
	require.NoError(t, err)
	require.NotNil(t, config)

	baseSubject := "Test Subject"
	variants := config.GenerateSubjectVariants(baseSubject)

	expected := []string{
		"Test Subject",
		"Re: Test Subject",
		"Fwd: Test Subject",
		"Ответ: Test Subject",
	}

	assert.Equal(t, expected, variants)
}

func TestEmailSearchConfig_GetSearchSince(t *testing.T) {
	config, err := NewEmailSearchConfig(
		30, 90, 180, 60*time.Second, true, nil,
	)
	require.NoError(t, err)

	now := time.Now()

	standardSince := config.GetSearchSince("standard")
	extendedSince := config.GetSearchSince("extended")
	maximumSince := config.GetSearchSince("maximum")
	defaultSince := config.GetSearchSince("unknown")

	// Проверяем что даты примерно соответствуют ожидаемым
	assert.InDelta(t, 30*24, now.Sub(standardSince).Hours(), 1.0)
	assert.InDelta(t, 90*24, now.Sub(extendedSince).Hours(), 1.0)
	assert.InDelta(t, 180*24, now.Sub(maximumSince).Hours(), 1.0)
	assert.InDelta(t, 30*24, now.Sub(defaultSince).Hours(), 1.0)
}

func TestEmailSearchConfig_Immutable(t *testing.T) {
	config, err := NewEmailSearchConfig(
		30, 90, 180, 60*time.Second, true,
		[]string{"Re:", "Fwd:"},
	)
	require.NoError(t, err)

	// Проверяем что геттеры возвращают ожидаемые значения
	assert.Equal(t, 30, config.DefaultDaysBack())
	assert.Equal(t, 90, config.ExtendedDaysBack())
	assert.Equal(t, 180, config.MaxDaysBack())
	assert.Equal(t, 60*time.Second, config.FetchTimeout())
	assert.True(t, config.IncludeSeenMessages())

	prefixes := config.SubjectPrefixes()
	assert.Equal(t, []string{"Re:", "Fwd:"}, prefixes)

	// Проверяем иммутабельность - изменения возвращенного слайса не влияют на оригинал
	prefixes[0] = "Modified:"
	assert.Equal(t, []string{"Re:", "Fwd:"}, config.SubjectPrefixes())
}

func TestEmailSearchConfig_DefaultSubjectPrefixes(t *testing.T) {
	config, err := NewEmailSearchConfig(
		30, 90, 180, 60*time.Second, true, nil, // nil prefixes
	)
	require.NoError(t, err)

	prefixes := config.SubjectPrefixes()
	assert.NotEmpty(t, prefixes)
	assert.Contains(t, prefixes, "Re:")
	assert.Contains(t, prefixes, "Fwd:")
	assert.Contains(t, prefixes, "Ответ:")
}

func TestEmailSearchConfig_NormalizePrefixes(t *testing.T) {
	config, err := NewEmailSearchConfig(
		30, 90, 180, 60*time.Second, true,
		[]string{"Re", "Fwd:", "  Reply  ", "", "Re:"}, // mixed formats
	)
	require.NoError(t, err)

	prefixes := config.SubjectPrefixes()
	expected := []string{"Re:", "Fwd:", "Reply:"} // normalized and deduplicated

	assert.Equal(t, expected, prefixes)
}
