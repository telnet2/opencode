# MemSh POSIX Compatibility Analysis

This document analyzes the POSIX compliance of commands implemented in go-memsh, comparing them against the POSIX.1-2017 specification.

## Executive Summary

**Overall Assessment**: The memsh implementation provides a **partially POSIX-compatible** shell with good coverage of common use cases. Most commands support the most frequently used flags and behaviors, though some advanced POSIX features are not implemented.

**Compliance Level**: ~75-80% POSIX compatible for implemented commands (improved from ~70%)

**Target Use Case**: The implementation is optimized for common scripting scenarios rather than full POSIX compliance.

## Recent POSIX Improvements (v0.2+)

The following enhancements have been made to improve POSIX compliance:

### v0.2 Core Improvements

✅ **`cd` command** (Compliance: 50% → 85%)
- Now supports `$HOME` environment variable for bare `cd`
- Added `cd -` to return to previous directory
- Sets `OLDPWD` environment variable

✅ **`echo` command** (Compliance: 70% → 95%)
- Added `-n` flag to suppress trailing newline

✅ **`env` command** (Compliance: 40% → 90%)
- Added command execution: `env VAR=value command args`
- Added `-i` / `--ignore-environment` flag
- Added `-u` / `--unset` flag for removing variables

✅ **`test` command** (Compliance: 60% → 75%)
- Added `-h` and `-L` for symbolic link tests
- Added `-b` for block special files
- Added `-c` for character special files
- Added `-p` for named pipes (FIFOs)
- Added `-S` for socket files

### v0.3 Quick-Win Flags

✅ **`ls` command** (Compliance: 40% → 45%)
- Added `-R` flag for recursive directory listing

✅ **`rm` command** (Compliance: 75% → 85%)
- Added `-i` flag for interactive confirmation

✅ **`cp` command** (Compliance: 50% → 65%)
- Added `-p` flag to preserve file attributes (permissions, timestamps)

✅ **`grep` command** (Compliance: 75% → 80%)
- Added `-q` flag for quiet mode (exit status only)

**Overall Impact**: These improvements bring the average compliance from ~70% to ~75-80% across all implemented commands.

---

## Shell Language Features

### ✅ SUPPORTED

- **Command execution**: Full support
- **Pipes** (`|`): Full support via mvdan/sh parser
- **Redirections**: `>`, `>>`, `<`, `2>&1` supported
- **Variable expansion**: `$VAR`, `${VAR}` supported
- **Control flow**: `if/then/else/fi`, `for` loops, `while` loops
- **Command substitution**: Supported via mvdan/sh
- **Arithmetic expansion**: Supported via mvdan/sh
- **Exit status**: `$?` supported

### ⚠️ PARTIAL SUPPORT

- **Quoting**: Single and double quotes supported, but escape sequences may have limitations
- **Here documents**: Supported by parser, not explicitly tested

### ❌ NOT IMPLEMENTED

- **Job control**: Background jobs (`&`), `fg`, `bg`, `jobs` commands
- **Aliases**: No alias support
- **Functions**: Not explicitly implemented
- **History expansion**: Not supported (no `!` history)
- **Command line editing**: Basic only (no vi/emacs mode)

---

## File Operations

### `pwd` - Print Working Directory

**POSIX Status**: ✅ **FULLY COMPLIANT**

- Correctly prints current working directory
- No options required by POSIX

**Compliance**: 100%

---

### `cd` - Change Directory

**POSIX Status**: ✅ **MOSTLY COMPLIANT** ⬆️ *Improved*

**Supported**:
- `cd [directory]` - Changes to specified directory
- ✅ **NEW**: `cd` with no arguments goes to `$HOME` (falls back to `/` if not set)
- ✅ **NEW**: `cd -` - Changes to previous directory and prints it
- ✅ **NEW**: Sets `OLDPWD` environment variable

**POSIX Deviations**:
- Does not support `$CDPATH`
- Does not support physical vs logical path resolution (`-L`, `-P`)

**Compliance**: ~85% ⬆️ *(improved from 50%)*

**Remaining Recommendations**: For full POSIX compliance:
- Add `$CDPATH` support for directory search path
- Add `-L` and `-P` flags for symbolic link handling

---

### `ls` - List Directory Contents

**POSIX Status**: ⚠️ **PARTIALLY COMPLIANT** ⬆️ *Improved*

**Supported**:
- `ls` - List current directory
- `ls [path...]` - List multiple paths
- `-a` - Show hidden files
- `-l` - Long format
- ✅ **NEW**: `-R` - Recursive directory listing

**POSIX Deviations**:
- Missing many standard flags: `-t` (sort by time), `-r` (reverse), `-S` (sort by size), `-1`, `-d`, `-i`, `-s`, `-u`
- Long format differs from POSIX: Shows Go's mode string format, not traditional format
- Does not show: number of links, owner, group (uses simplified format)
- No color support (not POSIX but common)

**Compliance**: ~45% ⬆️ *(improved from 40%)*

---

### `cat` - Concatenate Files

**POSIX Status**: ✅ **MOSTLY COMPLIANT**

**Supported**:
- `cat [file...]` - Concatenate files
- `cat` - Read from stdin
- Handles multiple files
- Proper error handling for directories

**POSIX Deviations**:
- Missing `-u` flag (unbuffered, rarely used)
- Does not support `-` for explicit stdin (minor issue)

**Compliance**: ~95%

---

### `mkdir` - Make Directory

**POSIX Status**: ⚠️ **PARTIALLY COMPLIANT**

**Supported**:
- `mkdir directory...` - Create directories
- `-p` - Create parent directories

**POSIX Deviations**:
- Missing `-m mode` flag for setting permissions
- Hardcoded to 0755 permissions
- Does not report which directories were created

**Compliance**: ~70%

---

### `rm` - Remove Files

**POSIX Status**: ✅ **MOSTLY COMPLIANT** ⬆️ *Improved*

**Supported**:
- `rm file...` - Remove files
- `-r`, `-R` - Recursive removal
- `-f` - Force (ignore non-existent files, no prompts)
- ✅ **NEW**: `-i` - Interactive prompt before removal

**POSIX Deviations**:
- Always uses `RemoveAll` when `-r` is specified (simpler but less granular)
- No protection against removing `.` or `..`

**Compliance**: ~85% ⬆️ *(improved from 75%)*

---

### `touch` - Change File Timestamps

**POSIX Status**: ⚠️ **PARTIALLY COMPLIANT**

**Supported**:
- `touch file...` - Create or update timestamp
- Creates file if doesn't exist
- Updates both access and modification time

**POSIX Deviations**:
- Missing `-a` (access time only)
- Missing `-m` (modification time only)
- Missing `-t` (specify time)
- Missing `-r` (use time from reference file)

**Compliance**: ~50%

---

### `cp` - Copy Files

**POSIX Status**: ⚠️ **PARTIALLY COMPLIANT** ⬆️ *Improved*

**Supported**:
- `cp source dest` - Copy file
- `cp source... directory` - Copy to directory
- `-r`, `-R` - Recursive copy
- ✅ **NEW**: `-p` - Preserve file attributes (permissions, timestamps)

**POSIX Deviations**:
- Missing `-i` (interactive prompt)
- Missing `-f` (force overwrite)
- Missing `-a` (archive mode)
- `-p` does not preserve ownership (not applicable in afero)

**Compliance**: ~65% ⬆️ *(improved from 50%)*

---

### `mv` - Move Files

**POSIX Status**: ⚠️ **PARTIALLY COMPLIANT**

**Supported**:
- `mv source dest` - Move/rename file
- Handles directory destination correctly

**POSIX Deviations**:
- Missing `-i` (interactive prompt)
- Missing `-f` (force overwrite)
- Does not support moving multiple files
- Syntax: `mv file1 file2 file3 directory` not supported

**Compliance**: ~60%

---

## Text Processing

### `echo` - Display Text

**POSIX Status**: ✅ **MOSTLY COMPLIANT** ⬆️ *Improved*

**Supported**:
- `echo [string...]` - Print arguments
- ✅ **NEW**: `-n` flag to suppress trailing newline

**POSIX Deviations**:
- No escape sequence processing (no `-e` support, though not POSIX)
- POSIX specifies implementation-defined behavior for backslash sequences
- Does not handle `\c` to suppress trailing newline (rare usage)

**Compliance**: ~95% ⬆️ *(improved from 70%)*

**Note**: POSIX echo behavior is notoriously underspecified. The `-n` flag addresses the most common use case.

---

### `grep` - Pattern Matching

**POSIX Status**: ✅ **MOSTLY COMPLIANT**

**Supported**:
- `grep pattern [file...]` - Search for pattern
- `-i` - Ignore case
- `-v` - Invert match
- `-n` - Show line numbers
- `-c` - Count matches
- ✅ **NEW**: `-q` - Quiet mode (exit status only, no output)
- Regular expressions (via Go regexp)
- stdin support

**POSIX Deviations**:
- Missing `-E` (extended regex, though Go regex is already extended)
- Missing `-F` (fixed string)
- Missing `-l` (files with matches)
- Missing `-s` (suppress errors)
- Missing `-x` (exact line match)

**Compliance**: ~80% ⬆️ *(improved from 75%)*

**Note**: Very good coverage of common use cases.

---

### `head` - Output File Beginning

**POSIX Status**: ✅ **FULLY COMPLIANT**

**Supported**:
- `head [file...]` - Show first 10 lines
- `-n count` - Specify line count
- `-count` - Shorthand (e.g., `-5`)
- Multiple files with headers
- stdin support

**POSIX Deviations**: None significant

**Compliance**: ~100%

---

### `tail` - Output File End

**POSIX Status**: ✅ **FULLY COMPLIANT**

**Supported**:
- `tail [file...]` - Show last 10 lines
- `-n count` - Specify line count
- `-count` - Shorthand
- Multiple files with headers
- stdin support

**POSIX Deviations**:
- Missing `-f` (follow mode, watches file)

**Compliance**: ~95%

**Note**: `-f` is not critical for in-memory filesystem use case.

---

### `wc` - Word Count

**POSIX Status**: ✅ **FULLY COMPLIANT**

**Supported**:
- `wc [file...]` - Count lines, words, bytes
- `-l` - Lines only
- `-w` - Words only
- `-c` - Bytes only
- Multiple files with totals
- stdin support

**POSIX Deviations**:
- Missing `-m` (character count, different from bytes for multibyte)

**Compliance**: ~95%

---

### `sort` - Sort Lines

**POSIX Status**: ⚠️ **PARTIALLY COMPLIANT**

**Supported**:
- `sort [file...]` - Sort lines
- `-r` - Reverse order
- `-u` - Unique (remove duplicates)
- `-n` - Numeric sort
- stdin support

**POSIX Deviations**:
- Missing many flags: `-b`, `-d`, `-f`, `-i`, `-k` (key fields), `-t` (delimiter)
- Missing `-o` (output file)
- No merge sort capability
- No field/column sorting

**Compliance**: ~50%

**Note**: Covers basic sorting well, but lacks advanced field-based sorting.

---

### `uniq` - Report Unique Lines

**POSIX Status**: ⚠️ **PARTIALLY COMPLIANT**

**Supported**:
- `uniq [file]` - Remove duplicate adjacent lines
- `-c` - Count occurrences

**POSIX Deviations**:
- Missing `-d` (only duplicates)
- Missing `-u` (only unique)
- Missing `-f` (skip fields)
- Missing `-s` (skip characters)

**Compliance**: ~50%

---

### `find` - Search Files

**POSIX Status**: ⚠️ **PARTIALLY COMPLIANT**

**Supported**:
- `find [path] [-name pattern] [-type f|d]`
- Glob pattern matching (`*`, `?`)
- Recursive directory traversal

**POSIX Deviations**:
- Missing MANY predicates: `-size`, `-user`, `-group`, `-perm`, `-mtime`, `-newer`, `-links`, `-empty`
- Missing operators: `-and`, `-or`, `-not`, `!`, `(`, `)`
- Missing actions: `-exec`, `-ok`, `-print0`, `-delete`, `-ls`
- No `-depth` or `-prune`
- Very simplified compared to POSIX find

**Compliance**: ~20%

**Note**: This is a minimal find implementation. For full POSIX find, significant work would be needed.

---

## Test/Conditional

### `test` / `[` - Evaluate Expression

**POSIX Status**: ⚠️ **PARTIALLY COMPLIANT** ⬆️ *Improved*

**Supported**:

**File tests**:
- `-e`, `-a` - File exists
- `-f` - Regular file
- `-d` - Directory
- `-r` - Readable (simplified)
- `-w` - Writable (simplified)
- `-x` - Executable
- `-s` - Non-empty file
- ✅ **NEW**: `-h`, `-L` - Symbolic link
- ✅ **NEW**: `-b` - Block special file
- ✅ **NEW**: `-c` - Character special file
- ✅ **NEW**: `-p` - Named pipe (FIFO)
- ✅ **NEW**: `-S` - Socket file

**String tests**:
- `-z` - Zero length
- `-n` - Non-zero length
- `=`, `==` - String equality
- `!=` - String inequality

**Numeric tests**:
- `-eq`, `-ne`, `-lt`, `-le`, `-gt`, `-ge`

**POSIX Deviations**:
- Missing file tests: `-g`, `-u`, `-G`, `-O`, `-N`
- Missing `-t` (file descriptor is terminal)
- Missing logical operators: `-a` (and), `-o` (or), `!` (not)
- Missing `\(` and `\)` for grouping
- Missing string comparison: `<`, `>` (lexicographic)
- Readable/writable checks are simplified (doesn't check actual permissions properly)

**Compliance**: ~75% ⬆️ *(improved from 60%)*

**Note**: Now covers most common file type tests. Logical operators remain unimplemented due to complexity.

---

## Environment Management

### `env` - Display Environment

**POSIX Status**: ✅ **MOSTLY COMPLIANT** ⬆️ *Improved*

**Supported**:
- `env` - Display exported variables
- ✅ **NEW**: `env VAR=value command [args...]` - Run command with modified environment
- ✅ **NEW**: `-i` / `--ignore-environment` - Start with empty environment
- ✅ **NEW**: `-u NAME` / `--unset NAME` - Remove variable from environment

**POSIX Deviations**:
- None significant

**Compliance**: ~90% ⬆️ *(improved from 40%)*

**Note**: Now implements full command execution functionality as per POSIX specification.

---

### `export` - Export Variables

**POSIX Status**: ✅ **MOSTLY COMPLIANT**

**Supported**:
- `export VAR=value` - Set and export
- `export VAR` - Mark existing as exported
- `export` (no args) - List exported

**POSIX Deviations**:
- Format of list output differs from POSIX
- Missing `-p` flag (though behavior matches)

**Compliance**: ~90%

---

### `unset` - Unset Variables

**POSIX Status**: ⚠️ **PARTIALLY COMPLIANT**

**Supported**:
- `unset VAR...` - Remove variables

**POSIX Deviations**:
- Missing `-f` (unset functions)
- Missing `-v` (unset variables, though this is default)
- POSIX allows unsetting readonly variables with special handling

**Compliance**: ~80%

---

### `set` - Set Variables (Non-standard)

**POSIX Status**: ❌ **NON-POSIX EXTENSION**

The `set` command in memsh is used for setting non-exported shell variables:
```bash
set VAR=value
```

In POSIX, `set` is used for setting shell options and positional parameters:
```bash
set -e          # Exit on error
set -x          # Print commands
set -- a b c    # Set positional parameters
```

**Note**: The memsh implementation of `set` does not match POSIX semantics at all. This is a significant deviation.

---

## Utilities

### `sleep` - Delay Execution

**POSIX Status**: ✅ **FULLY COMPLIANT**

**Supported**:
- `sleep seconds` - Sleep for specified seconds
- Context cancellation support

**POSIX Deviations**: None

**Compliance**: 100%

---

### `true` - Return Success

**POSIX Status**: ✅ **FULLY COMPLIANT**

**Supported**:
- `true` - Always returns 0

**Compliance**: 100%

---

### `false` - Return Failure

**POSIX Status**: ✅ **FULLY COMPLIANT**

**Supported**:
- `false` - Always returns 1

**Compliance**: 100%

---

### `exit` - Exit Shell

**POSIX Status**: ✅ **FULLY COMPLIANT**

**Supported**:
- `exit [n]` - Exit with status code

**Compliance**: 100%

---

## Non-POSIX Extensions

The following commands are **not part of POSIX** but are useful extensions:

### `curl` - HTTP Client ❌ NON-POSIX

A modern HTTP client implementation with common flags:
- `-X` (method), `-d` (data), `-H` (header)
- `-o` (output), `-s` (silent), `-i` (headers), `-L` (follow redirects)

**Note**: Commonly available on Unix systems but not in POSIX.

---

### `jq` - JSON Processor ❌ NON-POSIX

JSON query and manipulation tool:
- Full jq filter syntax via gojq library
- `-r` (raw output), `-c` (compact)

**Note**: Modern tool, not in POSIX.

---

### `import-file`, `import-dir`, `export-file`, `export-dir` ❌ NON-POSIX

Custom commands for moving data between host filesystem and in-memory filesystem.

**Note**: Specific to memsh's in-memory architecture.

---

## Summary of Compliance by Category

| Category | Commands | Avg. Compliance | Status | Notes |
|----------|----------|----------------|--------|-------|
| **Shell Language** | Core features | ~70% | Stable | Good coverage, missing job control |
| **File Operations** | pwd, cd, ls, cat, mkdir, rm, touch, cp, mv | ~70% ⬆️ | Improved | cd command now ~85% compliant |
| **Text Processing** | echo, grep, head, tail, wc, sort, uniq, find | ~75% ⬆️ | Improved | echo now ~95% compliant |
| **Test/Conditional** | test, [ | ~75% ⬆️ | Improved | Added 5 new file tests |
| **Environment** | env, export, unset, set | ~80% ⬆️ | Improved | env now ~90% compliant |
| **Utilities** | sleep, true, false, exit | ~100% | Stable | Fully compliant |
| **Extensions** | curl, jq, import/export | N/A | Stable | Useful additions |

**Overall Compliance**: ~75-80% ⬆️ *(improved from ~70%)*

---

## Recommendations for Improved POSIX Compliance

### ✅ Recently Completed

1. ✅ **`cd` command**: Added `$HOME` support and `cd -` for previous directory
2. ✅ **`echo` command**: Added `-n` flag to suppress trailing newline
3. ✅ **`env` command**: Added command execution with `-i` and `-u` flags
4. ✅ **`test` command**: Added `-h/-L`, `-b`, `-c`, `-p`, `-S` file tests
5. ✅ **Quick-win POSIX flags** (v0.3):
   - `ls -R` (recursive listing)
   - `rm -i` (interactive confirmation)
   - `cp -p` (preserve attributes)
   - `grep -q` (quiet mode)

### High Priority (Remaining)

1. **Fix `set` command** (Breaking Change):
   - Implement proper POSIX `set` for shell options (`set -e`, `set -x`, etc.)
   - Rename current `set VAR=value` syntax (non-POSIX) to `setvar` or deprecate

2. **Improve `test` command**:
   - Add logical operators (`-a`, `-o`, `!`)
   - Add grouping with `\(` and `\)`
   - Complex implementation - deferred

### Medium Priority

4. **Improve error messages**:
   - Match POSIX format more closely
   - Consistent error reporting

### Low Priority

7. **Add advanced features**:
   - `find` with `-exec`
   - `sort` with field-based sorting (`-k`)
   - Extended `ls` formatting

---

## Conclusion

**MemSh provides a pragmatic, mostly-POSIX-compatible shell environment** that:

✅ **Strengths**:
- Solid coverage of core shell features (pipes, redirections, control flow)
- Well-implemented common commands (cat, grep, head, tail, wc)
- Good for scripting common tasks
- Useful extensions (curl, jq) for modern workflows

⚠️ **Limitations**:
- Missing advanced flags on many commands
- `set` command is non-POSIX
- `cd` lacks $HOME and previous directory support
- Simplified permission handling (acceptable for in-memory FS)
- No job control

**Target Audience**: MemSh is well-suited for:
- Testing and automation scripts
- Embedded shell environments
- Learning shell scripting basics
- Situations where full POSIX compliance is not required

**Not Recommended For**:
- Replacing system shell (bash, sh)
- Scripts requiring strict POSIX compliance
- Advanced shell scripting with job control
- Production systems requiring full POSIX utilities

**Overall Grade**: **B** (Good, with room for improvement in POSIX compliance)
