CREATE TABLE metrics (
    "id"        VARCHAR(100) NOT NULL,
    "type"      VARCHAR(100) NOT NULL,
    "delta"     BIGINT,
    "value"     DOUBLE PRECISION,
    "hash"      TEXT,
    PRIMARY KEY ("id", "type")
);