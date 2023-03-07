CREATE TABLE IF NOT EXISTS cuttlink (
	id SERIAL,
	user_id VARCHAR(36) NOT NULL,
	original_url text NOT NULL
);