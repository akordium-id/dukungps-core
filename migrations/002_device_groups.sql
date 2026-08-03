-- Add device_groups table
CREATE TABLE device_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Junction table for device-group membership
CREATE TABLE device_group_members (
    id BIGSERIAL PRIMARY KEY,
    device_group_id BIGINT NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    device_id BIGINT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_group_id, device_id)
);