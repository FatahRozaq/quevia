CREATE TABLE branch_settings (
    id                          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                        UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id                   BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id                   BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,

    change_cutoff_minutes       INTEGER NOT NULL DEFAULT 180,
    check_in_open_before_minutes INTEGER NOT NULL DEFAULT 30,
    check_in_close_after_minutes INTEGER NOT NULL DEFAULT 15,
    no_show_after_minutes       INTEGER NOT NULL DEFAULT 10,
    slot_step_minutes           SMALLINT NOT NULL DEFAULT 30,
    appointment_priority        SMALLINT NOT NULL DEFAULT 2,
    walk_in_priority            SMALLINT NOT NULL DEFAULT 1,
    late_grace_minutes          INTEGER,
    notify_before_queue_count   SMALLINT NOT NULL DEFAULT 2,

    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                  BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by                  BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at                  TIMESTAMPTZ,
    deleted_by                  BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_branch_settings_guid UNIQUE (guid)
);

CREATE UNIQUE INDEX uq_branch_settings_branch ON branch_settings (branch_id) WHERE deleted_at IS NULL;
CREATE INDEX ix_branch_settings_tenant_id ON branch_settings (tenant_id);
CREATE INDEX ix_branch_settings_created_by ON branch_settings (created_by);
CREATE INDEX ix_branch_settings_updated_by ON branch_settings (updated_by);
CREATE INDEX ix_branch_settings_deleted_by ON branch_settings (deleted_by);

ALTER TABLE branch_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE branch_settings FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_branch_settings ON branch_settings
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
