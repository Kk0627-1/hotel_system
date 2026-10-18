package main

import (
	"testing"

	"hotelsystem/UserServices"
)

func bearerTokenForTest(t *testing.T, userID string) string {
	t.Helper()
	token, err := UserServices.IssueAccessToken(userID)
	if err != nil {
		t.Fatalf("issue test token: %v", err)
	}
	return "Bearer " + token
}
