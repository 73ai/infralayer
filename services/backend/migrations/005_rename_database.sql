-- Migration: Rename database and user from infragpt to infralayer
-- This migration should be run MANUALLY in production.
-- DO NOT run this automatically.

-- Step 1: Rename the database
-- NOTE: Must be run while NOT connected to the infragpt database.
-- Connect to 'postgres' database first, then run:
-- ALTER DATABASE infragpt RENAME TO infralayer;

-- Step 2: Rename the database user
-- ALTER USER infragpt RENAME TO infralayer;

-- Step 3: Update the user password
-- ALTER USER infralayer WITH PASSWORD 'infralayerisgreatsre';
