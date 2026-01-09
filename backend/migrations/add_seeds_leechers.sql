-- Migration: Add seeds and leechers columns to matches table
-- Date: 2026-01-09
-- Description: Adds seeds and leechers columns to support torrent metadata

-- Add seeds column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'matches' AND column_name = 'seeds'
    ) THEN
        ALTER TABLE matches ADD COLUMN seeds VARCHAR(20);
        RAISE NOTICE 'Added seeds column to matches table';
    ELSE
        RAISE NOTICE 'seeds column already exists';
    END IF;
END $$;

-- Add leechers column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'matches' AND column_name = 'leechers'
    ) THEN
        ALTER TABLE matches ADD COLUMN leechers VARCHAR(20);
        RAISE NOTICE 'Added leechers column to matches table';
    ELSE
        RAISE NOTICE 'leechers column already exists';
    END IF;
END $$;

-- Verify the columns were added
SELECT column_name, data_type, character_maximum_length 
FROM information_schema.columns 
WHERE table_name = 'matches' 
AND column_name IN ('seeds', 'leechers')
ORDER BY column_name;
