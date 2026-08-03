-- Add alerts table
CREATE TABLE alerts (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    type VARCHAR(64) NOT NULL,
    title VARCHAR(255),
    description TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alerts_device_id ON alerts (device_id);
CREATE INDEX idx_alerts_status ON alerts (status);