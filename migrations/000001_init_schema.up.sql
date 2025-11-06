CREATE TABLE IF NOT EXISTS recipients (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    username VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    last_sent_at TIMESTAMP,
    delivery_status VARCHAR(50) DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_recipients_user_id ON recipients(user_id);
CREATE INDEX idx_recipients_is_active ON recipients(is_active);
CREATE INDEX idx_recipients_delivery_status ON recipients(delivery_status);

CREATE TABLE IF NOT EXISTS message_logs (
    id BIGSERIAL PRIMARY KEY,
    message_id INTEGER NOT NULL,
    message_type VARCHAR(50),
    message_text TEXT,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_recipients INTEGER DEFAULT 0,
    successfully_sent INTEGER DEFAULT 0
);

CREATE INDEX idx_message_logs_message_id ON message_logs(message_id);
CREATE INDEX idx_message_logs_sent_at ON message_logs(sent_at);
