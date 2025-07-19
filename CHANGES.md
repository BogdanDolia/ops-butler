# Changes Made to Fix "fcntl64: symbol not found" Error

## Issue Description
The ops-butler-core container was failing with the error:
```
Error relocating /app/ops-butler-core: fcntl64: symbol not found
```

## Root Cause
The issue was caused by a mismatch between the build and runtime environments:
- The application was built in a glibc environment (golang:1.21 which is Debian-based)
- But it was run in an Alpine container that uses musl libc
- The binary was looking for glibc-specific symbols (like fcntl64) that aren't available in Alpine's musl libc

## Solution
Modified the Dockerfile.core to ensure compatibility between build and runtime environments:

1. Kept the build stage using the Debian-based `golang:1.21` image, which uses glibc
2. Changed the runtime stage from Alpine to Debian-based (`debian:bullseye-slim`), which also uses glibc
3. Updated the package installation commands to use apt-get instead of apk
4. Updated the user creation commands to use Debian-compatible syntax:
   - Changed `addgroup` to `groupadd`
   - Changed `adduser` with Alpine options to `useradd` with Debian options

This ensures that both the build and runtime environments use glibc, which fixes the "fcntl64: symbol not found" error.

## Alternative Solutions Considered
1. Building the application in an Alpine-based Go image with musl libc
   - This approach encountered compilation errors with the SQLite C code
   - Attempted to use build tags like `sqlite_omit_load_extension`, but still encountered errors

2. Using a different database driver that doesn't require CGO
   - This would require significant changes to the application code
   - SQLite is a core requirement of the application

## Testing
The container was successfully built using the build-and-push.sh script, and the build completed without errors.