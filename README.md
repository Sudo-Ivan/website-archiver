# Page Downloader

A tool for downloading web pages and their snapshots from the Wayback Machine.

## Features

- Downloads web pages and their resources
- Integrates with Internet Archive's Wayback Machine
- Supports recursive downloading with configurable depth
- Preserves page structure and converts links
- Creates timestamped output directories
- Handles both HTTP and HTTPS URLs
- Creates a ZIM file

## Usage

```bash
website-archiver [--zim|-z] [--all-snapshots|-as] [--snapshot|-s YYYYMMDDHHMMSS] <url1> [url2] [url3] ... [depth]
```

### Examples

Download a single page:
```bash
website-archiver https://example.com
```

Download with ZIM file creation:
```bash
website-archiver --zim https://example.com
```

Download all available snapshots:
```bash
website-archiver --all-snapshots https://example.com
```

Download a specific snapshot:
```bash
website-archiver --snapshot 20230101000000 https://example.com
```

## Dependencies

- wget
- ImageMagick (for ZIM file creation)
- zim-tools (for ZIM file creation)

## Prerequisites

- Go 1.24 or higher
- `wget` command-line tool
- `zimwriterfs` command-line tool (zim-tools)

## Output

The tool creates a directory named `downloads/<domain>_<timestamp>` containing the downloaded files. The timestamp format is `YYYYMMDD_HHMMSS`.

## Error Handling

- Invalid URLs are rejected
- Failed downloads trigger cleanup of partial downloads
- Wayback Machine integration failures fall back to direct downloads
- Invalid depth values are rejected

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 