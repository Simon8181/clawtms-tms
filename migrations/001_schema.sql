-- ClawTMS Database Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enum types
CREATE TYPE load_status AS ENUM ('pending', 'quoted', 'dispatched', 'in_transit', 'delivered');
CREATE TYPE equipment_type AS ENUM ('53FT_VAN', '26FT_BOX', 'FLATBED', 'STEP_DECK');
CREATE TYPE stop_type AS ENUM ('pickup', 'delivery');

-- Loads table (运单主表)
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

-- Stops table (站点明细表)
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

-- Quotes table (报价记录表)
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

-- Indexes
CREATE INDEX idx_stops_load_id ON stops(load_id);
CREATE INDEX idx_stops_zip_prefix ON stops(zip_prefix);
CREATE INDEX idx_quotes_load_id ON quotes(load_id);
CREATE INDEX idx_loads_status ON loads(status);
CREATE INDEX idx_loads_is_fba ON loads(is_fba);
