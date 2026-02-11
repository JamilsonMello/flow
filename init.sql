DROP TABLE IF EXISTS assertions;
DROP TABLE IF EXISTS points;
DROP TABLE IF EXISTS flows;
DROP TABLE IF EXISTS flow_events;

CREATE TABLE flows (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    identifier VARCHAR(255),
    status VARCHAR(50) DEFAULT 'ACTIVE',
    service VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE points (
    id BIGSERIAL PRIMARY KEY,
    flow_id BIGINT REFERENCES flows(id) ON DELETE CASCADE,
    description TEXT,
    expected JSONB,
    service_name VARCHAR(255),
    schema JSONB,
    timeout BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE assertions (
    id BIGSERIAL PRIMARY KEY,
    flow_id BIGINT REFERENCES flows(id) ON DELETE CASCADE,
    actual JSONB,
    service_name VARCHAR(255),
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_points_flow_id ON points(flow_id);
CREATE INDEX idx_assertions_flow_id ON assertions(flow_id);
CREATE INDEX idx_flows_name_status ON flows(name, status);
CREATE INDEX idx_flows_identifier ON flows(identifier);
