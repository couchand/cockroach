select statement_key, application_name, distributed, optimized, has_error, sum(automatic_retry_count) retries, max(automatic_retry_count) max_retries, count(id) as first_attempt_count, avg(rows_affected) as rows_affected, stddev(rows_affected) as rows_affected_stddev, avg(parse_lat::DECIMAL)::INTERVAL as parse_lat, stddev(parse_lat::DECIMAL)::INTERVAL as parse_lat_stddev, avg(plan_lat::DECIMAL)::INTERVAL as plan_lat, stddev(plan_lat::DECIMAL)::INTERVAL as plan_lat_stddev, avg(run_lat::DECIMAL)::INTERVAL as run_lat, stddev(run_lat::DECIMAL)::INTERVAL as run_lat_stddev, avg(service_lat::DECIMAL - parse_lat::DECIMAL - plan_lat::DECIMAL - run_lat::DECIMAL)::INTERVAL as overhead_lat, stddev(service_lat::DECIMAL - parse_lat::DECIMAL - plan_lat::DECIMAL - run_lat::DECIMAL)::INTERVAL as overhead_lat_stddev, avg(service_lat::DECIMAL)::INTERVAL as service_lat, stddev(service_lat::DECIMAL)::INTERVAL as service_lat_stddev from (select *, (CASE error WHEN '' THEN false ELSE true END) as has_error from system.statement_executions) group by statement_key, application_name, distributed, optimized, has_error order by service_lat;
