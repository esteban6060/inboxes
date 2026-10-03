-- +goose Up
ALTER TABLE aliases
  ADD COLUMN photo_data BYTEA,
  ADD COLUMN photo_version UUID,
  ADD CONSTRAINT aliases_photo_pair CHECK ((photo_data IS NULL) = (photo_version IS NULL));

-- +goose Down
ALTER TABLE aliases DROP CONSTRAINT aliases_photo_pair, DROP COLUMN photo_version, DROP COLUMN photo_data;
