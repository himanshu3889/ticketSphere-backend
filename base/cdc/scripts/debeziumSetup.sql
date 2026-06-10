-- (idempotent version)

-- Grant replication
DO $$
BEGIN
    ALTER USER ticketsphere WITH REPLICATION;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not alter user (might need superuser)';
END $$;

-- Permissions (idempotent by default)
GRANT CONNECT ON DATABASE ticketsphere TO ticketsphere;
GRANT USAGE ON SCHEMA public TO ticketsphere;
GRANT SELECT ON public.members TO ticketsphere;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO ticketsphere;

-- Replica identity (idempotent)
ALTER TABLE public.members REPLICA IDENTITY FULL;


-- Create publication only if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_publication WHERE pubname = 'ticketsphere_publication'
    ) THEN
        CREATE PUBLICATION ticketsphere_publication FOR TABLE public.members;
    END IF;
END $$;
