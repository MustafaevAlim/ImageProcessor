CREATE TABLE IF NOT EXISTS image_path (
    id SERIAL PRIMARY KEY,
    uploads_path VARCHAR(100) NOT NULL,
    processed_path VARCHAR(100) NOT NULL,
    processed BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);