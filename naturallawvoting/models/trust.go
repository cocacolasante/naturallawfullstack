package models

import "time"

type TrustVoteRequest struct {
	Score int `json:"score" binding:"required,min=1,max=100"`
}

type TrustVote struct {
	ID        int       `json:"id"`
	VoterID   int       `json:"voter_id"`
	SubjectID int       `json:"subject_id"`
	Score     int       `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TrustScore struct {
	UserID    int     `json:"user_id"`
	Score     float64 `json:"score"`
	VoteCount int     `json:"vote_count"`
}

type PublicUser struct {
	ID         int     `json:"id"`
	Username   string  `json:"username"`
	TrustScore float64 `json:"trust_score"`
	VoteCount  int     `json:"vote_count"`
}
