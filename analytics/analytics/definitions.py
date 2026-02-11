from dagster import Definitions, ScheduleDefinition, define_asset_job

from .assets import snapshot_sensor, snapshot_summary

snapshot_job = define_asset_job("snapshot_job", selection=["snapshot_summary"])

snapshot_schedule = ScheduleDefinition(
	job=snapshot_job,
	cron_schedule="0 2 * * *",
)

definitions = Definitions(
	assets=[snapshot_summary],
	schedules=[snapshot_schedule],
	sensors=[snapshot_sensor],
)
