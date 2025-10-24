package mirage_test

import (
	"testing"

	"github.com/CruGlobal/mirage-server/internal/mirage"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestType_String(t *testing.T) {
	tests := []struct {
		name     string
		input    mirage.Type
		expected string
	}{
		{
			name:     "REDIRECT",
			input:    mirage.TypeRedirect,
			expected: "REDIRECT",
		},
		{
			name:     "PROXY",
			input:    mirage.TypeProxy,
			expected: "PROXY",
		},
		{
			name:     "unknown type",
			input:    mirage.Type(42),
			expected: "REDIRECT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestType_MarshalDynamoDBAttributeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    mirage.Type
		expected *types.AttributeValueMemberS
	}{
		{
			name:     "valid REDIRECT type",
			input:    mirage.TypeRedirect,
			expected: &types.AttributeValueMemberS{Value: "REDIRECT"},
		},
		{
			name:     "valid PROXY type",
			input:    mirage.TypeProxy,
			expected: &types.AttributeValueMemberS{Value: "PROXY"},
		},
		{
			name:     "invalid type",
			input:    mirage.Type(42),
			expected: &types.AttributeValueMemberS{Value: "REDIRECT"},
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

func TestType_UnmarshalDynamoDBAttributeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    types.AttributeValue
		expected mirage.Type
	}{
		{
			name:     "REDIRECT",
			input:    &types.AttributeValueMemberS{Value: "REDIRECT"},
			expected: mirage.TypeRedirect,
		},
		{
			name:     "PROXY",
			input:    &types.AttributeValueMemberS{Value: "PROXY"},
			expected: mirage.TypeProxy,
		},
		{
			name:     "unknown type",
			input:    &types.AttributeValueMemberS{Value: "FOO"},
			expected: mirage.DefaultType,
		},
		{
			name:     "incorrect type",
			input:    &types.AttributeValueMemberN{Value: "0"},
			expected: mirage.DefaultType,
		},
		{
			name:     "nil",
			input:    nil,
			expected: mirage.DefaultType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p mirage.Type
			err := p.UnmarshalDynamoDBAttributeValue(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, p)
		})
	}
}
