package db

const listDevices = `
SELECT id, name, device_type, os, user_agent, browser, ip, last_active 
FROM devices 
WHERE user_id = $1
`

const getDevice = `
SELECT
	id,
	user_id,
	name,
	device_type,
	os,
	browser,
	user_agent,
	ip,
	last_active,
	created_at
FROM devices
WHERE id = $1 AND user_id = $2
`

const getDeviceByID = `
SELECT
	id,
	user_id,
	name,
	device_type,
	os,
	browser,
	user_agent,
	ip,
	last_active,
	created_at
FROM devices
WHERE id = $1
`

const updateDevice = `
UPDATE devices
SET name = $1
WHERE id = $2 AND user_id = $3
`

const deleteDevice = `
DELETE FROM devices
WHERE id = $1 AND user_id = $2
`
