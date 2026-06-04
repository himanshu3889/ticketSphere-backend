-- (idempotent version)

-- Grant replication
DO $$
BEGIN
    ALTER USER ticketspehere WITH REPLICATION;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not alter user (might need superuser)';
END $$;

-- Permissions (idempotent by default)
GRANT CONNECT ON DATABASE ticketspehere TO ticketspehere;
GRANT USAGE ON SCHEMA public TO ticketspehere;
GRANT SELECT ON public.members TO ticketspehere;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO ticketspehere;

-- Replica identity (idempotent)
ALTER TABLE public.members REPLICA IDENTITY FULL;


-- Create publication only if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_publication WHERE pubname = 'ticketspehere_publication'
    ) THEN
        CREATE PUBLICATION ticketspehere_publication FOR TABLE public.members;
    END IF;
END $$;
