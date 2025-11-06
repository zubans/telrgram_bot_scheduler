DROP INDEX IF EXISTS idx_recipients_allow_sending;
ALTER TABLE recipients DROP COLUMN IF EXISTS allow_sending;

