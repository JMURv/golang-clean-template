package http

import (
	"bytes"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserRoutes(t *testing.T) {
	ts, cleanup := setupTestServer()
	defer cleanup(t)

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

}

//t.Run("Test createUser endpoint", func(t *testing.T) {
//	// Prepare multipart form data
//	body := &bytes.Buffer{}
//	writer := multipart.NewWriter(body)
//
//	// Add JSON data
//	userData := map[string]interface{}{
//		"name":      "Test User",
//		"email":     "test@example.com",
//		"password":  "password123",
//		"is_active": true,
//	}
//	data, _ := json.Marshal(userData)
//	_ = writer.WriteField("data", string(data))
//
//	// Add avatar file
//	file, err := os.Open("testdata/avatar.png")
//	require.NoError(t, err)
//	defer file.Close()
//
//	part, err := writer.CreateFormFile("avatar", filepath.Base(file.Name()))
//	require.NoError(t, err)
//
//	_, err = io.Copy(part, file)
//	require.NoError(t, err)
//	writer.Close()
//
//	// Make request
//	req := httptest.NewRequest("POST", "/users", body)
//	req.Header.Set("Content-Type", writer.FormDataContentType())
//
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusCreated, w.Code)
//
//	var response map[string]interface{}
//	err = json.Unmarshal(w.Body.Bytes(), &response)
//	require.NoError(t, err)
//	assert.NotEmpty(t, response["id"])
//})
//
//t.Run("Test getUser endpoint", func(t *testing.T) {
//	// First create a user
//	userID := createTestUser(t, router)
//
//	// Test getting the user
//	req := httptest.NewRequest("GET", "/users/"+userID.String(), nil)
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusOK, w.Code)
//
//	var user map[string]interface{}
//	err := json.Unmarshal(w.Body.Bytes(), &user)
//	require.NoError(t, err)
//	assert.Equal(t, userID.String(), user["id"])
//})
//
//t.Run("Test listUsers endpoint", func(t *testing.T) {
//	// Create some test users
//	createTestUser(t, router)
//	createTestUser(t, router)
//
//	// Test listing users
//	req := httptest.NewRequest("GET", "/users?page=1&size=10", nil)
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusOK, w.Code)
//
//	var response map[string]interface{}
//	err := json.Unmarshal(w.Body.Bytes(), &response)
//	require.NoError(t, err)
//	assert.GreaterOrEqual(t, len(response["data"].([]interface{})), 2)
//})
//
//t.Run("Test updateUser endpoint", func(t *testing.T) {
//	// First create a user
//	userID := createTestUser(t, router)
//
//	// Prepare update data
//	updateBody := &bytes.Buffer{}
//	writer := multipart.NewWriter(updateBody)
//
//	updateData := map[string]interface{}{
//		"name":  "Updated Name",
//		"email": "updated@example.com",
//	}
//	data, _ := json.Marshal(updateData)
//	_ = writer.WriteField("data", string(data))
//	writer.Close()
//
//	// Test updating the user
//	req := httptest.NewRequest("PUT", "/users/"+userID.String(), updateBody)
//	req.Header.Set("Content-Type", writer.FormDataContentType())
//
//	// TODO: Add auth token to request
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusOK, w.Code)
//})
//
//t.Run("Test deleteUser endpoint", func(t *testing.T) {
//	// First create a user
//	userID := createTestUser(t, router)
//
//	// Test deleting the user
//	req := httptest.NewRequest("DELETE", "/users/"+userID.String(), nil)
//	// TODO: Add auth token to request
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusNoContent, w.Code)
//
//	// Verify user is deleted
//	req = httptest.NewRequest("GET", "/users/"+userID.String(), nil)
//	w = httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusNotFound, w.Code)
//})
//}

func createTestUser(t *testing.T, router *chi.Mux) uuid.UUID {
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

	req := httptest.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var response map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	userID, err := uuid.Parse(response["id"].(string))
	require.NoError(t, err)

	return userID
}
