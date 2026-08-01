ALTER TABLE irrigation_actions
DROP CONSTRAINT IF EXISTS irrigation_actions_command_id_fkey;

ALTER TABLE irrigation_actions
DROP COLUMN IF EXISTS command_id;