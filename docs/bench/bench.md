<script setup>
import { withBase } from 'vitepress'
</script>

# Benchmark

The benchmarks provided are meant to help development and offer a general sense of MangoSQL's performance. They are not intended to be exhaustive or compare every available method or library.

MangoSQL’s speed is not a primary objective, it's rather a byproduct of its design. By eliminating abstraction, reducing indirection, and directly interfacing with drivers, the system minimizes overhead.

## Postgres 

::: info

Library Tested:
* MangoSQL with PQ+SQLX driver
* MangoSQL with PGX
* [Gorm](https://gorm.io/) with PGX

Tested against a real postgres server `postgres:16` (tmpfs & fsync=off)

:::

### CPU (Operation per second)

::: tip Axis

**Horizontal:** Size of payload (number of items)

**Vertical:** Converted ns/op -> op/s, to get unit easier to visualize

:::

<iframe :src="withBase('/bench_postgres_insertmany_cpu.html')" width=576 height=320 frameBorder="0" scrolling="no" />

<iframe :src="withBase('/bench_postgres_findmany_cpu.html')" width=576 height=320 frameBorder="0" scrolling="no" />

### Memory Allocation

::: tip Axis

**Horizontal:** Size of payload (number of items)

**Vertical:** Go bench alloc/op

:::

<iframe :src="withBase('/bench_postgres_insertmany_alloc.html')" width=576 height=320 frameBorder="0" scrolling="no"/>

<iframe :src="withBase('/bench_postgres_findmany_alloc.html')" width=576 height=320 frameBorder="0" scrolling="no"/>

---

## SQLite

::: info

Library Tested:
* MangoSQL with modernc driver
* [Gorm](https://gorm.io/) with gorm sqlite driver

Tested in `:memory:` mode

:::

### CPU (Operation per second)

::: tip Axis

**Horizontal:** Size of payload (number of items)

**Vertical:** Converted ns/op -> op/s, to get unit easier to visualize

:::

<iframe :src="withBase('/bench_sqlite_insertmany_cpu.html')" width=576 height=320 frameBorder="0" scrolling="no" />

<iframe :src="withBase('/bench_sqlite_findmany_cpu.html')" width=576 height=320 frameBorder="0" scrolling="no" />

### Memory Allocation

::: tip Axis

**Horizontal:** Size of payload (number of items)

**Vertical:** Go bench alloc/op

:::

<iframe :src="withBase('/bench_sqlite_insertmany_alloc.html')" width=576 height=320 frameBorder="0" scrolling="no"/>

<iframe :src="withBase('/bench_sqlite_findmany_alloc.html')" width=576 height=320 frameBorder="0" scrolling="no"/>


---

## MariaDB

::: info

Library Tested:
* MangoSQL with go-sql-driver
* [Gorm](https://gorm.io/) with gorm sqlite driver

:::

### CPU (Operation per second)

::: tip Axis

**Horizontal:** Size of payload (number of items)

**Vertical:** Converted ns/op -> op/s, to get unit easier to visualize

:::

<iframe :src="withBase('/bench_mariadb_insertmany_cpu.html')" width=576 height=320 frameBorder="0" scrolling="no" />

<iframe :src="withBase('/bench_mariadb_findmany_cpu.html')" width=576 height=320 frameBorder="0" scrolling="no" />

### Memory Allocation

::: tip Axis

**Horizontal:** Size of payload (number of items)

**Vertical:** Go bench alloc/op

:::

<iframe :src="withBase('/bench_mariadb_insertmany_alloc.html')" width=576 height=320 frameBorder="0" scrolling="no"/>

<iframe :src="withBase('/bench_mariadb_findmany_alloc.html')" width=576 height=320 frameBorder="0" scrolling="no"/>
