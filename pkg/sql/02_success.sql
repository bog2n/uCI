ALTER TABLE logs ADD COLUMN success BOOLEAN;
UPDATE logs SET success = 1 WHERE data NOT LIKE  '%____/__/__ __:__:__ Error deploying: %';
UPDATE logs SET success = 0 WHERE data LIKE '%____/__/__ __:__:__ Error deploying: %';
