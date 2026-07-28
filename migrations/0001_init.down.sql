DROP INDEX IF EXISTS patients_hospital_phone_idx;
DROP INDEX IF EXISTS patients_hospital_email_lower_idx;
DROP INDEX IF EXISTS patients_hospital_last_name_en_lower_idx;
DROP INDEX IF EXISTS patients_hospital_first_name_en_lower_idx;
DROP INDEX IF EXISTS patients_hospital_date_of_birth_idx;
DROP INDEX IF EXISTS patients_hospital_last_name_en_idx;
DROP INDEX IF EXISTS patients_hospital_passport_id_idx;
DROP INDEX IF EXISTS patients_hospital_national_id_idx;
DROP INDEX IF EXISTS patients_hospital_passport_id_key;
DROP INDEX IF EXISTS patients_hospital_national_id_key;

DROP TABLE IF EXISTS patients;
DROP TABLE IF EXISTS staff;
