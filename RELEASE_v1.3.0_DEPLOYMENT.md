# Release v1.3.0 - Ready to Deploy! 🎉

## Release Summary

**Version**: v1.3.0  
**Date**: January 25, 2026  
**Type**: Minor version (new feature)  
**Status**: ✅ Built and ready

## What's New

### Major Feature: Topic Size Calculation

Calculate actual disk space used by Kafka topics across all brokers and partitions.

**New Flags:**
- `-topic-sizes` - Calculate and display topic sizes
- `-topic-sizes-output <file>` - Save report to JSON
- `-topic-list <topics>` - Filter specific topics

**Quick Example:**
```bash
./kmap -brokers localhost:9092 -topic-sizes
```

## Release Artifacts

All binaries built and ready in `bin/`:

```
✅ kmap-linux-amd64 (7.9M)
✅ kmap-linux-arm64 (7.3M)
✅ kmap-darwin-amd64 (8.0M) - macOS Intel
✅ kmap-darwin-arm64 (7.5M) - macOS Apple Silicon
✅ kmap-windows-amd64.exe (8.0M)
```

**Build Details:**
- Version: 1.3.0
- Build Time: 2026-01-25T20:19:32Z
- Git Commit: 0383128

## Files Updated

### New Files
- ✅ `topic_sizes.go` - Core implementation
- ✅ `TOPIC_SIZES.md` - Comprehensive documentation
- ✅ `QUICKSTART_TOPIC_SIZES.md` - Quick start guide
- ✅ `TOPIC_SIZE_IMPLEMENTATION.md` - Technical details
- ✅ `calculate-topic-sizes.sh` - Helper script
- ✅ `compare-topic-sizes.sh` - Comparison tool
- ✅ `examples/topic-sizes-example.sh` - Usage examples
- ✅ `release-notes-v1.3.0.md` - Release notes
- ✅ `CHANGELOG.md` - Changelog

### Modified Files
- ✅ `main.go` - Integrated topic sizes
- ✅ `README.md` - Updated documentation
- ✅ `Makefile` - Updated version and added release target
- ✅ `build.sh` - Updated default version

## Next Steps

### 1. Test the Release (Recommended)

```bash
# Test version
./bin/kmap-darwin-arm64 -version

# Test help
./bin/kmap-darwin-arm64 -h | grep topic-sizes

# Test on a local Kafka (if available)
./bin/kmap-darwin-arm64 -brokers localhost:9092 -topic-sizes
```

### 2. Commit and Tag

```bash
# Stage all release files
git add release-notes-v1.3.0.md CHANGELOG.md Makefile build.sh
git add topic_sizes.go TOPIC_SIZES.md QUICKSTART_TOPIC_SIZES.md
git add TOPIC_SIZE_IMPLEMENTATION.md *.sh examples/

# Commit
git commit -m "Release v1.3.0 - Topic Size Calculation Feature

- Add topic size calculation via DescribeLogDirs API
- Add topic filtering support
- Add JSON export for automation
- Add helper scripts and comprehensive documentation
- Update version to 1.3.0"

# Create annotated tag
git tag -a v1.3.0 -m "Release v1.3.0 - Topic Size Calculation

New Features:
- Topic size calculation with -topic-sizes flag
- Filter topics with -topic-list flag
- JSON export with -topic-sizes-output flag
- Helper scripts for common workflows
- Comprehensive documentation

See release-notes-v1.3.0.md for full details."

# Push to remote
git push origin main
git push origin v1.3.0
```

### 3. Create GitHub Release (Optional)

If you're using GitHub:

1. Go to: https://github.com/yourorg/kmap/releases/new
2. Tag: v1.3.0
3. Title: "kmap v1.3.0 - Topic Size Calculation"
4. Description: Copy from `release-notes-v1.3.0.md`
5. Upload binaries from `bin/`:
   - kmap-linux-amd64
   - kmap-linux-arm64
   - kmap-darwin-amd64
   - kmap-darwin-arm64
   - kmap-windows-amd64.exe

### 4. Announce the Release

Example announcement:

```
🎉 kmap v1.3.0 Released!

New feature: Calculate topic sizes directly!

Instead of complex kafka-log-dirs.sh pipelines, just:
  kmap -brokers kafka:9092 -topic-sizes

✨ Features:
  • Disk usage calculation for all topics
  • Filter specific topics
  • JSON export for automation
  • Human-readable output (GiB/TiB)
  • Summary statistics

📦 Download: https://github.com/yourorg/kmap/releases/tag/v1.3.0
📖 Docs: See TOPIC_SIZES.md

#kafka #devops #monitoring
```

## Verification Checklist

- [✅] All binaries built successfully
- [✅] Version is 1.3.0 in binaries
- [✅] Topic-sizes flag present in help
- [✅] Release notes created
- [✅] CHANGELOG updated
- [✅] Makefile version updated
- [✅] build.sh version updated
- [ ] Binaries tested on target platforms
- [ ] Git commit created
- [ ] Git tag created
- [ ] Changes pushed to remote
- [ ] GitHub release created (if applicable)
- [ ] Release announced

## Rollback Plan

If issues are discovered:

```bash
# Revert to previous version
git revert <commit-hash>
git push

# Delete tag locally and remotely
git tag -d v1.3.0
git push origin :refs/tags/v1.3.0

# Users can use previous release
Previous stable: v1.2.2
```

## Support

- Documentation: TOPIC_SIZES.md
- Quick Start: QUICKSTART_TOPIC_SIZES.md
- Examples: examples/topic-sizes-example.sh
- Issues: Create GitHub issue with logs and reproduction steps

---

**Release prepared by**: GitHub Copilot  
**Release date**: January 25, 2026  
**Status**: Ready for deployment ✅
