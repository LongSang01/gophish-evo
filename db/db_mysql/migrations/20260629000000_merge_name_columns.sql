
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Merge first_name and last_name into full_name for targets table
ALTER TABLE `targets` ADD COLUMN `full_name` varchar(255);
UPDATE `targets` SET `full_name` = TRIM(CONCAT(COALESCE(`first_name`, ''), ' ', COALESCE(`last_name`, '')));
ALTER TABLE `targets` DROP COLUMN `first_name`;
ALTER TABLE `targets` DROP COLUMN `last_name`;

-- Merge first_name and last_name into full_name for results table
ALTER TABLE `results` ADD COLUMN `full_name` varchar(255);
UPDATE `results` SET `full_name` = TRIM(CONCAT(COALESCE(`first_name`, ''), ' ', COALESCE(`last_name`, '')));
ALTER TABLE `results` DROP COLUMN `first_name`;
ALTER TABLE `results` DROP COLUMN `last_name`;

-- Merge first_name and last_name into full_name for email_requests table
ALTER TABLE `email_requests` ADD COLUMN `full_name` varchar(255);
UPDATE `email_requests` SET `full_name` = TRIM(CONCAT(COALESCE(`first_name`, ''), ' ', COALESCE(`last_name`, '')));
ALTER TABLE `email_requests` DROP COLUMN `first_name`;
ALTER TABLE `email_requests` DROP COLUMN `last_name`;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

-- Restore first_name and last_name columns for targets table
ALTER TABLE `targets` ADD COLUMN `first_name` varchar(255);
ALTER TABLE `targets` ADD COLUMN `last_name` varchar(255);
UPDATE `targets` SET `first_name` = SUBSTRING_INDEX(`full_name`, ' ', 1);
UPDATE `targets` SET `last_name` = CASE WHEN LOCATE(' ', `full_name`) > 0 THEN SUBSTRING(`full_name`, LOCATE(' ', `full_name`) + 1) ELSE '' END;
ALTER TABLE `targets` DROP COLUMN `full_name`;

-- Restore first_name and last_name columns for results table
ALTER TABLE `results` ADD COLUMN `first_name` varchar(255);
ALTER TABLE `results` ADD COLUMN `last_name` varchar(255);
UPDATE `results` SET `first_name` = SUBSTRING_INDEX(`full_name`, ' ', 1);
UPDATE `results` SET `last_name` = CASE WHEN LOCATE(' ', `full_name`) > 0 THEN SUBSTRING(`full_name`, LOCATE(' ', `full_name`) + 1) ELSE '' END;
ALTER TABLE `results` DROP COLUMN `full_name`;

-- Restore first_name and last_name columns for email_requests table
ALTER TABLE `email_requests` ADD COLUMN `first_name` varchar(255);
ALTER TABLE `email_requests` ADD COLUMN `last_name` varchar(255);
UPDATE `email_requests` SET `first_name` = SUBSTRING_INDEX(`full_name`, ' ', 1);
UPDATE `email_requests` SET `last_name` = CASE WHEN LOCATE(' ', `full_name`) > 0 THEN SUBSTRING(`full_name`, LOCATE(' ', `full_name`) + 1) ELSE '' END;
ALTER TABLE `email_requests` DROP COLUMN `full_name`;
