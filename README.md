# LSM Tree Key-Value Store

Simple Log-Structured Merge Tree (LSM Tree) key-value store written in Go. Also it exposes a REST API for basic CRUD operations and uses an in-memory memtable, bloom filter, and persistent SSTables.


## Try the Deployed Demo

You can try the deployed code here:

```sh
curl https://lsmtree.alhadad.me/your-key
curl -X PUT https://lsmtree.alhadad.me/your-key -d "very-secure-data"
```

## API Usage

- `GET /ping` — Health check
- `PUT /:key` — Set value (body = value)
- `GET /:key` — Get value
- `DELETE /:key` — Delete key

### Run Locally

```sh
go run main.go
```

Server runs at `http://localhost:8080`.

### Run with Docker

```sh
docker build -t lsm-tree .
docker run -p 8080:8080 lsm-tree
```

## Example

```sh
curl -X PUT localhost:8080/foo -d "bar"
curl localhost:8080/foo
curl -X DELETE localhost:8080/foo
```

## Next Steps

- Add SSTable compaction
