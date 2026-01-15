# Build Notes

## Migration to Official Docker SDK

The project has been migrated from `github.com/fsouza/go-dockerclient` to the official Docker SDK (`github.com/docker/docker/client`). This resolves the Go 1.24 compatibility issue and provides better long-term support.

### Changes Made

- **Docker Handler**: Updated `widgets/handlers/docker/docker.go` to use the official Docker SDK
- **Dependencies**: Replaced `github.com/fsouza/go-dockerclient` with `github.com/docker/docker`
- **API Changes**: 
  - Uses `client.NewClientWithOpts()` instead of `dockerApi.NewClient()`
  - Uses `container.ListOptions` and `container.Summary` types
  - Uses `context.Context` for API calls

### Status

- ✅ All dependencies have been upgraded to their latest versions
- ✅ The project structure has been migrated to Go modules
- ✅ Main function has been moved to `cmd/howe/howe.go`
- ✅ Makefile has been updated to use `go mod` instead of `dep`
- ✅ Docker widget now uses the official Docker SDK (compatible with Go 1.24)
