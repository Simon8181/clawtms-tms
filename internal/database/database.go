package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"clawtms/internal/models"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func Connect() (*DB, error) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "clawtms"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Connected to PostgreSQL database")
	return &DB{db}, nil
}

func (db *DB) InitSchema() error {
	schema := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	
	CREATE TYPE IF NOT EXISTS load_status AS ENUM ('pending', 'quoted', 'dispatched', 'in_transit', 'delivered');
	CREATE TYPE IF NOT EXISTS equipment_type AS ENUM ('53FT_VAN', '26FT_BOX', 'FLATBED', 'STEP_DECK');
	CREATE TYPE IF NOT EXISTS stop_type AS ENUM ('pickup', 'delivery');
	
	CREATE TABLE IF NOT EXISTS loads (
		load_id VARCHAR(50) PRIMARY KEY,
		status load_status DEFAULT 'pending',
		commodity VARCHAR(255) NOT NULL,
		equipment_required equipment_type NOT NULL,
		total_weight DECIMAL(10,2) NOT NULL,
		is_fba BOOLEAN DEFAULT FALSE,
		linear_feet INTEGER DEFAULT 0,
		special_instructions TEXT,
		pallet_count INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	
	CREATE TABLE IF NOT EXISTS stops (
		stop_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		load_id VARCHAR(50) NOT NULL REFERENCES loads(load_id) ON DELETE CASCADE,
		sequence INTEGER NOT NULL,
		type stop_type NOT NULL,
		location_name VARCHAR(255) NOT NULL,
		full_address TEXT NOT NULL,
		zip_code VARCHAR(10) NOT NULL,
		zip_prefix VARCHAR(3) NOT NULL,
		appt_time TIMESTAMP WITH TIME ZONE,
		isa_number VARCHAR(50),
		contact_info JSONB,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	
	CREATE TABLE IF NOT EXISTS quotes (
		quote_id VARCHAR(50) PRIMARY KEY,
		load_id VARCHAR(50) NOT NULL REFERENCES loads(load_id) ON DELETE CASCADE,
		carrier_name VARCHAR(255) NOT NULL,
		driver_name VARCHAR(255),
		buy_rate DECIMAL(10,2) NOT NULL,
		sell_rate DECIMAL(10,2) NOT NULL,
		distance_miles INTEGER,
		weather_info VARCHAR(100),
		route_link TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	
	CREATE INDEX IF NOT EXISTS idx_stops_load_id ON stops(load_id);
	CREATE INDEX IF NOT EXISTS idx_stops_zip_prefix ON stops(zip_prefix);
	CREATE INDEX IF NOT EXISTS idx_quotes_load_id ON quotes(load_id);
	CREATE INDEX IF NOT EXISTS idx_loads_status ON loads(status);
	`

	_, err := db.Exec(schema)
	return err
}

func (db *DB) CreateLoad(req models.CreateLoadRequest) (*models.Load, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO loads (load_id, commodity, equipment_required, total_weight, is_fba, linear_feet, special_instructions, pallet_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, req.LoadID, req.Commodity, req.EquipmentRequired, req.TotalWeight, req.IsFBA, req.LinearFeet, req.SpecialInstructions, req.PalletCount)
	if err != nil {
		return nil, err
	}

	for _, stop := range req.Stops {
		zipPrefix := stop.ZipCode
		if len(zipPrefix) >= 3 {
			zipPrefix = zipPrefix[:3]
		}

		var contactJSON []byte
		if stop.ContactInfo != nil {
			contactJSON, _ = json.Marshal(stop.ContactInfo)
		}

		_, err = tx.Exec(`
			INSERT INTO stops (load_id, sequence, type, location_name, full_address, zip_code, zip_prefix, appt_time, isa_number, contact_info)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, req.LoadID, stop.Sequence, stop.Type, stop.LocationName, stop.FullAddress, stop.ZipCode, zipPrefix, stop.ApptTime, stop.ISANumber, contactJSON)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return db.GetLoad(req.LoadID)
}

func (db *DB) GetLoad(loadID string) (*models.Load, error) {
	var load models.Load
	err := db.QueryRow(`
		SELECT load_id, status, commodity, equipment_required, total_weight, is_fba, linear_feet, special_instructions, pallet_count, created_at, updated_at
		FROM loads WHERE load_id = $1
	`, loadID).Scan(&load.LoadID, &load.Status, &load.Commodity, &load.EquipmentRequired, &load.TotalWeight, &load.IsFBA, &load.LinearFeet, &load.SpecialInstructions, &load.PalletCount, &load.CreatedAt, &load.UpdatedAt)
	if err != nil {
		return nil, err
	}

	stops, err := db.GetStopsByLoadID(loadID)
	if err != nil {
		return nil, err
	}
	load.Stops = stops

	return &load, nil
}

func (db *DB) GetStopsByLoadID(loadID string) ([]models.Stop, error) {
	rows, err := db.Query(`
		SELECT stop_id, load_id, sequence, type, location_name, full_address, zip_code, zip_prefix, appt_time, isa_number, contact_info, created_at
		FROM stops WHERE load_id = $1 ORDER BY sequence
	`, loadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stops []models.Stop
	for rows.Next() {
		var stop models.Stop
		var contactJSON []byte
		err := rows.Scan(&stop.StopID, &stop.LoadID, &stop.Sequence, &stop.Type, &stop.LocationName, &stop.FullAddress, &stop.ZipCode, &stop.ZipPrefix, &stop.ApptTime, &stop.ISANumber, &contactJSON, &stop.CreatedAt)
		if err != nil {
			return nil, err
		}
		if contactJSON != nil {
			var contact models.ContactInfo
			json.Unmarshal(contactJSON, &contact)
			stop.ContactInfo = &contact
		}
		stops = append(stops, stop)
	}
	return stops, nil
}

func (db *DB) GetAllLoads() ([]models.Load, error) {
	rows, err := db.Query(`
		SELECT load_id, status, commodity, equipment_required, total_weight, is_fba, linear_feet, special_instructions, pallet_count, created_at, updated_at
		FROM loads ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var loads []models.Load
	for rows.Next() {
		var load models.Load
		err := rows.Scan(&load.LoadID, &load.Status, &load.Commodity, &load.EquipmentRequired, &load.TotalWeight, &load.IsFBA, &load.LinearFeet, &load.SpecialInstructions, &load.PalletCount, &load.CreatedAt, &load.UpdatedAt)
		if err != nil {
			return nil, err
		}
		loads = append(loads, load)
	}
	return loads, nil
}

func (db *DB) UpdateLoadStatus(loadID string, status models.LoadStatus) error {
	_, err := db.Exec(`UPDATE loads SET status = $1, updated_at = NOW() WHERE load_id = $2`, status, loadID)
	return err
}

func (db *DB) CreateQuote(req models.CreateQuoteRequest) (*models.Quote, error) {
	quoteID := fmt.Sprintf("FBA-NJ-%s", time.Now().Format("20060102150405"))

	_, err := db.Exec(`
		INSERT INTO quotes (quote_id, load_id, carrier_name, driver_name, buy_rate, sell_rate)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, quoteID, req.LoadID, req.CarrierName, req.DriverName, req.BuyRate, req.SellRate)
	if err != nil {
		return nil, err
	}

	db.UpdateLoadStatus(req.LoadID, models.LoadStatusQuoted)

	return db.GetQuote(quoteID)
}

func (db *DB) GetQuote(quoteID string) (*models.Quote, error) {
	var quote models.Quote
	err := db.QueryRow(`
		SELECT quote_id, load_id, carrier_name, driver_name, buy_rate, sell_rate, distance_miles, weather_info, route_link, created_at
		FROM quotes WHERE quote_id = $1
	`, quoteID).Scan(&quote.QuoteID, &quote.LoadID, &quote.CarrierName, &quote.DriverName, &quote.BuyRate, &quote.SellRate, &quote.DistanceMiles, &quote.WeatherInfo, &quote.RouteLink, &quote.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &quote, nil
}

func (db *DB) GetQuotesByLoadID(loadID string) ([]models.Quote, error) {
	rows, err := db.Query(`
		SELECT quote_id, load_id, carrier_name, driver_name, buy_rate, sell_rate, distance_miles, weather_info, route_link, created_at
		FROM quotes WHERE load_id = $1 ORDER BY created_at DESC
	`, loadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotes []models.Quote
	for rows.Next() {
		var quote models.Quote
		err := rows.Scan(&quote.QuoteID, &quote.LoadID, &quote.CarrierName, &quote.DriverName, &quote.BuyRate, &quote.SellRate, &quote.DistanceMiles, &quote.WeatherInfo, &quote.RouteLink, &quote.CreatedAt)
		if err != nil {
			return nil, err
		}
		quotes = append(quotes, quote)
	}
	return quotes, nil
}

func (db *DB) GetStopsByZipPrefix(zipPrefix string) ([]models.Stop, error) {
	rows, err := db.Query(`
		SELECT s.stop_id, s.load_id, s.sequence, s.type, s.location_name, s.full_address, s.zip_code, s.zip_prefix, s.appt_time, s.isa_number, s.contact_info, s.created_at
		FROM stops s
		JOIN loads l ON s.load_id = l.load_id
		WHERE s.zip_prefix = $1 AND l.status IN ('pending', 'quoted')
		ORDER BY s.sequence
	`, zipPrefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stops []models.Stop
	for rows.Next() {
		var stop models.Stop
		var contactJSON []byte
		err := rows.Scan(&stop.StopID, &stop.LoadID, &stop.Sequence, &stop.Type, &stop.LocationName, &stop.FullAddress, &stop.ZipCode, &stop.ZipPrefix, &stop.ApptTime, &stop.ISANumber, &contactJSON, &stop.CreatedAt)
		if err != nil {
			return nil, err
		}
		if contactJSON != nil {
			var contact models.ContactInfo
			json.Unmarshal(contactJSON, &contact)
			stop.ContactInfo = &contact
		}
		stops = append(stops, stop)
	}
	return stops, nil
}

func (db *DB) UpdateQuoteRoute(quoteID string, distanceMiles int, weatherInfo, routeLink string) error {
	_, err := db.Exec(`
		UPDATE quotes SET distance_miles = $1, weather_info = $2, route_link = $3 WHERE quote_id = $4
	`, distanceMiles, weatherInfo, routeLink, quoteID)
	return err
}
