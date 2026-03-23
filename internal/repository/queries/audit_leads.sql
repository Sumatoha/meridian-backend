-- name: InsertAuditLead :one
INSERT INTO audit_leads (ig_username, ip_address, user_agent, locale, mock_score)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CountAuditLeads :one
SELECT COUNT(*) FROM audit_leads;

-- name: CountUniqueAuditLeads :one
SELECT COUNT(DISTINCT ig_username) FROM audit_leads;

-- name: ListRecentAuditLeads :many
SELECT * FROM audit_leads
ORDER BY created_at DESC
LIMIT $1;
