-- We need to declare start_timestamp to avoid missing metrics
-- happening while this migration runs.
DO $$ DECLARE start_timestamp timestamptz := NOW(); BEGIN

ALTER TABLE trials
    ADD COLUMN IF NOT EXISTS summary_metrics jsonb DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS summary_metrics_timestamp timestamptz DEFAULT NULL;


-- Invalidate summary_metrics_timestamp for trials that have a metric added since.

-- TODO test this aspect of the migration...
-- AND time it
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

--    end_time FROM STEPS
--    end_time FROM validations

-- training_trial_metrics
-- gets every trial_id and metric name pair with a count of non numeric metrics 
--  name  | trial_id | nonumbers 
-- -------+----------+-----------
--  loss  |        1 |          
--  loss  |        2 |          
--  loss2 |        2 |          
--  loss3 |        2 |          
--  loss  |        3 |          
--  loss2 |        3 |          
--  loss3 |        3 |          
--  loss  |        4 |          
--  loss2 |        4 |          
--  loss3 |        4 |          
--  loss  |        5 |          
--  loss2 |        5 |          
--  loss3 |        5 |          

-- training_numeric_trial_metrics
-- same as above but we drop rows with nonumbers
--  name  | trial_id | nonumbers 
-- -------+----------+-----------
--  loss  |        1 |          
--  loss  |        2 |          
--  loss2 |        2 |          
--  loss3 |        2 |          
--  loss  |        3 |          
--  loss2 |        3 |          
--  loss3 |        3 |          
--  loss  |        4 |          
--  loss2 |        4 |          
--  loss3 |        4 |          
--  loss  |        5 |          
--  loss2 |        5 |          
--  loss3 |        5 |          

-- training_trial_metric_aggs
-- computes count, sum, min, max of metrics / trial_id pairs
--  name  | trial_id | count_agg |       sum_agg       |        min_agg        |       max_agg       
-- -------+----------+-----------+---------------------+-----------------------+---------------------
--  loss  |        4 |       100 | 0.00311207065401935 |                     0 | 0.00310020474647165
--  loss  |        3 |       100 | 0.00133352620233214 | 2.49289751518464e-242 | 0.00133058893084141
--  loss2 |        4 |       100 |                   0 |                     0 |                   0
--  loss3 |        2 |       100 |   0.105468154486085 |                     0 |   0.101728458177556
--  loss3 |        3 |       100 | 0.00198129560316848 |                     0 | 0.00197523944970218
--  loss3 |        4 |       100 |  0.0116938120311594 |                     0 |  0.0116938120311594
--  loss  |        1 |        10 |    1.80937720835209 |    0.0793009176850319 |   0.584656655788422
--  loss2 |        5 |       100 |  0.0804982265233428 |                     0 |  0.0776193972233052
--  loss2 |        3 |       100 |                   0 |                     0 |                   0
--  loss  |        2 |       100 |  0.0357096241581005 | 6.32423761930295e-117 |  0.0308295342447146
--  loss  |        5 |       100 |                   0 |                     0 |                   0
--  loss2 |        2 |       100 | 0.00319892593160602 |                     0 | 0.00319892593160602
--  loss3 |        5 |       100 |                   0 |                     0 |                   0

-- latest_training
-- gets latest training value report for the metric / trial_ids pair
--  trial_id | name  | latest_value
-- ----------+-------+--------------------------------------------------------
--         1 | loss  | 0.07930091768503189
--         2 | loss  | 0.0000000000006324237619302949
--         2 | loss2 | 0
--         2 | loss3 | 0
--         3 | loss  | 0.000000000000000000000000000000000002492897515184636
--         3 | loss2 | 0
--         3 | loss3 | 0
--         4 | loss  | 0
--         4 | loss2 | 0
--         4 | loss3 | 0
--         5 | loss  | 0
--         5 | loss2 | 0
--         5 | loss3 | 0

-- training_combined_latest_agg
-- squishes training aggs + metrics together
--  trial_id | name  | count_agg |       sum_agg       |        min_agg        |       max_agg       |                                                                                                                            latest_value                                   
                                                                                          
-- ----------+-------+-----------+---------------------+-----------------------+---------------------+---------------------------------------------------------------------------------------------------------------------------------------------------------------------------
-- ------------------------------------------------------------------------------------------
--         1 | loss  |        10 |    1.80937720835209 |    0.0793009176850319 |   0.584656655788422 | 0.07930091768503189
--         2 | loss  |       100 |  0.0357096241581005 | 6.32423761930295e-117 |  0.0308295342447146 | 0.6324237619302949
--         2 | loss2 |       100 | 0.00319892593160602 |                     0 | 0.00319892593160602 | 0
--         2 | loss3 |       100 |   0.105468154486085 |                     0 |   0.101728458177556 | 0
--         3 | loss  |       100 | 0.00133352620233214 | 2.49289751518464e-242 | 0.00133058893084141 | 0.00000000000000000002492897515184636
--         3 | loss2 |       100 |                   0 |                     0 |                   0 | 0
--         3 | loss3 |       100 | 0.00198129560316848 |                     0 | 0.00197523944970218 | 0
--         4 | loss  |       100 | 0.00311207065401935 |                     0 | 0.00310020474647165 | 0
--         4 | loss2 |       100 |                   0 |                     0 |                   0 | 0
--         4 | loss3 |       100 |  0.0116938120311594 |                     0 |  0.0116938120311594 | 0
--         5 | loss  |       100 |                   0 |                     0 |                   0 | 0
--         5 | loss2 |       100 |  0.0804982265233428 |                     0 |  0.0776193972233052 | 0
--         5 | loss3 |       100 |                   0 |                     0 |                   0 | 0

-- training_trial_metrics_final
-- turns trial training metrics into jsonb
 -- trial_id |                                                                                                                                                                                                                                                                   
 --                                                                                                                              training_metrics                                                                                                                                
                                                                                                                                                                                                                                                                 
----------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
--         3 | {"loss": {"max": 0.00133058893084141, "min": 0.0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
-- 000000000000000000000000000000249289751518464, "sum": 0.00133352620233214, "last": 0.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
-- 000000000000000000000000000000000000000000000000000000002492897515184636, "count": 100}, "loss2": {"max": 0, "min": 0, "sum": 0, "last": 0, "count": 100}, "loss3": {"max": 0.00197523944970218, "min": 0, "sum": 0.00198129560316848, "last": 0, "count": 100}}
--         5 | {"loss": {"max": 0, "min": 0, "sum": 0, "last": 0, "count": 100}, "loss2": {"max": 0.0776193972233052, "min": 0, "sum": 0.0804982265233428, "last": 0, "count": 100}, "loss3": {"max": 0, "min": 0, "sum": 0, "last": 0, "count": 100}}
--         4 | {"loss": {"max": 0.00310020474647165, "min": 0, "sum": 0.00311207065401935, "last": 0, "count": 100}, "loss2": {"max": 0, "min": 0, "sum": 0, "last": 0, "count": 100}, "loss3": {"max": 0.0116938120311594, "min": 0, "sum": 0.0116938120311594, "last": 0, "coun
-- t": 100}}
--         2 | {"loss": {"max": 0.0308295342447146, "min": 0.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000632423761930295, "sum": 0.0357096241581005, "last": 0.000000000000000000000000000000000000000000
-- 000000000000000000000000000000000000000000000000000000000000000000000000006324237619302949, "count": 100}, "loss2": {"max": 0.00319892593160602, "min": 0, "sum": 0.00319892593160602, "last": 0, "count": 100}, "loss3": {"max": 0.101728458177556, "min": 0, "sum": 0.105468
-- 154486085, "last": 0, "count": 100}}
--         1 | {"loss": {"max": 0.584656655788422, "min": 0.0793009176850319, "sum": 1.80937720835209, "last": 0.07930091768503189, "count": 10}}




WITH training_trial_metrics as (
SELECT
	name,
	trial_id,
	sum(entries) FILTER (WHERE metric_type != 'number') as nonumbers
FROM (
	SELECT
	name,
	jsonb_typeof(steps.metrics->'avg_metrics'->name) as metric_type,
	trial_id,
	count(1) as entries
	FROM (
		SELECT DISTINCT
		jsonb_object_keys(s.metrics->'avg_metrics') as name
		FROM steps s
	) names, steps
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
	jsonb_typeof(validations.metrics->'validation_metrics'->name) as metric_type,
	trial_id,
	count(1) as entries
	FROM (
		SELECT DISTINCT
		jsonb_object_keys(s.metrics->'validation_metrics') as name
		FROM validations s
	) names, validations
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
    summary_metrics = vtcj.summary_metrics,
    summary_metrics_timestamp = start_timestamp
FROM validation_training_combined_json vtcj WHERE vtcj.trial_id = trials.id;

END$$;
