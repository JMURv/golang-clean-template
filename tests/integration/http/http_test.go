package http

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/cache/redis"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	hdl "github.com/JMURv/golang-clean-template/internal/hdl/http"
	"github.com/JMURv/golang-clean-template/internal/repo/db"
	"github.com/JMURv/golang-clean-template/internal/repo/s3"
	"github.com/JMURv/golang-clean-template/internal/smtp"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const getTables = `
SELECT tablename 
FROM pg_tables 
WHERE schemaname = 'public';
`

var rootDir = filepath.Join("..", "..", "..")

func GetRedis() testcontainers.Container {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.PortBindings = nat.PortMap{
				"6379/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "6379",
					},
				},
			}
		},
	}

	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}

	zap.L().Info("Redis container is ready")
	return redisC
}

func GetPostgres() testcontainers.Container {
	ctx := context.Background()
	pgPort := os.Getenv("POSTGRES_PORT")
	pgPortC := fmt.Sprintf("%s/tcp", pgPort)

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17.4-alpine",
		WaitingFor:   wait.ForHealthCheck(),
		ExposedPorts: []string{pgPortC},
		ConfigModifier: func(conf *container.Config) {
			conf.Healthcheck = &container.HealthConfig{
				Test:        []string{"CMD-SHELL", fmt.Sprintf("pg_isready -U %s -d %s", os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_DB"))},
				Interval:    5 * time.Second,
				Timeout:     2 * time.Second,
				Retries:     5,
				StartPeriod: 2 * time.Second,
			}
		},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.PortBindings = nat.PortMap{
				nat.Port(pgPortC): []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: pgPort,
					},
				},
			}
		},
		Env: map[string]string{
			"POSTGRES_DB":       os.Getenv("POSTGRES_DB"),
			"POSTGRES_USER":     os.Getenv("POSTGRES_USER"),
			"POSTGRES_PASSWORD": os.Getenv("POSTGRES_PASSWORD"),
			"POSTGRES_HOST":     os.Getenv("POSTGRES_HOST"),
			"POSTGRES_PORT":     os.Getenv("POSTGRES_PORT"),
		},
	}

	pgC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}

	return pgC
}

func GetMinio() testcontainers.Container {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image: "minio/minio:RELEASE.2025-06-13T11-33-47Z",
		Cmd:   []string{"server", "/data", "--console-address", ":9001"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("9000/tcp"),
			wait.ForHTTP("/minio/health/live").WithPort("9000/tcp"),
		),
		ExposedPorts: []string{"9000/tcp", "9001/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":            os.Getenv("MINIO_ROOT_USER"),
			"MINIO_ROOT_PASSWORD":        os.Getenv("MINIO_ROOT_PASSWORD"),
			"MINIO_PROMETHEUS_AUTH_TYPE": "public",
		},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.PortBindings = nat.PortMap{
				"9000/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "9000",
					},
				},
				"9001/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "9001",
					},
				},
			}
		},
	}

	minioC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}

	return minioC
}

func setupTestServer() (*httptest.Server, func(t *testing.T)) {
	zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))

	conf := config.MustLoad(
		filepath.ToSlash(
			filepath.Join(rootDir, "config", ".env.integration"),
		),
	)

	_ = os.Setenv("MIGRATIONS_PATH", filepath.ToSlash(
		filepath.Join(rootDir, "internal", "repo", "db", "migration"),
	))

	redisC := GetRedis()
	pgC := GetPostgres()
	minioC := GetMinio()

	au := auth.New(conf)
	cache := redis.New(conf)
	repo := db.New(conf)
	svc := ctrl.New(au, repo, cache, s3.New(conf), smtp.New(conf))
	h := hdl.New(au, svc)

	ts := httptest.NewServer(h.Router)

	cleanupFunc := func(t *testing.T) {
		ts.Close()

		conn, err := sql.Open(
			"pgx", fmt.Sprintf(
				"postgres://%s:%s@%s:%d/%s?sslmode=disable",
				conf.DB.User,
				conf.DB.Password,
				conf.DB.Host,
				conf.DB.Port,
				conf.DB.Database,
			),
		)
		if err != nil {
			zap.L().Fatal("Failed to connect to the database", zap.Error(err))
		}

		if err = conn.Ping(); err != nil {
			zap.L().Fatal("Failed to ping the database", zap.Error(err))
		}

		rows, err := conn.Query(getTables)
		if err != nil {
			zap.L().Fatal("Failed to fetch table names", zap.Error(err))
		}
		defer func(rows *sql.Rows) {
			if err := rows.Close(); err != nil {
				zap.L().Debug("Error while closing rows", zap.Error(err))
			}
		}(rows)

		var tables []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				zap.L().Fatal("Failed to scan table name", zap.Error(err))
			}
			tables = append(tables, name)
		}

		if len(tables) == 0 {
			return
		}

		_, err = conn.Exec(fmt.Sprintf("TRUNCATE TABLE %v RESTART IDENTITY CASCADE;", strings.Join(tables, ", ")))
		if err != nil {
			zap.L().Fatal("Failed to truncate tables", zap.Error(err))
		}

		testcontainers.CleanupContainer(t, redisC)
		testcontainers.CleanupContainer(t, pgC)
		testcontainers.CleanupContainer(t, minioC)
	}

	return ts, cleanupFunc
}

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
