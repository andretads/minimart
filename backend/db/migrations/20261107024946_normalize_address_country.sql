-- migrate:up
UPDATE addresses
SET country = 'BR'
WHERE lower(btrim(country)) IN ('brasil', 'brazil');

-- migrate:down
-- Intentionally a no-op: reverting would re-introduce the localized country
-- values that broke shipping zone matching, and there is no reliable way to
-- tell which rows were changed.
