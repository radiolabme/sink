# Build Configuration

This document explains Sink's build options and binary configurations.

## Quick Answer

**Current `make build`**: ❌ **NOT statically linked** (uses dynamic system libraries)

**For static builds**: ✅ Use `make build-static` (Linux only, fully portable)

## Build Targets

### `make build` - Default Build
Creates a binary for your current platform with **dynamic linking**.

```bash
make build
# Output: bin/sink
```

**Characteristics:**
- Uses system dynamic libraries
- Smaller binary size (~10MB on macOS)
- Requires compatible system libraries
- Best for: Local development and testing

**Dependencies (macOS example):**
```
/usr/lib/libSystem.B.dylib
/usr/lib/libresolv.9.dylib
/System/Library/Frameworks/CoreFoundation.framework
/System/Library/Frameworks/Security.framework
```

### `make build-static` - Static Linux Build
Creates a **fully static** Linux binary with zero external dependencies.

```bash
make build-static
# Output: bin/sink-linux-amd64-static
```

**Characteristics:**
- `CGO_ENABLED=0` - Disables CGO for pure Go build
- `-extldflags=-static` - Forces static linking
- `-tags netgo,osusergo` - Pure Go networking and user/group lookups
- `-ldflags="-s -w"` - Strips debug info and symbol table
- Fully portable across Linux distributions
- ~7.4MB binary size
- Best for: Production deployments, containers, CI/CD

**Verification:**
```bash
file bin/sink-linux-amd64-static
# Output: statically linked, stripped
```

### `make build-linux` - Dynamic Linux Build
Creates a Linux binary with dynamic linking.

```bash
make build-linux
# Output: bin/sink-linux-amd64
```

**Use case:** When you need a Linux binary but don't need portability.

### `make build-all` - Multi-Platform Build
Creates binaries for all supported platforms:

```bash
make build-all
# Output:
#   bin/sink-darwin-amd64          (macOS Intel)
#   bin/sink-darwin-arm64          (macOS Apple Silicon)
#   bin/sink-linux-amd64           (Linux x64, dynamic)
#   bin/sink-linux-arm64           (Linux ARM64, dynamic)
#   bin/sink-linux-amd64-static    (Linux x64, static)
```

**Best for:** Release preparation, testing cross-platform compatibility

## Static vs Dynamic Linking

### Static Linking Benefits
✅ **Portability** - Works on any Linux distribution  
✅ **No dependencies** - Single self-contained binary  
✅ **Container-friendly** - Can use `FROM scratch` base image  
✅ **Deployment simplicity** - Just copy the binary  
✅ **Version isolation** - No library version conflicts

### Static Linking Tradeoffs
⚠️ **Larger binary** - ~7.4MB vs potentially smaller dynamic  
⚠️ **No shared libraries** - Can't benefit from system lib updates  
⚠️ **Limited CGO** - Can't use CGO-dependent packages

### Dynamic Linking Benefits
✅ **Smaller binary** - System libraries not included  
✅ **Shared resources** - Benefits from system library updates  
✅ **Full CGO support** - Can use any Go package

### Dynamic Linking Tradeoffs
⚠️ **System dependencies** - Requires compatible libraries  
⚠️ **Version sensitivity** - May break with library updates  
⚠️ **Distribution-specific** - Binary may not work across distros

## Recommended Builds by Use Case

### Local Development (macOS/Linux)
```bash
make build
```
Fast builds, easy debugging, platform-native.

### Production Linux Deployment
```bash
make build-static
```
Maximum portability, works everywhere, container-ready.

### Docker Containers
```bash
make build-static
```

Use in Dockerfile:
```dockerfile
FROM scratch
COPY bin/sink-linux-amd64-static /sink
ENTRYPOINT ["/sink"]
```

### Release Distribution
```bash
make build-all
```
Provides binaries for all platforms and architectures.

### CI/CD Pipelines
```bash
make build-static
```
Consistent behavior across different CI environments.

## GitHub Actions Release Builds

The release workflow (`.github/workflows/release.yml`) builds:

- `sink-linux-amd64` (static)
- `sink-linux-arm64` (static)
- `sink-darwin-amd64` (dynamic, macOS requirements)
- `sink-darwin-arm64` (dynamic, macOS requirements)

All Linux releases are **static** for maximum portability.  
macOS releases are dynamic because:
- macOS doesn't support fully static binaries
- System frameworks (CoreFoundation, Security) are required
- Dynamic linking is the macOS standard

## Verifying Build Type

### Linux Binary
```bash
file bin/sink-linux-amd64-static
# statically linked = ✅ Static
# dynamically linked = ❌ Dynamic

ldd bin/sink-linux-amd64-static
# "not a dynamic executable" = ✅ Static
# Shows libraries = ❌ Dynamic
```

### macOS Binary
```bash
file bin/sink
# Mach-O 64-bit executable

otool -L bin/sink
# Lists dynamic libraries required
```

## Build Flags Explained

### `-ldflags="-s -w"`
- `-s` - Omit symbol table
- `-w` - Omit DWARF debug info
- Result: Smaller binary, harder to debug

### `-ldflags="-extldflags=-static"`
- Forces static linking of external libraries
- Only effective with CGO enabled
- Ensures no dynamic dependencies

### `-tags netgo,osusergo`
- `netgo` - Pure Go networking (no CGO DNS)
- `osusergo` - Pure Go user/group lookups
- Enables static builds without CGO

### `CGO_ENABLED=0`
- Disables CGO completely
- Forces pure Go implementation
- Required for truly static builds
- Breaks packages that require CGO

## Size Comparison

```bash
# Build all variants
make build build-static build-linux

# Compare sizes
ls -lh bin/sink*
```

Typical sizes:
- macOS dynamic: ~10MB
- Linux dynamic: ~9MB
- Linux static: ~7.4MB

Static is actually *smaller* because:
- Stripped debug info (`-s -w`)
- No CGO overhead
- Pure Go implementations are efficient

## Common Issues

### "Cannot find library"
**Problem**: Dynamic binary on system without required libraries  
**Solution**: Use `make build-static` for portable Linux builds

### "Exec format error"
**Problem**: Wrong architecture (ARM vs x64)  
**Solution**: Use correct build target:
- `GOARCH=amd64` for x86-64
- `GOARCH=arm64` for ARM64

### "DNS resolution fails"
**Problem**: Static build with network issues  
**Solution**: Already handled - we use `-tags netgo` for pure Go DNS

## Future Enhancements

Potential build improvements:

1. **Compressed releases** - UPX compression for smaller binaries
2. **Darwin static** - Explore static linking on macOS (limited support)
3. **Windows builds** - Add Windows support
4. **Build cache** - Speed up CI with build caching
5. **Cross-compilation in CI** - Build for all platforms in one job

## Summary

| Build Command | Output | Type | Size | Portability | Use Case |
|--------------|---------|------|------|-------------|----------|
| `make build` | `sink` | Dynamic | 10MB | Current OS | Development |
| `make build-static` | `sink-linux-amd64-static` | **Static** | 7.4MB | **All Linux** | **Production** |
| `make build-linux` | `sink-linux-amd64` | Dynamic | 9MB | Similar Linux | Testing |
| `make build-all` | Multiple | Mixed | Various | All platforms | Release |

**Recommendation for production**: Always use `make build-static` for Linux deployments to ensure maximum portability and eliminate dependency issues.
