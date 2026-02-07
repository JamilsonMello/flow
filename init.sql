DROP TABLE IF EXISTS assertions;
DROP TABLE IF EXISTS points;
DROP TABLE IF EXISTS flows;
DROP TABLE IF EXISTS flow_events;

CREATE TABLE flows (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(50) DEFAULT 'ACTIVE', -- ACTIVE, FINISHED
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE points (
    id BIGSERIAL PRIMARY KEY,
    flow_id BIGINT REFERENCES flows(id) ON DELETE CASCADE,
    description TEXT,
    expected JSONB,
    service_name VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE assertions (
    id BIGSERIAL PRIMARY KEY,
    flow_id BIGINT REFERENCES flows(id) ON DELETE CASCADE,
    actual JSONB,
    service_name VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_points_flow_id ON points(flow_id);
CREATE INDEX idx_assertions_flow_id ON assertions(flow_id);
