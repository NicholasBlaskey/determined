BEGIN;

CREATE OR REPLACE FUNCTION safe_sum_accumulate(float8, float8, OUT float8)
  RETURNS float8 AS $$
  BEGIN
    -- Check for potential overflow
    BEGIN
      IF $1 IS NULL THEN
        $3 := $2;
      ELSIF $2 IS NULL THEN
        $3 := $1;
      ELSE
        $3 := $1 + $2;
      END IF;
    EXCEPTION
      WHEN numeric_value_out_of_range THEN
        IF $1 < 0 THEN
          $3 := '-Infinity';
        ELSE
          $3 := 'Infinity';
        END IF;
    END;
  END;
$$ LANGUAGE plpgsql;

DROP AGGREGATE IF EXISTS safe_sum(float8);

CREATE AGGREGATE safe_sum(float8) (
  SFUNC = safe_sum_accumulate,
  STYPE = float8
);

CREATE TEMPORARY TABLE metric_values (
  id SERIAL,
  trial_id INT,
  name TEXT,
  value TEXT,
  type TEXT,
  end_time timestamptz
);

CREATE TEMPORARY TABLE numeric_aggs (
  id SERIAL,
  trial_id INT,
  name TEXT,
  count INT,
  sum FLOAT8,
  min FLOAT8,
  max FLOAT8
);

CREATE TEMPORARY TABLE metric_types (
  id SERIAL,
  trial_id INT,
  name TEXT,
  type TEXT
);

CREATE TEMPORARY TABLE metric_latest (
  id SERIAL,
  trial_id INT,
  name TEXT,
  value jsonb
);

CREATE TEMPORARY TABLE validation_summary_metrics (
  id SERIAL,
  trial_id INT,
  val_summary_metrics JSONB
);

-- Extract training metrics.
EXPLAIN ANALYZE INSERT INTO metric_values(trial_id, name, value, type, end_time)
SELECT
    trial_id AS trial_id,
    key AS name,
    CASE value
        WHEN '"NaN"' THEN 'NaN'
        WHEN '"Infinity"' THEN 'Infinity'
        WHEN '"-Infinity"' THEN '-Infinity'
        ELSE value::text
    END AS value,
    CASE
        WHEN jsonb_typeof(value) = 'string' THEN
            CASE
                WHEN value::text = '"Infinity"'::text THEN 'number'
                WHEN value::text = '"-Infinity"'::text THEN 'number'
                WHEN value::text = '"NaN"'::text THEN 'number'
                WHEN value::text ~
                    '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$' THEN 'date'
                ELSE 'string'
            END
        ELSE jsonb_typeof(value)::text
    END AS type,
    end_time AS end_time
FROM (
    SELECT trial_id, (jsonb_each(metrics->'validation_metrics')).key, (jsonb_each(metrics->'validation_metrics')).value, end_time
    FROM validations
) AS subquery;

-- Numeric aggregates.
EXPLAIN ANALYZE INSERT INTO numeric_aggs(trial_id, name, count, sum, min, max)
SELECT
    trial_id AS trial_id,
    name AS name,
    COUNT(*) AS count,
    safe_sum(value::double precision) AS sum,
    MIN(value::double precision) AS min,
    MAX(value::double precision) AS max
FROM metric_values
WHERE type = 'number'
GROUP BY trial_id, name;

-- Types.
EXPLAIN ANALYZE INSERT INTO metric_types(trial_id, name, type)
SELECT
    trial_id AS trial_id,
    name AS name,
    CASE
        WHEN COUNT(DISTINCT type) = 1 THEN MAX(type)
        ELSE 'string'
    END AS type
FROM metric_values
GROUP BY trial_id, name;

-- Latest.
EXPLAIN ANALYZE INSERT INTO metric_latest(trial_id, name, value)
SELECT
    s.trial_id AS trial_id,
    unpacked.key as name,
    unpacked.value as value
FROM (
    SELECT s.*,
        ROW_NUMBER() OVER(
            PARTITION BY s.trial_id
            ORDER BY s.end_time DESC
        ) as rank
    FROM validations s
    JOIN trials ON s.trial_id = trials.id
) s, jsonb_each(s.metrics->'validation_metrics') unpacked
WHERE s.rank = 1;

-- Summary metrics.
EXPLAIN ANALYZE INSERT INTO validation_summary_metrics(trial_id, val_summary_metrics)
SELECT
    trial_id, jsonb_collect(jsonb_build_object(
        name, jsonb_build_object(
        'count', sub.count,
        'sum', sub.sum,
        'min', CASE WHEN sub.max = 'NaN'::double precision
            THEN 'NaN'::double precision ELSE sub.min END,
        'max', sub.max,
        'last', sub.latest,
        'type', sub.type
    )
)) as val_summary_metrics
FROM (SELECT
    numeric_aggs.trial_id,
    numeric_aggs.name,
    count,
    sum,
    min,
    max,
    metric_types.type AS type,
    metric_latest.value AS latest
FROM numeric_aggs
LEFT JOIN metric_types ON
     numeric_aggs.trial_id = metric_types.trial_id AND
     numeric_aggs.name = metric_types.name
LEFT JOIN metric_latest ON
     numeric_aggs.trial_id = metric_latest.trial_id AND
     numeric_aggs.name = metric_latest.name) sub
GROUP BY trial_id;

ROLLBACK;
