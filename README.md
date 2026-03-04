# CLI Music Rating App

## Installation
**Just install release for your os and cpu arch if you're not an advanced user**
`https://github.com/Blixza/CLI-Music-Rating/releases/tag/v1.0.0`

## Example JSON file
**Make sure your JSON is formatted exactly like this**
**Also you're NOT meant to edit the JSON file by yourself!**

```json
{
    "title": "Some Album Title",
    "authors": [
        "An Artist", "Some Other Artist"
    ],
    "rating": 0,
    "songs": [
        {
            "title": "Track",
            "rating": 0
        },
        {
            "title": "Another track",
            "rating": 0
        }
    ]
}
```

## Run
**Just run the binary or exe depending on your os**
**You also must provide path to the JSON file for rating**
**Also keep in mind that the app supports few language (English and Russian yet). Just add `--lang` flag**
**Example:** `./rymcli-linux-amd64 --json album1.json --lang en`

### Build from Source
```bash
go mod tidy
go build -o rymcli
```