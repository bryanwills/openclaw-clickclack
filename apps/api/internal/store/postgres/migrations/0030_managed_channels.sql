ALTER TABLE channels ADD COLUMN external_managed INTEGER NOT NULL DEFAULT 0;
ALTER TABLE channels ADD COLUMN external_ref TEXT;
ALTER TABLE channels ADD COLUMN external_url TEXT;
ALTER TABLE channels ADD COLUMN sidebar_section TEXT;
