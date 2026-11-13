-- db/seed.sql
INSERT INTO wallets (id, owner_id, currency, balance, status) 
VALUES 
('be5c1322-a240-4d86-9b52-fa3e1d63c33c', gen_random_uuid(), 'KES', 50000, 'ACTIVE'),
('f67568a2-5f0f-4fbb-93f2-c8918e56ff9c', gen_random_uuid(), 'KES', 10000, 'ACTIVE'),
('72a30e9a-5968-43e3-8ba3-8b0a72620276', gen_random_uuid(), 'KES', 500, 'ACTIVE'),
('821ccdb2-ceae-4cc8-bfd6-c326cd5877b7', gen_random_uuid(), 'KES', 245, 'SUSPENDED'),
('1bcac58a-9fad-444a-a4fb-b8002407d6ff', gen_random_uuid(), 'KES', 500, 'SUSPENDED'),
('33f3a2c2-5e3d-4267-bfce-258cab8e3a67', gen_random_uuid(), 'KES', 7500, 'CLOSED'),
('3e91d03a-1e8a-4caf-9160-c591c25402ea', gen_random_uuid(), 'KES', 0, 'CLOSED')

ON CONFLICT (id) DO NOTHING;