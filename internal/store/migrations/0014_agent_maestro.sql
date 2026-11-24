-- An agent opts in before its Codex runs use the isolated Maestro home.
ALTER TABLE agents ADD COLUMN maestro_enabled INTEGER NOT NULL DEFAULT 0 CHECK (maestro_enabled IN (0, 1));
