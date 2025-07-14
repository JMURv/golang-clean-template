package http

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestDeviceRoutes(t *testing.T) {
	ts, cleanup := setupTestServer()
	t.Cleanup(func() {
		cleanup(t)
	})

	_, userData := createTestUser(t, ts)
	access, _ := loginUser(t, ts, userData)

	// List devices of the newly created user
	req, err := http.NewRequest("GET", ts.URL+"/device", nil)
	require.NoError(t, err)

	req.AddCookie(access)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response []map[string]any
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	require.Equal(t, 1, len(response))
	dID := response[0]["id"].(string)

	// Get first device from list
	t.Run("Get created device", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/device/"+dID, nil)
		require.NoError(t, err)
		req.AddCookie(access)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var deviceResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&deviceResponse)
		require.NoError(t, err)

		assert.Equal(t, dID, deviceResponse["id"])
	})

	// Update device
	t.Run("Update device", func(t *testing.T) {
		updateBody, err := json.Marshal(map[string]any{
			"name": "Updated Device",
		})
		require.NoError(t, err)

		req, err := http.NewRequest("PUT", ts.URL+"/device/"+dID, bytes.NewReader(updateBody))
		require.NoError(t, err)
		req.AddCookie(access)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Check update was success
		req, err = http.NewRequest("GET", ts.URL+"/device/"+dID, nil)
		require.NoError(t, err)
		req.AddCookie(access)

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var updatedDevice map[string]any
		err = json.NewDecoder(resp.Body).Decode(&updatedDevice)
		require.NoError(t, err)

		assert.Equal(t, "Updated Device", updatedDevice["name"])
		assert.Equal(t, dID, updatedDevice["id"])
	})

	// 5. Получаем список устройств (LIST)
	t.Run("List devices", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/device", nil)
		require.NoError(t, err)
		req.AddCookie(access)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResponse []map[string]any
		err = json.NewDecoder(resp.Body).Decode(&listResponse)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(listResponse))
	})

	// Delete device
	t.Run("Delete device", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.URL+"/device/"+dID, nil)
		require.NoError(t, err)
		req.AddCookie(access)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		req, err = http.NewRequest("GET", ts.URL+"/device/"+dID, nil)
		require.NoError(t, err)
		req.AddCookie(access)

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
