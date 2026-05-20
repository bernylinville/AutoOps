// Package model provides database migration helpers for inspection tables.
package model

// InspectionMigrationSQL returns the SQL statements for inspection table migration.
// These are run after AutoMigrate to create constraints that GORM cannot express.
func InspectionMigrationSQL() []string {
	return []string{
		// 防 cron 任务重复运行：同一任务同一天只能有一条 cron 记录。
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_run_unique_daily
		 ON inspection_run (task_id, run_date)
		 WHERE trigger_type = 'cron'`,

		// 级联删除约束：删除 run 时自动清理子记录（兜底，业务层已通过 CleanupService 处理）。
		`DO $$ BEGIN
		 ALTER TABLE inspection_target_result
		   ADD CONSTRAINT fk_result_run FOREIGN KEY (run_id) REFERENCES inspection_run(id) ON DELETE CASCADE;
		 EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
		 ALTER TABLE inspection_alert
		   ADD CONSTRAINT fk_alert_run FOREIGN KEY (run_id) REFERENCES inspection_run(id) ON DELETE CASCADE;
		 EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
		 ALTER TABLE inspection_report_artifact
		   ADD CONSTRAINT fk_report_run FOREIGN KEY (run_id) REFERENCES inspection_run(id) ON DELETE CASCADE;
		 EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
		 ALTER TABLE inspection_notification
		   ADD CONSTRAINT fk_notification_run FOREIGN KEY (run_id) REFERENCES inspection_run(id) ON DELETE CASCADE;
		 EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	}
}
