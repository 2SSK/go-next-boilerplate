-- Todos belong to a user: the `sub` claim of their access token. Rows from
-- before this migration get an empty owner, which no user can ever match.
ALTER TABLE todos ADD COLUMN owner_id text NOT NULL DEFAULT '';
ALTER TABLE todos ALTER COLUMN owner_id DROP DEFAULT;

-- Every query filters by owner and lists newest first.
CREATE INDEX todos_owner_created_idx ON todos (owner_id, created_at DESC, id DESC);
