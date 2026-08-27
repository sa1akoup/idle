-- V1.3 藏身处离散作业的通用目标、输入和产出字段。
ALTER TABLE facility_jobs ADD COLUMN target_ref TEXT NOT NULL DEFAULT '';
ALTER TABLE facility_jobs ADD COLUMN payload_json TEXT NOT NULL DEFAULT '';
ALTER TABLE facility_jobs ADD COLUMN result_json TEXT NOT NULL DEFAULT '';
