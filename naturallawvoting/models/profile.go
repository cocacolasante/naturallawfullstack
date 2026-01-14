package models

import (
	"time"

	"github.com/lib/pq"
)

type UserProfile struct {
	UserID             int            `json:"user_id" db:"user_id"`
	Email              string         `json:"email" db:"email"`
	FullName           string         `json:"full_name" db:"full_name"`
	Birthday           *time.Time     `json:"birthday" db:"birthday"`
	Gender             string         `json:"gender" db:"gender"`
	MothersMaidenName  string         `json:"mothers_maiden_name" db:"mothers_maiden_name"`
	PhoneNumber        string         `json:"phone_number" db:"phone_number"`
	AdditionalEmails   pq.StringArray `json:"additional_emails" db:"additional_emails"`
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at" db:"updated_at"`
}

type UserAddress struct {
	UserID       int       `json:"user_id" db:"user_id"`
	StreetNumber string    `json:"street_number" db:"street_number"`
	StreetName   string    `json:"street_name" db:"street_name"`
	AddressLine2 string    `json:"address_line_2" db:"address_line_2"`
	City         string    `json:"city" db:"city"`
	State        string    `json:"state" db:"state"`
	ZipCode      string    `json:"zip_code" db:"zip_code"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type UserPoliticalAffiliation struct {
	UserID           int       `json:"user_id" db:"user_id"`
	PartyAffiliation string    `json:"party_affiliation" db:"party_affiliation"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type UserReligiousAffiliation struct {
	UserID                 int            `json:"user_id" db:"user_id"`
	Religion               string         `json:"religion" db:"religion"`
	SupportingReligion     *int           `json:"supporting_religion" db:"supporting_religion"`
	ReligiousServicesTypes pq.StringArray `json:"religious_services_types" db:"religious_services_types"`
	CreatedAt              time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at" db:"updated_at"`
}

type UserRaceEthnicity struct {
	UserID    int            `json:"user_id" db:"user_id"`
	Race      pq.StringArray `json:"race" db:"race"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// Request types for creating/updating profiles
type CreateUserProfileRequest struct {
	FullName          string   `json:"full_name"`
	Birthday          string   `json:"birthday"` // Format: YYYY-MM-DD
	Gender            string   `json:"gender"`
	MothersMaidenName string   `json:"mothers_maiden_name"`
	PhoneNumber       string   `json:"phone_number"`
	AdditionalEmails  []string `json:"additional_emails"`
}

type UpdateUserProfileRequest struct {
	FullName          *string  `json:"full_name"`
	Birthday          *string  `json:"birthday"` // Format: YYYY-MM-DD
	Gender            *string  `json:"gender"`
	MothersMaidenName *string  `json:"mothers_maiden_name"`
	PhoneNumber       *string  `json:"phone_number"`
	AdditionalEmails  []string `json:"additional_emails"`
}

type CreateUserAddressRequest struct {
	StreetNumber string `json:"street_number"`
	StreetName   string `json:"street_name"`
	AddressLine2 string `json:"address_line_2"`
	City         string `json:"city"`
	State        string `json:"state"`
	ZipCode      string `json:"zip_code"`
}

type UpdateUserAddressRequest struct {
	StreetNumber *string `json:"street_number"`
	StreetName   *string `json:"street_name"`
	AddressLine2 *string `json:"address_line_2"`
	City         *string `json:"city"`
	State        *string `json:"state"`
	ZipCode      *string `json:"zip_code"`
}

type CreateUserPoliticalAffiliationRequest struct {
	PartyAffiliation string `json:"party_affiliation"`
}

type UpdateUserPoliticalAffiliationRequest struct {
	PartyAffiliation *string `json:"party_affiliation"`
}

type CreateUserReligiousAffiliationRequest struct {
	Religion               string   `json:"religion"`
	SupportingReligion     *int     `json:"supporting_religion" binding:"omitempty,min=0,max=10"`
	ReligiousServicesTypes []string `json:"religious_services_types"`
}

type UpdateUserReligiousAffiliationRequest struct {
	Religion               *string  `json:"religion"`
	SupportingReligion     *int     `json:"supporting_religion" binding:"omitempty,min=0,max=10"`
	ReligiousServicesTypes []string `json:"religious_services_types"`
}

type CreateUserRaceEthnicityRequest struct {
	Race []string `json:"race"`
}

type UpdateUserRaceEthnicityRequest struct {
	Race []string `json:"race"`
}

type EconomicInfo struct {
	UserID                       int            `json:"user_id" db:"user_id"`
	ForCurrentPoliticalStructure string         `json:"for_current_political_structure" db:"for_current_political_structure"`
	ForCapitalism                string         `json:"for_capitalism" db:"for_capitalism"`
	ForLaws                      string         `json:"for_laws" db:"for_laws"`
	GoodsServices                pq.StringArray `json:"goods_services" db:"goods_services"`
	Affiliations                 pq.StringArray `json:"affiliations" db:"affiliations"`
	SupportOfAltEcon             string         `json:"support_of_alt_econ" db:"support_of_alt_econ"`
	SupportAltComm               string         `json:"support_alt_comm" db:"support_alt_comm"`
	AdditionalText               string         `json:"additional_text" db:"additional_text"`
	CreatedAt                    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt                    time.Time      `json:"updated_at" db:"updated_at"`
}

type CreateEconomicInfoRequest struct {
	ForCurrentPoliticalStructure string   `json:"for_current_political_structure"`
	ForCapitalism                string   `json:"for_capitalism"`
	ForLaws                      string   `json:"for_laws"`
	GoodsServices                []string `json:"goods_services"`
	Affiliations                 []string `json:"affiliations"`
	SupportOfAltEcon             string   `json:"support_of_alt_econ"`
	SupportAltComm               string   `json:"support_alt_comm"`
	AdditionalText               string   `json:"additional_text"`
}

type UpdateEconomicInfoRequest struct {
	ForCurrentPoliticalStructure *string  `json:"for_current_political_structure"`
	ForCapitalism                *string  `json:"for_capitalism"`
	ForLaws                      *string  `json:"for_laws"`
	GoodsServices                []string `json:"goods_services"`
	Affiliations                 []string `json:"affiliations"`
	SupportOfAltEcon             *string  `json:"support_of_alt_econ"`
	SupportAltComm               *string  `json:"support_alt_comm"`
	AdditionalText               *string  `json:"additional_text"`
}
