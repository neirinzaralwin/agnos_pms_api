# Agnos Candidate Assignment

**CONFIDENTIAL**

Each candidate will need to complete the assignment within 3 days of receiving the task.

## Back-end developer

### Tasks

1. Develop APIs for a Hospital Middleware system to search and display patient information from Hospital Information Systems (HIS) using the following details:

**Hospital A API:**

- Route: `GET https://hospital-a.api.co.th/patient/search/{id}`
  - Request Parameters:
    - `id` (string): Can be either `national_id` or `passport_id`
  - Response Body (JSON):
    - `first_name_th`
    - `middle_name_th`
    - `last_name_th`
    - `first_name_en`
    - `middle_name_en`
    - `last_name_en`
    - `date_of_birth`
    - `patient_hn`
    - `national_id`
    - `passport_id`
    - `phone_number`
    - `email`
    - `gender` (M, F)

2. Design a database schema for a "Patient" model that is compatible with hospital data structures.

3. Design a database schema for a "Staff" model where each staff can only search for patients in the same hospital as the staff.

4. Implement the following APIs:
   - API to create a new hospital staff member with login credentials (`/staff/create`):
     - Input: `username`, `password`, `hospital`
   - API for staff login (`/staff/login`):
     - Input: `username`, `password`, `hospital`
   - API to search for a patient (`/patient/search`):
     - Requires login
     - Input: (all fields are optional)
       - `national_id`
       - `passport_id`
       - `first_name`
       - `middle_name`
       - `last_name`
       - `date_of_birth`
       - `phone_number`
       - `email`
     - Output: Patients matching the criteria and belonging to the same hospital as the staff member

5. Create unit tests for each API, covering both positive and negative test scenarios.

### Tech Stack

1. Go
2. Gin Framework
3. Docker
4. Nginx
5. Postgres

> \*The resources are proprietary and are intellectual property of Agnos health co. ltd. DO NOT SHARE WITH OTHERS\*

## Deliverables

1. Development planning documentations — share as Google Doc
   - Project Structure
   - API Spec
   - ER Diagram
2. Setup server with docker compose containing nginx, golang service, postgresql
3. Share your GitHub

## Evaluation Criteria

1. **Requirement satisfaction**
2. **Code quality** (readability, maintainability, structure)
3. **Unit test coverage**
4. **Documentation clarity**
