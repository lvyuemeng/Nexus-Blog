# Hugo content contract fixtures

Run the complete host-level contract from the repository root:

```sh
go test ./tests
```

The Go test launches the installed Hugo executable with the isolated fixture
configuration in this directory. Each build uses a disposable destination and
cache. The checked-in `valid` fixture must render and publish its bundle
resources; mechanical invalid bundles are generated in temporary directories
and must fail with their named admission errors.

These fixtures do not import Noteloom and are never part of the production site.
The normal Nexus build remains the separate provider-integration check.
