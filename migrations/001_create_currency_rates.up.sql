CREATE TABLE IF NOT EXISTS currency_rates (
    char_code   VARCHAR(3) PRIMARY KEY,
    name        TEXT        NOT NULL,
    nominal     INTEGER     NOT NULL CHECK (nominal > 0),
    value       NUMERIC(20, 4) NOT NULL CHECK (value >= 0),
    num_code    VARCHAR(3),
    updated_at  TIMESTAMP   NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_currency_char_code ON currency_rates(char_code);