-- Core catalog + enrollment schema (v1). Enrollment capacity enforcement is application-level
-- (transactions + SELECT ... FOR UPDATE); see domain tests later.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE class_status AS ENUM ('draft', 'published', 'archived');

CREATE TYPE enrollment_status AS ENUM ('active', 'withdrawn');

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    idp_subject text NOT NULL UNIQUE,
    display_name text,
    email text,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE classes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title text NOT NULL,
    description text,
    capacity int NOT NULL CHECK (capacity >= 0),
    status class_status NOT NULL DEFAULT 'draft',
    enrollment_opens_at timestamptz,
    enrollment_closes_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE enrollments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    class_id uuid NOT NULL REFERENCES classes (id) ON DELETE RESTRICT,
    status enrollment_status NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT enrollments_user_class_unique UNIQUE (user_id, class_id)
);

CREATE INDEX enrollments_class_id_idx ON enrollments (class_id);
CREATE INDEX enrollments_user_id_idx ON enrollments (user_id);
