-- Add description column to alert_rules table if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'alert_rules' 
        AND column_name = 'description'
    ) THEN
        ALTER TABLE alert_rules ADD COLUMN description VARCHAR(500);
    END IF;
END $$;