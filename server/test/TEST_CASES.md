# Moonbase Test Case Registry

This document is the master registry for Moonbase test cases.

Each test case has a unique ID which is used consistently across:
- test implementation
- test execution
- test results
- test reports

## Test Status

- PLANNED: Test has been defined but not implemented.
- IMPLEMENTED: Test code exists but has not yet been executed.
- PASS: Test executed successfully.
- FAIL: Test executed and actual behavior differed from expected behavior.
- BLOCKED: Test cannot currently be executed because a required feature or dependency is incomplete.

---

# Section 2 - Functional & Data Layer Testing

## Unit Tests

| ID | Test Case | Status |
|---|---|---|
| UT-FUNC-001 | User input validation | PLANNED |
| UT-FUNC-002 | Conversation input validation | PLANNED |
| UT-FUNC-003 | Message validation | PLANNED |
| UT-FUNC-004 | Session input/lookup logic | PLANNED |
| UT-FUNC-005 | Input parsing | PLANNED |

## Database Layer Testing

### User Operations

| ID | Test Case | Status |
|---|---|---|
| IT-DB-001 | Valid user creation | PLANNED |
| IT-DB-002 | Duplicate username | PLANNED |
| IT-DB-003 | Invalid user lookup | PLANNED |

### Conversation Operations

| ID | Test Case | Status |
|---|---|---|
| IT-DB-004 | Create conversation | PLANNED |
| IT-DB-005 | Retrieve conversation by ID | PLANNED |
| IT-DB-006 | Retrieve user's conversations | PLANNED |
| IT-DB-007 | Add conversation member | PLANNED |
| IT-DB-008 | Remove conversation member | PLANNED |
| IT-DB-009 | Add non-existent user | PLANNED |
| IT-DB-010 | Remove non-member | PLANNED |
| IT-DB-011 | Delete conversation | PLANNED |
| IT-DB-012 | Foreign-key relationship enforcement | PLANNED |

### Message Persistence

| ID | Test Case | Status |
|---|---|---|
| IT-DB-013 | Valid message creation | PLANNED |
| IT-DB-014 | Non-existent user | PLANNED |
| IT-DB-015 | Non-existent conversation | PLANNED |
| IT-DB-016 | Invalid conversation membership | PLANNED |
| IT-DB-017 | Empty message | PLANNED |
| IT-DB-018 | Message exceeds permitted length | PLANNED |
| IT-DB-019 | Invalid user identifier | PLANNED |
| IT-DB-020 | Invalid conversation identifier | PLANNED |
| IT-DB-021 | Database unchanged after failed validation | PLANNED |