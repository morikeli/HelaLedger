-- Wallet 1 Opening Balance & Transaction
INSERT INTO transactions (id, reference_id, transaction_type, amount, currency, status, completed_at)
VALUES ('a1111111-1111-1111-1111-111111111111', 'REF-OPENING-1', 'WALLET_TRANSFER', 50000, 'KES', 'COMPLETED', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO wallets (id, owner_id, currency, balance, status) 
VALUES ('be5c1322-a240-4d86-9b52-fa3e1d63c33c', gen_random_uuid(), 'KES', 50000, 'ACTIVE')
ON CONFLICT (id) DO NOTHING;

INSERT INTO postings (id, transaction_id, wallet_id, amount, direction)
VALUES (gen_random_uuid(), 'a1111111-1111-1111-1111-111111111111', 'be5c1322-a240-4d86-9b52-fa3e1d63c33c', 50000, 'CREDIT')
ON CONFLICT DO NOTHING;

-- Wallet 2 Opening Balance & Transaction
INSERT INTO transactions (id, reference_id, transaction_type, amount, currency, status, completed_at)
VALUES ('a2222222-2222-2222-2222-222222222222', 'REF-OPENING-2', 'WALLET_TRANSFER', 10000, 'KES', 'COMPLETED', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO wallets (id, owner_id, currency, balance, status) 
VALUES ('f67568a2-5f0f-4fbb-93f2-c8918e56ff9c', gen_random_uuid(), 'KES', 10000, 'ACTIVE')
ON CONFLICT (id) DO NOTHING;

INSERT INTO postings (id, transaction_id, wallet_id, amount, direction)
VALUES (gen_random_uuid(), 'a2222222-2222-2222-2222-222222222222', 'f67568a2-5f0f-4fbb-93f2-c8918e56ff9c', 10000, 'CREDIT')
ON CONFLICT DO NOTHING;

-- Wallet 3 Opening Balance & Transaction
INSERT INTO transactions (id, reference_id, transaction_type, amount, currency, status, completed_at)
VALUES ('a3333333-3333-3333-3333-333333333333', 'REF-OPENING-3', 'WALLET_TRANSFER', 500, 'KES', 'COMPLETED', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO wallets (id, owner_id, currency, balance, status) 
VALUES ('72a30e9a-5968-43e3-8ba3-8b0a72620276', gen_random_uuid(), 'KES', 500, 'ACTIVE')
ON CONFLICT (id) DO NOTHING;

INSERT INTO postings (id, transaction_id, wallet_id, amount, direction)
VALUES (gen_random_uuid(), 'a3333333-3333-3333-3333-333333333333', '72a30e9a-5968-43e3-8ba3-8b0a72620276', 500, 'CREDIT')
ON CONFLICT DO NOTHING;

-- Wallet 4 Opening Balance & Transaction
INSERT INTO transactions (id, reference_id, transaction_type, amount, currency, status, completed_at)
VALUES ('a4444444-4444-4444-4444-444444444444', 'REF-OPENING-4', 'WALLET_TRANSFER', 245, 'KES', 'COMPLETED', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO wallets (id, owner_id, currency, balance, status) 
VALUES ('821ccdb2-ceae-4cc8-bfd6-c326cd5877b7', gen_random_uuid(), 'KES', 245, 'SUSPENDED')
ON CONFLICT (id) DO NOTHING;

INSERT INTO postings (id, transaction_id, wallet_id, amount, direction)
VALUES (gen_random_uuid(), 'a4444444-4444-4444-4444-444444444444', '821ccdb2-ceae-4cc8-bfd6-c326cd5877b7', 245, 'CREDIT')
ON CONFLICT DO NOTHING;

-- Wallet 5 Opening Balance & Transaction
INSERT INTO transactions (id, reference_id, transaction_type, amount, currency, status, completed_at)
VALUES ('a5555555-5555-5555-5555-555555555555', 'REF-OPENING-5', 'WALLET_TRANSFER', 500, 'KES', 'COMPLETED', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO wallets (id, owner_id, currency, balance, status) 
VALUES ('1bcac58a-9fad-444a-a4fb-b8002407d6ff', gen_random_uuid(), 'KES', 500, 'SUSPENDED')
ON CONFLICT (id) DO NOTHING;

INSERT INTO postings (id, transaction_id, wallet_id, amount, direction)
VALUES (gen_random_uuid(), 'a5555555-5555-5555-5555-555555555555', '1bcac58a-9fad-444a-a4fb-b8002407d6ff', 500, 'CREDIT')
ON CONFLICT DO NOTHING;

-- Wallet 6 Opening Balance & Transaction
INSERT INTO transactions (id, reference_id, transaction_type, amount, currency, status, completed_at)
VALUES ('a6666666-6666-6666-6666-666666666666', 'REF-OPENING-6', 'WALLET_TRANSFER', 7500, 'KES', 'COMPLETED', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO wallets (id, owner_id, currency, balance, status) 
VALUES ('33f3a2c2-5e3d-4267-bfce-258cab8e3a67', gen_random_uuid(), 'KES', 7500, 'CLOSED')
ON CONFLICT (id) DO NOTHING;

INSERT INTO postings (id, transaction_id, wallet_id, amount, direction)
VALUES (gen_random_uuid(), 'a6666666-6666-6666-6666-666666666666', '33f3a2c2-5e3d-4267-bfce-258cab8e3a67', 7500, 'CREDIT')
ON CONFLICT DO NOTHING;


-- Wallet 7 (Zero balance - optional transaction, or skip posting if balance is 0)
INSERT INTO wallets (id, owner_id, currency, balance, status) 
VALUES ('3e91d03a-1e8a-4caf-9160-c591c25402ea', gen_random_uuid(), 'KES', 0, 'CLOSED')
ON CONFLICT (id) DO NOTHING;