# Changelog

All notable changes to kmap will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-01-25

### Added
- **Topic Size Calculation** - Calculate actual disk space used by topics
  - New `-topic-sizes` flag to enable size calculation
  - New `-topic-sizes-output` flag to export JSON report
  - New `-topic-list` flag to filter specific topics
  - Human-readable size formatting (KiB, MiB, GiB, TiB)
  - Summary statistics with total cluster storage
- **Helper Scripts**
  - `calculate-topic-sizes.sh` - User-friendly wrapper
  - `compare-topic-sizes.sh` - Compare size reports over time
  - `examples/topic-sizes-example.sh` - Usage examples
- **Documentation**
  - `TOPIC_SIZES.md` - Comprehensive guide
  - `QUICKSTART_TOPIC_SIZES.md` - Quick reference
  - `TOPIC_SIZE_IMPLEMENTATION.md` - Technical details

### Changed
- Updated README.md with topic sizes feature documentation

### Performance
- Efficient parallel querying of all brokers via DescribeLogDirs API
- Optimized data aggregation across partitions

## [1.2.2] - Previous Release

See [release-notes-v1.2.2.md](release-notes-v1.2.2.md) for details.

## [1.2.1] - Previous Release

See [release-notes-v1.2.1.md](release-notes-v1.2.1.md) for details.

## [1.2.0] - Previous Release

See [release-notes-v1.2.0.md](release-notes-v1.2.0.md) for details.

---

[1.3.0]: https://github.com/yourorg/kmap/compare/v1.2.2...v1.3.0
