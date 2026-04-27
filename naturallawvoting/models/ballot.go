package models

import (
	"time"
)

type Ballot struct {
	ID          int          `json:"id" db:"id"`
	Title       string       `json:"title" db:"title"`
	Description string       `json:"description" db:"description"`
	Category    string       `json:"category" db:"category"`
	Superstate  string       `json:"superstate" db:"superstate"`
	State       string       `json:"state" db:"state"`
	District    string       `json:"district" db:"district"`
	CreatorID   int          `json:"creator_id" db:"creator_id"`
	IsActive    bool         `json:"is_active" db:"is_active"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
	Items       []BallotItem `json:"options,omitempty"` // Frontend expects "options"
}

type BallotItem struct {
	ID          int    `json:"id" db:"id"`
	BallotID    int    `json:"ballot_id" db:"ballot_id"`
	Title       string `json:"title" db:"title"`
	Description string `json:"description" db:"description"`
}

type Vote struct {
	ID           int       `json:"id" db:"id"`
	UserID       int       `json:"user_id" db:"user_id"`
	BallotID     int       `json:"ballot_id" db:"ballot_id"`
	BallotItemID int       `json:"ballot_item_id" db:"ballot_item_id"`
	Score        int       `json:"score" db:"score"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type CreateBallotRequest struct {
	Title       string                    `json:"title" binding:"required,min=1,max=200"`
	Description string                    `json:"description" binding:"max=1000"`
	Category    string                    `json:"category" binding:"max=100"`
	Superstate  string                    `json:"superstate" binding:"max=100"`
	State       string                    `json:"state" binding:"max=100"`
	District    string                    `json:"district" binding:"max=100"`
	Items       []CreateBallotItemRequest `json:"items" binding:"required,min=2"`
}

type CreateBallotItemRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=200"`
	Description string `json:"description" binding:"max=500"`
}

// ScoreEntry is one option's score within a bulk vote submission.
type ScoreEntry struct {
	BallotItemID int  `json:"ballot_item_id"`
	OptionID     int  `json:"option_id"` // Frontend alias for ballot_item_id
	Score        *int `json:"score" binding:"required"`
}

// VoteRequest carries a user's scores for every option on a ballot.
type VoteRequest struct {
	Scores []ScoreEntry `json:"scores" binding:"required,min=1,dive"`
}