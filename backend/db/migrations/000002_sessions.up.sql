-- Server-side sessions after OAuth login. Tokens stay on the server; browser holds opaque session id.

CREATE TABLE sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
