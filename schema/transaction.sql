CREATE TABLE transaction
(
    timestamp  timestamp without time zone NOT NULL,
    gas_wanted integer                              DEFAULT 0,
    gas_used   integer                              DEFAULT 0,
    height     integer                     NOT NULL REFERENCES block (height),
    hash       character varying(64)       NOT NULL UNIQUE PRIMARY KEY,
    messages   jsonb                       NOT NULL DEFAULT '[]'::jsonb,
    fee        jsonb                       NOT NULL DEFAULT '{}'::jsonb,
    signatures jsonb                       NOT NULL DEFAULT '[]'::jsonb,
    memo       character varying(256)
);
