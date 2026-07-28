CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE staff (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    hospital      TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT staff_username_hospital_key UNIQUE (username, hospital)
);

CREATE TABLE patients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital        TEXT NOT NULL,
    first_name_th   TEXT,
    middle_name_th  TEXT,
    last_name_th    TEXT,
    first_name_en   TEXT,
    middle_name_en  TEXT,
    last_name_en    TEXT,
    date_of_birth   DATE,
    patient_hn      TEXT,
    national_id     TEXT,
    passport_id     TEXT,
    phone_number    TEXT,
    email           TEXT,
    gender          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT patients_gender_check CHECK (gender IS NULL OR gender IN ('M', 'F'))
);

-- Upsert identity: prefer national_id when present, else passport_id.
CREATE UNIQUE INDEX patients_hospital_national_id_key
    ON patients (hospital, national_id)
    WHERE national_id IS NOT NULL;

CREATE UNIQUE INDEX patients_hospital_passport_id_key
    ON patients (hospital, passport_id)
    WHERE national_id IS NULL AND passport_id IS NOT NULL;

-- Search performance indexes
CREATE INDEX patients_hospital_national_id_idx ON patients (hospital, national_id);
CREATE INDEX patients_hospital_passport_id_idx ON patients (hospital, passport_id);
CREATE INDEX patients_hospital_last_name_en_idx ON patients (hospital, last_name_en);
CREATE INDEX patients_hospital_date_of_birth_idx ON patients (hospital, date_of_birth);
CREATE INDEX patients_hospital_first_name_en_lower_idx ON patients (hospital, lower(first_name_en));
CREATE INDEX patients_hospital_last_name_en_lower_idx ON patients (hospital, lower(last_name_en));
CREATE INDEX patients_hospital_email_lower_idx ON patients (hospital, lower(email));
CREATE INDEX patients_hospital_phone_idx ON patients (hospital, phone_number);
