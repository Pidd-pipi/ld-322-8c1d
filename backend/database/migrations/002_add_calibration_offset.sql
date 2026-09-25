-- 002_add_calibration_offset: 传感器校准偏移列。
-- Schema 仍由 GORM AutoMigrate 在应用启动时管理，此文件仅记录迁移边界供 DBA 参考。
-- 已有传感器的 calibration_offset 默认为 0，即保持原始读数不变。
ALTER TABLE sensors ADD COLUMN calibration_offset DOUBLE NOT NULL DEFAULT 0;
