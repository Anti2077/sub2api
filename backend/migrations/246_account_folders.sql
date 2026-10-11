-- Organizational folders are independent of scheduling/billing groups.
CREATE TABLE IF NOT EXISTS account_folders (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(80) NOT NULL,
    parent_id BIGINT REFERENCES account_folders(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE UNIQUE INDEX IF NOT EXISTS account_folders_sibling_name
    ON account_folders (COALESCE(parent_id, 0), LOWER(name));
CREATE TABLE IF NOT EXISTS account_folder_members (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    folder_id BIGINT NOT NULL REFERENCES account_folders(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS account_folder_members_folder_id ON account_folder_members(folder_id);
