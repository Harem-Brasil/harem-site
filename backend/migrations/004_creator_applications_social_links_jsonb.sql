-- social_links: text[] -> jsonb (mesmo formato que a API JSON; evita confusão em clientes .NET/gestores)

ALTER TABLE IF EXISTS creator_applications
    ALTER COLUMN social_links TYPE jsonb
    USING (
        CASE pg_typeof(social_links)::text
            WHEN 'text[]' THEN to_jsonb(social_links)
            ELSE social_links::jsonb
        END
    );

ALTER TABLE creator_applications
    ALTER COLUMN social_links SET DEFAULT '[]'::jsonb;
