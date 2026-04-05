# Rubbish Rules

This directory stores text rule files used by the `rubbish` command.

## Files

- `rubbish_rules.txt`: default rubbish matching rules

## Rule Format

- One rule per line
- Empty lines are ignored
- Lines starting with `#` are comments
- Lines starting with `!` are exclude rules

## Examples

```txt
.DS_Store
*.tmp
cache/*.part
!important.tmp
!/System/*
```

## Usage

```bash
pikpakcli rubbish --rules rules/rubbish_rules.txt
pikpakcli rubbish --rules rules/rubbish_rules.txt --delete
```
