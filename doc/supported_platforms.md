# Supported Host Platforms

`vfkit` is a macOS-only virtualization tool that uses Apple's Virtualization framework to manage virtual machines.

## Host Requirements

The project aims to support the most recent macOS major version at all times, along with the two previous major versions. Support for the oldest supported version will be dropped three years after its release date or when Apple discontinues support, whichever occurs first.

**macOS 15 or later** is recommended and fully supported.

## CI Testing

The project is tested on:
- macOS 15 (Sequoia) - Apple Silicon
- macOS 26 (Tahoe) - Apple Silicon
- macOS 26 (Tahoe) - Intel x86_64
- macOS 27 (Golden Gate) - Apple Silicon

Integration tests with virtualization are run on **macos-26-intel** runners, as virtualization is not supported on Apple Silicon runners in GitHub Actions. These Intel runners may be deprecated in the second half of 2028. See: [actions/runner-images#13739](https://github.com/actions/runner-images/issues/13045)

## Supported Architectures

- Intel x86_64
- Apple Silicon (arm64)
