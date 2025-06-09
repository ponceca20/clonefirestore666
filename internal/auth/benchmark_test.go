package auth_test

import (
	"testing"

	"firestore-clone/internal/auth/domain/model"

	"golang.org/x/crypto/bcrypt"
)

func BenchmarkPasswordHashing(b *testing.B) {
	password := []byte("SuperSecurePassword123!")
	for i := 0; i < b.N; i++ {
		_, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
		if err != nil {
			b.Fatalf("bcrypt error: %v", err)
		}
	}
}

func BenchmarkPasswordCompare(b *testing.B) {
	password := []byte("SuperSecurePassword123!")
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		b.Fatalf("bcrypt error: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := bcrypt.CompareHashAndPassword(hash, password); err != nil {
			b.Fatalf("bcrypt compare error: %v", err)
		}
	}
}

func BenchmarkUserStructCopy(b *testing.B) {
	user := &model.User{
		ID:    "user-123",
		Email: "test@example.com",
	}
	for i := 0; i < b.N; i++ {
		_ = *user // struct copy
	}
}
