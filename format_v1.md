## V1 Layer data

- `image id` is the HEX part of the checksum of the canonical config JSON.  JSON is produced from top layer's `v1Compatibility` (first item in the `history` array returned with v2 manifest) with `ChainID` and `history` added.  Generated using `MakeConfigFromV1Config` in the docker repo.
- `diff id` is the HEX part of the checksum of the layer's TAR file
- `blobsum` is the checksum of the layer's GZIP file (includes `layer.tar`, `json`, and `VERSION` files)

### Directory structure

```
repositories
manifest.json
<image id>.json
<diff id> -|
           |-- json
           |-- layer.tar
           |-- VERSION
<diff id> -|
           |-- json
           |-- layer.tar
           |-- VERSION
<diff id> -|
           |-- json
           |-- layer.tar
           |-- VERSION
```

### File: repositories

```
{"name":{"tag":"<image id>"}}
```

### File: manifest.json

```json
[{
	"Config": "<image id>.json",
	"RepoTags": ["name:tag"],
	"Layers": ["<diff id>/layer.tar", "<diff id>/layer.tar", "<diff id>/layer.tar"]
}]
```

### File: \<image id\>.json

Config file, contains data from which `image id` is calculated.

```json
{"architecture":"amd64","config":{...}..."os":"linux","rootfs":{"type":"layers","diff_ids":[...]}}
```

### File: json

The `json` is produced using an empty image config object with ID and parent ID (when exists) added.  Top layer is the only one that uses the contents of the corresponding `history.v1Compatibility` item.

```json
{"id":"<layer id>","parent":"<layer id>","created":"0001-01-01T00:00:00Z","container_config"
:{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":null,"Cmd":null,"Image":"","Volumes":null,
"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null}}
```

### File: VERSION

This is simply hardcoded in docker

```
1.0
```
