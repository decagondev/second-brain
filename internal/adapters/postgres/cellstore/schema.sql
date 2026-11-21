-- A cell's own records (SB-1508): JSON documents by tenant, kind and id, with the time,
-- user and space the queries need. Names, ids and decisions only; document content never
-- reaches these tables (the index holds chunks, under its own access rules).
CREATE TABLE IF NOT EXISTS cell_docs (
    tenant text NOT NULL,
    kind   text NOT NULL,
    id     text NOT NULL,
    at     timestamptz NOT NULL,
    usr    text NOT NULL DEFAULT '',
    space  text NOT NULL DEFAULT '',
    body   jsonb NOT NULL,
    PRIMARY KEY (tenant, kind, id)
);
CREATE INDEX IF NOT EXISTS cell_docs_at ON cell_docs (tenant, kind, at);
CREATE INDEX IF NOT EXISTS cell_docs_usr ON cell_docs (tenant, kind, usr);

-- Append-only queues (the chat events inbox): a position from 0, gap-free.
CREATE TABLE IF NOT EXISTS cell_queue (
    tenant text NOT NULL,
    kind   text NOT NULL,
    pos    bigint NOT NULL CHECK (pos >= 0),
    body   jsonb NOT NULL,
    PRIMARY KEY (tenant, kind, pos)
);
