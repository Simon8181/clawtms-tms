package models

import (
	"encoding/json"
	"time"
)

type LoadStatus string
type EquipmentType string
type StopType string

const (
	LoadStatusPending    LoadStatus = "pending"
	LoadStatusQuoted     LoadStatus = "quoted"
	LoadStatusDispatched LoadStatus = "dispatched"
	LoadStatusInTransit  LoadStatus = "in_transit"
	LoadStatusDelivered  LoadStatus = "delivered"
)

const (
	Equipment53FTVan  EquipmentType = "53FT_VAN"
	Equipment26FTBox  EquipmentType = "26FT_BOX"
	EquipmentFlatbed  EquipmentType = "FLATBED"
	EquipmentStepDeck EquipmentType = "STEP_DECK"
)

const (
	StopTypePickup   StopType = "pickup"
	StopTypeDelivery StopType = "delivery"
)

type ContactInfo struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type Load struct {
	LoadID              string        `json:"load_id" db:"load_id"`
	Status              LoadStatus    `json:"status" db:"status"`
	Commodity           string        `json:"commodity" db:"commodity"`
	EquipmentRequired   EquipmentType `json:"equipment_required" db:"equipment_required"`
	TotalWeight         float64       `json:"total_weight" db:"total_weight"`
	IsFBA               bool          `json:"is_fba" db:"is_fba"`
	LinearFeet          int           `json:"linear_feet" db:"linear_feet"`
	SpecialInstructions string        `json:"special_instructions" db:"special_instructions"`
	PalletCount         int           `json:"pallet_count" db:"pallet_count"`
	CreatedAt           time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at" db:"updated_at"`
	Stops               []Stop        `json:"stops,omitempty"`
}

type Stop struct {
	StopID       string       `json:"stop_id" db:"stop_id"`
	LoadID       string       `json:"load_id" db:"load_id"`
	Sequence     int          `json:"sequence" db:"sequence"`
	Type         StopType     `json:"type" db:"type"`
	LocationName string       `json:"location_name" db:"location_name"`
	FullAddress  string       `json:"full_address" db:"full_address"`
	ZipCode      string       `json:"zip_code" db:"zip_code"`
	ZipPrefix    string       `json:"zip_prefix" db:"zip_prefix"`
	ApptTime     *time.Time   `json:"appt_time" db:"appt_time"`
	ISANumber    string       `json:"isa_number" db:"isa_number"`
	ContactInfo  *ContactInfo `json:"contact_info,omitempty" db:"contact_info"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
}

type Quote struct {
	QuoteID       string    `json:"quote_id" db:"quote_id"`
	LoadID        string    `json:"load_id" db:"load_id"`
	CarrierName   string    `json:"carrier_name" db:"carrier_name"`
	DriverName    string    `json:"driver_name" db:"driver_name"`
	BuyRate       float64   `json:"buy_rate" db:"buy_rate"`
	SellRate      float64   `json:"sell_rate" db:"sell_rate"`
	DistanceMiles int       `json:"distance_miles" db:"distance_miles"`
	WeatherInfo   string    `json:"weather_info" db:"weather_info"`
	RouteLink     string    `json:"route_link" db:"route_link"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type CreateLoadRequest struct {
	LoadID              string    `json:"load_id"`
	Commodity           string    `json:"commodity"`
	EquipmentRequired   string    `json:"equipment_required"`
	TotalWeight         float64   `json:"total_weight"`
	IsFBA               bool      `json:"is_fba"`
	LinearFeet          int       `json:"linear_feet"`
	SpecialInstructions string    `json:"special_instructions"`
	PalletCount         int       `json:"pallet_count"`
	Stops               []StopReq `json:"stops"`
}

type StopReq struct {
	Sequence     int          `json:"sequence"`
	Type         string       `json:"type"`
	LocationName string       `json:"location_name"`
	FullAddress  string       `json:"full_address"`
	ZipCode      string       `json:"zip_code"`
	ApptTime     *time.Time   `json:"appt_time"`
	ISANumber    string       `json:"isa_number"`
	ContactInfo  *ContactInfo `json:"contact_info"`
}

type CreateQuoteRequest struct {
	LoadID      string  `json:"load_id"`
	CarrierName string  `json:"carrier_name"`
	DriverName  string  `json:"driver_name"`
	BuyRate     float64 `json:"buy_rate"`
	SellRate    float64 `json:"sell_rate"`
}

type RouteResult struct {
	LoadID        string   `json:"load_id"`
	QuoteID       string   `json:"quote_id,omitempty"`
	Distance      string   `json:"distance"`
	DistanceMiles int      `json:"distance_miles"`
	Duration      string   `json:"duration"`
	RouteLink     string   `json:"route_link"`
	Weather       string   `json:"weather"`
	ZipPrefixes   []string `json:"zip_prefixes"`
	Stops         []Stop   `json:"stops"`
}

type RouteCalculationRequest struct {
	LoadIDs []string `json:"load_ids"`
}

type RouteCalculationResponse struct {
	Routes        []RouteResult `json:"routes"`
	TotalDistance string        `json:"total_distance"`
	TotalMiles    int           `json:"total_miles"`
}

type FBACheckResult struct {
	LoadID          string  `json:"load_id"`
	IsOverweight    bool    `json:"is_overweight"`
	WeightPerPallet float64 `json:"weight_per_pallet"`
	Message         string  `json:"message"`
}

func (c *ContactInfo) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, c)
	case string:
		return json.Unmarshal([]byte(v), c)
	}
	return nil
}
