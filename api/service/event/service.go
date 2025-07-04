package eventservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"treffly/api/models"
	"treffly/apperror"
	db "treffly/db/sqlc"
	"treffly/util"
)

type Service struct {
	store  db.Store
	config util.Config
}

func New(store db.Store, config util.Config) *Service {
	return &Service{store: store, config: config}
}

func (s *Service) Create(ctx context.Context, params models.CreateParams) (models.Event, error) {
	eventArg := db.CreateEventTxParams{
		Name:        params.Name,
		Description: params.Description,
		Capacity:    params.Capacity,
		Latitude:    params.Latitude,
		Longitude:   params.Longitude,
		Address:     params.Address,
		Date:        params.Date,
		IsPrivate:   params.IsPrivate,
		OwnerID:     params.OwnerID,
		Tags:        params.Tags,
		ImageID:     params.ImageID,
	}

	imageArg := db.CreateImageParams{
		ID:   params.ImageID,
		Path: params.Path,
	}

	event, err := s.store.CreateEventTx(ctx, eventArg, imageArg)
	if err != nil {
		return models.Event{}, err
	}

	resp := ConvertGetEventRow(event, true, false)

	return resp, nil
}

func (s *Service) List(ctx context.Context, params models.ListParams) ([]models.Event, error) {
	arg := db.ListEventsParams{
		UserLat:    params.Lat,
		UserLon:    params.Lon,
		SearchTerm: params.Search,
		TagIds:     params.TagIDs,
		DateRange:  params.DateRange,
	}

	rows, err := s.store.ListEvents(ctx, arg)
	if err != nil {
		return nil, err
	}

	result := convertEventType(rows)

	return result, nil
}

func (s *Service) Update(ctx context.Context, params models.UpdateParams) (models.Event, error) {
	getArg := db.GetEventParams{
		ID:      params.EventID,
		OwnerID: params.UserID,
	}
	event, err := s.store.GetEvent(ctx, getArg)
	if err != nil {
		return models.Event{}, err
	}

	if event.OwnerID != params.UserID {
		err = errors.New("owner id missmatch")
		return models.Event{}, err
	}

	imageID := params.NewImageID
	path := params.Path
	if params.DeleteImage {
		imageID = uuid.Nil
		path = ""
	}
	if !params.DeleteImage && params.NewImageID == uuid.Nil {
		imageID = event.ImageID.Bytes
		path = event.EventImagePath.String
	}

	arg := db.UpdateEventTxParams{
		EventID:     params.EventID,
		Name:        params.Name,
		Description: params.Description,
		Capacity:    params.Capacity,
		Latitude:    params.Latitude,
		Longitude:   params.Longitude,
		Address:     params.Address,
		Date:        params.Date,
		IsPrivate:   params.IsPrivate,
		Tags:        params.Tags,
		NewImageID:  imageID,
		NewPath:     path,
		OldImageID:  params.OldImageID,
	}

	err = s.store.UpdateEventTx(ctx, arg)
	if err != nil {
		return models.Event{}, err
	}

	event, err = s.store.GetEvent(ctx, getArg)

	resp := ConvertGetEventRow(event, true, false)

	return resp, nil
}

func (s *Service) Delete(ctx context.Context, params models.DeleteParams) error {
	getArg := db.GetEventParams{
		ID:      params.EventID,
		OwnerID: params.UserID,
	}
	event, err := s.store.GetEvent(ctx, getArg)
	if err != nil {
		return err
	}

	if event.OwnerID != params.UserID {
		err = fmt.Errorf("owner id missmatch")
		return apperror.Forbidden.WithCause(err)
	}

	return s.store.DeleteEvent(ctx, params.EventID)
}

func (s *Service) GetHomeForUser(ctx context.Context, params models.GetHomeParams) (models.HomeEvents, error) {
	premium, latest, popular, err := s.getHome(ctx)
	if err != nil {
		return models.HomeEvents{}, err
	}

	arg := db.GetUserRecommendedEventsParams{
		UserID:  params.UserID,
		UserLon: params.Lon,
		UserLat: params.Lat,
	}

	recommended, err := s.store.GetUserRecommendedEvents(ctx, arg)
	if err != nil {
		return models.HomeEvents{}, err
	}

	resp := ConvertHomeEvents(premium, recommended, latest, popular)

	return resp, nil
}

func (s *Service) GetHomeForGuest(ctx context.Context, params models.GetHomeParams) (models.HomeEvents, error) {
	premium, latest, popular, err := s.getHome(ctx)
	if err != nil {
		return models.HomeEvents{}, err
	}

	arg := db.GetGuestRecommendedEventsParams{
		UserLon: params.Lon,
		UserLat: params.Lat,
	}

	recommended, err := s.store.GetGuestRecommendedEvents(ctx, arg)
	if err != nil {
		return models.HomeEvents{}, err
	}

	resp := ConvertHomeEvents(premium, recommended, latest, popular)

	return resp, nil
}

func (s *Service) getHome(ctx context.Context) ([]db.GetPremiumEventsRow, []db.GetLatestEventsRow, []db.GetPopularEventsRow, error) {
	premium, err := s.store.GetPremiumEvents(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	latest, err := s.store.GetLatestEvents(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	popular, err := s.store.GetPopularEvents(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return premium, latest, popular, nil
}

func (s *Service) getPremiumEvents(ctx context.Context) ([]db.GetPremiumEventsRow, error) {
	rows, err := s.store.GetPremiumEvents(ctx)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) getRecommendedEvents(ctx context.Context, params models.GetHomeParams) ([]db.GetUserRecommendedEventsRow, []db.GetGuestRecommendedEventsRow, error) {
	var (
		userRecommended  []db.GetUserRecommendedEventsRow
		guestRecommended []db.GetGuestRecommendedEventsRow
		err              error
	)
	if params.UserID > 0 {
		arg := db.GetUserRecommendedEventsParams{
			UserID:  params.UserID,
			UserLat: params.Lat,
			UserLon: params.Lon,
		}

		userRecommended, err = s.store.GetUserRecommendedEvents(ctx, arg)
		if err != nil {
			return []db.GetUserRecommendedEventsRow{}, []db.GetGuestRecommendedEventsRow{}, err
		}
		return userRecommended, guestRecommended, nil
	}

	arg := db.GetGuestRecommendedEventsParams{
		UserLat: params.Lat,
		UserLon: params.Lon,
	}

	guestRecommended, err = s.store.GetGuestRecommendedEvents(ctx, arg)
	if err != nil {
		return []db.GetUserRecommendedEventsRow{}, []db.GetGuestRecommendedEventsRow{}, err
	}
	return userRecommended, guestRecommended, nil
}

func (s *Service) getLatestEvents(ctx context.Context) ([]db.GetLatestEventsRow, error) {
	rows, err := s.store.GetLatestEvents(ctx)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) getPopularEvents(ctx context.Context) ([]db.GetPopularEventsRow, error) {
	rows, err := s.store.GetPopularEvents(ctx)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) Subscribe(ctx context.Context, params models.SubscriptionParams) (models.Event, error) {
	arg := db.SubscribeToEventParams{
		EventID: params.EventID,
		UserID:  params.UserID,
		Token:   params.Token,
	}

	getArg := db.GetEventParams{
		ID:      params.EventID,
		OwnerID: params.UserID,
		Token:   params.Token,
	}

	event, err := s.store.GetEvent(ctx, getArg)
	if err != nil {
		return models.Event{}, err
	}

	if event.OwnerID == params.UserID {
		return models.Event{}, fmt.Errorf("user is owner")
	}

	allowed, err := s.store.SubscribeToEvent(ctx, arg)
	if err != nil {
		return models.Event{}, err
	}
	if allowed.Valid && !allowed.Bool {
		return models.Event{}, fmt.Errorf("event is full")
	}

	return s.GetEvent(ctx, params.EventID, params.UserID, params.Token)
}

func (s *Service) Unsubscribe(ctx context.Context, params models.SubscriptionParams) (models.Event, error) {
	arg := db.UnsubscribeFromEventParams{
		EventID: params.EventID,
		UserID:  params.UserID,
	}

	if err := s.store.UnsubscribeFromEvent(ctx, arg); err != nil {
		return models.Event{}, err
	}

	event, err := s.GetEvent(ctx, params.EventID, params.UserID, params.Token)
	if err != nil {
		return models.Event{}, err
	}

	return event, err
}

func (s *Service) GetEvent(ctx context.Context, eventID, userID int32, token string) (models.Event, error) {
	getArg := db.GetEventParams{
		ID:      eventID,
		OwnerID: userID,
		Token:   token,
	}
	event, err := s.store.GetEvent(ctx, getArg)
	if err != nil {
		return models.Event{}, err
	}

	participantArg := db.IsParticipantParams{
		EventID: eventID,
		UserID:  userID,
	}

	isParticipant, err := s.store.IsParticipant(ctx, participantArg)
	if err != nil {
		return models.Event{}, err
	}

	isOwner := event.OwnerID == userID

	resp := ConvertGetEventRow(event, isOwner, isParticipant)

	return resp, nil
}

func (s *Service) GetUpcomingUserEvents(ctx context.Context, userID int32) ([]models.Event, error) {
	rows, err := s.store.GetUpcomingUserEvents(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := convertEventType(rows)

	return resp, nil
}

func (s *Service) GetPastUserEvents(ctx context.Context, userID int32) ([]models.Event, error) {
	rows, err := s.store.GetPastUserEvents(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := convertEventType(rows)

	return resp, nil
}

func (s *Service) GetOwnedUserEvents(ctx context.Context, userID int32) ([]models.Event, error) {
	rows, err := s.store.GetOwnedUserEvents(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := convertEventType(rows)

	return resp, nil
}

func (s *Service) ListAll(ctx context.Context, params models.ListParams) ([]models.Event, error) {
	arg := db.ListAllEventsParams{
		TagIds: params.TagIDs,
		SearchTerm: params.Search,
		DateRange: params.DateRange,
	}

	rows, err := s.store.ListAllEvents(ctx, arg)
	if err != nil {
		return nil, err
	}

	result := convertEventType(rows)

	return result, nil
}

func (s *Service) AdminDelete(ctx context.Context, id int32) error {
	return s.store.DeleteEvent(ctx, id)
}

func (s *Service) CreatePremiumOrder(ctx context.Context, params models.PremiumOrderParams) (models.PremiumOrder, error) {
	arg := db.CreatePremiumOrderParams{
		UserID: params.UserID,
		EventID: params.EventID,
		Shop: params.Shop,
		Price: util.Float64ToNumeric(params.Price),
	}

	order, err := s.store.CreatePremiumOrder(ctx, arg)
	if err != nil {
		return models.PremiumOrder{}, err
	}

	resp := ConvertPremiumOrder(order)

	return resp, nil
}

func (s *Service) GetPremiumOrder(ctx context.Context, id int32) (models.PremiumOrder, error) {
	order, err := s.store.GetPremiumOrder(ctx, id)
	if err != nil {
		return models.PremiumOrder{}, err
	}

	resp := ConvertPremiumOrder(order)

	return resp, nil
}

func (s *Service) CompletePremiumOrder(ctx context.Context, id int32) error {
	err := s.store.SetEventPremium(ctx, id)

	return err
}