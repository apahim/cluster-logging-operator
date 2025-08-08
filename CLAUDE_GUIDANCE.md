# Claude Code Implementation Guidance

## Project Context
This is the OpenShift Cluster Logging Operator, which manages log collection and forwarding using Vector as the collector. The operator generates Vector configurations from ClusterLogForwarder CRDs.

## Key Architecture Components

### 1. API Structure (`api/observability/v1/`)
- **Core CRD**: ClusterLogForwarder defines inputs, outputs, and pipelines
- **Output Types**: CloudWatch, Elasticsearch, Loki, etc. with authentication specs
- **Field Requirements**: All API fields need kubebuilder validation tags and operator-sdk CSV annotations

### 2. Vector Configuration Generation (`internal/generator/vector/output/`)
- **Templates**: Each output type has Go templates that generate Vector TOML configuration
- **Authentication**: Different auth methods generate different Vector auth configs
- **Testing**: Each generator has comprehensive tests with expected TOML files

### 3. Collector Deployment (`internal/collector/`)
- **Pod Generation**: Factory pattern creates DaemonSet/Deployment specs
- **Environment Variables**: Set in `NewCollectorContainer()` function
- **Volume Mounts**: Secrets mounted at `/var/run/ocp-collector/secrets/`
- **ServiceAccount Tokens**: Projected volumes mounted at `/var/run/ocp-collector/serviceaccount/`

### 4. Secret Collection (`internal/api/observability/`)
- **Critical**: All secret references must be collected for pod mounting
- **Function**: `SecretReferences()` and type-specific functions like `cloudwatchAuthKeys()`
- **Failure Mode**: Uncollected secrets cause Vector crashes due to missing mounts

## AWS CloudWatch Authentication Patterns

### ServiceAccount Web Identity Token Authentication
**Required Components:**
1. `AWS_ROLE_ARN` environment variable (primary role ARN from secret)
2. `AWS_WEB_IDENTITY_TOKEN_FILE` environment variable (points to token file)
3. ServiceAccount token mounted as projected volume
4. Optional: `auth.assume_role` in Vector config (for secondary role)

**Vector Configuration:**
- **Primary role only**: No auth fields in Vector config (uses env vars)
- **With assume role**: Only `auth.assume_role` field (env vars + assume role)

### Implementation Checklist for CloudWatch Authentication

#### 1. API Changes (`api/observability/v1/output_types.go`)
- [ ] Add new fields with proper validation tags
- [ ] Include comprehensive documentation comments
- [ ] Add operator-sdk CSV annotations for OpenShift Console

#### 2. Secret Collection (`internal/api/observability/output_types.go`)
- [ ] Update `cloudwatchAuthKeys()` to include new secret references
- [ ] **Critical**: Missing secrets cause Vector crashes - verify in tests

#### 3. Vector Configuration (`internal/generator/vector/output/cloudwatch/`)
- [ ] Update `authConfig()` function logic
- [ ] Consider ServiceAccount vs Secret token authentication paths
- [ ] Never generate conflicting auth fields
- [ ] Update `auth.go` struct if new Vector auth fields needed

#### 4. Environment Variables (`internal/collector/collector.go`)
- [ ] Add logic in `NewCollectorContainer()` to set AWS env vars
- [ ] Extract role ARNs from secrets when using ServiceAccount tokens
- [ ] Set both `AWS_ROLE_ARN` and `AWS_WEB_IDENTITY_TOKEN_FILE`

#### 5. Validation (`internal/validations/observability/outputs/`)
- [ ] Add validation for new fields
- [ ] Ensure secret references are valid
- [ ] Test with invalid configurations

#### 6. Testing Strategy
- [ ] **Vector Generation Tests**: Update expected TOML files
- [ ] **Secret Collection Tests**: Verify all secrets are collected
- [ ] **Validation Tests**: Test both valid and invalid configs
- [ ] **Integration Tests**: Test actual Vector deployment

#### 7. Code Generation
- [ ] Run `make generate` to update CRDs and deepcopy methods
- [ ] Verify generated CRD includes new fields with proper validation

## Common Pitfalls & Solutions

### Authentication Issues
**Problem**: Vector authentication enum errors
**Cause**: Conflicting auth methods (credentials file + assume role)
**Solution**: Use environment variables for primary auth, Vector config for secondary

### Secret Mounting Issues
**Problem**: Vector crashes with "file not found" errors
**Cause**: Secret references not collected in `SecretReferences()` functions
**Solution**: Always update secret collection when adding new secret fields

### Test Failures
**Problem**: Tests fail after auth changes
**Cause**: Expected TOML files don't match generated config
**Solution**: Update expected TOML files to match new authentication logic

### Environment Variable Issues
**Problem**: ServiceAccount authentication fails
**Cause**: Missing `AWS_ROLE_ARN` or `AWS_WEB_IDENTITY_TOKEN_FILE` env vars
**Solution**: Extract from secrets and set in collector container

## File Patterns

### Adding New CloudWatch Authentication Fields
1. `api/observability/v1/output_types.go` - Add field to struct
2. `internal/api/observability/output_types.go` - Update secret collection
3. `internal/generator/vector/output/cloudwatch/cloudwatch.go` - Update Vector config
4. `internal/collector/collector.go` - Update environment variables (if needed)
5. `internal/validations/observability/outputs/cloudwatch.go` - Add validation
6. Tests in respective `*_test.go` files
7. `make generate` for CRD/deepcopy updates


## Testing Commands
```bash
# Unit tests
go test ./internal/generator/vector/output/cloudwatch/... -v
go test ./internal/api/observability/... -v
go test ./internal/collector/... -v

# Generate CRDs and deepcopy
make generate

# Validation tests
go test ./internal/validations/observability/outputs/... -v
```

## Key Constants
- `constants.AWSRoleArnEnvVarKey` = "AWS_ROLE_ARN"
- `constants.AWSWebIdentityTokenEnvVarKey` = "AWS_WEB_IDENTITY_TOKEN_FILE"
- `constants.ServiceAccountSecretPath` = "/var/run/ocp-collector/serviceaccount"
- `constants.TokenKey` = "token"

## Vector Authentication Reference
- **Environment Variables**: AWS_ROLE_ARN + AWS_WEB_IDENTITY_TOKEN_FILE
- **Access Keys**: auth.access_key_id + auth.secret_access_key
- **Credentials File**: auth.credentials_file + auth.profile
- **Assume Role**: auth.assume_role (can combine with any above)

Never mix credentials file with environment variable authentication in the same Vector sink configuration.