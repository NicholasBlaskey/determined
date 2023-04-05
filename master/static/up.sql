-- We need to declare start_timestamp before this migration starts
-- to avoid missing metrics reported while this migration runs in background cases.
DO $$ DECLARE start_timestamp timestamptz := NOW(); BEGIN

ALTER TABLE trials
    ADD COLUMN IF NOT EXISTS summary_metrics jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS summary_metrics_timestamp timestamptz DEFAULT NULL;

-- Invalidate summary_metrics_timestamp for trials that have a metric added since.


-- 250,000,000 rows in steps => 155 seconds?
WITH max_training AS (
     SELECT trial_id, max(steps.end_time) AS last_reported_metric FROM steps
     JOIN trials ON trials.id = trial_id          -- TODO Does this join optimizie?
     WHERE summary_metrics_timestamp IS NOT NULL
     GROUP BY trial_id
)
UPDATE trials SET summary_metrics_timestamp = NULL FROM max_training WHERE
    max_training.trial_id = trials.id AND
    summary_metrics_timestamp IS NOT NULL AND
    last_reported_metric > summary_metrics_timestamp;

-- 2,500,000 rows in validation => 1 second
WITH max_validation AS (
     SELECT trial_id, max(validations.end_time) AS last_reported_metric FROM validations
     JOIN trials ON trials.id = trial_id          -- TODO Does this join optimizie?
     WHERE summary_metrics_timestamp IS NOT NULL
     GROUP BY trial_id
)
UPDATE trials SET summary_metrics_timestamp = NULL FROM max_validation WHERE
    max_validation.trial_id = trials.id AND
    summary_metrics_timestamp IS NOT NULL AND
    last_reported_metric > summary_metrics_timestamp;

-- 7.25 ms with all metrics incrementally done.
-- trials needing summary metrics calculated
--   UPDATE trials SET summary_metrics_timestamp = null WHERE id >= 10000;
--   1 => .8s, 1.4s
--   UPDATE trials SET summary_metrics_timestamp = null WHERE id >= 9990;
--   10 => 8.7s, 13.8s
--   UPDATE trials SET summary_metrics_timestamp = null WHERE id >= 9900;
--   100 => 109.9s, 180.8s
--   UPDATE trials SET summary_metrics_timestamp = null WHERE id >= 9000;
--   1000 =>  2249.3s, 2362.7s
--   UPDATE trials SET summary_metrics_timestamp = null;
--   10000 (all) => 166.667 minutes
--   

EXPLAIN ANALYZE WITH training_trial_metrics as (
SELECT
	name,
	trial_id,
	sum(entries) FILTER (WHERE metric_type != 'number') as nonumbers
FROM (
	SELECT
	name,
    CASE
        WHEN (metrics->'avg_metrics'->name)::text = '"Infinity"'::text THEN 'number'
        WHEN (metrics->'avg_metrics'->name)::text = '"-Infinity"'::text THEN 'number'
        WHEN (metrics->'avg_metrics'->name)::text = '"NaN"'::text THEN 'number'
        ELSE jsonb_typeof(metrics->'avg_metrics'->name)
    END AS metric_type,
	trial_id,
	count(1) as entries
	FROM (
		SELECT DISTINCT
		jsonb_object_keys(s.metrics->'avg_metrics') as name
		FROM steps s
        JOIN trials ON s.trial_id = trials.id
        WHERE trials.summary_metrics_timestamp IS NULL
	) names, steps
    JOIN trials ON trial_id = trials.id
    WHERE trials.summary_metrics_timestamp IS NULL
	GROUP BY name, metric_type, trial_id
) typed
where metric_type is not null
GROUP BY name, trial_id
ORDER BY trial_id, name
),
training_numeric_trial_metrics as (
SELECT name, trial_id
FROM training_trial_metrics
WHERE nonumbers IS null
),
training_trial_metric_aggs as (
SELECT
	name,
	ntm.trial_id,
	count(1) as count_agg,
	sum((steps.metrics->'avg_metrics'->>name)::double precision) as sum_agg,
	min((steps.metrics->'avg_metrics'->>name)::double precision) as min_agg,
	max((steps.metrics->'avg_metrics'->>name)::double precision) as max_agg
FROM training_numeric_trial_metrics ntm INNER JOIN steps
ON steps.trial_id=ntm.trial_id
WHERE steps.metrics->'avg_metrics'->name IS NOT NULL
GROUP BY 1, 2
UNION
SELECT
    name,
    trial_id,
    NULL AS count_agg,
    NULL AS sum,
    NULL AS min,
    NULL AS max
FROM training_trial_metrics
WHERE nonumbers IS NOT null
),
latest_training AS (
  SELECT s.trial_id,
	unpacked.key as name,
	unpacked.value as latest_value
  FROM (
      SELECT s.*,
        ROW_NUMBER() OVER(
          PARTITION BY s.trial_id
          ORDER BY s.end_time DESC
        ) AS rank
      FROM steps s
      JOIN trials ON s.trial_id = trials.id
      WHERE trials.summary_metrics_timestamp IS NULL
    ) s, jsonb_each(s.metrics->'avg_metrics') unpacked
  WHERE s.rank = 1
),
training_combined_latest_agg as (SELECT
 coalesce(lt.trial_id, tma.trial_id) as trial_id,
 coalesce(lt.name, tma.name) as name,
 tma.count_agg,
 tma.sum_agg,
 tma.min_agg,
 tma.max_agg,
 lt.latest_value
FROM latest_training lt FULL OUTER JOIN training_trial_metric_aggs tma ON
	lt.trial_id = tma.trial_id AND lt.name = tma.name
),
training_trial_metrics_final as (
	SELECT
		trial_id, jsonb_collect(jsonb_build_object(
			name, jsonb_build_object(
				'count', count_agg,
				'sum', sum_agg,
				'min', min_agg,
				'max', max_agg,
				'last', latest_value
			)
		)) as training_metrics
	FROM training_combined_latest_agg
	GROUP BY trial_id
),
validation_trial_metrics as (
SELECT
	name,
	trial_id,
	sum(entries) FILTER (WHERE metric_type != 'number') as nonumbers
FROM (
	SELECT
	name,
    CASE
        WHEN (metrics->'validation_metrics'->name)::text = '"Infinity"'::text THEN 'number'
        WHEN (metrics->'validation_metrics'->name)::text = '"-Infinity"'::text THEN 'number'
        WHEN (metrics->'validation_metrics'->name)::text = '"NaN"'::text THEN 'number'
        ELSE jsonb_typeof(metrics->'validation_metrics'->name)
    END AS metric_type,
	trial_id,
	count(1) as entries
	FROM (
		SELECT DISTINCT
		jsonb_object_keys(s.metrics->'validation_metrics') as name
		FROM validations s
        JOIN trials ON s.trial_id = trials.id
        WHERE trials.summary_metrics_timestamp IS NULL
	) names, validations
    JOIN trials ON trial_id = trials.id
    WHERE trials.summary_metrics_timestamp IS NULL
	GROUP BY name, metric_type, trial_id
) typed
where metric_type is not null
GROUP BY name, trial_id
ORDER BY trial_id, name
),
validation_numeric_trial_metrics as (
SELECT name, trial_id
FROM validation_trial_metrics
WHERE nonumbers IS null
),
validation_trial_metric_aggs as (
SELECT
	name,
	ntm.trial_id,
	count(1) as count_agg,
	sum((validations.metrics->'validation_metrics'->>name)::double precision) as sum_agg,
	min((validations.metrics->'validation_metrics'->>name)::double precision) as min_agg,
	max((validations.metrics->'validation_metrics'->>name)::double precision) as max_agg
FROM validation_numeric_trial_metrics ntm INNER JOIN validations
ON validations.trial_id=ntm.trial_id
WHERE validations.metrics->'validation_metrics'->name IS NOT NULL
GROUP BY 1, 2
UNION
SELECT
    name,
    trial_id,
    NULL AS count_agg,
    NULL AS sum,
    NULL AS min,
    NULL AS max
FROM validation_trial_metrics
WHERE nonumbers IS NOT null
),
latest_validation AS (
  SELECT s.trial_id,
	unpacked.key as name,
	unpacked.value as latest_value
  FROM (
      SELECT s.*,
        ROW_NUMBER() OVER(
          PARTITION BY s.trial_id
          ORDER BY s.end_time DESC
        ) AS rank
      FROM validations s
      JOIN trials ON s.trial_id = trials.id
      WHERE trials.summary_metrics_timestamp IS NULL
    ) s, jsonb_each(s.metrics->'validation_metrics') unpacked
  WHERE s.rank = 1
),
validation_combined_latest_agg as (SELECT
 coalesce(lt.trial_id, tma.trial_id) as trial_id,
 coalesce(lt.name, tma.name) as name,
 tma.count_agg,
 tma.sum_agg,
 tma.min_agg,
 tma.max_agg,
 lt.latest_value
FROM latest_validation lt FULL OUTER JOIN validation_trial_metric_aggs tma ON
	lt.trial_id = tma.trial_id AND lt.name = tma.name
),
validation_trial_metrics_final as (
	SELECT
		trial_id, jsonb_collect(jsonb_build_object(
			name, jsonb_build_object(
				'count', count_agg,
				'sum', sum_agg,
				'min', min_agg,
				'max', max_agg,
				'last', latest_value
			)
		)) as validation_metrics
	FROM validation_combined_latest_agg
	GROUP BY trial_id
),
validation_training_combined_json as (
	SELECT
	coalesce(ttm.trial_id, vtm.trial_id) as trial_id,
    (CASE
        WHEN ttm.training_metrics IS NOT NULL AND vtm.validation_metrics IS NOT NULL THEN
            jsonb_build_object(
                'avg_metrics', ttm.training_metrics, 
                'validation_metrics', vtm.validation_metrics
            ) 
        WHEN ttm.training_metrics IS NOT NULL THEN
            jsonb_build_object(
                'avg_metrics', ttm.training_metrics
            )
        WHEN vtm.validation_metrics IS NOT NULL THEN jsonb_build_object(
                'validation_metrics', vtm.validation_metrics
           )
        ELSE '{}'::jsonb END) AS summary_metrics
	FROM training_trial_metrics_final ttm FULL OUTER JOIN validation_trial_metrics_final vtm
	ON ttm.trial_id = vtm.trial_id
)
UPDATE trials SET
    summary_metrics = vtcj.summary_metrics
FROM validation_training_combined_json vtcj WHERE vtcj.trial_id = trials.id;

UPDATE trials SET
    summary_metrics_timestamp = start_timestamp;

END$$;
