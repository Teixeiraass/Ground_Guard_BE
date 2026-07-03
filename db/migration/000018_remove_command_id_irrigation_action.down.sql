ALTER TABLE irrigation_actions
ADD COLUMN command_id BIGINT;

ALTER TABLE irrigation_actions
ADD CONSTRAINT irrigation_actions_command_id_fkey
FOREIGN KEY (command_id)
REFERENCES irrigation_commands(id)
ON DELETE SET NULL;

CREATE INDEX idx_irrigation_actions_command_id
ON irrigation_actions(command_id);