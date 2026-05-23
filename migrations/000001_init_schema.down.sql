DROP TABLE IF EXISTS users CASCADE;
DROP INDEX IF EXISTS users_email_idx CASCADE;

DROP TABLE IF EXISTS devices CASCADE;
DROP INDEX IF EXISTS idx_devices_user_id CASCADE;
DROP INDEX IF EXISTS idx_devices_ip CASCADE;
DROP INDEX IF EXISTS idx_devices_last_active CASCADE;

DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP INDEX IF EXISTS idx_refresh_tokens_user CASCADE;
DROP INDEX IF EXISTS idx_refresh_tokens_expires CASCADE;
DROP INDEX IF EXISTS idx_refresh_tokens_device_id CASCADE;
