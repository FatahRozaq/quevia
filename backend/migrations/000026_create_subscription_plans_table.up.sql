CREATE TABLE subscription_plans (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    code                VARCHAR(30) NOT NULL,
    name                VARCHAR(100) NOT NULL,
    max_branches        INTEGER,
    max_staff           INTEGER,
    features            JSONB NOT NULL DEFAULT '{}'::jsonb,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_subscription_plans_guid UNIQUE (guid),
    CONSTRAINT ck_subscription_plans_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_subscription_plans_code ON subscription_plans (code) WHERE deleted_at IS NULL;
CREATE INDEX ix_subscription_plans_created_by ON subscription_plans (created_by);
CREATE INDEX ix_subscription_plans_updated_by ON subscription_plans (updated_by);
CREATE INDEX ix_subscription_plans_deleted_by ON subscription_plans (deleted_by);
