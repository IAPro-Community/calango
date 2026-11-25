-- Existing agents retain the explicit fallback chain; capacity routing is opt-in.
ALTER TABLE agents ADD COLUMN capacity_routing INTEGER NOT NULL DEFAULT 0 CHECK (capacity_routing IN (0, 1));
