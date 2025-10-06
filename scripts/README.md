# Sink Scripts

This directory contains utility scripts that extend Sink's core functionality with operational tooling for remote deployment and system bootstrapping. These scripts address real-world scenarios where configuration management must reach beyond local execution to provision remote systems over SSH.

## Remote Bootstrap Overview

Deploying system configurations to remote hosts presents several challenges. Manual SSH sessions are error-prone and don't scale to multiple hosts. Configuration files must be transferred securely, and binaries need distribution to target systems. Without proper tooling, administrators resort to ad-hoc shell scripts that lack verification, security checking, and proper error handling.

The bootstrap-remote.sh script solves these problems by providing a secure, verifiable, and automated deployment workflow. It handles binary transfer, configuration delivery with integrity verification, and remote execution with proper context checking. The script works with local files, HTTPS URLs with TLS verification, or HTTP URLs with mandatory SHA256 checksums. All transfers occur over SSH, ensuring encryption in transit regardless of configuration source.

The result is a single command that can bootstrap remote systems reliably. The same script works for deploying to individual hosts or orchestrating fleet-wide configuration updates. Security is enforced through cryptographic verification, and the idempotent design allows safe re-execution if initial attempts encounter transient failures.

## Quick Start

The bootstrap script requires an SSH target specification and a configuration source. For local configurations, provide the file path directly. The script transfers both the Sink binary and configuration over encrypted SSH, then executes on the remote system:

```bash
./scripts/bootstrap-remote.sh user@host config.json
```

For centrally managed configurations served over HTTPS, provide the URL. The remote system downloads the configuration with TLS certificate verification providing cryptographic assurance of authenticity:

```bash
./scripts/bootstrap-remote.sh user@host https://configs.example.com/prod.json
```

When configurations must be served over HTTP, SHA256 checksums provide integrity verification. The script rejects HTTP URLs without checksums, preventing deployment of potentially compromised configurations:

```bash
./scripts/bootstrap-remote.sh user@host \
  http://configs.example.com/setup.json \
  --sha256 $(sha256sum config.json | cut -d' ' -f1)
```

These three patterns cover the most common deployment scenarios while maintaining strong security guarantees throughout.

## Command Interface

The bootstrap script accepts an SSH target and configuration source as required arguments, with optional flags controlling behavior and security verification. The SSH target uses standard SSH syntax supporting custom ports and authentication methods configured in SSH config files.

```
./bootstrap-remote.sh <ssh-target> <config-source> [options]
```

The SSH target specifies the remote host and credentials in standard format: `user@hostname` or `user@hostname:port`. SSH configuration from `~/.ssh/config` applies normally, including key selection, proxy commands, and connection multiplexing.

Configuration sources accept three forms. Local file paths transfer configurations directly via SCP, leveraging SSH encryption for security. HTTPS URLs direct the remote system to download configurations with TLS certificate validation. HTTP URLs require explicit SHA256 checksums passed via the `--sha256` flag, with the script rejecting HTTP sources lacking integrity verification.

Optional flags modify behavior while maintaining security guarantees. The `--binary` flag specifies an alternative Sink binary path, defaulting to `./bin/sink`. The `--no-cleanup` flag preserves temporary files on the remote system for debugging. The `--dry-run` flag previews operations without execution. The `--platform` flag overrides automatic platform detection. The `--yes` flag skips interactive confirmation, enabling automation. The `--verbose` flag provides detailed output for troubleshooting.

## Security Model

Configuration delivery security depends on the source type and verification mechanisms applied. The script enforces different requirements based on how configurations are retrieved, balancing convenience with security guarantees appropriate to each delivery method.

Local file sources transfer over SSH with encryption in transit. Since files originate from the administrator's trusted system and transfer through authenticated SSH channels, no additional verification is required. The SSH connection provides both confidentiality and authenticity through the established trust relationship between client and server.

HTTPS URLs rely on TLS certificate validation for authenticity and encryption. The remote system downloads configurations directly from the HTTPS endpoint, with the TLS handshake providing cryptographic proof of the server's identity. Certificate validation through the system's trusted root certificates prevents man-in-the-middle attacks. No separate checksum is required because TLS provides end-to-end security.

HTTP URLs lack transport security and server authentication, creating opportunity for interception or tampering. The script addresses this by mandating SHA256 checksums for HTTP sources. Before execution, the downloaded configuration is hashed and compared to the provided checksum. Mismatches cause immediate failure, preventing execution of compromised configurations. This approach provides integrity verification but not confidentiality during transfer.

HTTP URLs without SHA256 checksums are rejected outright. The script refuses to process HTTP configurations lacking integrity verification, displaying an error and exiting. This prevents accidental deployment of potentially compromised configurations and enforces the security policy that all configurations must be cryptographically verified.

The security model assumes trusted administrators and secure workstations. Local file sources presume the administrator's system is secure and files haven't been tampered with. HTTPS sources trust the system's root certificate store and TLS implementation. HTTP sources require administrators generate and protect checksums appropriately. The script cannot defend against compromised administrator workstations or stolen SSH credentials.

## Usage Examples

### Local Configuration Deployment

Deploying configurations from local files suits development workflows and initial provisioning. The administrator maintains configuration files on their workstation and deploys them to remote systems as needed. This pattern works well for small-scale deployments and testing scenarios.

Build the Sink binary locally before deployment:

```bash
make build
```

Deploy the local configuration to a remote host:

```bash
./scripts/bootstrap-remote.sh user@remote-host setup.json
```

The script transfers both the binary and configuration via SCP, validates the configuration on the remote system, then executes with interactive confirmation. This workflow provides immediate feedback and suits iterative development cycles.

### HTTPS Configuration Delivery

Organizations serving configurations from central repositories benefit from HTTPS delivery. Configuration updates are published to HTTPS endpoints, and remote systems download directly. This centralizes configuration management and enables version control through URL structure.

Deploy from an HTTPS endpoint with TLS verification:

```bash
./scripts/bootstrap-remote.sh user@remote-host \
  https://configs.example.com/production.json
```

The remote system downloads the configuration with automatic TLS certificate validation. No additional verification is required because TLS provides cryptographic authenticity and encryption. This pattern scales to large fleets where centralized configuration management is essential.

### GitHub Release Configurations

GitHub releases provide versioned, immutable configuration storage with built-in checksums. Tagging releases creates permanent references that never change, enabling reproducible deployments. The raw.githubusercontent.com service delivers files with HTTPS and supports version pinning through URL structure.

Deploy from a pinned GitHub release:

```bash
./scripts/bootstrap-remote.sh user@remote-host \
  https://raw.githubusercontent.com/myorg/configs/v1.0.0/prod.json
```

The script detects GitHub URLs and validates version pinning, displaying confirmation that the configuration is immutably pinned to a specific release tag. This provides both security and reproducibility for production deployments.

For configurations with accompanying checksum files, the script automatically fetches and verifies them:

```bash
./scripts/bootstrap-remote.sh user@remote-host \
  https://raw.githubusercontent.com/myorg/configs/v1.0.0/prod.json
```

If `prod.json.sha256` exists alongside the configuration, the script fetches it automatically and performs verification. This adds integrity checking on top of TLS security, useful for high-security environments requiring defense-in-depth.

### Commit SHA Pinning

Pinning to specific Git commit SHAs provides maximum immutability. Unlike tags which can theoretically be moved, commit SHAs are cryptographically guaranteed to reference specific content. This suits compliance requirements where exact configuration versions must be traceable.

Deploy a configuration pinned to a specific commit:

```bash
./scripts/bootstrap-remote.sh user@remote-host \
  https://raw.githubusercontent.com/myorg/configs/a1b2c3d4e5f6/prod.json
```

The script recognizes commit SHA pinning and confirms the immutable reference. This pattern provides the strongest guarantee that deployed configurations exactly match reviewed versions.

### HTTP Delivery with Verification

Legacy infrastructure may only support HTTP for configuration delivery. While not ideal, HTTP becomes acceptable when combined with mandatory SHA256 verification. This provides integrity checking even without transport security.

Generate a checksum for the configuration:

```bash
SHA256=$(sha256sum config.json | cut -d' ' -f1)
```

Deploy with explicit checksum verification:

```bash
./scripts/bootstrap-remote.sh user@remote-host \
  http://configs.example.com/config.json \
  --sha256 $SHA256
```

The remote system downloads the configuration via HTTP, computes its SHA256 hash, and compares it to the provided value. Mismatches cause immediate failure, preventing execution of tampered configurations. While this protects integrity, it doesn't provide confidentiality during transfer.

### Fleet Deployment

Deploying configurations to multiple hosts requires orchestration that handles failures gracefully and provides clear feedback. Shell loops combined with the `--yes` flag enable sequential provisioning of multiple systems.

Deploy to a range of web servers:

```bash
for host in web{1..5}.example.com; do
  echo "Bootstrapping $host..."
  ./scripts/bootstrap-remote.sh user@$host setup.json --yes
done
```

Each host is provisioned in sequence with automatic confirmation. Failures on individual hosts don't prevent subsequent deployments from proceeding. This pattern suits small to medium deployments where parallel execution isn't required.

### Preview Mode

Understanding what operations will occur before executing them prevents surprises and catches configuration errors. Dry-run mode performs all validation steps and displays the execution plan without making system changes.

Preview a deployment:

```bash
./scripts/bootstrap-remote.sh user@host config.json --dry-run
```

The script transfers files, validates configurations, and shows what would execute, then exits without running commands. This allows validating SSH connectivity, configuration syntax, and deployment readiness without system modifications.

### Custom Binary Deployment

Different environments may require specific Sink binary versions. Testing pre-release versions or deploying architecture-specific builds requires overriding the default binary location.

Deploy with a custom binary:

```bash
./scripts/bootstrap-remote.sh user@host config.json \
  --binary ~/Downloads/sink-v0.2.0
```

The specified binary is transferred instead of the default `./bin/sink`. This supports testing scenarios and multi-architecture deployments where different binaries are maintained for various target platforms.

### Platform Override

Automatic platform detection occasionally misidentifies systems or configurations need explicit platform targeting. The platform override forces execution for a specific operating system regardless of detection results.

Force Linux platform execution:

```bash
./scripts/bootstrap-remote.sh user@host config.json \
  --platform linux
```

This bypasses platform detection and selects Linux installation steps directly. Use this for systems where detection fails or when configurations should run against specific platforms for testing purposes.

### Debugging Deployments

Failed deployments require investigation of remote system state. Preserving temporary files after execution enables inspection of transferred configurations and binary locations.

Preserve remote files for debugging:

```bash
./scripts/bootstrap-remote.sh user@host config.json --no-cleanup
```

Temporary files remain in `/tmp` on the remote system after execution. This allows examining exactly what was transferred and executed, supporting troubleshooting of deployment failures.

## Execution Workflow

The bootstrap script orchestrates several operations that together establish a complete deployment. Understanding this workflow helps diagnose issues and optimize deployment processes. Each phase builds on the previous one, creating a pipeline from local preparation through remote execution.

The script begins by testing SSH connectivity to verify it can reach the target system and authenticate successfully. Once connectivity is confirmed, it transfers the Sink binary to the remote system using SCP. Configuration handling follows, with the approach depending on the source type: local files are transferred via SCP alongside the binary, HTTPS URLs are downloaded directly on the remote system with automatic TLS verification, and HTTP URLs require SHA256 checksums for integrity verification.

After both binary and configuration reach the remote system, the script validates the configuration to catch errors before execution begins. This validation runs Sink's built-in checking to ensure the JSON schema is correct and all referenced steps are valid. Following successful validation, the script prompts for user confirmation before executing, displaying the target host, user account, and step count. Once confirmed, execution proceeds with real-time output streamed back to the terminal. The script concludes by cleaning up temporary files from the remote system, removing both the configuration and binary unless the no-cleanup flag is set.

### Output Example

```bash
$ ./scripts/bootstrap-remote.sh user@webserver setup.json

🚀 Sink Remote Bootstrap
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   Target: user@webserver
   Config: setup.json

▶  Transferring sink binary...
✅ Binary transferred (3.2M)
▶  Transferring config file...
✅ Config transferred (2.1K)
▶  Validating config on remote host...
✅ Config is valid

▶  Executing sink on remote host...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Gathering facts...
🖥️  Platform: Ubuntu (linux)
📝 Steps: 5

🔍 Execution Context:
   Host:      webserver
   User:      deploy
   Work Dir:  /home/deploy
   OS/Arch:   Linux/amd64
   Transport: local

⚠️  You are about to execute 5 steps on webserver as deploy
   Continue? [yes/no]: yes

[1/5] Update apt cache...
      ✓ Success
[2/5] Install nginx...
      ✓ Success
[3/5] Start nginx...
      ✓ Success
[4/5] Enable nginx...
      ✓ Success
[5/5] Verify nginx...
      ✓ Success

✅ Execution complete: 5 steps succeeded
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ Bootstrap complete!
```

## Error Conditions

Deployment failures fall into several categories, each with distinct causes and remediation approaches. Understanding these failure modes enables rapid diagnosis and resolution. The script provides specific error messages for each condition rather than generic failures.

Connectivity failures occur when SSH cannot reach or authenticate to the target system. Network issues, incorrect hostnames, authentication problems, or firewall restrictions all manifest as connection failures. The script tests connectivity early to fail fast rather than after file transfers begin. Resolution involves verifying SSH connectivity manually with direct SSH commands before retrying the bootstrap.

File availability errors happen when required local files cannot be found. Missing Sink binaries typically indicate the project hasn't been built yet, while missing configuration files suggest incorrect paths or working directories. The script checks for file existence before attempting transfers, providing explicit paths in error messages. Building the project or verifying file paths resolves these conditions.

Security verification failures protect against executing tampered or corrupted configurations. HTTP URLs without SHA256 checksums are rejected immediately as policy, preventing acceptance of unverified content. Checksum mismatches during download indicate corruption or modification, triggering rejection before execution. These failures require generating fresh checksums and verifying configuration integrity.

Configuration validation failures catch syntax errors, schema violations, and invalid step definitions before execution begins. Running validation locally first identifies these issues earlier in the development cycle. The script reports specific validation errors with line numbers and field names, enabling precise fixes. Correcting the configuration based on error messages resolves these conditions.

Execution failures occur when Sink runs successfully but individual installation steps fail on the remote system. These represent environmental issues rather than script problems - missing permissions, unavailable packages, or incorrect system state. The script streams execution output in real-time, showing exactly which step failed and why. Resolution depends on addressing the underlying system condition that caused the step to fail.

## Security Guidance

Secure deployment requires attention to both transport security and configuration integrity. Multiple layers of verification protect against various threat models. Following these practices reduces risk while maintaining operational efficiency.

Transport security begins with preferring HTTPS for all configuration downloads. TLS provides both encryption during transit and server authentication through certificate validation. When HTTPS is unavailable and HTTP must be used, mandatory SHA256 verification becomes essential. Never accept HTTP configurations without checksums, as this creates vulnerability to man-in-the-middle attacks and tampering. The script enforces this policy by rejecting HTTP URLs that lack checksum parameters.

Checksum management requires generating fresh values for each configuration version. Reusing old checksums defeats integrity checking because modified configurations will match outdated hashes. Generate checksums immediately before deployment, store them separately from configurations, and never embed them in configuration files themselves. Separate storage prevents attackers from modifying both the configuration and its embedded checksum simultaneously.

Authentication security depends on using SSH key-based authentication rather than passwords. Keys provide stronger cryptographic security and enable automation without embedding credentials in scripts. Configure SSH key forwarding or agent authentication for deployments from CI/CD systems. Validate SSH connectivity and authentication before attempting deployments to catch authentication issues early.

Configuration review before deployment catches errors in development rather than production. Validate configurations locally using Sink's validation mode, test against staging environments, and use dry-run mode to preview operations before execution. This multi-stage validation reduces the risk of deploying broken configurations that fail during execution. Preview mode particularly helps verify that all steps are appropriate for the target environment before making any system changes.

## Troubleshooting Common Issues

Deployment problems typically fall into predictable categories with straightforward solutions. Understanding the diagnostic approach for each category accelerates problem resolution. Most issues can be reproduced and fixed locally before attempting remote deployment.

### Connection Failures

When the script reports inability to connect to the SSH target, the problem lies in network connectivity, DNS resolution, or SSH authentication. Test SSH connectivity directly using a simple command: `ssh user@host echo "test"`. If this fails, the issue is not with the bootstrap script but with underlying SSH configuration. Verify that SSH keys are properly configured in `~/.ssh/authorized_keys` on the remote system and that your private key is loaded in the SSH agent. Check that firewalls allow TCP port 22 from your source IP to the target system. Many cloud providers require explicit security group rules for SSH access.

### Binary Availability

The script expects to find the Sink binary at `./bin/sink` relative to the project root. When this file doesn't exist, the project hasn't been built yet. Run `make build` to compile the binary, which places it in the expected location. If working with a pre-built binary from another location, use the `--binary` flag to specify its path explicitly: `--binary ~/Downloads/sink`. This commonly occurs when testing release binaries or architecture-specific builds that aren't built locally.

### HTTP Security Requirements

The script rejects HTTP URLs without accompanying SHA256 checksums as security policy. This prevents accepting configurations that cannot be verified for integrity. Generate a checksum for the configuration file using `sha256sum config.json | cut -d' ' -f1`, then add it to the command with the `--sha256` flag. Store this checksum separately from the configuration itself - never embed checksums in the files they verify. This ensures that tampering with the configuration cannot simultaneously update its checksum.

### Checksum Verification Failures

When the downloaded configuration's checksum doesn't match the provided value, either the configuration was modified after checksum generation or network corruption occurred during transfer. Check whether the configuration file was edited after you generated its checksum - this is the most common cause. Configuration files in version control may have different line endings or whitespace than local copies, causing checksum mismatches. Always generate checksums from the exact file that will be deployed, and regenerate checksums after any modifications. For persistent mismatches, download the configuration manually and verify its contents match expectations.

### Validation Errors

Configuration validation failures indicate JSON syntax errors, schema violations, or invalid step definitions. The error message includes specific details about what failed validation and where in the configuration file the problem occurs. Validate configurations locally before attempting remote deployment using `./bin/sink validate config.json`. This provides the same validation output without requiring SSH connectivity or file transfers. Fix reported errors in the configuration, revalidate locally to confirm the fix, then retry deployment. Most validation failures result from typos in field names, incorrect JSON structure, or referencing undefined steps.

## CI/CD Integration

Automating deployments through continuous integration pipelines requires adapting the bootstrap script to work without interactive prompts. The `--yes` flag enables automatic confirmation, allowing the script to run unattended in automation contexts. SSH authentication in CI requires loading deployment keys into the environment before script execution.

GitHub Actions workflows integrate bootstrap operations into automated deployment pipelines. Store SSH private keys as encrypted secrets, load them into the SSH agent during workflow execution, and use the `--yes` flag for non-interactive operation:

```yaml
- name: Bootstrap production servers
  run: |
    ./scripts/bootstrap-remote.sh deploy@prod-server \
      https://configs.example.com/prod.json \
      --yes
```

Configure SSH key setup in preceding steps, and consider using environment-specific configurations by constructing URLs from repository variables.

GitLab CI environments follow similar patterns but use GitLab's variable system for SSH keys and target hosts. Configure deployment keys in repository settings, inject them into runner environments, and reference configuration variables:

```yaml
deploy:
  script:
    - ./scripts/bootstrap-remote.sh deploy@$SERVER_HOST setup.json --yes
  only:
    - main
```

Restricting deployment to specific branches prevents accidental production deployments from development branches.

Jenkins integrations leverage Jenkins' credential management for SSH keys and parameterized builds for target selection. Configure SSH credentials as Jenkins secrets, inject them into build environments, and use parameterized builds for flexible target selection:

```groovy
sh './scripts/bootstrap-remote.sh deploy@${SERVER} config.json --yes'
```

Jenkins' credential binding plugins handle SSH key injection securely.

## Advanced Patterns

### Parallel Fleet Deployment

Deploying to large server fleets benefits from parallel execution rather than sequential processing. GNU Parallel orchestrates concurrent deployments while managing connection limits and error handling:

```bash
parallel -j 10 ./scripts/bootstrap-remote.sh user@{} setup.json --yes \
  ::: host{1..10}.example.com
```

The `-j 10` flag limits concurrency to 10 simultaneous connections, preventing overwhelming network capacity or triggering rate limits. Each host's output is buffered and displayed after completion, maintaining readability. Failed deployments don't stop processing of remaining hosts.

### Ansible Orchestration

Integrating bootstrap operations with Ansible inventory management combines Sink's configuration approach with Ansible's host management:

```yaml
- name: Bootstrap with Sink
  command: ./scripts/bootstrap-remote.sh "{{ ansible_user }}@{{ inventory_hostname }}" setup.json --yes
  delegate_to: localhost
```

The `delegate_to: localhost` directive executes the bootstrap script from the Ansible control node rather than on target hosts, maintaining the script's SSH-based transfer approach. Ansible variables like `ansible_user` and `inventory_hostname` enable the same configuration to deploy across different inventories.

### Version-Controlled Configurations

Storing configurations in Git repositories enables version control, code review, and environment-specific configuration management:

```bash
git clone https://github.com/org/sink-configs.git /tmp/configs
./scripts/bootstrap-remote.sh user@host /tmp/configs/prod.json
```

This pattern separates configuration management from deployment tooling. Different branches can maintain environment-specific configurations, tags can mark tested configuration versions, and pull requests provide review workflows before configuration changes deploy to production.

## Related Documentation

The complete remote bootstrap guide in [docs/REMOTE_BOOTSTRAP.md](../docs/REMOTE_BOOTSTRAP.md) provides comprehensive coverage of deployment patterns, security considerations, and advanced scenarios. This document expands on the quick start examples shown here with detailed explanations of edge cases and production deployment strategies.

Configuration examples demonstrating various use cases appear in [examples/README.md](../examples/README.md), showing how to structure configurations for common installation scenarios. These examples include platform-specific installations, multi-step workflows, and conditional configuration patterns.

Future development plans for native SSH transport appear in [docs/REST_AND_SSH.md](../docs/REST_AND_SSH.md), describing how Sink will eventually support direct SSH execution without requiring the bootstrap script as an intermediary layer.

---

**Requirements:** SSH client, sink binary  
**Platforms:** macOS, Linux (any Unix-like with SSH)  
**License:** Same as Sink
