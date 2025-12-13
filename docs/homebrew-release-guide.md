# Rebranding TS OpenCode and Publishing via Homebrew

This guide explains how to rebrand the OpenCode CLI for your business, build signed binaries, and distribute them through a Homebrew tap.

## Prerequisites
- Bun installed
- Access to your GitHub fork (for release assets and tap repository)
- `GITHUB_TOKEN` with permission to push to your tap repo

## 1) Rename the CLI for your brand
- Update `packages/opencode/package.json`:
  - Set the package name and binary entry (`name`, `bin`) to your brand.
  - Adjust the `build` script or related scripts if you keep using them.
- In your fork, update URLs/branding (README, website links) so the Homebrew formula will point at your repo and release assets.

## 2) Build branded binaries for all platforms
- From `packages/opencode`, run `bun run ./script/build.ts` to cross-compile for Linux (glibc/musl, AVX2 and non-AVX2), macOS (arm64/x64), and Windows x64. Binaries land in `dist/<name>/bin/opencode` per target.
- For quick iteration, use `--single` to build only for the current platform, or `--skip-install` if platform-specific deps are already installed, e.g. `bun run ./script/build.ts --single --skip-install`.

## 3) Package and publish npm artifacts (optional)
- Still in `packages/opencode`, run `bun run ./script/publish.ts` to bundle binaries, create per-platform npm tarballs, and publish them with the configured channel tags.
- The script smoke-tests the built binary before packing and writes an installer package that depends on the per-platform binary packages, matching the `pkg.name` you set.

## 4) Generate release archives and checksums
- For non-preview builds, the publish script tars/zips each platform’s `bin` output and calculates SHA256 checksums for Homebrew and other package managers.

## 5) Create your Homebrew formula and push your tap
- The publish script assembles a Ruby formula with download URLs and SHA256 values and pushes it to the tap repo it clones (requires `GITHUB_TOKEN`).
- To rebrand:
  - Fork or create your own tap repo (e.g., `github.com/your-org/homebrew-yourtap`).
  - Edit the formula template in `publish.ts` to change the class name (`class Opencode < Formula`), `homepage`, `desc`, and each `url` so they point to your fork’s GitHub release assets (matching filenames from step 4).
  - Keep the `sha256` values emitted by the publish run.
- After editing, rerun `bun run ./script/publish.ts` with `GITHUB_TOKEN` set to a token that can push to your tap; the script rewrites the formula file and pushes the commit.

## 6) Create a GitHub Release with your assets
- Upload the archives (`opencode-<os>-<arch>.zip`/`.tar.gz` from `dist`) produced in step 4 to a GitHub Release whose tag/version matches the one used by `publish.ts` URLs.

## 7) Point users to your tap
- In your docs, instruct users to install from your tap:
  ```bash
  brew tap your-org/yourtap
  brew install yourtool   # matches the formula class/filename
  ```
- Ensure the formula filename in your tap (e.g., `yourtool.rb`) matches the class name.

## 8) Smoke-test the Homebrew flow
- After pushing the tap, run `brew update && brew install yourtool` on macOS and Linux to confirm install success and `yourtool --version` output.
