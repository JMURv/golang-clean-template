package db

const createUserDevice = `
INSERT INTO devices (id, user_id, name, device_type, os, browser, user_agent, ip)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE 
SET last_active = NOW()
`

const getTokenByDevice = `
SELECT id, expires_at, revoked
FROM refresh_tokens
WHERE user_id = $1 AND device_id = $2
ORDER BY created_at DESC
LIMIT 1
`

const createRefreshToken = `
INSERT INTO refresh_tokens (user_id, token_hash, expires_at, device_id)
VALUES ($1, $2, $3, $4)
`

const isValidToken = `
SELECT token_hash
FROM refresh_tokens 
WHERE user_id = $1 AND device_id = $2 AND expires_at > NOW() AND revoked IS FALSE
ORDER BY expires_at DESC
LIMIT 1
`

const revokeToken = `
UPDATE refresh_tokens 
SET revoked = TRUE 
WHERE user_id = $1
`

const revokeTokenByDevice = `
UPDATE refresh_tokens
SET revoked = TRUE
WHERE user_id = $1 AND device_id = $2
`
