# scs

`scs` (Shortest Common Superstring) is a utility for computing highly optimized, memory-efficient string dictionaries. It compresses lists of immutable strings by calculating an approximate shortest common superstring—eliminating exact duplicates, subsuming substrings, and overlapping identical prefixes and suffixes.

The output is a single contiguous block of text accompanied by a tightly packed binary footer mapping the offsets and lengths of the original strings.

## The Algorithm

The Shortest Common Superstring (SCS) problem asks for the shortest possible sequence of characters that contains every string in a given set as a contiguous substring.

To visualize the concept, imagine assembling a jigsaw puzzle where the pieces are strips of semi-transparent glass with words printed on them. If you have one strip reading `micro` and another reading `crobe`, you do not lay them end-to-end to spell `microcrobe`. Instead, you slide them over one another until the overlapping letters align perfectly, fusing the sequence into `microbe`.

Applied computationally at scale, `scs` first eliminates exact duplicates and strings entirely subsumed by others (for example, the word `rob` is completely swallowed by `microbe` and requires no separate sequence). It then calculates the optimal way to overlap the remaining prefixes and suffixes. Because finding the absolute shortest superstring is mathematically NP-hard, the tool employs a pipeline: exact dynamic programming for small clusters, deterministic greedy algorithms for fast assembly, and a time-bounded genetic algorithm to search for deeper, non-obvious overlaps.

## Motivation

When systems programs (written in C, C++, Rust, or Go) load a large list of words into memory, the raw character data is only a fraction of the total memory footprint. Each string typically requires a pointer and a length integer (e.g., 16 bytes per string on a 64-bit architecture), plus memory allocator padding. For millions of small strings, this structural overhead vastly exceeds the payload itself.

It is inaccurate to say `scs` merely "packs" strings together. Packing implies placing discrete items side-by-side in a compact format. Instead, `scs` overlaps and fuses strings at their boundaries, generating a single, unified byte sequence that is mathematically shorter than the sum of its parts.

The necessary metadata (lengths, offsets, and ordering) to extract the original words is delta-encoded and bit-packed into a highly compressed footer. Because the resulting `.scs` file inherently includes this mapping data, it functions as a cache-local, zero-allocation data structure when loaded directly into memory as a continuous blob or sidecar. It is specifically designed to optimize memory usage in read-heavy Unix programs (such as spellcheckers or dictionary tools) where massive string datasets are immutable.

## File Format

The `.scs` file consists of a fixed-size 12-byte header, a raw contiguous text payload, and a metadata footer. This structure is designed for instant memory mapping without sequential delimiter scanning.

```text
+---------------------------------------------------------------+
|                      Header (12 Bytes)                        |
+--------+--------+--------+--------+--------+------------------+
|   'S'  |   'C'  |   'S'  |  0x02  |  Sep   | Footer Offset &  |
|  (1B)  |  (1B)  |  (1B)  |  (1B)  |  (1B)  | IsOrdered (7B)   |
+--------+--------+--------+--------+--------+------------------+
|                                                               |
|                      Superstring Payload                      |
|                                                               |
|   (Raw, contiguous bytes of overlapping strings. Unpadded,    |
|    no delimiters, entirely determined by footer metadata.)    |
|                                                               |
+---------------------------------------------------------------+ <-- Footer Offset
|                                                               |
|                        Metadata Footer                        |
|                                                               |
|  - Ordered:   ULEB128 maximums + bitpacked length/offset arrays
|  - Unordered: Length groups + ULEB128 delta-encoded offsets   |
+---------------------------------------------------------------+

```

The 56-bit (7-byte) footer offset and flag field is stored in Little-Endian:

* **Bit 55:** `IsOrdered` flag (1 = ordered, 0 = unordered).
* **Bits 0-54:** Absolute byte offset of the footer.

## Usage

```text
scs <command> [options]

```

### build

Constructs an `.scs` file from a standard newline-separated text file. By default, it preserves the chronological order of the input strings using a fast deterministic greedy algorithm.

```text
scs build -i input.txt -o output.scs

```

* `--unordered`: Discards chronological ordering, grouping strings by length to delta-encode offsets for maximum metadata compression.
* `--ga-time <duration>`: Time budget for the genetic algorithm optimizer (e.g., `10m`, `2h`) to search for deeper overlaps.
* `-k, --min-overlap`: Minimum meaningful overlap threshold (default: 3).
* `--dp-limit`: Exact dynamic programming threshold for assembly.
* `-v, --verbose`: Enable progress updates.

### merge

Combines an update `.scs` file into a primary `.scs` file, eliminating duplicate substrings against the primary payload without requiring a full rebuild.

```text
scs merge --primary base.scs --update patch.scs -o merged.scs

```

### cat

Reconstructs and outputs the original text lines from the `.scs` archive to stdout.

```text
scs cat data.scs

```

### search

Performs an exact match query for a specific word within the archive.

```text
scs search "query" data.scs

```

## Experimental Results

Benchmarks across various word lists demonstrate consistent disk and memory reduction. The compression ratio is heavily dependent on the dataset's linguistic redundancy and the chosen optimization strategy (fast greedy vs. time-bounded genetic algorithm). Unordered sets compress significantly better than ordered ones due to optimal delta-encoding of offsets.

Ballpark figures for file size reduction (comparing the raw `.txt` input to the optimal `.scs` output):

* **~1,000 words:** ~0% reduction. The dataset is too small; the metadata footer overhead cancels out the overlap savings.
* **~10,000 words:** ~36% reduction (e.g., 79 KB raw text reduces to ~51 KB).
* **~100,000 words:** ~39% reduction (e.g., 834 KB raw text reduces to ~595 KB).
* **~1,000,000 words:** ~26% reduction (e.g., 8.5 MB raw text reduces to ~6.3 MB).
* **Massive real-world datasets (~14,000,000 words, e.g., rockyou):** ~16% reduction (e.g., 140 MB raw text reduces to ~117.5 MB).

### Memory Efficiency Note

It is critical to understand that the `.scs` file natively includes all positions and lengths necessary for extracting the strings. The percentages above reflect the strict on-disk footprint.

If a program were to load an 8.5 MB raw text file of 1,000,000 words into memory, it would inherently require parallel allocations for language-level pointers and string structures. This pointer overhead can add an additional 16 MB of RAM usage (16 bytes per string on a 64-bit architecture), bringing the total footprint to nearly 25 MB.

Therefore, an on-disk footprint reduction of 26% (yielding a 6.3 MB `.scs` file) translates to a significantly steeper reduction in runtime memory utilization, as the `scs` format completely bypasses native address bloat.
