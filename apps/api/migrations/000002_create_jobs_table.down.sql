-- Rollback da Migration 000002: Remoção da tabela jobs e seus índices associados
DROP TABLE IF EXISTS jobs;
