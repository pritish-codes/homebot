-- +goose Up
ALTER TABLE maintenance_entries ADD COLUMN is_recurring BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE maintenance_entries ADD COLUMN recurrence_interval_months INTEGER NULL;
