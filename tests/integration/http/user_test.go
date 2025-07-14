package http

import (
	"bytes"
	"fmt"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUserRoutes(t *testing.T) {
	ts, cleanup := setupTestServer()
	t.Cleanup(func() {
		cleanup(t)
	})

	t.Run("Test health endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Test existsUser endpoint", func(t *testing.T) {
		reqBody := map[string]string{"email": "nonexistent@example.com"}
		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		resp, err := http.Post(ts.URL+"/users/exists", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]any
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		require.Equal(t, false, response["exists"])
	})

	t.Run("Test createUser endpoint with avatar", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		userData := map[string]any{
			"name":     "Test User",
			"email":    fmt.Sprintf("test-%v@example.com", uuid.New()),
			"password": "password123",
		}
		data, err := json.Marshal(userData)
		require.NoError(t, err)

		err = writer.WriteField("data", string(data))
		require.NoError(t, err)

		file, err := os.Open("files/avatar.png")
		require.NoError(t, err)
		defer file.Close()

		part, err := writer.CreateFormFile("avatar", filepath.Base(file.Name()))
		require.NoError(t, err)

		_, err = io.Copy(part, file)
		require.NoError(t, err)
		writer.Close()

		resp, err := http.Post(ts.URL+"/users", writer.FormDataContentType(), body)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]any
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.NotEmpty(t, response["id"])
	})

	t.Run("Test getUser endpoint", func(t *testing.T) {
		userID, _ := createTestUser(t, ts)

		resp, err := http.Get(ts.URL + "/users/" + userID.String())
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var user map[string]any
		err = json.NewDecoder(resp.Body).Decode(&user)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), user["id"])
	})

	t.Run("Test listUsers endpoint", func(t *testing.T) {
		createTestUser(t, ts)
		createTestUser(t, ts)

		resp, err := http.Get(ts.URL + "/users?page=1&size=10")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]any
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(response["data"].([]any)), 2)
	})

	t.Run("Test updateUser endpoint", func(t *testing.T) {
		userID, userData := createTestUser(t, ts)
		access, _ := loginUser(t, ts, userData)

		updateBody := &bytes.Buffer{}
		writer := multipart.NewWriter(updateBody)

		updateData := map[string]interface{}{
			"name":  "Updated Name",
			"email": "updated@example.com",
		}
		data, err := json.Marshal(updateData)
		require.NoError(t, err)

		err = writer.WriteField("data", string(data))
		require.NoError(t, err)
		writer.Close()

		req, err := http.NewRequest("PUT", ts.URL+"/users/"+userID.String(), updateBody)
		require.NoError(t, err)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(access)

		cli := &http.Client{}
		resp, err := cli.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Test deleteUser endpoint", func(t *testing.T) {
		userID, userData := createTestUser(t, ts)
		access, _ := loginUser(t, ts, userData)

		req, err := http.NewRequest("DELETE", ts.URL+"/users/"+userID.String(), nil)
		require.NoError(t, err)
		req.AddCookie(access)

		cli := &http.Client{}
		resp, err := cli.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		resp, err = http.Get(ts.URL + "/users/" + userID.String())
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func createTestUser(t *testing.T, ts *httptest.Server) (uuid.UUID, map[string]any) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	userData := map[string]any{
		"name":     "Test User",
		"email":    fmt.Sprintf("test-%s@example.com", uuid.New().String()),
		"password": "password123",
	}
	data, err := json.Marshal(userData)
	require.NoError(t, err)

	err = writer.WriteField("data", string(data))
	writer.Close()
	require.NoError(t, err)

	resp, err := http.Post(ts.URL+"/users", writer.FormDataContentType(), body)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]any
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	userID, err := uuid.Parse(response["id"].(string))
	require.NoError(t, err)

	return userID, userData
}

func loginUser(t *testing.T, ts *httptest.Server, userData map[string]any) (*http.Cookie, *http.Cookie) {
	loginData := map[string]any{
		"email":    userData["email"],
		"password": userData["password"],
		"token":    "test-token",
	}
	data, err := json.Marshal(loginData)
	require.NoError(t, err)

	resp, err := http.Post(ts.URL+"/auth/jwt", "application/json", bytes.NewReader(data))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	accessCookie := &http.Cookie{}
	refreshCookie := &http.Cookie{}
	for _, cookie := range resp.Cookies() {
		switch cookie.Name {
		case config.AccessCookieName:
			accessCookie = cookie
		case config.RefreshCookieName:
			refreshCookie = cookie
		}
	}

	require.NotEmpty(t, accessCookie.Value, "Access token not found")
	require.NotEmpty(t, refreshCookie.Value, "Refresh token cookie not set")
	return accessCookie, refreshCookie
}
