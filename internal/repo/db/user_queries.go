package db

const userSelectQ = `
SELECT COUNT(DISTINCT u.id)
FROM users u
`

const userListQ = `
SELECT 
	u.id, 
	u.name, 
	u.email, 
	u.avatar,
	u.is_active,
	u.is_email_verified,
	u.created_at, 
	u.updated_at
FROM users u
LIMIT $1 OFFSET $2
`

const userGetByIDQ = `
SELECT 
	u.id, 
	u.name, 
	u.email, 
	u.avatar,
	u.is_active,
	u.is_email_verified,
	u.created_at, 
	u.updated_at
FROM users u
WHERE u.id = $1
GROUP BY u.id
`

const userGetByEmailQ = `
SELECT 
    u.id, 
    u.name, 
    u.email, 
    u.password,
    u.avatar,
	u.is_active,
	u.is_email_verified,
    u.created_at, 
    u.updated_at
FROM users u
WHERE email = $1
GROUP BY u.id
`

const userCreateQ = `
INSERT INTO users (name, password, email, avatar, is_active, is_email_verified) 
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`

const userUpdateQ = `
UPDATE users 
SET name = $1, 
    email = $2,
    avatar = $3,
	is_active = $4,
	is_email_verified = $5
WHERE id = $6`

const userUpdateWithPassQ = `
UPDATE users 
SET name = $1, 
    email = $2,
	password = $3,
    avatar = $4,
	is_active = $5,
	is_email_verified = $6
WHERE id = $7`

const userDeleteQ = `
DELETE FROM users 
WHERE id = $1
`
