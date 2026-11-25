# Project conventions

## Size vs Len

- `Size()` returns **bytes**. For the memtable this is the sum of encoded key
  bytes (`key.Size()`, the `ToBytes` length) and value bytes. Byte sizes drive
  flush decisions.
- `Len()` returns the **number of entries**, like Go's built-in `len`.
- Parameters and fields measured in bytes carry the unit in their name, e.g.
  `maxMemtableBytes`.
