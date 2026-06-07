-- Initialize FolioTrack Database schema
CREATE DATABASE IF NOT EXISTS `foliotrack` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `foliotrack`;

-- Optional: Initial SQL comments or indexing optimization
-- GORM in Go backend will automatically perform AutoMigrate to create schemas for:
-- `users`, `assets`, `holdings`, `combos`, and `holding_combos`.
