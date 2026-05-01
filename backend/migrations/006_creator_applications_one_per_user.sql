-- Uma candidatura de criador por usuário (evita duplicatas e reenvios).
-- Remove linhas duplicadas antigas, mantendo a inscrição mais antiga por user_id.
DELETE FROM creator_applications ca
WHERE ca.id NOT IN (
    SELECT DISTINCT ON (user_id) id
    FROM creator_applications
    ORDER BY user_id, submitted_at ASC
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_creator_applications_user_id_unique
    ON creator_applications (user_id);
