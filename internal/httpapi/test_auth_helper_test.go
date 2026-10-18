package httpapi_test

import (
	"testing"

	"hotelsystem/internal/auth"
)

func bearerTokenForTest(t *testing.T, userID string) string {
	t.Helper()
	token, err := auth.IssueAccessToken(userID)
	if err != nil {
		t.Fatalf("issue test token: %v", err)
	}
	return "Bearer " + token
}
