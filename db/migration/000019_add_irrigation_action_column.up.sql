ALTER TABLE irrigation_commands
ADD COLUMN irrigation_action_id BIGINT;

ALTER TABLE irrigation_commands
ADD CONSTRAINT irrigation_commands_irrigation_action_id_fkey
FOREIGN KEY (irrigation_action_id)
REFERENCES irrigation_actions(id)
ON DELETE SET NULL;

CREATE INDEX idx_irrigation_commands_irrigation_action_id
ON irrigation_commands(irrigation_action_id);