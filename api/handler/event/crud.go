package event

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"strings"
	"treffly/api/common"
	eventdto "treffly/api/dto/event"
	"treffly/api/models"
	"treffly/apperror"
)

//go:generate mockgen -destination=./mock/service.go -package=mock . crudService,ImageService

type crudService interface {
	Create(ctx context.Context, params models.CreateParams) (models.Event, error)
	List(ctx context.Context, params models.ListParams) ([]models.Event, error)
	GetEvent(ctx context.Context, eventID int32, userID int32, token string) (models.Event, error)
	Update(ctx context.Context, params models.UpdateParams) (models.Event, error)
	Delete(ctx context.Context, params models.DeleteParams) error
}

type CRUDHandler struct {
	BaseHandler
	crudService crudService
	imageService ImageService
	converter *eventdto.EventConverter
}

func NewEventCRUDHandler(crudService crudService, imageService ImageService, converter *eventdto.EventConverter) *CRUDHandler {
	return &CRUDHandler{crudService: crudService, imageService: imageService, converter: converter}
}

func (h *CRUDHandler) Create(ctx *gin.Context) {
	userID := common.GetUserIDFromContextPayload(ctx)

	var req eventdto.CreateEventRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(apperror.BadRequest.WithCause(err))
		return
	}

	var (
		imageID = uuid.Nil
		path    string
	)

	file, header, err := ctx.Request.FormFile("image")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		ctx.Error(apperror.BadRequest.WithCause(err))
		return
	}

	if err == nil && file != nil {
		imageID = uuid.New()
		path, err = h.imageService.Upload(file, header, "event", imageID.String())
		if err != nil {
			ctx.Error(apperror.BadRequest.WithCause(err))
			return
		}
	}
	params := models.CreateParams{
		Name:        req.Name,
		Description: req.Description,
		Capacity:    req.Capacity,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Address:     req.Address,
		Date:        req.Date,
		IsPrivate:   req.IsPrivate,
		Tags:        req.Tags,
		OwnerID:     userID,
		ImageID:     imageID,
		Path:        path,
	}

	createdEvent, err := h.crudService.Create(ctx, params)
	if err != nil {
		if path != "" {
			_ = h.imageService.Delete(path)
		}
		ctx.Error(apperror.WrapDBError(err))
		return
	}

	response := h.converter.ToEventResponse(createdEvent)

	ctx.JSON(http.StatusOK, response)
}

func (h *CRUDHandler) List(ctx *gin.Context) {
	lat, lon, err := common.GetUserLocation(ctx)
	if err != nil {
		ctx.Error(apperror.BadRequest.WithCause(err))
		return
	}

	tagIDs, err := parseTagIDs(ctx.Query("tags"))
	if err != nil {
		ctx.Error(apperror.BadRequest.WithCause(err))
		return
	}

	params := models.ListParams{
		Lat:       lat,
		Lon:       lon,
		Search:    ctx.Query("keywords"),
		TagIDs:    tagIDs,
		DateRange: ctx.Query("dateWithin"),
	}

	events, err := h.crudService.List(ctx, params)
	if err != nil {
		ctx.Error(apperror.WrapDBError(err))
		return
	}

	resp := h.converter.ToEventsResponse(events)

	ctx.JSON(http.StatusOK, resp)
}

func (h *CRUDHandler) GetByID(ctx *gin.Context) {
	eventID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Error(apperror.BadRequest.WithCause(err))
		return
	}

	token := ctx.Query("invite")

	userID := common.GetUserIDFromSoftAuth(ctx)

	Event, err := h.crudService.GetEvent(ctx, int32(eventID), userID, token)
	if err != nil {
		ctx.Error(apperror.WrapDBError(err))
		return
	}

	resp := h.converter.ToEventResponse(Event)

	ctx.JSON(http.StatusOK, resp)
}

func (h *CRUDHandler) Update(ctx *gin.Context) {
	userID := common.GetUserIDFromContextPayload(ctx)
	eventID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Error(apperror.BadRequest.WithCause(err))
		return
	}

	var req eventdto.UpdateEventRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(apperror.BadRequest.WithCause(err))
		return
	}

	var (
		oldImageID uuid.UUID
		oldPath    string
	)

	oldImageID, oldPath, err = h.imageService.GetDBImageByEventID(ctx, int32(eventID))
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			ctx.Error(apperror.WrapDBError(err))
			return
		}
	}

	var (
		imageID = uuid.Nil
		path    string
	)

	file, header, err := ctx.Request.FormFile("image")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		ctx.Error(apperror.BadRequest.WithCause(err))
		return
	}

	if err == nil && file != nil && !req.DeleteImage {
		imageID = uuid.New()
		path, err = h.imageService.Upload(file, header, "event", imageID.String())
		if err != nil {
			ctx.Error(apperror.BadRequest.WithCause(err))
			return
		}
	}

	params := models.UpdateParams{
		EventID:     int32(eventID),
		Name:        req.Name,
		Description: req.Description,
		Capacity:    req.Capacity,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Address:     req.Address,
		Date:        req.Date,
		IsPrivate:   req.IsPrivate,
		Tags:        req.Tags,
		UserID:      userID,
		Path:        path,
		NewImageID:  imageID,
		DeleteImage: req.DeleteImage,
		OldImageID:  oldImageID,
	}

	updatedEvent, err := h.crudService.Update(ctx, params)
	if err != nil {
		ctx.Error(apperror.WrapDBError(err))
		return
	}

	if req.DeleteImage && oldPath != "" {
		err = h.imageService.Delete(oldPath) //TODO: make deletes transactional
		if err != nil {
			ctx.Error(apperror.WrapDBError(err))
			return
		}
	}

	if oldPath != "" && file != nil {
		_ = h.imageService.Delete(oldPath)
	}

	resp := h.converter.ToEventResponse(updatedEvent)

	ctx.JSON(http.StatusOK, resp)
}

func (h *CRUDHandler) Delete(ctx *gin.Context) {
	userID := common.GetUserIDFromContextPayload(ctx)
	eventID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Error(apperror.BadRequest.WithCause(err))
		return
	}

	_, path, err := h.imageService.GetDBImageByEventID(ctx, int32(eventID)) //TODO: make deletes transactional
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		ctx.Error(apperror.WrapDBError(err))
		return
	}

	if path != "" {
		err = h.imageService.Delete(path)
		if err != nil {
			ctx.Error(apperror.WrapDBError(err))
			return
		}
	}

	err = h.crudService.Delete(ctx, models.DeleteParams{
		EventID: int32(eventID),
		UserID:  userID,
	})
	if err != nil {
		ctx.Error(apperror.WrapDBError(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}

func parseTagIDs(tagsStr string) ([]int32, error) {
	if tagsStr == "" {
		return []int32{}, nil
	}

	strIDs := strings.Split(tagsStr, ",")
	result := make([]int32, 0, len(strIDs))

	for _, strID := range strIDs {
		cleaned := strings.TrimSpace(strID)
		if cleaned == "" {
			continue
		}

		id, err := strconv.ParseInt(cleaned, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid tag ID: %s", strID)
		}

		result = append(result, int32(id))
	}

	return result, nil
}