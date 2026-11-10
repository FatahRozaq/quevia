-- Migration: 000001_init_schema.up.sql
-- Description: Core 26 tables for Quevia Smart Queue & Appointment Management with RLS & GiST

CREATE EXTENSION IF NOT EXISTS btree_gist;

-- ============================================================================
-- 1. TENANT & IDENTITY
-- ============================================================================

-- users (Global, No RLS)
CREATE TABLE users (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    email               VARCHAR(255) NOT NULL,
    password_hash       VARCHAR(255),
    full_name           VARCHAR(150) NOT NULL,
    phone               VARCHAR(30),
    platform_role       VARCHAR(20),
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    is_system           BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified_at   TIMESTAMPTZ,
    last_login_at       TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT,

    CONSTRAINT uq_users_guid UNIQUE (guid),
    CONSTRAINT ck_users_platform_role CHECK (platform_role IS NULL OR platform_role IN ('SUPER_ADMIN')),
    CONSTRAINT ck_users_status CHECK (status IN ('ACTIVE', 'INVITED', 'DISABLED'))
);

CREATE UNIQUE INDEX uq_users_email ON users (lower(email)) WHERE deleted_at IS NULL;
CREATE INDEX ix_users_created_by ON users (created_by);
CREATE INDEX ix_users_updated_by ON users (updated_by);
CREATE INDEX ix_users_deleted_by ON users (deleted_by);

ALTER TABLE users ADD CONSTRAINT fk_users_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE users ADD CONSTRAINT fk_users_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE users ADD CONSTRAINT fk_users_deleted_by FOREIGN KEY (deleted_by) REFERENCES users(id) ON DELETE RESTRICT;

-- tenants
CREATE TABLE tenants (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    name                VARCHAR(150) NOT NULL,
    slug                VARCHAR(80) NOT NULL,
    business_type       VARCHAR(30),
    contact_email       VARCHAR(255) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_tenants_guid UNIQUE (guid),
    CONSTRAINT ck_tenants_status CHECK (status IN ('ACTIVE', 'SUSPENDED'))
);

CREATE UNIQUE INDEX uq_tenants_slug ON tenants (slug) WHERE deleted_at IS NULL;
CREATE INDEX ix_tenants_created_by ON tenants (created_by);
CREATE INDEX ix_tenants_updated_by ON tenants (updated_by);
CREATE INDEX ix_tenants_deleted_by ON tenants (deleted_by);

-- branches
CREATE TABLE branches (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    code                VARCHAR(20) NOT NULL,
    name                VARCHAR(150) NOT NULL,
    address             TEXT,
    phone               VARCHAR(30),
    timezone            VARCHAR(50) NOT NULL DEFAULT 'Asia/Jakarta',
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_branches_guid UNIQUE (guid),
    CONSTRAINT ck_branches_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_branches_tenant_code ON branches (tenant_id, code) WHERE deleted_at IS NULL;
CREATE INDEX ix_branches_tenant_id ON branches (tenant_id);
CREATE INDEX ix_branches_created_by ON branches (created_by);
CREATE INDEX ix_branches_updated_by ON branches (updated_by);
CREATE INDEX ix_branches_deleted_by ON branches (deleted_by);

-- memberships
CREATE TABLE memberships (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    branch_id           BIGINT REFERENCES branches(id) ON DELETE RESTRICT,
    role                VARCHAR(20) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_memberships_guid UNIQUE (guid),
    CONSTRAINT ck_memberships_role CHECK (role IN ('TENANT_ADMIN', 'STAFF')),
    CONSTRAINT ck_memberships_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_memberships_scope ON memberships (user_id, tenant_id, COALESCE(branch_id, 0)) WHERE deleted_at IS NULL;
CREATE INDEX ix_memberships_tenant_id ON memberships (tenant_id);
CREATE INDEX ix_memberships_user_id ON memberships (user_id);
CREATE INDEX ix_memberships_branch_id ON memberships (branch_id);
CREATE INDEX ix_memberships_created_by ON memberships (created_by);
CREATE INDEX ix_memberships_updated_by ON memberships (updated_by);
CREATE INDEX ix_memberships_deleted_by ON memberships (deleted_by);

-- invitations
CREATE TABLE invitations (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT REFERENCES branches(id) ON DELETE RESTRICT,
    email               VARCHAR(255) NOT NULL,
    role                VARCHAR(20) NOT NULL,
    token_hash          VARCHAR(128) NOT NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    accepted_at         TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_invitations_guid UNIQUE (guid),
    CONSTRAINT ck_invitations_role CHECK (role IN ('TENANT_ADMIN', 'STAFF')),
    CONSTRAINT ck_invitations_status CHECK (status IN ('PENDING', 'ACCEPTED', 'EXPIRED', 'REVOKED'))
);

CREATE UNIQUE INDEX uq_invitations_token_hash ON invitations (token_hash) WHERE deleted_at IS NULL;
CREATE INDEX ix_invitations_tenant_id ON invitations (tenant_id);
CREATE INDEX ix_invitations_branch_id ON invitations (branch_id);
CREATE INDEX ix_invitations_created_by ON invitations (created_by);
CREATE INDEX ix_invitations_updated_by ON invitations (updated_by);
CREATE INDEX ix_invitations_deleted_by ON invitations (deleted_by);

-- ============================================================================
-- 2. MASTER & KONFIGURASI
-- ============================================================================

-- service_categories
CREATE TABLE service_categories (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    name                VARCHAR(100) NOT NULL,
    sort_order          SMALLINT NOT NULL DEFAULT 0,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_service_categories_guid UNIQUE (guid)
);

CREATE UNIQUE INDEX uq_service_categories_name ON service_categories (tenant_id, lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX ix_service_categories_tenant_id ON service_categories (tenant_id);
CREATE INDEX ix_service_categories_created_by ON service_categories (created_by);
CREATE INDEX ix_service_categories_updated_by ON service_categories (updated_by);
CREATE INDEX ix_service_categories_deleted_by ON service_categories (deleted_by);

-- services
CREATE TABLE services (
    id                      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                    UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id               BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id               BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    category_id             BIGINT REFERENCES service_categories(id) ON DELETE RESTRICT,
    name                    VARCHAR(150) NOT NULL,
    description             TEXT,
    duration_minutes        INTEGER NOT NULL,
    buffer_minutes          INTEGER NOT NULL DEFAULT 0,
    price                   NUMERIC(14,2),
    queue_prefix            VARCHAR(5) NOT NULL,
    required_specialization VARCHAR(100),
    status                  VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by              BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at              TIMESTAMPTZ,
    deleted_by              BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_services_guid UNIQUE (guid),
    CONSTRAINT ck_services_duration CHECK (duration_minutes > 0),
    CONSTRAINT ck_services_buffer CHECK (buffer_minutes >= 0),
    CONSTRAINT ck_services_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_services_queue_prefix ON services (branch_id, queue_prefix) WHERE deleted_at IS NULL;
CREATE INDEX ix_services_tenant_id ON services (tenant_id);
CREATE INDEX ix_services_branch_id ON services (branch_id);
CREATE INDEX ix_services_category_id ON services (category_id);
CREATE INDEX ix_services_created_by ON services (created_by);
CREATE INDEX ix_services_updated_by ON services (updated_by);
CREATE INDEX ix_services_deleted_by ON services (deleted_by);

-- staff_profiles
CREATE TABLE staff_profiles (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    display_name        VARCHAR(150) NOT NULL,
    specialization      VARCHAR(100),
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_staff_profiles_guid UNIQUE (guid),
    CONSTRAINT ck_staff_profiles_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_staff_profiles_user_branch ON staff_profiles (user_id, branch_id) WHERE deleted_at IS NULL;
CREATE INDEX ix_staff_profiles_tenant_id ON staff_profiles (tenant_id);
CREATE INDEX ix_staff_profiles_branch_id ON staff_profiles (branch_id);
CREATE INDEX ix_staff_profiles_user_id ON staff_profiles (user_id);
CREATE INDEX ix_staff_profiles_created_by ON staff_profiles (created_by);
CREATE INDEX ix_staff_profiles_updated_by ON staff_profiles (updated_by);
CREATE INDEX ix_staff_profiles_deleted_by ON staff_profiles (deleted_by);

-- staff_services
CREATE TABLE staff_services (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    staff_id            BIGINT NOT NULL REFERENCES staff_profiles(id) ON DELETE RESTRICT,
    service_id          BIGINT NOT NULL REFERENCES services(id) ON DELETE RESTRICT,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_staff_services_guid UNIQUE (guid)
);

CREATE UNIQUE INDEX uq_staff_services_staff_service ON staff_services (staff_id, service_id) WHERE deleted_at IS NULL;
CREATE INDEX ix_staff_services_tenant_id ON staff_services (tenant_id);
CREATE INDEX ix_staff_services_staff_id ON staff_services (staff_id);
CREATE INDEX ix_staff_services_service_id ON staff_services (service_id);
CREATE INDEX ix_staff_services_created_by ON staff_services (created_by);
CREATE INDEX ix_staff_services_updated_by ON staff_services (updated_by);
CREATE INDEX ix_staff_services_deleted_by ON staff_services (deleted_by);

-- service_counters
CREATE TABLE service_counters (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    code                VARCHAR(20) NOT NULL,
    name                VARCHAR(100) NOT NULL,
    display_order       SMALLINT NOT NULL DEFAULT 0,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_service_counters_guid UNIQUE (guid),
    CONSTRAINT ck_service_counters_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_service_counters_code ON service_counters (branch_id, code) WHERE deleted_at IS NULL;
CREATE INDEX ix_service_counters_tenant_id ON service_counters (tenant_id);
CREATE INDEX ix_service_counters_branch_id ON service_counters (branch_id);
CREATE INDEX ix_service_counters_created_by ON service_counters (created_by);
CREATE INDEX ix_service_counters_updated_by ON service_counters (updated_by);
CREATE INDEX ix_service_counters_deleted_by ON service_counters (deleted_by);

-- display_screens
CREATE TABLE display_screens (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    name                VARCHAR(100) NOT NULL,
    access_token_hash   VARCHAR(128) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    last_seen_at        TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_display_screens_guid UNIQUE (guid),
    CONSTRAINT ck_display_screens_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE INDEX ix_display_screens_tenant_id ON display_screens (tenant_id);
CREATE INDEX ix_display_screens_branch_id ON display_screens (branch_id);
CREATE INDEX ix_display_screens_access_token ON display_screens (access_token_hash);
CREATE INDEX ix_display_screens_created_by ON display_screens (created_by);
CREATE INDEX ix_display_screens_updated_by ON display_screens (updated_by);
CREATE INDEX ix_display_screens_deleted_by ON display_screens (deleted_by);

-- branch_settings
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

-- ============================================================================
-- 3. PENJADWALAN
-- ============================================================================

-- business_hours
CREATE TABLE business_hours (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    day_of_week         SMALLINT NOT NULL,
    open_time           TIME NOT NULL,
    close_time          TIME NOT NULL,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_business_hours_guid UNIQUE (guid),
    CONSTRAINT ck_business_hours_day CHECK (day_of_week BETWEEN 1 AND 7),
    CONSTRAINT ck_business_hours_time CHECK (close_time > open_time)
);

CREATE UNIQUE INDEX uq_business_hours_scope ON business_hours (branch_id, day_of_week, open_time) WHERE deleted_at IS NULL;
CREATE INDEX ix_business_hours_tenant_id ON business_hours (tenant_id);
CREATE INDEX ix_business_hours_branch_id ON business_hours (branch_id);
CREATE INDEX ix_business_hours_created_by ON business_hours (created_by);
CREATE INDEX ix_business_hours_updated_by ON business_hours (updated_by);
CREATE INDEX ix_business_hours_deleted_by ON business_hours (deleted_by);

-- staff_schedules
CREATE TABLE staff_schedules (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    staff_id            BIGINT NOT NULL REFERENCES staff_profiles(id) ON DELETE RESTRICT,
    day_of_week         SMALLINT NOT NULL,
    start_time          TIME NOT NULL,
    end_time            TIME NOT NULL,
    break_start         TIME,
    break_end           TIME,
    valid_from          DATE NOT NULL,
    valid_until         DATE,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_staff_schedules_guid UNIQUE (guid),
    CONSTRAINT ck_staff_schedules_day CHECK (day_of_week BETWEEN 1 AND 7),
    CONSTRAINT ck_staff_schedules_time CHECK (end_time > start_time),
    CONSTRAINT ck_staff_schedules_break CHECK (
        (break_start IS NULL AND break_end IS NULL) OR
        (break_start IS NOT NULL AND break_end IS NOT NULL AND break_end > break_start AND break_start >= start_time AND break_end <= end_time)
    )
);

CREATE INDEX ix_staff_schedules_tenant_id ON staff_schedules (tenant_id);
CREATE INDEX ix_staff_schedules_staff_day ON staff_schedules (staff_id, day_of_week);
CREATE INDEX ix_staff_schedules_created_by ON staff_schedules (created_by);
CREATE INDEX ix_staff_schedules_updated_by ON staff_schedules (updated_by);
CREATE INDEX ix_staff_schedules_deleted_by ON staff_schedules (deleted_by);

-- schedule_exceptions
CREATE TABLE schedule_exceptions (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    staff_id            BIGINT REFERENCES staff_profiles(id) ON DELETE RESTRICT,
    exception_type      VARCHAR(20) NOT NULL,
    start_date          DATE NOT NULL,
    end_date            DATE NOT NULL,
    start_time          TIME,
    end_time            TIME,
    reason              VARCHAR(255),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_schedule_exceptions_guid UNIQUE (guid),
    CONSTRAINT ck_schedule_exceptions_type CHECK (exception_type IN ('HOLIDAY', 'LEAVE', 'CUSTOM_HOURS')),
    CONSTRAINT ck_schedule_exceptions_dates CHECK (end_date >= start_date),
    CONSTRAINT ck_schedule_exceptions_times CHECK (
        (start_time IS NULL AND end_time IS NULL) OR
        (start_time IS NOT NULL AND end_time IS NOT NULL AND end_time > start_time)
    )
);

CREATE INDEX ix_schedule_exceptions_tenant_id ON schedule_exceptions (tenant_id);
CREATE INDEX ix_schedule_exceptions_branch_dates ON schedule_exceptions (branch_id, start_date, end_date);
CREATE INDEX ix_schedule_exceptions_staff_id ON schedule_exceptions (staff_id);
CREATE INDEX ix_schedule_exceptions_created_by ON schedule_exceptions (created_by);
CREATE INDEX ix_schedule_exceptions_updated_by ON schedule_exceptions (updated_by);
CREATE INDEX ix_schedule_exceptions_deleted_by ON schedule_exceptions (deleted_by);

-- ============================================================================
-- 4. CUSTOMER & APPOINTMENT
-- ============================================================================

-- customers
CREATE TABLE customers (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    user_id             BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    full_name           VARCHAR(150) NOT NULL,
    email               VARCHAR(255),
    phone               VARCHAR(30),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_customers_guid UNIQUE (guid)
);

CREATE UNIQUE INDEX uq_customers_email ON customers (tenant_id, lower(email)) WHERE deleted_at IS NULL AND email IS NOT NULL;
CREATE INDEX ix_customers_tenant_id ON customers (tenant_id);
CREATE INDEX ix_customers_user_id ON customers (user_id);
CREATE INDEX ix_customers_created_by ON customers (created_by);
CREATE INDEX ix_customers_updated_by ON customers (updated_by);
CREATE INDEX ix_customers_deleted_by ON customers (deleted_by);

-- appointments
CREATE TABLE appointments (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    customer_id         BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    service_id          BIGINT NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
    staff_id            BIGINT NOT NULL REFERENCES staff_profiles(id) ON DELETE RESTRICT,

    booking_code        VARCHAR(30) NOT NULL,
    qr_token            VARCHAR(64) NOT NULL,
    start_at            TIMESTAMPTZ NOT NULL,
    end_at              TIMESTAMPTZ NOT NULL,
    status              VARCHAR(30) NOT NULL DEFAULT 'CONFIRMED',
    notes               TEXT,
    rescheduled_count   SMALLINT NOT NULL DEFAULT 0,
    cancelled_at        TIMESTAMPTZ,
    cancel_reason       VARCHAR(255),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_appointments_guid UNIQUE (guid),
    CONSTRAINT ck_appointments_time CHECK (end_at > start_at),
    CONSTRAINT ck_appointments_status CHECK (
        status IN ('CONFIRMED', 'CHECKED_IN', 'COMPLETED', 'CANCELLED', 'NO_SHOW')
    ),
    CONSTRAINT ex_appointments_no_overlap EXCLUDE USING gist (
        staff_id WITH =,
        tstzrange(start_at, end_at) WITH &&
    ) WHERE (status IN ('CONFIRMED', 'CHECKED_IN') AND deleted_at IS NULL)
);

CREATE UNIQUE INDEX uq_appointments_booking_code ON appointments (booking_code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_appointments_qr_token ON appointments (qr_token) WHERE deleted_at IS NULL;
CREATE INDEX ix_appointments_tenant_id ON appointments (tenant_id);
CREATE INDEX ix_appointments_branch_start ON appointments (branch_id, start_at) WHERE deleted_at IS NULL;
CREATE INDEX ix_appointments_customer_history ON appointments (customer_id, start_at DESC);
CREATE INDEX ix_appointments_status_start ON appointments (start_at) WHERE status = 'CONFIRMED' AND deleted_at IS NULL;
CREATE INDEX ix_appointments_staff_id ON appointments (staff_id);
CREATE INDEX ix_appointments_service_id ON appointments (service_id);
CREATE INDEX ix_appointments_created_by ON appointments (created_by);
CREATE INDEX ix_appointments_updated_by ON appointments (updated_by);
CREATE INDEX ix_appointments_deleted_by ON appointments (deleted_by);

-- ============================================================================
-- 5. ANTREAN (QUEUE ENGINE)
-- ============================================================================

-- queue_counters
CREATE TABLE queue_counters (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    service_id          BIGINT NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
    queue_date          DATE NOT NULL,
    last_number         INTEGER NOT NULL DEFAULT 0,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_queue_counters_guid UNIQUE (guid)
);

CREATE UNIQUE INDEX uq_queue_counters_scope ON queue_counters (branch_id, service_id, queue_date) WHERE deleted_at IS NULL;
CREATE INDEX ix_queue_counters_tenant_id ON queue_counters (tenant_id);
CREATE INDEX ix_queue_counters_created_by ON queue_counters (created_by);
CREATE INDEX ix_queue_counters_updated_by ON queue_counters (updated_by);
CREATE INDEX ix_queue_counters_deleted_by ON queue_counters (deleted_by);

-- queue_tickets
CREATE TABLE queue_tickets (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    service_id          BIGINT NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
    customer_id         BIGINT REFERENCES customers(id) ON DELETE RESTRICT,
    appointment_id      BIGINT REFERENCES appointments(id) ON DELETE RESTRICT,
    counter_id          BIGINT REFERENCES service_counters(id) ON DELETE RESTRICT,
    staff_id            BIGINT REFERENCES staff_profiles(id) ON DELETE RESTRICT,

    queue_date          DATE NOT NULL,
    queue_number        INTEGER NOT NULL,
    queue_code          VARCHAR(20) NOT NULL,
    source              VARCHAR(20) NOT NULL,
    priority            INTEGER NOT NULL DEFAULT 0,
    status              VARCHAR(30) NOT NULL DEFAULT 'WAITING',

    check_in_method     VARCHAR(20),
    checked_in_at       TIMESTAMPTZ,
    called_at           TIMESTAMPTZ,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    recall_count        SMALLINT NOT NULL DEFAULT 0,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_queue_tickets_guid UNIQUE (guid),
    CONSTRAINT ck_queue_tickets_source CHECK (source IN ('WALK_IN', 'APPOINTMENT')),
    CONSTRAINT ck_queue_tickets_status CHECK (
        status IN ('WAITING', 'CALLED', 'SERVING', 'COMPLETED', 'SKIPPED', 'CANCELLED', 'NO_SHOW')
    ),
    CONSTRAINT ck_queue_tickets_source_appointment CHECK (
        (source = 'APPOINTMENT') = (appointment_id IS NOT NULL)
    ),
    CONSTRAINT ck_queue_tickets_check_in_method CHECK (
        check_in_method IS NULL OR check_in_method IN ('WEB', 'QR', 'BOOKING_CODE', 'OVERRIDE')
    )
);

CREATE UNIQUE INDEX uq_queue_tickets_appointment ON queue_tickets (appointment_id) WHERE appointment_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_queue_tickets_number ON queue_tickets (branch_id, service_id, queue_date, queue_number) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_queue_tickets_code ON queue_tickets (branch_id, queue_date, queue_code) WHERE deleted_at IS NULL;
CREATE INDEX ix_queue_tickets_tenant_id ON queue_tickets (tenant_id);
CREATE INDEX ix_queue_tickets_call_next ON queue_tickets (branch_id, queue_date, status, priority DESC, queue_number);
CREATE INDEX ix_queue_tickets_completed ON queue_tickets (branch_id, queue_date) WHERE status = 'COMPLETED' AND deleted_at IS NULL;
CREATE INDEX ix_queue_tickets_customer_id ON queue_tickets (customer_id);
CREATE INDEX ix_queue_tickets_counter_id ON queue_tickets (counter_id);
CREATE INDEX ix_queue_tickets_staff_id ON queue_tickets (staff_id);
CREATE INDEX ix_queue_tickets_created_by ON queue_tickets (created_by);
CREATE INDEX ix_queue_tickets_updated_by ON queue_tickets (updated_by);
CREATE INDEX ix_queue_tickets_deleted_by ON queue_tickets (deleted_by);

-- ============================================================================
-- 6. NOTIFIKASI & SISTEM
-- ============================================================================

-- notification_templates
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

-- notifications
CREATE TABLE notifications (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    appointment_id      BIGINT REFERENCES appointments(id) ON DELETE RESTRICT,
    queue_ticket_id     BIGINT REFERENCES queue_tickets(id) ON DELETE RESTRICT,
    customer_id         BIGINT REFERENCES customers(id) ON DELETE RESTRICT,
    recipient_email     VARCHAR(255),
    event_type          VARCHAR(40) NOT NULL,
    channel             VARCHAR(10) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    subject             VARCHAR(255),
    body                TEXT NOT NULL,
    scheduled_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at             TIMESTAMPTZ,
    attempts            SMALLINT NOT NULL DEFAULT 0,
    last_error          TEXT,
    read_at             TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_notifications_guid UNIQUE (guid),
    CONSTRAINT ck_notifications_channel CHECK (channel IN ('EMAIL', 'IN_APP')),
    CONSTRAINT ck_notifications_status CHECK (status IN ('PENDING', 'SENT', 'FAILED', 'CANCELLED')),
    CONSTRAINT ck_notifications_email_recipient CHECK (channel != 'EMAIL' OR recipient_email IS NOT NULL)
);

CREATE INDEX ix_notifications_tenant_id ON notifications (tenant_id);
CREATE INDEX ix_notifications_worker_pending ON notifications (status, scheduled_at) WHERE status = 'PENDING';
CREATE INDEX ix_notifications_appointment_id ON notifications (appointment_id);
CREATE INDEX ix_notifications_queue_ticket_id ON notifications (queue_ticket_id);
CREATE INDEX ix_notifications_customer_id ON notifications (customer_id);
CREATE INDEX ix_notifications_created_by ON notifications (created_by);
CREATE INDEX ix_notifications_updated_by ON notifications (updated_by);
CREATE INDEX ix_notifications_deleted_by ON notifications (deleted_by);

-- audit_logs (Append-Only)
CREATE TABLE audit_logs (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT REFERENCES branches(id) ON DELETE RESTRICT,
    action              VARCHAR(60) NOT NULL,
    entity_type         VARCHAR(60) NOT NULL,
    entity_guid         UUID NOT NULL,
    before_data         JSONB,
    after_data          JSONB,
    ip_address          INET,
    request_id          VARCHAR(64),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_audit_logs_guid UNIQUE (guid)
);

CREATE INDEX ix_audit_logs_tenant_created ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX ix_audit_logs_entity ON audit_logs (entity_type, entity_guid);
CREATE INDEX ix_audit_logs_branch_id ON audit_logs (branch_id);
CREATE INDEX ix_audit_logs_created_by ON audit_logs (created_by);

-- outbox_events (Operational, No Soft Delete)
CREATE TABLE outbox_events (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    aggregate_type      VARCHAR(60) NOT NULL,
    aggregate_guid      UUID NOT NULL,
    event_type          VARCHAR(80) NOT NULL,
    payload             JSONB NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    attempts            SMALLINT NOT NULL DEFAULT 0,
    available_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at        TIMESTAMPTZ,
    last_error          TEXT,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_outbox_events_guid UNIQUE (guid),
    CONSTRAINT ck_outbox_events_status CHECK (status IN ('PENDING', 'PROCESSED', 'FAILED'))
);

CREATE INDEX ix_outbox_events_worker_pending ON outbox_events (status, available_at) WHERE status = 'PENDING';
CREATE INDEX ix_outbox_events_tenant_id ON outbox_events (tenant_id);
CREATE INDEX ix_outbox_events_created_by ON outbox_events (created_by);
CREATE INDEX ix_outbox_events_updated_by ON outbox_events (updated_by);

-- idempotency_keys (Operational, No Soft Delete)
CREATE TABLE idempotency_keys (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    scope               VARCHAR(60) NOT NULL,
    key                 VARCHAR(100) NOT NULL,
    request_hash        VARCHAR(64) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'IN_PROGRESS',
    response_status     SMALLINT,
    response_body       JSONB,
    expires_at          TIMESTAMPTZ NOT NULL,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_idempotency_keys_guid UNIQUE (guid),
    CONSTRAINT ck_idempotency_keys_status CHECK (status IN ('IN_PROGRESS', 'COMPLETED'))
);

CREATE UNIQUE INDEX uq_idempotency_keys_scope ON idempotency_keys (tenant_id, scope, key);
CREATE INDEX ix_idempotency_keys_expires ON idempotency_keys (expires_at);
CREATE INDEX ix_idempotency_keys_tenant_id ON idempotency_keys (tenant_id);
CREATE INDEX ix_idempotency_keys_created_by ON idempotency_keys (created_by);
CREATE INDEX ix_idempotency_keys_updated_by ON idempotency_keys (updated_by);

-- ============================================================================
-- 7. LANGGANAN (SIMULASI SAAS)
-- ============================================================================

-- subscription_plans (Global, No RLS)
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

-- tenant_subscriptions
CREATE TABLE tenant_subscriptions (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    plan_id             BIGINT NOT NULL REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    status              VARCHAR(20) NOT NULL DEFAULT 'TRIAL',
    started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at             TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_tenant_subscriptions_guid UNIQUE (guid),
    CONSTRAINT ck_tenant_subscriptions_status CHECK (status IN ('TRIAL', 'ACTIVE', 'CANCELLED', 'EXPIRED'))
);

CREATE UNIQUE INDEX uq_tenant_subscriptions_active ON tenant_subscriptions (tenant_id) WHERE status IN ('TRIAL', 'ACTIVE') AND deleted_at IS NULL;
CREATE INDEX ix_tenant_subscriptions_tenant_id ON tenant_subscriptions (tenant_id);
CREATE INDEX ix_tenant_subscriptions_plan_id ON tenant_subscriptions (plan_id);
CREATE INDEX ix_tenant_subscriptions_created_by ON tenant_subscriptions (created_by);
CREATE INDEX ix_tenant_subscriptions_updated_by ON tenant_subscriptions (updated_by);
CREATE INDEX ix_tenant_subscriptions_deleted_by ON tenant_subscriptions (deleted_by);

-- ============================================================================
-- 8. ROW-LEVEL SECURITY (RLS) POLICIES
-- ============================================================================

-- tenants policy (Self-isolation by id)
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_tenants ON tenants
    USING (id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);

-- macro for tenant_id scoped tables
DO $$
DECLARE
    tbl text;
    tables text[] := ARRAY[
        'branches', 'memberships', 'invitations',
        'service_categories', 'services', 'staff_profiles', 'staff_services',
        'service_counters', 'display_screens', 'branch_settings',
        'business_hours', 'staff_schedules', 'schedule_exceptions',
        'customers', 'appointments', 'queue_counters', 'queue_tickets',
        'notification_templates', 'notifications',
        'outbox_events', 'idempotency_keys', 'tenant_subscriptions'
    ];
BEGIN
    FOREACH tbl IN ARRAY tables LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY;', tbl);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY;', tbl);
        EXECUTE format($sql$CREATE POLICY tenant_isolation_%1$I ON %1$I USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT) WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);$sql$, tbl);
    END LOOP;
END $$;

-- audit_logs policy (nullable tenant_id: visible only if tenant_id matches or platform admin)
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);

-- ============================================================================
-- 9. SEED DATA
-- ============================================================================

-- Reserved system user guest
INSERT INTO users (guid, email, full_name, is_system, status)
VALUES ('00000000-0000-0000-0000-000000000001', 'guest@system.quevia.app', 'Guest User', TRUE, 'ACTIVE')
ON CONFLICT DO NOTHING;

-- Subscription plans
INSERT INTO subscription_plans (code, name, max_branches, max_staff, features, status)
VALUES
    ('FREE', 'Free Plan', 1, 3, '{"analytics": false}'::jsonb, 'ACTIVE'),
    ('PRO', 'Pro Plan', 5, 20, '{"analytics": true}'::jsonb, 'ACTIVE'),
    ('BUSINESS', 'Business Plan', NULL, NULL, '{"analytics": true, "integrations": true}'::jsonb, 'ACTIVE')
ON CONFLICT DO NOTHING;
