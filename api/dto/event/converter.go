package eventdto

import (
	"treffly/api/common"
	"treffly/api/models"
)

type EventConverter struct {
	env    string
	domain string
}

func NewEventConverter(env, domain string) *EventConverter {
	return &EventConverter{
		env:    env,
		domain: domain,
	}
}

func (c *EventConverter) ToEventResponse(e models.Event) EventResponse {
	return EventResponse{
		ID:               e.ID,
		Name:             e.Name,
		Description:      e.Description,
		Capacity:         e.Capacity,
		Latitude:         e.Latitude,
		Longitude:        e.Longitude,
		Address:          e.Address,
		Date:             e.Date,
		IsPrivate:        e.IsPrivate,
		IsPremium:        e.IsPremium,
		CreatedAt:        e.CreatedAt,
		OwnerUsername: 	  e.OwnerUsername,
		IsOwner:          e.IsOwner,
		IsParticipant:    e.IsParticipant,
		Tags:             c.convertTagsToResponse(e.Tags),
		ParticipantCount: e.ParticipantCount,
		ImageEventURL:    common.ImageURL(c.env, c.domain, e.ImagePath),
	}
}

func (c *EventConverter) convertTagsToResponse(tags []models.Tag) []TagResponse {
	result := make([]TagResponse, len(tags))
	for i, t := range tags {
		result[i] = TagResponse{
			ID:   t.ID,
			Name: t.Name,
		}
	}
	return result
}

func (c *EventConverter) ToEventsResponse(events []models.Event) []EventResponse {
	result := make([]EventResponse, len(events))
	for i, e := range events {
		result[i] = c.ToEventResponse(e)
	}
	return result
}

func (c *EventConverter) ToHomeEventsResponse(h models.HomeEvents) HomeEventsResponse {
	return HomeEventsResponse{
		Premium:     c.ToEventsResponse(h.Premium),
		Recommended: c.ToEventsResponse(h.Recommended),
		Latest:      c.ToEventsResponse(h.Latest),
		Popular:     c.ToEventsResponse(h.Popular),
	}
}
