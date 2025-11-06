CREATE TABLE IF NOT EXISTS sent_events (
    id BIGSERIAL PRIMARY KEY,
    event_date DATE NOT NULL,
    event_description TEXT NOT NULL,
    event_hash VARCHAR(64) NOT NULL UNIQUE,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sent_events_event_hash ON sent_events(event_hash);
CREATE INDEX idx_sent_events_event_date ON sent_events(event_date);
CREATE INDEX idx_sent_events_sent_at ON sent_events(sent_at);

