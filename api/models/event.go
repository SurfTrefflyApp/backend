package models

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

type Event struct {
	ID               int32
	Name             string
	Description      string
	Capacity         int32
	Latitude         float64
	Longitude        float64
	Address          string
	Date             time.Time
	IsPrivate        bool
	IsPremium        bool
	CreatedAt        time.Time
	OwnerUsername    string
	IsOwner          bool
	IsParticipant    bool
	Tags             []Tag
	ParticipantCount int
	ImagePath        string
	OwnerImagePath   string
}

type HomeEvents struct {
	Premium     []Event
	Recommended []Event
	Latest      []Event
	Popular     []Event
}

type CreateParams struct {
	Name        string
	Description string
	Capacity    int32
	Latitude    pgtype.Numeric
	Longitude   pgtype.Numeric
	Address     string
	Date        time.Time
	IsPrivate   bool
	Tags        []int32
	OwnerID     int32
	ImageID     uuid.UUID
	Path        string
}

type ListParams struct {
	Lat       pgtype.Numeric
	Lon       pgtype.Numeric
	Search    string
	TagIDs    []int32
	DateRange string
}

type UpdateParams struct {
	EventID     int32
	Name        string
	Description string
	Capacity    int32
	Latitude    pgtype.Numeric
	Longitude   pgtype.Numeric
	Address     string
	Date        time.Time
	IsPrivate   bool
	Tags        []int32
	UserID      int32
	NewImageID  uuid.UUID
	Path        string
	DeleteImage bool
	OldImageID  uuid.UUID
}

type DeleteParams struct {
	EventID int32
	UserID  int32
}

type GetHomeParams struct {
	UserID int32
	Lat    pgtype.Numeric
	Lon    pgtype.Numeric
}

type SubscriptionParams struct {
	EventID int32
	UserID  int32
	Token   string
}

type UserEventsParams struct {
	UserID int32
}

type PremiumOrder struct {
	ID        int32     `json:"id"`
	UserID    int32     `json:"user_id"`
	EventID   int32     `json:"event_id"`
	Shop      string    `json:"shop"`
	Price     float64   `json:"price"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type PremiumOrderParams struct {
	UserID    int32     `json:"user_id"`
	EventID   int32     `json:"event_id"`
	Shop      string    `json:"shop"`
	Price     float64   `json:"price"`
}
