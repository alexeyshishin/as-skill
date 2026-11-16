# README skeletons

Section sets per project type, the EN ↔ RU heading map, and one full skeleton pair. The order and the rules behind it are in `docs-conventions`; this file only says which sections a type usually needs.

## Detecting the project type

| signal | type |
|--------|------|
| `bin` in `package.json`, `[project.scripts]` in `pyproject.toml`, `cmd/` or `package main` in Go, `[[bin]]` in `Cargo.toml` | CLI |
| `Dockerfile`, `docker-compose*.yml`, `Chart.yaml`, an HTTP server entry point, `.env.example` | service |
| `exports`/`main` without `bin`, `[lib]`, a Go module without `main`, published to a registry | library |
| several manifests in subdirectories, `workspaces`, `go.work`, `[workspace]`, `pnpm-workspace.yaml` | monorepo |

A project can be two types (a library that ships a CLI): take the union, Usage shows the main one first.

## Sections per type

| section | library | CLI | service | monorepo root |
|---------|:-------:|:---:|:-------:|:-------------:|
| title + opening line | ✓ | ✓ | ✓ | ✓ |
| Why | ✓ | ✓ | ✓ | ✓ |
| Install | package manager command | install command + binary download | run with Docker / from source | — |
| Usage | minimal import + call + result | one command + its output | start command + one request + response | — |
| API overview | short, link to reference docs | — | endpoints table or link to OpenAPI | — |
| Configuration | — | flags table | environment variables table | — |
| Supported versions | language/runtime versions table | OS / arch table | — | — |
| Packages | — | — | — | table: package, one line, link |
| Documentation | ✓ if docs exist | ✓ if docs exist | ✓ if docs exist | ✓ |
| Contributing | if `CONTRIBUTING.md` exists | same | same | same |
| License | ✓ | ✓ | ✓ | ✓ |

## EN ↔ RU heading map

| EN | RU |
|----|----|
| Why | Зачем |
| Install / Installation | Установка |
| Quick start | Быстрый старт |
| Usage | Использование |
| Examples | Примеры |
| Configuration | Конфигурация |
| API | API |
| Supported versions | Поддерживаемые версии |
| Packages | Пакеты |
| Documentation | Документация |
| Contributing | Участие в разработке |
| License | Лицензия |
| Table of contents | Оглавление |

## Skeleton: CLI

`README.md`:

````markdown
# <name>

🇷🇺 [Русская версия](README_RUS.md)

<One sentence: what it does, for whom, searchable nouns first.>

## Why

<2–3 sentences: the problem and what it replaces.>

## Install

Requires <runtime> <version>+.

```bash
<one install command>
```

## Usage

```bash
<one command>
```

```text
<its real output>
```

| flag | default | description |
|------|---------|-------------|
| `--<flag>` | `<value>` | <what it does> |

## Documentation

- [<doc title>](<relative path>)

## License

[<SPDX>](LICENSE)
````

`README_RUS.md`:

````markdown
# <name>

🇬🇧 [English version](README.md)

<Одно предложение: что делает и для кого.>

## Зачем

<2–3 предложения.>

## Установка

Нужен <runtime> <version>+.

```bash
<та же команда>
```

## Использование

```bash
<та же команда>
```

```text
<тот же вывод>
```

| флаг | по умолчанию | описание |
|------|--------------|----------|
| `--<flag>` | `<value>` | <что делает> |

## Документация

- [<название>](<путь, _RUS-версия если есть>)

## Лицензия

[<SPDX>](LICENSE)
````
