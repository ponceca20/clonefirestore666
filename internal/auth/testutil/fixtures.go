package testutil

import (
	"time"

	"firestore-clone/internal/auth/domain/model"

	"golang.org/x/crypto/bcrypt"
)

// UserFixture provides test data for User model
type UserFixture struct{}

// NewUserFixture creates a new UserFixture instance
func NewUserFixture() *UserFixture {
	return &UserFixture{}
}

// ValidUser returns a valid user for testing
func (f *UserFixture) ValidUser() *model.User {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	return &model.User{
		ID:           "test-user-id-123",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// UserWithEmail returns a user with specific email
func (f *UserFixture) UserWithEmail(email string) *model.User {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	return &model.User{
		ID:           "user-" + email,
		Email:        email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// UserWithPassword returns a user with specific password
func (f *UserFixture) UserWithPassword(email, password string) *model.User {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return &model.User{
		ID:           "user-" + email,
		Email:        email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// SessionFixture provides test data for Session model
type SessionFixture struct{}

// NewSessionFixture creates a new SessionFixture instance
func NewSessionFixture() *SessionFixture {
	return &SessionFixture{}
}

// ValidSession returns a valid session for testing
func (f *SessionFixture) ValidSession() *model.Session {
	return &model.Session{
		ID:        "test-session-id-123",
		UserID:    "test-user-id-123",
		Token:     "test-session-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
}

// SessionForUser returns a session for specific user
func (f *SessionFixture) SessionForUser(userID string) *model.Session {
	return &model.Session{
		ID:        "session-for-" + userID,
		UserID:    userID,
		Token:     "session-token-" + userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
}

// ExpiredSession returns an expired session
func (f *SessionFixture) ExpiredSession() *model.Session {
	return &model.Session{
		ID:        "expired-session-id",
		UserID:    "test-user-id",
		Token:     "expired-session-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
}

// TestData provides all fixtures
type TestData struct {
	Users    *UserFixture
	Sessions *SessionFixture
}

// NewTestData creates a new TestData instance with all fixtures
func NewTestData() *TestData {
	return &TestData{
		Users:    NewUserFixture(),
		Sessions: NewSessionFixture(),
	}
}

// Common test emails for validation testing
var (
	ValidEmails = []string{
		"test@example.com",
		"user.name@domain.co.uk",
		"user+tag@example.org",
		"firstname.lastname@company.com",
	}

	InvalidEmails = []string{
		"",
		"invalid-email",
		"@example.com",
		"test@",
		"test.example.com",
		"test@.com",
		"test@com.",
		"test space@example.com",
	}

	ValidPasswords = []string{
		"password123",
		"StrongP@ssw0rd",
		"MySecurePassword2024!",
		"12345678", // Minimum length
	}

	InvalidPasswords = []string{
		"",
		"123",     // Too short
		"1234567", // Still too short
		"short",   // Too short
	}
)

// CleanupFunc represents a cleanup function for tests
type CleanupFunc func()

// NoOpCleanup is a no-operation cleanup function
func NoOpCleanup() {}
