CREATE TABLE notification_templates (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    event_type          VARCHAR(40) NOT NULL,
    channel             VARCHAR(10) NOT NULL,
    subject             VARCHAR(255),
    body_template       TEXT NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_notification_templates_guid UNIQUE (guid),
    CONSTRAINT ck_notification_templates_channel CHECK (channel IN ('EMAIL', 'IN_APP')),
    CONSTRAINT ck_notification_templates_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_notification_templates_scope ON notification_templates (tenant_id, event_type, channel) WHERE deleted_at IS NULL;
CREATE INDEX ix_notification_templates_tenant_id ON notification_templates (tenant_id);
CREATE INDEX ix_notification_templates_created_by ON notification_templates (created_by);
CREATE INDEX ix_notification_templates_updated_by ON notification_templates (updated_by);
CREATE INDEX ix_notification_templates_deleted_by ON notification_templates (deleted_by);

ALTER TABLE notification_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_templates FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_notification_templates ON notification_templates
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
