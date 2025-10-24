package mirage_test

import (
	"testing"

	"github.com/CruGlobal/mirage-server/internal/mirage"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		input    mirage.Status
		expected string
	}{
		{
			name:     "temporary",
			input:    mirage.StatusTemporary,
			expected: "TEMPORARY",
		},
		{
			name:     "permanent",
			input:    mirage.StatusPermanent,
			expected: "PERMANENT",
		},
		{
			name:     "unknown status",
			input:    mirage.Status(42),
			expected: "TEMPORARY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatus_StatusCode(t *testing.T) {
	tests := []struct {
		name     string
		input    mirage.Status
		expected int
	}{
		{
			name:     "temporary",
			input:    mirage.StatusTemporary,
			expected: 302,
		},
		{
			name:     "permanent",
			input:    mirage.StatusPermanent,
			expected: 301,
		},
		{
			name:     "unknown status",
			input:    mirage.Status(42),
			expected: 302,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.StatusCode()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatus_MarshalDynamoDBAttributeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    mirage.Status
		expected *types.AttributeValueMemberS
	}{
		{
			name:     "temporary",
			input:    mirage.StatusTemporary,
			expected: &types.AttributeValueMemberS{Value: "TEMPORARY"},
		},
		{
			name:     "permanent",
			input:    mirage.StatusPermanent,
			expected: &types.AttributeValueMemberS{Value: "PERMANENT"},
		},
		{
			name:     "invalid status",
			input:    mirage.Status(42),
			expected: &types.AttributeValueMemberS{Value: "TEMPORARY"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.input.MarshalDynamoDBAttributeValue()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatus_UnmarshalDynamoDBAttributeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    types.AttributeValue
		expected mirage.Status
	}{
		{
			name:     "temporary",
			input:    &types.AttributeValueMemberS{Value: "TEMPORARY"},
			expected: mirage.StatusTemporary,
		},
		{
			name:     "permanent",
			input:    &types.AttributeValueMemberS{Value: "PERMANENT"},
			expected: mirage.StatusPermanent,
		},
		{
			name:     "unknown status",
			input:    &types.AttributeValueMemberS{Value: "FOO"},
			expected: mirage.DefaultStatus,
		},
		{
			name:     "incorrect status",
			input:    &types.AttributeValueMemberN{Value: "0"},
			expected: mirage.DefaultStatus,
		},
		{
			name:     "nil",
			input:    nil,
			expected: mirage.DefaultStatus,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s mirage.Status
			err := s.UnmarshalDynamoDBAttributeValue(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, s)
		})
	}
}
