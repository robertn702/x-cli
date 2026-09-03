---
name: x-cli
description: "Read X (Twitter) with the x CLI — tweets, profiles, timelines, threads, search, follow graphs, trends, and crawls into a local SQLite store you can query with SQL. Use this whenever a task touches X or Twitter data, even if the x CLI is not mentioned: a twitter.com or x.com link to read, a tweet or thread to fetch, a user's tweets or profile to pull, tweet search or counting topic activity over time, followers/likers/retweeters, trending topics, or archiving an account locally for SQL or RDF analysis. Strictly read-only; it never posts, likes, or follows."
---

# x CLI

`x` is one pure-Go binary that reads X (Twitter) over X's free public
surfaces and crawls accounts into a local SQLite store. Strictly read-only:
no command posts, likes, follows, or otherwise changes an account.

Full docs: <https://x-cli.tamnd.com> — command map in the
[CLI reference](https://x-cli.tamnd.com/reference/cli/).

## Health check first

```bash
x version      # is it installed, which build
x doctor       # probe all eight surfaces live; non-zero exit means something is down
```

Install if missing: `go install github.com/tamnd/x-cli/cmd/x@latest`, a
binary from the [releases](https://github.com/tamnd/x-cli/releases), or
`docker run --rm ghcr.io/tamnd/x:latest <command>`. Build from source with
`make build`.

## The one mental model: credential tiers

x picks the cheapest credential that can answer a read:

- **Tier 0 — no auth.** Public syndication/oEmbed. Serves `tweet`, `user`,
  `timeline` (recent window), `thread`, `replies`, `media`, `poll`, `embed`,
  `trends`, `places`, the graph reads (`edges`, `graph`, `rdf`), and `crawl`.
- **Tier 1 — `--guest`.** Guest GraphQL. X answers only five operations:
  the profile read (by handle or `--id`), the deeper `timeline` walk, the
  `tweet` read, and `space`. Everything else on that surface 404s.
- **Tier 2 — your session.** `x auth import --auth-token <t> --ct0 <c>`
  (cookies from your browser) unlocks what X reserves for logged-in
  clients: `search`, `counts`, `quotes`, `mentions`, `followers`/
  `following`, `likers`/`retweeters`, `likes`, `list`, `home`, `bookmarks`.

A read that needs a tier you have not enabled exits **4** and names the
tier. Fix: add `--guest` (only helps `user`, `timeline`, `space`), or
import a session once and re-run. Check with `x auth status`.

## Command map

| Question | Command | Needs |
| --- | --- | --- |
| What does this tweet say? | `x tweet <ref>` | nothing |
| What is this account? | `x user <handle>` | nothing |
| What did they post lately? | `x timeline <user>` | nothing (deeper with `--guest`) |
| What conversation is this tweet in? | `x thread <ref>` | nothing |
| Who replied? | `x replies <ref>` | nothing (whole tree with a session) |
| Pictures/video on this tweet or profile? | `x media <ref>` [`--download dir`] | nothing |
| Which tweets match a query? | `x search <query>` | session |
| When did a topic spike? | `x counts <query>` | session |
| Who quote-tweeted / mentioned them? | `x quotes <ref>` / `x mentions <user>` | session |
| Who follows / is followed? | `x followers <user>` / `x following <user>` | session |
| Who liked / retweeted? | `x likers <ref>` / `x retweeters <ref>` | session |
| My timeline, my bookmarks? | `x home` / `x bookmarks` | session |
| An audio Space? | `x space <ref>` | `--guest` |
| What is trending? | `x trends [place\|woeid]`, `x places <query>` | nothing |
| Anything a ref points at | `x get <ref>...` | depends |

A `<ref>` is a tweet id, a status URL, or anything x resolves to a tweet; a
`<user>` is a handle, or a numeric id with `--id`. Search takes X's own
query syntax (`from:`, `to:`, `#tag`, `filter:`, `since:`/`until:`, quoted
phrases).

## Core workflows

Search a topic and keep the results:

```bash
x search "webb telescope" -n 50 -o jsonl > hits.jsonl
x counts "webb telescope"            # per-day counts, no paging
```

Read a thread root-first, then its replies:

```bash
x thread 1903142823316049977
x replies 1903142823316049977
```

Pull a user's tweets, then keep a queryable local copy:

```bash
x timeline nasa --guest -n 200 -o jsonl
x crawl nasa --depth 1 --max 200    # breadth-first into the SQLite store
x db stats                          # what is in it
x query "select predicate, count(*) from edges group by 1 order by 2 desc"
x export --format nq > nasa.nq      # whole store as RDF; or: x export nasa ./out
```

`x query`/`x db query` and `x export` never touch the network: crawl once,
then answer from the store.

## Output shaping

- Piped output is JSONL by default; a terminal gets a readable list.
- `-o list|table|jsonl|json|csv|tsv|markdown|url|raw`, plus
  `--fields id,likes,text` and `--template '{{.username}}'`.
- `-n N` caps rows. Tweet and account ids are always strings, so nothing
  loses precision in `jq` or a spreadsheet.

## Exit codes and rate limits

- `4` needs-auth — add `--guest` or `x auth import`; the message names which.
- `5` rate-limited — x already retries 429s and paces at `--rate` (1s
  default); slow down (`--rate 3s`), keep the cache on, or wait out the window.
- `6` not-found — deleted, suspended, protected, or a bad id; the message
  says which.

Crawls cap spend with `--budget` (upstream requests) and `--max` (stored
nodes) and say on stderr when they stop early, so a partial crawl never
reads as a finished one. `timeline` is the account's own tweets only —
reply parents and reposts are deliberately excluded, so `x timeline jack`
never prints tweets @jack did not write.

## Prefer MCP over shelling out?

`x mcp` runs the same reads as an MCP stdio server (24 tools, one per read
command). It takes the global flags, so `x mcp --guest` serves at the guest
tier. `x serve` exposes the same reads over HTTP as NDJSON.

## Let x describe itself

`x tiers` (what each credential buys), `x routes`, `x surfaces`, `x fields
tweet`, and `x <command> --help` print tables from the binary itself —
prefer them over guessing when behavior seems to have shifted.
