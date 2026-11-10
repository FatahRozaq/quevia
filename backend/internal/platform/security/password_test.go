package security

import (
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	raw := "SecureP@ssw0rd123!"

	hash, err := HashPassword(raw)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("empty password hash")
	}

	match, err := VerifyPassword(raw, hash)
	if err != nil {
		t.Fatalf("failed to verify password: %v", err)
	}
	if !match {
		t.Fatal("expected password to match hash")
	}

	wrongMatch, err := VerifyPassword("WrongPassword!", hash)
	if err != nil {
		t.Fatalf("failed to check wrong password: %v", err)
	}
	if wrongMatch {
		t.Fatal("expected wrong password not to match")
	}
}
