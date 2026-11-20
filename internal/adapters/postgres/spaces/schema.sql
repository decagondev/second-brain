-- A cell's space registry (SB-1002). Names and configuration only; no content.
-- Only `sb tenant apply` writes it.
CREATE TABLE IF NOT EXISTS spaces (
    tenant text NOT NULL,
    id     text NOT NULL CHECK (id ~ '^spc_[a-z][a-z0-9-]{1,30}$'),
    space  jsonb NOT NULL,
    PRIMARY KEY (tenant, id)
);
