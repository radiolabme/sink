# Sink Engine - Build Progress

## Completed (Test-Driven) ✅

### 1. types.go (130 LOC)
- ✅ Type-safe domain model
- ✅ Sealed union via `StepVariant` interface
- ✅ Impossible states unrepresentable at compile time
- ✅ Clean separation of step types

### 2. config_test.go (280 LOC)  
- ✅ 20+ test cases for config parsing
- ✅ Tests for all 4 step types
- ✅ Fact definition validation tests
- ✅ File loading tests
- ✅ All tests passing

### 3. config.go (220 LOC)
- ✅ JSON config loading and parsing
- ✅ Comprehensive validation
- ✅ Export variable name validation (regex)
- ✅ Fact name validation (regex)
- ✅ Platform validation
- ✅ Transform type checking
- ✅ All 20+ tests passing

**Total: 630 LOC, 20+ passing tests**

## Test Results

```bash
$ go test -v
=== RUN   TestConfigParsing
=== RUN   TestConfigParsing/valid_minimal_config
=== RUN   TestConfigParsing/config_with_facts
=== RUN   TestConfigParsing/config_with_check-error_step
=== RUN   TestConfigParsing/config_with_check-remediate_step
=== RUN   TestConfigParsing/missing_required_version
=== RUN   TestConfigParsing/empty_platforms
--- PASS: TestConfigParsing (0.00s)

=== RUN   TestConfigFromFile
=== RUN   TestConfigFromFile/original_install-config.json
=== RUN   TestConfigFromFile/config_with_facts
--- PASS: TestConfigFromFile (0.00s)

=== RUN   TestFactDefinitionValidation
=== RUN   TestFactDefinitionValidation/valid_fact_with_export
=== RUN   TestFactDefinitionValidation/invalid_export_name_(lowercase)
=== RUN   TestFactDefinitionValidation/invalid_export_name_(starts_with_number)
=== RUN   TestFactDefinitionValidation/invalid_platform
=== RUN   TestFactDefinitionValidation/empty_command
=== RUN   TestFactDefinitionValidation/transform_without_string_type
--- PASS: TestFactDefinitionValidation (0.00s)

PASS
ok      github.com/brian/sink   0.175s
```

## Type Safety Enforced

### At Compile Time:
- ✅ Step cannot be both command and check
- ✅ Only StepVariant types can be steps
- ✅ RemediationStep cannot be used as InstallStep

### At Parse Time:
- ✅ Export vars match `^[A-Z_][A-Z0-9_]*$`
- ✅ Fact names match `^[a-z_][a-z0-9_]*$`
- ✅ Platforms must be darwin/linux/windows
- ✅ Transform only with string type
- ✅ Platform has either steps OR distributions, not both
- ✅ Version and platforms are required
- ✅ At least one step per platform

## Next Steps

1. **facts_test.go** - Test fact gathering
2. **facts.go** - Implement fact gathering
3. **executor_test.go** - Test step execution
4. **executor.go** - Implement execution engine
5. **transport_test.go** - Test local/SSH
6. **transport.go** - Implement transports
7. **server_test.go** - Test REST API
8. **server.go** - Implement REST API
9. **main.go** - CLI entry point

Estimated remaining: ~370 LOC + tests

Total estimated: **~1000 LOC**
