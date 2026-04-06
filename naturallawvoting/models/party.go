package models

import "time"

type Party struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"`
	Slug          string          `json:"slug"`
	Description   string          `json:"description"`
	IntroText     string          `json:"intro_text"`
	SortOrder     int             `json:"sort_order"`
	ExternalLinks []ExternalLink  `json:"external_links,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type ExternalLink struct {
	ID        int    `json:"id"`
	PartyID   int    `json:"party_id"`
	URL       string `json:"url"`
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
}

type PartyEngagement struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	PartySlug string    `json:"party_slug"`
	Q1Answer  string    `json:"q1_answer"`
	Q2Answer  string    `json:"q2_answer"`
	Q3Answer  string    `json:"q3_answer"`
	Q4Answer  string    `json:"q4_answer"`
	Comments  string    `json:"comments"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PartyEngagementRequest struct {
	Q1 string `json:"q1"`
	Q2 string `json:"q2"`
	Q3 string `json:"q3"`
	Q4 string `json:"q4"`
	Comments string `json:"comments"`
}
