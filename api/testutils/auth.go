package testutils

import (
	"context"
	"github.com/google/uuid"
	"net/http"
	"time"
	"treffly/api/common"
	"treffly/token"
)

func AddUserIDToContext(request *http.Request, userID int32) {
	payload := &token.Payload{
		ID:        uuid.New(),
		UserID:    userID,
		IsAdmin:   false,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(time.Hour),
	}
	ctx := request.Context()
	ctx = context.WithValue(ctx, common.AuthorizationPayloadKey, payload)
	*request = *request.WithContext(ctx)
}

func AddSoftAuthUserIDToContext(request *http.Request, userID int32) {
	ctx := request.Context()
	ctx = context.WithValue(ctx, "user_id", userID)
	*request = *request.WithContext(ctx)
} 