ALTER TABLE recipients ADD COLUMN IF NOT EXISTS allow_sending BOOLEAN DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_recipients_allow_sending ON recipients(allow_sending);

UPDATE recipients SET allow_sending = true WHERE allow_sending IS NULL;

