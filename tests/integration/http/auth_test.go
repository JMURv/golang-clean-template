package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthRoutes(t *testing.T) {
	ts := NewTestEnv(t)

	_, userData := createTestUser(t, ts)
	_, refresh := loginUser(t, ts, userData)

	// Try to log out without access
	resp, err := http.Post(ts.Server.URL+"/auth/logout", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Send refresh request
	time.Sleep(time.Second * 1) // Need other sec for creating unique uuid
	req, err := http.NewRequest("POST", ts.Server.URL+"/auth/jwt/refresh", nil)
	require.NoError(t, err)

	req.AddCookie(refresh)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Get new cookies from response
	newAccess := &http.Cookie{}
	for _, cookie := range resp.Cookies() {
		switch cookie.Name {
		case config.AccessCookieName:
			newAccess = cookie
		}
	}

	// Logout
	req, err = http.NewRequest("POST", ts.Server.URL+"/auth/logout", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	req.AddCookie(newAccess)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
