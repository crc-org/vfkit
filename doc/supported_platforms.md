# Supported Host Platforms

`vfkit` is a macOS-only virtualization tool that uses Apple's Virtualization framework to manage virtual machines.

## Host Requirements

The project aims to support the most recent macOS major version at all times, along with the two previous major versions. Support for the oldest supported version will be dropped three years after its release date or when Apple discontinues support, whichever occurs first.

**macOS 14.0 or later** is recommended and fully supported.

## CI Testing

The project is tested on:
- macOS 14 (Sonoma) - Apple Silicon
- macOS 15 (Sequoia) - Apple Silicon
- macOS 15 (Sequoia) - Intel x86_64
- macOS 26 (Tahoe) - Apple Silicon

Integration tests with virtualization are run on **macos-15-intel** runners, as virtualization is not supported on Apple Silicon runners in GitHub Actions. These Intel runners will be deprecated by August 2027. See: [actions/runner-images#13045](https://github.com/actions/runner-images/issues/13045)
## Supported Architectures

- Intel x86_64
- Apple Silicon (arm64)
