package event

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"treffly/api/common"
	eventdto "treffly/api/dto/event"
	eventservice "treffly/api/service/event"
	imageservice "treffly/api/service/image"
	db "treffly/db/sqlc"
	"treffly/image"
	"treffly/token"
	"treffly/util"
)

func TestCreateEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	eventService := eventservice.New(testStore, testConfig)
	iStore, err := image.NewLocalStorage(testConfig.ImageBasePath)
	require.NoError(t, err)
	imageService := imageservice.New(iStore, testConfig, testStore)
	converter := eventdto.NewEventConverter(testConfig.Environment, testConfig.Domain)
	handler := NewEventCRUDHandler(eventService, imageService, converter)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		err := v.RegisterValidation("event_name", func(fl validator.FieldLevel) bool { return true })
		require.NoError(t, err)
		err = v.RegisterValidation("latitude", func(fl validator.FieldLevel) bool { return true })
		require.NoError(t, err)
		err = v.RegisterValidation("longitude", func(fl validator.FieldLevel) bool { return true })
		require.NoError(t, err)
		err = v.RegisterValidation("valid_date", func(fl validator.FieldLevel) bool { return true })
		require.NoError(t, err)
		err = v.RegisterValidation("positive", func(fl validator.FieldLevel) bool { return true })
		require.NoError(t, err)
	}

	user, err := testStore.CreateUser(context.Background(), db.CreateUserParams{
		Username:     util.RandomUsername(),
		Email:        util.RandomEmail(),
		PasswordHash: "password",
	})
	require.NoError(t, err)
	t.Logf("Created test user: %+v", user)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	eventDate := time.Now().Add(24 * time.Hour)
	formFields := map[string]string{
		"name":        "Test Event",
		"description": "This is a very detailed description of the test event that meets the minimum length requirement of 50 characters",
		"capacity":    "100",
		"latitude":    "51.6683",
		"longitude":   "39.1919",
		"address":     "Test Address",
		"date":        eventDate.Format(time.RFC3339),
		"is_private":  "false",
	}

	for key, value := range formFields {
		err = writer.WriteField(key, value)
		require.NoError(t, err)
		t.Logf("Added form field %s: %s", key, value)
	}

	// Add tags as separate fields
	err = writer.WriteField("tags", "1")
	require.NoError(t, err)
	err = writer.WriteField("tags", "2")
	require.NoError(t, err)
	t.Logf("Added tags: [1, 2]")

	err = writer.Close()
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request, err = http.NewRequest(http.MethodPost, "/events", body)
	require.NoError(t, err)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	t.Logf("Content-Type: %s", writer.FormDataContentType())

	authPayload := &token.Payload{
		ID:        uuid.New(),
		UserID:    user.ID,
		IsAdmin:   false,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(time.Hour),
	}
	c.Set(common.AuthorizationPayloadKey, authPayload)
	t.Logf("Set auth payload: %+v", authPayload)

	handler.Create(c)

	t.Logf("Response status code: %d", recorder.Code)
	t.Logf("Response headers: %+v", recorder.Header())
	
	responseBody, err := io.ReadAll(recorder.Body)
	require.NoError(t, err)
	t.Logf("Response body: %s", string(responseBody))

	if len(c.Errors) > 0 {
		t.Logf("Gin context errors: %+v", c.Errors)
		t.FailNow()
	}

	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d", http.StatusOK, recorder.Code)
	}

	var resp eventdto.EventResponse
	err = json.Unmarshal(responseBody, &resp)
	if err != nil {
		t.Logf("Failed to unmarshal response: %v", err)
		t.Logf("Response body: %s", string(responseBody))
		t.FailNow()
	}
	
	require.NotZero(t, resp.ID)
}