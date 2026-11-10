-- Reserved system user guest
INSERT INTO users (guid, email, full_name, is_system, status)
VALUES ('00000000-0000-0000-0000-000000000001', 'guest@system.quevia.app', 'Guest User', TRUE, 'ACTIVE')
ON CONFLICT DO NOTHING;

-- Default subscription plans
INSERT INTO subscription_plans (code, name, max_branches, max_staff, features, status)
VALUES
    ('FREE', 'Free Plan', 1, 3, '{"analytics": false}'::jsonb, 'ACTIVE'),
    ('PRO', 'Pro Plan', 5, 20, '{"analytics": true}'::jsonb, 'ACTIVE'),
    ('BUSINESS', 'Business Plan', NULL, NULL, '{"analytics": true, "integrations": true}'::jsonb, 'ACTIVE')
ON CONFLICT DO NOTHING;
