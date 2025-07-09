-- USERS
CREATE TABLE IF NOT EXISTS users (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name              VARCHAR(50)  NOT NULL,
    password          VARCHAR(255) NULL,
    email             VARCHAR(50)  NOT NULL UNIQUE,
    avatar            VARCHAR(255),
    is_active         BOOLEAN          DEFAULT FALSE,
    is_email_verified BOOLEAN          DEFAULT FALSE,
    created_at        TIMESTAMPTZ      DEFAULT NOW(),
    updated_at        TIMESTAMPTZ      DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS users_email_idx ON users (email);

-- DEVICES
CREATE TABLE IF NOT EXISTS devices (
    id          VARCHAR(36) PRIMARY KEY,
    user_id     UUID         NOT NULL,
    name        VARCHAR(100) NOT NULL,
    device_type VARCHAR(50),
    os          VARCHAR(50),
    browser     VARCHAR(50),
    user_agent  TEXT,
    ip          VARCHAR(45),
    last_active TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices (user_id);
CREATE INDEX IF NOT EXISTS idx_devices_ip ON devices (ip);
CREATE INDEX IF NOT EXISTS idx_devices_last_active ON devices (last_active);

-- REFRESH TOKEN
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id           SERIAL PRIMARY KEY,
    user_id      UUID        NOT NULL,
    token_hash   TEXT        NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked      BOOLEAN     NOT NULL DEFAULT FALSE,
    device_id    VARCHAR(36) NOT NULL,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_device FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_device_id ON refresh_tokens (device_id);
