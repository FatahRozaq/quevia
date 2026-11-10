DELETE FROM users WHERE email = 'guest@system.quevia.app';
DELETE FROM subscription_plans WHERE code IN ('FREE', 'PRO', 'BUSINESS');
