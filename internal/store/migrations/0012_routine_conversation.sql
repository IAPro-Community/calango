-- Routines run in a conversation of their own, and routine_finish ends a routine for good.
ALTER TABLE routines ADD COLUMN conversation_id TEXT REFERENCES conversations(id) ON DELETE SET NULL;
ALTER TABLE routines ADD COLUMN ended_at TEXT;
