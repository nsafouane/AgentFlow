# Implementation Plan - Control Plane API Skeleton

This implementation plan converts the control plane API design into a series of prompts for code-generation that will implement each step in a test-driven manner. Each task builds incrementally on previous tasks and focuses on discrete, manageable coding steps that can be executed by a coding agent.

- [x] 1. HTTP Server & Routing (/api/v1) + Middleware Stack (logging, tracing, recovery)
















  - Create HTTP server with /api/v1 routing and configurable timeouts, TLS support, and graceful shutdown
  - Implement middleware stack in correct order: recovery, logging, tracing with OpenTelemetry integration from Q1.2
  - Build middleware registration system with proper error handling and context propagation
  - Integrate structured logging from Q1.2 with correlation IDs and trace context propagation
  - Write unit tests for server lifecycle, middleware ordering, error recovery, and routing functionality
  - Create manual test: start server, verify middleware execution order through logs, test graceful shutdown and routing
  - Document server configuration options, middleware chain behavior, and troubleshooting guide
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 2. AuthN (JWT Dev Secret) & Optional OIDC Flag






  - Implement JWT token issuance and validation with configurable dev secret and claims extraction (tenant_id, user_id, roles)
  - Create authentication middleware with token extraction from Authorization header and context propagation
  - Build optional OIDC integration with feature flag (oidc.enabled) and graceful fallback to internal JWT
  - Add token lifecycle management with expiration, refresh, and revocation capabilities
  - Write unit tests for auth success/failure scenarios, token validation, OIDC integration, and fallback behavior
  - Create manual test: OIDC flag off/on flows, token issuance/validation, test authentication endpoints
  - Document auth flows, token claims structure, OIDC configuration, and security considerations
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

- [x] 3. Multi-Tenancy Enforcement





  - Implement tenant scoping for database queries with automatic tenant_id WHERE clause injection
  - Create message bus subject prefixes for tenant isolation using Q1.2 messaging integration
  - Build tenant context extraction from JWT claims with validation and cross-tenant access prevention
  - Add tenant isolation middleware with audit logging for cross-tenant access attempts
  - Write unit tests for cross-tenant access denial, query scoping, subject prefixing, and isolation validation
  - Create manual test: seed two tenants, ensure data isolation, attempt cross-tenant access, verify blocking and audit logs
  - Document tenancy model, isolation architecture, security boundaries, and operational procedures
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [ ] 4. RBAC Seed Roles (admin, developer, viewer)
  - Implement role binding and middleware enforcement using Q1.3 rbac_roles and rbac_bindings tables
  - Create predefined roles (admin, developer, viewer) with comprehensive permission matrices
  - Build RBAC middleware with resource-action validation, permission checking, and audit logging
  - Add role inheritance support and dynamic permission updates without service restart
  - Write unit tests for negative mutation tests for viewer role, permission validation, and role enforcement
  - Create manual test: role switch smoke test, assign roles to users, test permission boundaries, verify audit logs
  - Document RBAC matrix, role definitions, permission model, and administrative procedures
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

- [ ] 5. Rate Limiting (Redis)
  - Implement sliding window or token bucket rate limiting algorithms using Redis from Q1.3 integration
  - Create configurable rate limits per tenant, endpoint, and user with burst handling capabilities
  - Build rate limiting middleware with 429 responses, retry-after headers, and quota information
  - Add rate limit configuration management with dynamic updates and tenant-specific overrides
  - Write unit tests for burst + sustained rate limiting scenarios, Redis integration, and quota enforcement
  - Create manual test: configure rate limits, generate traffic, verify 429 header responses and quota semantics
  - Document quota semantics, rate limiting headers, configuration options, and performance tuning guide
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

- [ ] 6. OpenAPI Contract + SDK Codegen (Python/JS stubs)
  - Implement OpenAPI 3.0 specification generation with complete endpoint documentation and schema validation
  - Create automated SDK generation pipeline for Python and JavaScript stubs with proper error handling
  - Build semantic diff CI integration for breaking change detection and approval workflow
  - Add interactive API explorer with authentication examples and request/response samples
  - Write unit tests for schema lint validation, SDK generation accuracy, and compatibility checks
  - Create manual test: generate OpenAPI spec, build and import generated SDKs, test basic API calls
  - Document API versioning policy, SDK usage guides, and integration examples
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [ ] 7. Budgets & Cost Estimate Endpoint (Heuristic)
  - Implement PlanCostModel.Estimate with token estimation, tool cost calculation, and confidence scoring
  - Create /api/v1/plans/{id}/estimate endpoint with structured cost breakdown and budget validation
  - Build budget warning system with threshold detection, overage alerts, and notification triggers
  - Add cost model calibration with actual vs estimated variance tracking for model improvement
  - Write unit tests for deterministic output fixtures, cost estimation accuracy, and budget validation scenarios
  - Create manual test: submit workflow plans, verify cost estimates, trigger over-budget warning logs
  - Document cost estimation limitations, methodology, budget configuration, and calibration procedures
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

- [ ] 8. ExecProfileCompiler (deny-by-default tool perms) – Stub Enforcement
  - Implement profile synthesis for tools with deny-by-default permissions and resource limit enforcement
  - Create tool permission validation with network allow-lists, filesystem restrictions, and timeout enforcement
  - Build ExecProfile attachment to tools with runtime validation and audit logging
  - Add permission enforcement middleware with violation detection, blocking, and security event logging
  - Write unit tests for missing permission denial scenarios, profile compilation, and enforcement validation
  - Create manual test: register tools with permissions, attempt unauthorized host calls, verify blocking and audit
  - Document permission taxonomy, tool security profiles, and operational security procedures
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

- [ ] 9. Data Minimization Middleware (Feature Flag)
  - Implement redaction rules engine with configurable patterns, replacement strategies, and data type targeting
  - Create data minimization middleware with feature flag control (data_minimization.enabled) and performance optimization
  - Build PII detection patterns for common data types (emails, phones, SSNs, IPs) with custom tenant patterns
  - Add redaction validation with golden dataset testing and zero-leakage verification
  - Write unit tests for golden redaction tests, pattern matching accuracy, and performance impact assessment
  - Create manual test: enable feature flag, process sample data, scan logs for PII, verify complete redaction
  - Document minimization strategy, redaction patterns, configuration options, and compliance validation procedures
  - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_

- [ ] 10. Residency Policy Gate (Strict Mode)
  - Implement egress filter with allow-list validation, geographic restrictions, and strict mode enforcement
  - Create residency policy engine with real-time policy updates and network request blocking
  - Build residency enforcement middleware with audit logging for egress attempts and violations
  - Add data sovereignty controls with classification-based routing and compliance reporting
  - Write unit tests for external host blocking, policy enforcement, and geographic validation scenarios
  - Create manual test: toggle strict mode flag, attempt external egress requests, verify blocking and audit logs
  - Document residency configuration, data sovereignty controls, and compliance procedures
  - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

## Exit Criteria (Gate G3)

This implementation plan ensures all Gate G3 exit criteria are satisfied:

- **Multi-tenant auth & RBAC enforced**: Tasks 2-5 implement comprehensive authentication, authorization, and tenant isolation
- **OpenAPI stable**: Task 7 implements specification generation with breaking change detection
- **Cost estimate warnings**: Task 8 implements cost estimation with budget integration and warning systems
- **Residency strict mode blocks egress**: Task 11 implements data sovereignty controls with egress filtering
- **SDK smoke tests pass**: Task 7 includes SDK generation and validation testing
- **Data minimization golden tests achieve 0 leaked sample PII tokens**: Task 10 implements PII redaction with zero-leakage validation

## Additional Quantitative Assertions (Gate G3 Augmentation)

- **OpenAPI semantic diff job records no unapproved breaking changes**: Task 7 implements automated breaking change detection with approval workflow
- **Tenancy isolation matrix executed**: Task 4 includes comprehensive cross-tenant access testing and validation
- **Rate limiting enforces quotas under load**: Task 6 implements robust rate limiting with burst handling and performance validation
- **RBAC denies unauthorized access**: Task 5 includes negative access testing and permission boundary validation
- **PII redaction achieves 100% detection rate**: Task 10 includes golden dataset testing with comprehensive PII pattern coverage

Each task follows the test-driven development approach with implementation, unit tests, manual validation, and documentation components as specified in the development plan.