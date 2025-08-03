CREATE TABLE IF NOT EXISTS historical_currency_rates (
    char_code   VARCHAR(3) NOT NULL,
    date        DATE NOT NULL,
    name        TEXT        NOT NULL,
    nominal     INTEGER     NOT NULL CHECK (nominal > 0),
    value       NUMERIC(20, 4) NOT NULL CHECK (value >= 0),
    num_code    VARCHAR(3),
    PRIMARY KEY (char_code, date)
);

CREATE INDEX IF NOT EXISTS idx_historical_currency_date ON historical_currency_rates(date);
CREATE INDEX IF NOT EXISTS idx_historical_currency_char_code ON historical_currency_rates(char_code)