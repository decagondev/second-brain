-- Control-plane tenant registry (SB-105). Names, ids and configuration only; no tenant content.
CREATE TABLE IF NOT EXISTS applied (
    tenant    text PRIMARY KEY,
    resources jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS cells (
    tenant        text PRIMARY KEY,
    region        text NOT NULL,
    endpoint      text NOT NULL,
    custody       text NOT NULL CHECK (custody IN ('factory', 'tenant', 'detached')),
    placement     text NOT NULL,
    key_custodian text NOT NULL CHECK (key_custodian IN ('factory', 'tenant'))
);

-- A host routes to exactly one cell: the primary key makes a shared host impossible.
CREATE TABLE IF NOT EXISTS cell_hosts (
    host   text PRIMARY KEY,
    tenant text NOT NULL REFERENCES cells (tenant) ON DELETE CASCADE,
    ord    int  NOT NULL
);

CREATE TABLE IF NOT EXISTS journal (
    id       bigserial PRIMARY KEY,
    tenant   text NOT NULL,
    apply_id text NOT NULL,
    step     int  NOT NULL,
    op       text NOT NULL,
    kind     text NOT NULL,
    name     text NOT NULL,
    phase    text NOT NULL CHECK (phase IN ('start', 'done')),
    at       timestamptz NOT NULL DEFAULT now()
);
