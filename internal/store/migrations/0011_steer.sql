-- Steer: a continuation run that carries the owner's guidance for a stopped run.
ALTER TABLE runs ADD COLUMN steer_of TEXT REFERENCES runs(id) ON DELETE SET NULL;
