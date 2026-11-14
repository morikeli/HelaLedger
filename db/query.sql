-- name: GetWalletForUpdate :one
SELECT id, owner_id, currency, balance, created_at, updated_at
FROM wallets WHERE id=$1 FOR UPDATE;

-- name: GetWalletByID :one
SELECT id, owner_id, currency, balance, created_at, updated_at
FROM wallets WHERE id=$1;

-- name: ListPostingsByWalletID :many
SELECT id, transaction_id, wallet_id, amount, direction, created_at
FROM postings WHERE wallet_id=$1 ORDER BY created_at DESC, id DESC LIMIT $2;

-- name: ReconcileWalletBalances :many
SELECT w.id, w.balance,
       COALESCE(SUM(CASE WHEN p.direction='CREDIT' THEN p.amount ELSE -p.amount END), 0)::BIGINT AS ledger_balance
FROM wallets w 
LEFT JOIN postings p ON p.wallet_id = w.id
GROUP BY w.id, w.balance
HAVING w.balance <> COALESCE(SUM(CASE WHEN p.direction='CREDIT' THEN p.amount ELSE -p.amount END), 0);

-- Run periodically using a scheduled maintenance job. Keep idempotency keys
-- for at least the documented client retry window.
-- name: DeleteExpiredIdempotencyKeys :exec
DELETE FROM idempotency_keys WHERE expires_at < NOW() AND status <> 'PROCESSING';

-- name: ClaimIdempotencyKey :execrows
INSERT INTO idempotency_keys (
    idempotency_key,
    payload_hash,
    status,
    created_at,
    updated_at,
    expires_at
)
VALUES (
    $1,
    $2,
    'PROCESSING',
    NOW(),
    NOW(),
    NOW() + INTERVAL '24 hours'
)
ON CONFLICT (idempotency_key)
DO UPDATE SET
    payload_hash = EXCLUDED.payload_hash,
    status = 'PROCESSING',
    response_status = NULL,
    response_body = NULL,
    created_at = NOW(),
    updated_at = NOW(),
    expires_at = NOW() + INTERVAL '24 hours'
WHERE idempotency_keys.expires_at <= NOW();

-- name: GetIdempotencyKey :one
SELECT payload_hash, status, response_status, response_body
FROM idempotency_keys WHERE idempotency_key = $1;

-- name: CompleteIdempotencyKey :exec
UPDATE idempotency_keys
SET status = 'COMPLETED', response_status = $1, response_body = $2, updated_at = NOW()
WHERE idempotency_key = $3;

-- name: LockTransferWallets :many
SELECT id::text, currency, balance, status
FROM wallets WHERE id IN ($1::uuid, $2::uuid) ORDER BY id FOR UPDATE;

-- name: CreateTransferTransaction :one
INSERT INTO transactions (reference_id, transaction_type, amount, currency, status, completed_at)
VALUES ($1, 'WALLET_TRANSFER', $2, $3, 'COMPLETED', NOW()) RETURNING id;

-- name: DebitWallet :execrows
UPDATE wallets SET balance = balance - $1, updated_at = NOW()
WHERE id = $2 AND balance >= $1;

-- name: CreditWallet :execrows
UPDATE wallets SET balance = balance + $1, updated_at = NOW() 
WHERE id = $2;

-- name: CreateTransferPostings :exec
INSERT INTO postings (transaction_id, wallet_id, amount, direction)
VALUES ($1, $2, $3, 'DEBIT'), ($1, $4, $3, 'CREDIT');
