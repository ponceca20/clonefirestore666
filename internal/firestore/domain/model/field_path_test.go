package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFieldPath_Valid(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedRaw    string
		expectedDepth  int
		expectedRoot   string
		expectedNested string
		isNested       bool
	}{
		{
			name:           "Simple field",
			input:          "status",
			expectedRaw:    "status",
			expectedDepth:  1,
			expectedRoot:   "status",
			expectedNested: "",
			isNested:       false,
		},
		{
			name:           "Nested field level 1",
			input:          "customer.ruc",
			expectedRaw:    "customer.ruc",
			expectedDepth:  2,
			expectedRoot:   "customer",
			expectedNested: "ruc",
			isNested:       true,
		},
		{
			name:           "Nested field level 2",
			input:          "customer.address.city",
			expectedRaw:    "customer.address.city",
			expectedDepth:  3,
			expectedRoot:   "customer",
			expectedNested: "address.city",
			isNested:       true,
		},
		{
			name:           "Deep nested field",
			input:          "user.profile.preferences.theme.colors.primary",
			expectedRaw:    "user.profile.preferences.theme.colors.primary",
			expectedDepth:  6,
			expectedRoot:   "user",
			expectedNested: "profile.preferences.theme.colors.primary",
			isNested:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fp, err := NewFieldPath(tc.input)
			require.NoError(t, err)
			require.NotNil(t, fp)

			assert.Equal(t, tc.expectedRaw, fp.Raw())
			assert.Equal(t, tc.expectedDepth, fp.Depth())
			assert.Equal(t, tc.expectedRoot, fp.Root())
			assert.Equal(t, tc.expectedNested, fp.NestedPath())
			assert.Equal(t, tc.isNested, fp.IsNested())
		})
	}
}

func TestNewFieldPath_Invalid(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedError error
	}{
		{
			name:          "Empty path",
			input:         "",
			expectedError: ErrEmptyFieldPath,
		},
		{
			name:          "Path starting with dot",
			input:         ".customer",
			expectedError: ErrInvalidFieldPathFormat,
		},
		{
			name:          "Path ending with dot",
			input:         "customer.",
			expectedError: ErrInvalidFieldPathFormat,
		},
		{
			name:          "Double dots",
			input:         "customer..ruc",
			expectedError: ErrInvalidFieldPathFormat,
		},
		{
			name:          "Invalid character slash",
			input:         "customer/ruc",
			expectedError: ErrInvalidFieldName,
		},
		{
			name:          "Invalid character bracket",
			input:         "customer[0]",
			expectedError: ErrInvalidFieldName,
		},
		{
			name:          "Reserved prefix",
			input:         "__metadata",
			expectedError: ErrInvalidFieldName,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fp, err := NewFieldPath(tc.input)
			assert.Error(t, err)
			assert.Nil(t, fp)
			assert.ErrorIs(t, err, tc.expectedError)
		})
	}
}

func TestFieldPath_Parent(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedParent string
		hasParent      bool
	}{
		{
			name:           "Simple field has no parent",
			input:          "status",
			expectedParent: "",
			hasParent:      false,
		},
		{
			name:           "Nested field level 1",
			input:          "customer.ruc",
			expectedParent: "customer",
			hasParent:      true,
		},
		{
			name:           "Nested field level 2",
			input:          "customer.address.city",
			expectedParent: "customer.address",
			hasParent:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fp := MustNewFieldPath(tc.input)
			parent := fp.Parent()

			if tc.hasParent {
				require.NotNil(t, parent)
				assert.Equal(t, tc.expectedParent, parent.Raw())
			} else {
				assert.Nil(t, parent)
			}
		})
	}
}

func TestFieldPath_Child(t *testing.T) {
	fp := MustNewFieldPath("customer")

	child, err := fp.Child("address")
	require.NoError(t, err)
	assert.Equal(t, "customer.address", child.Raw())

	grandchild, err := child.Child("city")
	require.NoError(t, err)
	assert.Equal(t, "customer.address.city", grandchild.Raw())

	// Test invalid child
	_, err = fp.Child("invalid/name")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidFieldName)
}

func TestFieldPath_Equal(t *testing.T) {
	fp1 := MustNewFieldPath("customer.ruc")
	fp2 := MustNewFieldPath("customer.ruc")
	fp3 := MustNewFieldPath("customer.name")

	assert.True(t, fp1.Equal(fp2))
	assert.False(t, fp1.Equal(fp3))
	assert.False(t, fp1.Equal(nil))
}

func TestFieldPath_Segments(t *testing.T) {
	fp := MustNewFieldPath("customer.address.city")
	segments := fp.Segments()

	expected := []string{"customer", "address", "city"}
	assert.Equal(t, expected, segments)

	// Test immutability - modifying returned slice shouldn't affect original
	segments[0] = "modified"
	assert.Equal(t, "customer", fp.Root())
}

func TestFieldPath_Validate(t *testing.T) {
	// Valid paths
	validPaths := []string{
		"status",
		"customer.ruc",
		"user.profile.settings",
	}

	for _, path := range validPaths {
		fp := MustNewFieldPath(path)
		assert.NoError(t, fp.Validate())
	}

	// Test max depth validation by creating a very deep path
	deepSegments := make([]string, MaxFieldPathDepth+1)
	for i := range deepSegments {
		deepSegments[i] = "field"
	}

	// This should fail during creation due to depth
	fp := &FieldPath{
		segments: deepSegments,
		raw:      "too.deep.path",
	}

	err := fp.Validate()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrFieldPathTooDeep)
}

func TestMustNewFieldPath_Panic(t *testing.T) {
	assert.Panics(t, func() {
		MustNewFieldPath("invalid..path")
	})
}

// Benchmark tests for performance
func BenchmarkNewFieldPath_Simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewFieldPath("status")
	}
}

func BenchmarkNewFieldPath_Nested(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewFieldPath("customer.address.city")
	}
}

func BenchmarkFieldPath_NestedPath(b *testing.B) {
	fp := MustNewFieldPath("customer.address.city.district.zone")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = fp.NestedPath()
	}
}
