-- social_links: text[] -> jsonb (mesmo formato que a API JSON; evita confusão em clientes .NET/gestores)
-- to_jsonb converte text[] para JSON array e jsonb permanece inalterado.

ALTER TABLE IF EXISTS creator_applications
    ALTER COLUMN social_links TYPE jsonb
    USING to_jsonb(social_links);

ALTER TABLE creator_applications
    ALTER COLUMN social_links SET DEFAULT '[]'::jsonb;
